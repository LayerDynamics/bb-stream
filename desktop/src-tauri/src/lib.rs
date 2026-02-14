use std::sync::atomic::{AtomicU16, AtomicBool, Ordering};
use std::sync::{Arc, Mutex};
use std::time::Duration;
use tauri::{Manager, Emitter, AppHandle};
use tauri::menu::{Menu, MenuItem, Submenu, PredefinedMenuItem};
use tauri_plugin_shell::ShellExt;
use tauri_plugin_shell::process::{CommandChild, CommandEvent};
use tokio::sync::mpsc;

// Backend status states
#[derive(Clone, serde::Serialize)]
#[serde(rename_all = "snake_case")]
enum BackendStatus {
    Starting,
    Healthy,
    Unhealthy,
    Crashed { error: String },
    Restarting,
}

// Global state for the sidecar process
struct AppState {
    sidecar: Mutex<Option<CommandChild>>,
    port: AtomicU16,
    is_healthy: AtomicBool,
    shutdown: AtomicBool,
}

impl AppState {
    fn new() -> Self {
        Self {
            sidecar: Mutex::new(None),
            port: AtomicU16::new(0),
            is_healthy: AtomicBool::new(false),
            shutdown: AtomicBool::new(false),
        }
    }
}

#[tauri::command]
fn get_api_port(state: tauri::State<Arc<AppState>>) -> u16 {
    state.port.load(Ordering::SeqCst)
}

#[tauri::command]
fn restart_backend(app: AppHandle, state: tauri::State<Arc<AppState>>) {
    let state = Arc::clone(&state);
    tauri::async_runtime::spawn(async move {
        restart_sidecar(&app, &state).await;
    });
}

// Find an available port
fn find_available_port() -> Option<u16> {
    // Try the default port first
    if portpicker::is_free(8765) {
        return Some(8765);
    }
    // Otherwise pick a random available port
    portpicker::pick_unused_port()
}

// Start the sidecar process
async fn start_sidecar(app: &AppHandle, state: &Arc<AppState>) -> Result<(), String> {
    // Find an available port
    let port = find_available_port().ok_or("No available ports")?;
    state.port.store(port, Ordering::SeqCst);

    log::info!("Starting BB Stream sidecar on port {}", port);

    // Emit starting status
    emit_backend_status(app, BackendStatus::Starting);

    let shell = app.shell();
    let sidecar_command = shell
        .sidecar("bb-stream")
        .map_err(|e| format!("Failed to create sidecar command: {}", e))?;

    let (mut rx, child) = sidecar_command
        .args(["serve", "--port", &port.to_string()])
        .spawn()
        .map_err(|e| format!("Failed to spawn sidecar: {}", e))?;

    // Store the child process
    {
        let mut guard = state.sidecar.lock().unwrap();
        *guard = Some(child);
    }

    // Create a channel for sidecar events
    let (tx, mut event_rx) = mpsc::channel::<String>(10);

    // Log sidecar output in background
    let app_handle = app.clone();
    let state_clone = Arc::clone(state);
    tauri::async_runtime::spawn(async move {
        while let Some(event) = rx.recv().await {
            match event {
                CommandEvent::Stdout(line) => {
                    let msg = String::from_utf8_lossy(&line);
                    log::info!("[bb-stream] {}", msg);
                }
                CommandEvent::Stderr(line) => {
                    let msg = String::from_utf8_lossy(&line);
                    log::warn!("[bb-stream] {}", msg);
                    let _ = tx.send(msg.to_string()).await;
                }
                CommandEvent::Error(err) => {
                    log::error!("[bb-stream] Error: {}", err);
                    let _ = tx.send(format!("Error: {}", err)).await;
                }
                CommandEvent::Terminated(status) => {
                    log::info!("[bb-stream] Terminated with status: {:?}", status);
                    state_clone.is_healthy.store(false, Ordering::SeqCst);

                    // If not shutting down, report crash
                    if !state_clone.shutdown.load(Ordering::SeqCst) {
                        let error = format!("Process exited with status: {:?}", status);
                        emit_backend_status(&app_handle, BackendStatus::Crashed { error });

                        // Auto-restart after a delay
                        tokio::time::sleep(Duration::from_secs(2)).await;
                        if !state_clone.shutdown.load(Ordering::SeqCst) {
                            restart_sidecar(&app_handle, &state_clone).await;
                        }
                    }
                    break;
                }
                _ => {}
            }
        }
    });

    // Start health check loop
    let app_handle = app.clone();
    let state_clone = Arc::clone(state);
    tauri::async_runtime::spawn(async move {
        let mut consecutive_failures = 0;

        // Wait for initial startup
        tokio::time::sleep(Duration::from_millis(500)).await;

        loop {
            if state_clone.shutdown.load(Ordering::SeqCst) {
                break;
            }

            let port = state_clone.port.load(Ordering::SeqCst);
            let health_url = format!("http://localhost:{}/health", port);

            match check_health(&health_url).await {
                Ok(()) => {
                    consecutive_failures = 0;
                    if !state_clone.is_healthy.swap(true, Ordering::SeqCst) {
                        // Transitioned from unhealthy to healthy
                        emit_backend_status(&app_handle, BackendStatus::Healthy);
                    }
                }
                Err(e) => {
                    consecutive_failures += 1;
                    log::warn!("Health check failed ({}): {}", consecutive_failures, e);

                    if consecutive_failures >= 3 {
                        state_clone.is_healthy.store(false, Ordering::SeqCst);
                        emit_backend_status(&app_handle, BackendStatus::Unhealthy);
                    }
                }
            }

            tokio::time::sleep(Duration::from_secs(5)).await;
        }
    });

    Ok(())
}

