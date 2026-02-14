package sync

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// FileInfo represents a file for comparison
type FileInfo struct {
	Path      string
	Size      int64
	ModTime   int64
	SHA1      string
	IsDir     bool
	IsRemote  bool
}

// DiffResult contains the result of comparing two file sets
type DiffResult struct {
	ToUpload   []FileInfo // Files that need to be uploaded (local → remote)
	ToDownload []FileInfo // Files that need to be downloaded (remote → local)
	ToDelete   []FileInfo // Files that need to be deleted
	Unchanged  []FileInfo // Files that are the same
}

// DiffOptions configures the diff operation
type DiffOptions struct {
	DeleteExtra bool   // Delete files that exist only in destination
	Checksum    bool   // Use SHA1 checksum for comparison (slower but more accurate)
	IgnorePatterns []string // Patterns to ignore
}

// DefaultDiffOptions returns sensible defaults
func DefaultDiffOptions() *DiffOptions {
	return &DiffOptions{
		DeleteExtra: false,
		Checksum:    false,
		IgnorePatterns: []string{
			".git",
			".DS_Store",
			"node_modules",
			"__pycache__",
			"*.pyc",
			".env",
		},
	}
}

// Diff compares local and remote file lists
func Diff(local, remote []FileInfo, opts *DiffOptions) *DiffResult {
	if opts == nil {
		opts = DefaultDiffOptions()
	}

	result := &DiffResult{
		ToUpload:   []FileInfo{},
		ToDownload: []FileInfo{},
		ToDelete:   []FileInfo{},
		Unchanged:  []FileInfo{},
	}

	// Create maps for quick lookup
	localMap := make(map[string]FileInfo)
	remoteMap := make(map[string]FileInfo)

	for _, f := range local {
		if !shouldIgnore(f.Path, opts.IgnorePatterns) {
			localMap[f.Path] = f
		}
	}

	for _, f := range remote {
		if !shouldIgnore(f.Path, opts.IgnorePatterns) {
			remoteMap[f.Path] = f
		}
	}

	// Find files to upload (in local but not in remote, or different)
	for path, localFile := range localMap {
		if localFile.IsDir {
			continue // Skip directories
		}

		remoteFile, exists := remoteMap[path]
		if !exists {
			// File exists locally but not remotely - upload
			result.ToUpload = append(result.ToUpload, localFile)
		} else if !filesEqual(localFile, remoteFile, opts.Checksum) {
			// File exists in both but is different - upload
			result.ToUpload = append(result.ToUpload, localFile)
		} else {
			// Files are the same
			result.Unchanged = append(result.Unchanged, localFile)
		}
	}

	// Find files to download (in remote but not in local)
	for path, remoteFile := range remoteMap {
		if remoteFile.IsDir {
			continue
		}

		_, exists := localMap[path]
		if !exists {
			// File exists remotely but not locally - download
			result.ToDownload = append(result.ToDownload, remoteFile)
		}
	}

	// Find files to delete if DeleteExtra is enabled
	if opts.DeleteExtra {
		for path, remoteFile := range remoteMap {
			if remoteFile.IsDir {
				continue
			}
			if _, exists := localMap[path]; !exists {
				result.ToDelete = append(result.ToDelete, remoteFile)
			}
		}
	}

	return result
}

// filesEqual compares two files for equality
func filesEqual(local, remote FileInfo, useChecksum bool) bool {
	// Size must match
	if local.Size != remote.Size {
		return false
	}

	// If using checksum, compare SHA1
	if useChecksum && local.SHA1 != "" && remote.SHA1 != "" {
		return local.SHA1 == remote.SHA1
	}

	// Otherwise, compare by modification time
	// B2 returns timestamps in milliseconds, local files use Unix seconds
	// Truncate both to second precision for reliable comparison
	localSeconds := local.ModTime
	remoteSeconds := remote.ModTime

	// If remote timestamp appears to be in milliseconds (very large), convert to seconds
	if remoteSeconds > 1e12 {
		remoteSeconds = remoteSeconds / 1000
	}

	// Allow 1 second tolerance for filesystem timestamp differences
	diff := localSeconds - remoteSeconds
	if diff < 0 {
		diff = -diff
	}
	return diff <= 1
}

// shouldIgnore checks if a path should be ignored
func shouldIgnore(path string, patterns []string) bool {
	for _, pattern := range patterns {
		// Simple matching - check if pattern appears in path
		if strings.Contains(path, pattern) {
			return true
		}
		// Also check just the filename
		filename := filepath.Base(path)
		matched, _ := filepath.Match(pattern, filename)
		if matched {
			return true
		}
	}
	return false
}

// ScanLocalDir scans a local directory and returns file info
func ScanLocalDir(root string, computeChecksum bool) ([]FileInfo, error) {
	var files []FileInfo

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		// Skip root
		if relPath == "." {
			return nil
		}

		// Normalize path separators
		relPath = filepath.ToSlash(relPath)

		fileInfo := FileInfo{
			Path:     relPath,
			Size:     info.Size(),
			ModTime:  info.ModTime().Unix(),
			IsDir:    info.IsDir(),
			IsRemote: false,
		}

		// Compute SHA1 if requested and it's a file
		if computeChecksum && !info.IsDir() {
			sha1, err := computeSHA1(path)
			if err == nil {
				fileInfo.SHA1 = sha1
			}
		}

		files = append(files, fileInfo)
		return nil
	})

	return files, err
}

// computeSHA1 computes the SHA1 hash of a file
func computeSHA1(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// Summary returns a summary of the diff result
func (d *DiffResult) Summary() DiffSummary {
	return DiffSummary{
		ToUploadCount:   len(d.ToUpload),
		ToDownloadCount: len(d.ToDownload),
		ToDeleteCount:   len(d.ToDelete),
		UnchangedCount:  len(d.Unchanged),
		ToUploadSize:    sumSize(d.ToUpload),
		ToDownloadSize:  sumSize(d.ToDownload),
	}
}

// DiffSummary provides a quick overview of changes
type DiffSummary struct {
	ToUploadCount   int
	ToDownloadCount int
	ToDeleteCount   int
	UnchangedCount  int
	ToUploadSize    int64
	ToDownloadSize  int64
}

func sumSize(files []FileInfo) int64 {
	var total int64
	for _, f := range files {
		total += f.Size
	}
	return total
}
