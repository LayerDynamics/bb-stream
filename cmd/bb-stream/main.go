package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/ryanoboyle/bb-stream/internal/api"
	"github.com/ryanoboyle/bb-stream/internal/b2"
	"github.com/ryanoboyle/bb-stream/internal/config"
	"github.com/ryanoboyle/bb-stream/internal/sync"
	"github.com/ryanoboyle/bb-stream/internal/watch"
	"github.com/ryanoboyle/bb-stream/pkg/progress"
	"github.com/spf13/cobra"
)

// Version information
const (
	Version    = "0.1.0"
	APIVersion = 1
)

var rootCmd = &cobra.Command{
	Use:   "bb-stream",
	Short: "Backblaze B2 streaming file manager",
	Long: `bb-stream is a CLI tool for streaming files to and from Backblaze B2 cloud storage.

Features:
  - Upload and download files with streaming
  - Sync directories with B2 buckets
  - Watch directories for auto-upload
  - HTTP API for programmatic access
  - Live Read support for reading files while they upload`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip config init for config commands
		if cmd.Name() == "init" || cmd.Name() == "show" || cmd.Parent().Name() == "config" {
			return nil
		}
		return config.Init()
	},
}

// Version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("bb-stream version %s (API version %d)\n", Version, APIVersion)
	},
}

// Config commands
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize configuration interactively",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.Init(); err != nil {
			return err
		}

		reader := bufio.NewReader(os.Stdin)

		fmt.Print("Enter B2 Key ID: ")
		keyID, _ := reader.ReadString('\n')
		keyID = strings.TrimSpace(keyID)

		fmt.Print("Enter B2 Application Key: ")
		appKey, _ := reader.ReadString('\n')
		appKey = strings.TrimSpace(appKey)

		fmt.Print("Enter default bucket (optional): ")
		bucket, _ := reader.ReadString('\n')
		bucket = strings.TrimSpace(bucket)

		config.SetCredentials(keyID, appKey)
		if bucket != "" {
			config.SetDefaultBucket(bucket)
		}

		if err := config.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("Configuration saved to %s\n", config.GetConfigPath())
		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.Init(); err != nil {
			return err
		}

		cfg := config.Get()
		fmt.Printf("Config file: %s\n", config.GetConfigPath())
		fmt.Printf("Key ID: %s\n", maskKey(cfg.KeyID))
		fmt.Printf("Application Key: %s\n", maskKey(cfg.ApplicationKey))
		fmt.Printf("Default Bucket: %s\n", cfg.DefaultBucket)
		fmt.Printf("API Port: %d\n", cfg.APIPort)
		return nil
	},
}

// List command
var lsCmd = &cobra.Command{
	Use:   "ls [bucket] [path]",
	Short: "List buckets or files",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := b2.NewFromConfig(ctx)
		if err != nil {
			return err
		}

		if len(args) == 0 {
			// List buckets
			buckets, err := client.ListBucketInfo(ctx)
			if err != nil {
				return err
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tTYPE")
			for _, b := range buckets {
				fmt.Fprintf(w, "%s\t%s\n", b.Name, b.Type)
			}
			w.Flush()
		} else {
			// List files in bucket
			bucket := args[0]
			prefix := ""
			if len(args) > 1 {
				prefix = args[1]
			}

			objects, err := client.ListObjects(ctx, bucket, prefix)
			if err != nil {
				return err
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tSIZE\tMODIFIED")
			for _, obj := range objects {
				fmt.Fprintf(w, "%s\t%s\t%s\n",
					obj.Name,
					formatSize(obj.Size),
					time.Unix(obj.Timestamp, 0).Format(time.RFC3339))
			}
			w.Flush()
		}

		return nil
	},
}

// Upload command
var uploadCmd = &cobra.Command{
	Use:   "upload <file> <bucket/path>",
	Short: "Upload a file to B2",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		localFile := args[0]
		remotePath := args[1]

		// Parse bucket/path
		parts := strings.SplitN(remotePath, "/", 2)
		if len(parts) < 2 {
			return fmt.Errorf("remote path must be in format: bucket/path")
		}
		bucket, path := parts[0], parts[1]

		// Open file
		f, err := os.Open(localFile)
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}
		defer f.Close()

		info, err := f.Stat()
		if err != nil {
			return fmt.Errorf("failed to stat file: %w", err)
		}

		ctx := context.Background()
		client, err := b2.NewFromConfig(ctx)
		if err != nil {
			return err
		}

		// Progress callback using progress.Callback type
		var progressCb progress.Callback = func(transferred, total int64) {
			percent := float64(transferred) / float64(total) * 100
			fmt.Printf("\rUploading: %s / %s (%.1f%%)", formatSize(transferred), formatSize(total), percent)
		}

		fmt.Printf("Uploading %s to %s/%s\n", localFile, bucket, path)
		err = client.UploadWithProgress(ctx, bucket, path, f, info.Size(), progressCb)
		if err != nil {
			return err
		}

		fmt.Println("\nUpload complete!")
		return nil
	},
}

