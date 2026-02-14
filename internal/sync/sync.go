package sync

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ryanoboyle/bb-stream/internal/b2"
	"github.com/ryanoboyle/bb-stream/pkg/progress"
)

// validateRelativePath ensures a relative path from diff results is safe.
// It prevents path traversal attacks by verifying the joined path stays under basePath.
func validateRelativePath(basePath, relativePath string) (string, error) {
	// Clean and join
	joined := filepath.Join(basePath, relativePath)

	// Ensure the result is still under basePath
	absBase, err := filepath.Abs(basePath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve base path: %w", err)
	}
	absJoined, err := filepath.Abs(joined)
	if err != nil {
		return "", fmt.Errorf("failed to resolve joined path: %w", err)
	}

	// Must be under base path (with separator to avoid prefix matching issues)
	if !strings.HasPrefix(absJoined, absBase+string(filepath.Separator)) && absJoined != absBase {
		return "", fmt.Errorf("path escapes base directory: %s", relativePath)
	}

	return joined, nil
}

// Direction specifies the sync direction
type Direction int

const (
	ToRemote Direction = iota // Local → B2
	ToLocal                   // B2 → Local
	Bidirectional             // Both directions
)

// SyncOptions configures a sync operation
type SyncOptions struct {
	Direction       Direction
	DryRun          bool
	Delete          bool // Delete files in destination that don't exist in source
	Checksum        bool // Use checksum for comparison
	Concurrent      int  // Number of concurrent transfers
	IgnorePatterns  []string
	ProgressCallback func(status SyncStatus)
}

// SyncStatus represents the current sync progress
type SyncStatus struct {
	Phase           string
	CurrentFile     string
	FilesTotal      int
	FilesCompleted  int
	BytesTotal      int64
	BytesTransferred int64
	Errors          []string
}

// DefaultSyncOptions returns sensible defaults
func DefaultSyncOptions() *SyncOptions {
	return &SyncOptions{
		Direction:  ToRemote,
		DryRun:     false,
		Delete:     false,
		Checksum:   false,
		Concurrent: 4,
		IgnorePatterns: []string{
			".git",
			".DS_Store",
			"node_modules",
			"__pycache__",
		},
	}
}

// Syncer handles sync operations
type Syncer struct {
	client *b2.Client
	opts   *SyncOptions
}

// NewSyncer creates a new syncer
func NewSyncer(client *b2.Client, opts *SyncOptions) *Syncer {
	if opts == nil {
		opts = DefaultSyncOptions()
	}
	return &Syncer{
		client: client,
		opts:   opts,
	}
}

// SyncResult contains the results of a sync operation
type SyncResult struct {
	Uploaded   int
	Downloaded int
	Deleted    int
	Skipped    int
	Errors     []error
	Duration   time.Duration
}