// Check health endpoint
async fn check_health(url: &str) -> Result<(), String> {
    let client = reqwest::Client::builder()
        .timeout(Duration::from_secs(2))
        .build()
        .map_err(|e| e.to_string())?;

    let resp = client.get(url).send().await.map_err(|e| e.to_string())?;

    if resp.status().is_success() {
        Ok(())
    } else {
        Err(format!("Health check returned status: {}", resp.status()))
    }
}

// Restart the sidecar
async fn restart_sidecar(app: &AppHandle, state: &Arc<AppState>) {
    log::info!("Restarting BB Stream sidecar...");
    emit_backend_status(app, BackendStatus::Restarting);

    // Kill existing process
    {
        let mut guard = state.sidecar.lock().unwrap();
        if let Some(child) = guard.take() {
            let _ = child.kill();
        }
    }

    // Wait a bit before restarting
    tokio::time::sleep(Duration::from_millis(500)).await;

    // Start new process
    if let Err(e) = start_sidecar(app, state).await {
        log::error!("Failed to restart sidecar: {}", e);
        emit_backend_status(app, BackendStatus::Crashed { error: e });
    }
}

// Emit backend status to frontend
fn emit_backend_status(app: &AppHandle, status: BackendStatus) {
    if let Some(window) = app.get_webview_window("main") {
        let _ = window.emit("backend-status", status);
    }
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_http::init())
        .manage(Arc::new(AppState::new()))
        .invoke_handler(tauri::generate_handler![get_api_port, restart_backend])
        .setup(|app| {
            // Setup logging in debug mode
            if cfg!(debug_assertions) {
                app.handle().plugin(
                    tauri_plugin_log::Builder::default()
                        .level(log::LevelFilter::Info)
                        .build(),
                )?;
            }

            // Create the application menu
            let app_menu = Submenu::with_items(
                app,
                "BB Stream",
                true,
                &[
                    &PredefinedMenuItem::about(app, Some("About BB Stream"), None)?,
                    &PredefinedMenuItem::separator(app)?,
                    &MenuItem::with_id(app, "preferences", "Preferences...", true, Some("CmdOrCtrl+,"))?,
                    &PredefinedMenuItem::separator(app)?,
                    &PredefinedMenuItem::services(app, None)?,
                    &PredefinedMenuItem::separator(app)?,
                    &PredefinedMenuItem::hide(app, None)?,
                    &PredefinedMenuItem::hide_others(app, None)?,
                    &PredefinedMenuItem::show_all(app, None)?,
                    &PredefinedMenuItem::separator(app)?,
                    &PredefinedMenuItem::quit(app, None)?,
                ],
            )?;

            let file_menu = Submenu::with_items(
                app,
                "File",
                true,
                &[
                    &MenuItem::with_id(app, "upload", "Upload Files...", true, Some("CmdOrCtrl+U"))?,
                    &MenuItem::with_id(app, "new_folder", "New Folder", true, Some("CmdOrCtrl+Shift+N"))?,
                    &PredefinedMenuItem::separator(app)?,
                    &MenuItem::with_id(app, "download", "Download Selected", true, Some("CmdOrCtrl+D"))?,
                    &MenuItem::with_id(app, "delete", "Delete Selected", true, Some("CmdOrCtrl+Backspace"))?,
                    &PredefinedMenuItem::separator(app)?,
                    &MenuItem::with_id(app, "refresh", "Refresh", true, Some("CmdOrCtrl+R"))?,
                    &PredefinedMenuItem::separator(app)?,
                    &PredefinedMenuItem::close_window(app, None)?,
                ],
            )?;

            let edit_menu = Submenu::with_items(
                app,
                "Edit",
                true,
                &[
                    &PredefinedMenuItem::undo(app, None)?,
                    &PredefinedMenuItem::redo(app, None)?,
                    &PredefinedMenuItem::separator(app)?,
                    &PredefinedMenuItem::cut(app, None)?,
                    &PredefinedMenuItem::copy(app, None)?,
                    &PredefinedMenuItem::paste(app, None)?,
                    &PredefinedMenuItem::select_all(app, None)?,
                    &PredefinedMenuItem::separator(app)?,
                    &MenuItem::with_id(app, "copy_url", "Copy URL", true, Some("CmdOrCtrl+Shift+C"))?,
                ],
            )?;

            let view_menu = Submenu::with_items(
                app,
                "View",
                true,
                &[
                    &MenuItem::with_id(app, "toggle_sidebar", "Toggle Sidebar", true, Some("CmdOrCtrl+\\"))?,
                    &PredefinedMenuItem::separator(app)?,
                    &PredefinedMenuItem::fullscreen(app, None)?,
                ],
            )?;

            let window_menu = Submenu::with_items(
                app,
                "Window",
                true,
                &[
                    &PredefinedMenuItem::minimize(app, None)?,
                    &PredefinedMenuItem::maximize(app, None)?,
                    &PredefinedMenuItem::separator(app)?,
                    &PredefinedMenuItem::close_window(app, None)?,
                ],
            )?;

            let help_menu = Submenu::with_items(
                app,
                "Help",
                true,
                &[
                    &MenuItem::with_id(app, "documentation", "Documentation", true, None::<&str>)?,
                    &MenuItem::with_id(app, "github", "GitHub Repository", true, None::<&str>)?,
                ],
            )?;

            let menu = Menu::with_items(
                app,
                &[&app_menu, &file_menu, &edit_menu, &view_menu, &window_menu, &help_menu],
            )?;

            app.set_menu(menu)?;

            // Start the sidecar
            let app_handle = app.handle().clone();
            let state: tauri::State<Arc<AppState>> = app.state();
            let state = Arc::clone(&state);

            tauri::async_runtime::spawn(async move {
                if let Err(e) = start_sidecar(&app_handle, &state).await {
                    log::error!("Failed to start sidecar: {}", e);
                    emit_backend_status(&app_handle, BackendStatus::Crashed { error: e });
                }
            });

            Ok(())
        })
        .on_menu_event(|app, event| {
            let id = event.id().as_ref();
            match id {
                "upload" => {
                    if let Some(window) = app.get_webview_window("main") {
                        let _ = window.emit("menu-upload", ());
                    }
                }
                "download" => {
                    if let Some(window) = app.get_webview_window("main") {
                        let _ = window.emit("menu-download", ());
                    }
                }
                "delete" => {
                    if let Some(window) = app.get_webview_window("main") {
                        let _ = window.emit("menu-delete", ());
                    }
                }
                "refresh" => {
                    if let Some(window) = app.get_webview_window("main") {
                        let _ = window.emit("menu-refresh", ());
                    }
                }
                "copy_url" => {
                    if let Some(window) = app.get_webview_window("main") {
                        let _ = window.emit("menu-copy-url", ());
                    }
                }
                "toggle_sidebar" => {
                    if let Some(window) = app.get_webview_window("main") {
                        let _ = window.emit("menu-toggle-sidebar", ());
                    }
                }
                "preferences" => {
                    if let Some(window) = app.get_webview_window("main") {
                        let _ = window.emit("menu-preferences", ());
                    }
                }
                "documentation" => {
                    let _ = open::that("https://github.com/ryanoboyle/bb-stream#readme");
                }
                "github" => {
                    let _ = open::that("https://github.com/ryanoboyle/bb-stream");
                }
                _ => {}
            }
        })
        .on_window_event(|window, event| {
            // Kill sidecar when window closes
            if let tauri::WindowEvent::CloseRequested { .. } = event {
                let state: tauri::State<Arc<AppState>> = window.state();
                state.shutdown.store(true, Ordering::SeqCst);
                let mut guard = state.sidecar.lock().unwrap();
                if let Some(child) = guard.take() {
                    let _ = child.kill();
                    log::info!("BB Stream sidecar stopped");
                }
                drop(guard);
            }
        })
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