// Download command
var downloadCmd = &cobra.Command{
	Use:   "download <bucket/path> <file>",
	Short: "Download a file from B2",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		remotePath := args[0]
		localFile := args[1]

		// Parse bucket/path
		parts := strings.SplitN(remotePath, "/", 2)
		if len(parts) < 2 {
			return fmt.Errorf("remote path must be in format: bucket/path")
		}
		bucket, path := parts[0], parts[1]

		ctx := context.Background()
		client, err := b2.NewFromConfig(ctx)
		if err != nil {
			return err
		}

		// Get file info for size
		objInfo, err := client.GetObjectInfo(ctx, bucket, path)
		if err != nil {
			return fmt.Errorf("failed to get object info: %w", err)
		}

		// Create local file
		f, err := os.Create(localFile)
		if err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}
		defer f.Close()

		// Progress callback using progress.Callback type
		var progressCb progress.Callback = func(transferred, total int64) {
			percent := float64(transferred) / float64(total) * 100
			fmt.Printf("\rDownloading: %s / %s (%.1f%%)", formatSize(transferred), formatSize(total), percent)
		}

		fmt.Printf("Downloading %s/%s to %s\n", bucket, path, localFile)
		err = client.DownloadWithProgress(ctx, bucket, path, f, progressCb)
		if err != nil {
			return err
		}

		fmt.Printf("\nDownload complete! (%s)\n", formatSize(objInfo.Size))
		return nil
	},
}

// Remove command
var rmCmd = &cobra.Command{
	Use:   "rm <bucket/path>",
	Short: "Delete a file from B2",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		remotePath := args[0]

		// Parse bucket/path
		parts := strings.SplitN(remotePath, "/", 2)
		if len(parts) < 2 {
			return fmt.Errorf("remote path must be in format: bucket/path")
		}
		bucket, path := parts[0], parts[1]

		ctx := context.Background()
		client, err := b2.NewFromConfig(ctx)
		if err != nil {
			return err
		}

		force, _ := cmd.Flags().GetBool("force")
		if !force {
			fmt.Printf("Delete %s/%s? [y/N]: ", bucket, path)
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			if strings.ToLower(strings.TrimSpace(response)) != "y" {
				fmt.Println("Aborted")
				return nil
			}
		}

		if err := client.DeleteObject(ctx, bucket, path); err != nil {
			return err
		}

		fmt.Printf("Deleted %s/%s\n", bucket, path)
		return nil
	},
}

// Stream upload command
var streamUpCmd = &cobra.Command{
	Use:   "stream-up <bucket/path>",
	Short: "Stream stdin to B2",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		remotePath := args[0]

		// Parse bucket/path
		parts := strings.SplitN(remotePath, "/", 2)
		if len(parts) < 2 {
			return fmt.Errorf("remote path must be in format: bucket/path")
		}
		bucket, path := parts[0], parts[1]

		ctx := context.Background()
		client, err := b2.NewFromConfig(ctx)
		if err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "Streaming stdin to %s/%s...\n", bucket, path)
		if err := client.StreamUpload(ctx, bucket, path, os.Stdin, nil); err != nil {
			return err
		}

		fmt.Fprintln(os.Stderr, "Stream upload complete!")
		return nil
	},
}

// Stream download command
var streamDownCmd = &cobra.Command{
	Use:   "stream-down <bucket/path>",
	Short: "Stream B2 file to stdout",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		remotePath := args[0]

		// Parse bucket/path
		parts := strings.SplitN(remotePath, "/", 2)
		if len(parts) < 2 {
			return fmt.Errorf("remote path must be in format: bucket/path")
		}
		bucket, path := parts[0], parts[1]

		ctx := context.Background()
		client, err := b2.NewFromConfig(ctx)
		if err != nil {
			return err
		}

		return client.StreamDownload(ctx, bucket, path, os.Stdout, nil)
	},
}