// Sync performs a sync operation between local directory and B2 bucket
func (s *Syncer) Sync(ctx context.Context, localPath, bucketName, remotePath string) (*SyncResult, error) {
	startTime := time.Now()
	result := &SyncResult{}

	// Normalize paths
	localPath = filepath.Clean(localPath)
	remotePath = filepath.ToSlash(remotePath)
	if remotePath != "" && remotePath[len(remotePath)-1] != '/' {
		remotePath += "/"
	}
	if remotePath == "/" {
		remotePath = ""
	}

	// Report status
	s.reportStatus(SyncStatus{Phase: "Scanning local files"})

	// Scan local files
	localFiles, err := ScanLocalDir(localPath, s.opts.Checksum)
	if err != nil {
		return nil, fmt.Errorf("failed to scan local directory: %w", err)
	}

	// Report status
	s.reportStatus(SyncStatus{Phase: "Scanning remote files"})

	// Get remote files
	remoteObjects, err := s.client.ListObjects(ctx, bucketName, remotePath)
	if err != nil {
		return nil, fmt.Errorf("failed to list remote objects: %w", err)
	}

	// Convert remote objects to FileInfo
	remoteFiles := make([]FileInfo, len(remoteObjects))
	for i, obj := range remoteObjects {
		// Remove remote path prefix for comparison
		name := obj.Name
		if remotePath != "" && len(name) > len(remotePath) {
			name = name[len(remotePath):]
		}
		remoteFiles[i] = FileInfo{
			Path:     name,
			Size:     obj.Size,
			ModTime:  obj.Timestamp,
			IsRemote: true,
		}
	}

	// Calculate diff
	diffOpts := &DiffOptions{
		DeleteExtra:    s.opts.Delete,
		Checksum:       s.opts.Checksum,
		IgnorePatterns: s.opts.IgnorePatterns,
	}
	diff := Diff(localFiles, remoteFiles, diffOpts)
	summary := diff.Summary()

	// Report plan
	s.reportStatus(SyncStatus{
		Phase:      "Planning",
		FilesTotal: summary.ToUploadCount + summary.ToDownloadCount + summary.ToDeleteCount,
		BytesTotal: summary.ToUploadSize + summary.ToDownloadSize,
	})

	// Handle dry run
	if s.opts.DryRun {
		result.Uploaded = summary.ToUploadCount
		result.Downloaded = summary.ToDownloadCount
		result.Deleted = summary.ToDeleteCount
		result.Skipped = summary.UnchangedCount
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Perform uploads
	if s.opts.Direction == ToRemote || s.opts.Direction == Bidirectional {
		for _, file := range diff.ToUpload {
			select {
			case <-ctx.Done():
				return result, ctx.Err()
			default:
			}

			s.reportStatus(SyncStatus{
				Phase:       "Uploading",
				CurrentFile: file.Path,
			})

			localFilePath, err := validateRelativePath(localPath, file.Path)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("invalid path %s: %w", file.Path, err))
				continue
			}
			remoteFilePath := remotePath + file.Path

			err = s.uploadFile(ctx, localFilePath, bucketName, remoteFilePath)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("upload %s: %w", file.Path, err))
			} else {
				result.Uploaded++
			}
		}
	}

	// Perform downloads
	if s.opts.Direction == ToLocal || s.opts.Direction == Bidirectional {
		for _, file := range diff.ToDownload {
			select {
			case <-ctx.Done():
				return result, ctx.Err()
			default:
			}

			s.reportStatus(SyncStatus{
				Phase:       "Downloading",
				CurrentFile: file.Path,
			})

			localFilePath, err := validateRelativePath(localPath, file.Path)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("invalid path %s: %w", file.Path, err))
				continue
			}
			remoteFilePath := remotePath + file.Path

			err = s.downloadFile(ctx, bucketName, remoteFilePath, localFilePath)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("download %s: %w", file.Path, err))
			} else {
				result.Downloaded++
			}
		}
	}

	// Perform deletions
	if s.opts.Delete {
		for _, file := range diff.ToDelete {
			select {
			case <-ctx.Done():
				return result, ctx.Err()
			default:
			}

			s.reportStatus(SyncStatus{
				Phase:       "Deleting",
				CurrentFile: file.Path,
			})

			remoteFilePath := remotePath + file.Path
			err := s.client.DeleteObject(ctx, bucketName, remoteFilePath)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("delete %s: %w", file.Path, err))
			} else {
				result.Deleted++
			}
		}
	}

	result.Skipped = summary.UnchangedCount
	result.Duration = time.Since(startTime)

	return result, nil
}

// uploadFile uploads a single file
func (s *Syncer) uploadFile(ctx context.Context, localPath, bucketName, remotePath string) error {
	f, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}

	return s.client.Upload(ctx, bucketName, remotePath, f, info.Size(), nil)
}