// Sync command
var syncCmd = &cobra.Command{
	Use:   "sync <source> <dest>",
	Short: "Sync files between local and B2",
	Long: `Sync files between a local directory and a B2 bucket.

Examples:
  bb-stream sync ./local-folder mybucket/backup --to-remote
  bb-stream sync mybucket/backup ./local-folder --to-local`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		source := args[0]
		dest := args[1]

		toRemote, _ := cmd.Flags().GetBool("to-remote")
		toLocal, _ := cmd.Flags().GetBool("to-local")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		delete, _ := cmd.Flags().GetBool("delete")

		ctx := context.Background()
		client, err := b2.NewFromConfig(ctx)
		if err != nil {
			return err
		}

		opts := sync.DefaultSyncOptions()
		opts.DryRun = dryRun
		opts.Delete = delete
		opts.ProgressCallback = func(status sync.SyncStatus) {
			fmt.Printf("\r%s: %s", status.Phase, status.CurrentFile)
		}

		var localPath, bucketName, remotePath string

		if toRemote {
			opts.Direction = sync.ToRemote
			localPath = source
			parts := strings.SplitN(dest, "/", 2)
			bucketName = parts[0]
			if len(parts) > 1 {
				remotePath = parts[1]
			}
		} else if toLocal {
			opts.Direction = sync.ToLocal
			parts := strings.SplitN(source, "/", 2)
			bucketName = parts[0]
			if len(parts) > 1 {
				remotePath = parts[1]
			}
			localPath = dest
		} else {
			return fmt.Errorf("must specify --to-remote or --to-local")
		}

		syncer := sync.NewSyncer(client, opts)
		result, err := syncer.Sync(ctx, localPath, bucketName, remotePath)
		if err != nil {
			return err
		}

		fmt.Println()
		if dryRun {
			fmt.Println("Dry run - no changes made")
		}
		fmt.Printf("Uploaded: %d, Downloaded: %d, Deleted: %d, Skipped: %d\n",
			result.Uploaded, result.Downloaded, result.Deleted, result.Skipped)
		fmt.Printf("Duration: %s\n", result.Duration)

		if len(result.Errors) > 0 {
			fmt.Printf("Errors: %d\n", len(result.Errors))
			for _, err := range result.Errors {
				fmt.Printf("  - %v\n", err)
			}
		}

		return nil
	},
}

// Watch command
var watchCmd = &cobra.Command{
	Use:   "watch <local-path> <bucket/path>",
	Short: "Watch a directory and auto-upload changes",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		localPath := args[0]
		remotePath := args[1]

		// Parse bucket/path
		parts := strings.SplitN(remotePath, "/", 2)
		if len(parts) < 2 {
			return fmt.Errorf("remote path must be in format: bucket/path")
		}
		bucket, path := parts[0], parts[1]

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		client, err := b2.NewFromConfig(ctx)
		if err != nil {
			return err
		}

		absPath, _ := filepath.Abs(localPath)
		fmt.Printf("Watching %s for changes...\n", absPath)
		fmt.Printf("Auto-uploading to %s/%s\n", bucket, path)
		fmt.Println("Press Ctrl+C to stop")

		autoUploader, err := watch.NewAutoUploader(client, localPath, bucket, path, nil)
		if err != nil {
			return err
		}

		autoUploader.OnUpload = func(path string, err error) {
			if err != nil {
				fmt.Printf("[ERROR] %s: %v\n", path, err)
			} else {
				fmt.Printf("[UPLOADED] %s\n", path)
			}
		}

		if err := autoUploader.Start(ctx); err != nil {
			return err
		}

		// Wait for interrupt
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		fmt.Println("\nStopping watcher...")
		autoUploader.Stop()
		return nil
	},
}

// Serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP API server",
	RunE: func(cmd *cobra.Command, args []string) error {
		port, _ := cmd.Flags().GetInt("port")

		ctx := context.Background()
		client, err := b2.NewFromConfig(ctx)
		if err != nil {
			return err
		}

		server := api.NewServer(client, port)

		fmt.Printf("Starting API server on http://localhost:%d\n", port)
		fmt.Println("Press Ctrl+C to stop")

		// Handle shutdown
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			<-sigCh
			fmt.Println("\nShutting down...")
			_ = server.Shutdown(context.Background())
		}()

		return server.Start()
	},
}

func init() {
	// Version command
	rootCmd.AddCommand(versionCmd)

	// Config commands
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configShowCmd)
	rootCmd.AddCommand(configCmd)

	// File commands
	rootCmd.AddCommand(lsCmd)
	rootCmd.AddCommand(uploadCmd)
	rootCmd.AddCommand(downloadCmd)

	rmCmd.Flags().BoolP("force", "f", false, "Skip confirmation")
	rootCmd.AddCommand(rmCmd)

	// Stream commands
	rootCmd.AddCommand(streamUpCmd)
	rootCmd.AddCommand(streamDownCmd)

	// Sync command
	syncCmd.Flags().Bool("to-remote", false, "Sync local to B2")
	syncCmd.Flags().Bool("to-local", false, "Sync B2 to local")
	syncCmd.Flags().Bool("dry-run", false, "Show what would be synced without making changes")
	syncCmd.Flags().Bool("delete", false, "Delete files in destination that don't exist in source")
	rootCmd.AddCommand(syncCmd)

	// Watch command
	rootCmd.AddCommand(watchCmd)

	// Serve command
	serveCmd.Flags().IntP("port", "p", 8080, "Port to listen on")
	rootCmd.AddCommand(serveCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// Helper functions

func maskKey(key string) string {
	if len(key) < 8 {
		return "****"
	}
	return key[:4] + "..." + key[len(key)-4:]
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