// downloadFile downloads a single file
func (s *Syncer) downloadFile(ctx context.Context, bucketName, remotePath, localPath string) error {
	// Ensure directory exists
	dir := filepath.Dir(localPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	f, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return s.client.Download(ctx, bucketName, remotePath, f, nil)
}

// reportStatus calls the progress callback if set
func (s *Syncer) reportStatus(status SyncStatus) {
	if s.opts.ProgressCallback != nil {
		s.opts.ProgressCallback(status)
	}
}

// ConcurrentSyncer handles concurrent sync operations
type ConcurrentSyncer struct {
	*Syncer
	workers int
}

// NewConcurrentSyncer creates a syncer with concurrent workers
func NewConcurrentSyncer(client *b2.Client, opts *SyncOptions) *ConcurrentSyncer {
	workers := 4
	if opts != nil && opts.Concurrent > 0 {
		workers = opts.Concurrent
	}
	return &ConcurrentSyncer{
		Syncer:  NewSyncer(client, opts),
		workers: workers,
	}
}

// SyncConcurrent performs sync with concurrent transfers
func (cs *ConcurrentSyncer) SyncConcurrent(ctx context.Context, localPath, bucketName, remotePath string) (*SyncResult, error) {
	startTime := time.Now()
	result := &SyncResult{}

	// Normalize paths
	localPath = filepath.Clean(localPath)
	remotePath = filepath.ToSlash(remotePath)
	if remotePath != "" && remotePath[len(remotePath)-1] != '/' {
		remotePath += "/"
	}
	if remotePath == "/" {
		remotePath = ""
	}

	// Scan and diff
	localFiles, err := ScanLocalDir(localPath, cs.opts.Checksum)
	if err != nil {
		return nil, fmt.Errorf("failed to scan local directory: %w", err)
	}

	remoteObjects, err := cs.client.ListObjects(ctx, bucketName, remotePath)
	if err != nil {
		return nil, fmt.Errorf("failed to list remote objects: %w", err)
	}

	remoteFiles := make([]FileInfo, len(remoteObjects))
	for i, obj := range remoteObjects {
		name := obj.Name
		if remotePath != "" && len(name) > len(remotePath) {
			name = name[len(remotePath):]
		}
		remoteFiles[i] = FileInfo{
			Path:     name,
			Size:     obj.Size,
			ModTime:  obj.Timestamp,
			IsRemote: true,
		}
	}

	diffOpts := &DiffOptions{
		DeleteExtra:    cs.opts.Delete,
		Checksum:       cs.opts.Checksum,
		IgnorePatterns: cs.opts.IgnorePatterns,
	}
	diff := Diff(localFiles, remoteFiles, diffOpts)

	if cs.opts.DryRun {
		summary := diff.Summary()
		result.Uploaded = summary.ToUploadCount
		result.Downloaded = summary.ToDownloadCount
		result.Deleted = summary.ToDeleteCount
		result.Skipped = summary.UnchangedCount
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Thread-safe error collection
	var errorsMu sync.Mutex
	var errors []error

	// Process uploads concurrently
	if cs.opts.Direction == ToRemote || cs.opts.Direction == Bidirectional {
		var uploaded int64
		var wg sync.WaitGroup
		uploadCh := make(chan FileInfo, len(diff.ToUpload))
		for _, f := range diff.ToUpload {
			uploadCh <- f
		}
		close(uploadCh)

		for i := 0; i < cs.workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for file := range uploadCh {
					select {
					case <-ctx.Done():
						return
					default:
					}

					localFilePath, err := validateRelativePath(localPath, file.Path)
					if err != nil {
						errorsMu.Lock()
						errors = append(errors, fmt.Errorf("invalid path %s: %w", file.Path, err))
						errorsMu.Unlock()
						continue
					}
					remoteFilePath := remotePath + file.Path

					cs.reportStatus(SyncStatus{
						Phase:       "Uploading",
						CurrentFile: file.Path,
					})

					if err := cs.uploadFile(ctx, localFilePath, bucketName, remoteFilePath); err != nil {
						errorsMu.Lock()
						errors = append(errors, fmt.Errorf("upload %s: %w", file.Path, err))
						errorsMu.Unlock()
					} else {
						atomic.AddInt64(&uploaded, 1)
					}
				}
			}()
		}
		wg.Wait()
		result.Uploaded = int(atomic.LoadInt64(&uploaded))
	}

	// Process downloads concurrently
	if cs.opts.Direction == ToLocal || cs.opts.Direction == Bidirectional {
		var downloaded int64
		var wg sync.WaitGroup
		downloadCh := make(chan FileInfo, len(diff.ToDownload))
		for _, f := range diff.ToDownload {
			downloadCh <- f
		}
		close(downloadCh)

		for i := 0; i < cs.workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for file := range downloadCh {
					select {
					case <-ctx.Done():
						return
					default:
					}

					localFilePath, err := validateRelativePath(localPath, file.Path)
					if err != nil {
						errorsMu.Lock()
						errors = append(errors, fmt.Errorf("invalid path %s: %w", file.Path, err))
						errorsMu.Unlock()
						continue
					}
					remoteFilePath := remotePath + file.Path

					cs.reportStatus(SyncStatus{
						Phase:       "Downloading",
						CurrentFile: file.Path,
					})

					if err := cs.downloadFile(ctx, bucketName, remoteFilePath, localFilePath); err != nil {
						errorsMu.Lock()
						errors = append(errors, fmt.Errorf("download %s: %w", file.Path, err))
						errorsMu.Unlock()
					} else {
						atomic.AddInt64(&downloaded, 1)
					}
				}
			}()
		}
		wg.Wait()
		result.Downloaded = int(atomic.LoadInt64(&downloaded))
	}

	// Process deletions concurrently
	if cs.opts.Delete {
		var deleted int64
		var wg sync.WaitGroup
		deleteCh := make(chan FileInfo, len(diff.ToDelete))
		for _, f := range diff.ToDelete {
			deleteCh <- f
		}
		close(deleteCh)

		for i := 0; i < cs.workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for file := range deleteCh {
					select {
					case <-ctx.Done():
						return
					default:
					}

					cs.reportStatus(SyncStatus{
						Phase:       "Deleting",
						CurrentFile: file.Path,
					})

					remoteFilePath := remotePath + file.Path
					if err := cs.client.DeleteObject(ctx, bucketName, remoteFilePath); err != nil {
						errorsMu.Lock()
						errors = append(errors, fmt.Errorf("delete %s: %w", file.Path, err))
						errorsMu.Unlock()
					} else {
						atomic.AddInt64(&deleted, 1)
					}
				}
			}()
		}
		wg.Wait()
		result.Deleted = int(atomic.LoadInt64(&deleted))
	}

	result.Errors = errors
	result.Skipped = len(diff.Unchanged)
	result.Duration = time.Since(startTime)

	return result, nil
}

// ProgressWriter wraps an io.Writer with progress reporting
type ProgressWriter struct {
	writer   io.Writer
	total    int64
	written  int64
	callback progress.Callback
}

// Write implements io.Writer
func (pw *ProgressWriter) Write(p []byte) (int, error) {
	n, err := pw.writer.Write(p)
	pw.written += int64(n)
	if pw.callback != nil {
		pw.callback(pw.written, pw.total)
	}
	return n, err
}
