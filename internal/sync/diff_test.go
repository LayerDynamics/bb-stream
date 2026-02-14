package sync

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiff_NewFilesToUpload(t *testing.T) {
	local := []FileInfo{
		{Path: "file1.txt", Size: 100, ModTime: 1000},
		{Path: "file2.txt", Size: 200, ModTime: 2000},
	}
	remote := []FileInfo{}

	result := Diff(local, remote, nil)

	if len(result.ToUpload) != 2 {
		t.Errorf("Expected 2 files to upload, got %d", len(result.ToUpload))
	}
	if len(result.ToDownload) != 0 {
		t.Errorf("Expected 0 files to download, got %d", len(result.ToDownload))
	}
}

func TestDiff_NewFilesToDownload(t *testing.T) {
	local := []FileInfo{}
	remote := []FileInfo{
		{Path: "file1.txt", Size: 100, ModTime: 1000, IsRemote: true},
		{Path: "file2.txt", Size: 200, ModTime: 2000, IsRemote: true},
	}

	result := Diff(local, remote, nil)

	if len(result.ToUpload) != 0 {
		t.Errorf("Expected 0 files to upload, got %d", len(result.ToUpload))
	}
	if len(result.ToDownload) != 2 {
		t.Errorf("Expected 2 files to download, got %d", len(result.ToDownload))
	}
}

func TestDiff_UnchangedFiles(t *testing.T) {
	local := []FileInfo{
		{Path: "file1.txt", Size: 100, ModTime: 1000},
	}
	remote := []FileInfo{
		{Path: "file1.txt", Size: 100, ModTime: 1000, IsRemote: true},
	}

	result := Diff(local, remote, nil)

	if len(result.Unchanged) != 1 {
		t.Errorf("Expected 1 unchanged file, got %d", len(result.Unchanged))
	}
	if len(result.ToUpload) != 0 {
		t.Errorf("Expected 0 files to upload, got %d", len(result.ToUpload))
	}
}

func TestDiff_ModifiedFiles(t *testing.T) {
	local := []FileInfo{
		{Path: "file1.txt", Size: 150, ModTime: 2000}, // Larger and newer
	}
	remote := []FileInfo{
		{Path: "file1.txt", Size: 100, ModTime: 1000, IsRemote: true},
	}

	result := Diff(local, remote, nil)

	if len(result.ToUpload) != 1 {
		t.Errorf("Expected 1 file to upload (modified), got %d", len(result.ToUpload))
	}
}

func TestDiff_DeleteExtra(t *testing.T) {
	local := []FileInfo{
		{Path: "keep.txt", Size: 100, ModTime: 1000},
	}
	remote := []FileInfo{
		{Path: "keep.txt", Size: 100, ModTime: 1000, IsRemote: true},
		{Path: "delete.txt", Size: 50, ModTime: 500, IsRemote: true},
	}

	opts := &DiffOptions{DeleteExtra: true}
	result := Diff(local, remote, opts)

	if len(result.ToDelete) != 1 {
		t.Errorf("Expected 1 file to delete, got %d", len(result.ToDelete))
	}
	if result.ToDelete[0].Path != "delete.txt" {
		t.Errorf("Expected delete.txt to be deleted, got %s", result.ToDelete[0].Path)
	}
}

func TestDiff_IgnorePatterns(t *testing.T) {
	local := []FileInfo{
		{Path: "file.txt", Size: 100, ModTime: 1000},
		{Path: ".git/config", Size: 50, ModTime: 500},
		{Path: "node_modules/pkg/index.js", Size: 200, ModTime: 600},
	}
	remote := []FileInfo{}

	opts := &DiffOptions{
		IgnorePatterns: []string{".git", "node_modules"},
	}
	result := Diff(local, remote, opts)

	if len(result.ToUpload) != 1 {
		t.Errorf("Expected 1 file to upload (ignoring .git and node_modules), got %d", len(result.ToUpload))
	}
	if result.ToUpload[0].Path != "file.txt" {
		t.Errorf("Expected file.txt, got %s", result.ToUpload[0].Path)
	}
}

func TestFilesEqual_SameSize(t *testing.T) {
	local := FileInfo{Path: "file.txt", Size: 100, ModTime: 1000}
	remote := FileInfo{Path: "file.txt", Size: 100, ModTime: 1000}

	if !filesEqual(local, remote, false) {
		t.Error("Files with same size and time should be equal")
	}
}

func TestFilesEqual_DifferentSize(t *testing.T) {
	local := FileInfo{Path: "file.txt", Size: 100, ModTime: 1000}
	remote := FileInfo{Path: "file.txt", Size: 200, ModTime: 1000}

	if filesEqual(local, remote, false) {
		t.Error("Files with different sizes should not be equal")
	}
}

func TestFilesEqual_TimeTolerance(t *testing.T) {
	local := FileInfo{Path: "file.txt", Size: 100, ModTime: 1000}
	remote := FileInfo{Path: "file.txt", Size: 100, ModTime: 1001} // 1 second difference

	if !filesEqual(local, remote, false) {
		t.Error("Files with 1 second time difference should be equal (tolerance)")
	}
}

func TestFilesEqual_Checksum(t *testing.T) {
	local := FileInfo{Path: "file.txt", Size: 100, ModTime: 1000, SHA1: "abc123"}
	remote := FileInfo{Path: "file.txt", Size: 100, ModTime: 2000, SHA1: "abc123"}

	if !filesEqual(local, remote, true) {
		t.Error("Files with same checksum should be equal regardless of time")
	}

	remote.SHA1 = "different"
	if filesEqual(local, remote, true) {
		t.Error("Files with different checksums should not be equal")
	}
}

func TestShouldIgnore(t *testing.T) {
	patterns := []string{".git", "node_modules", "*.pyc"}

	tests := []struct {
		path     string
		expected bool
	}{
		{".git/config", true},
		{"node_modules/pkg/index.js", true},
		{"__pycache__/file.pyc", true},
		{"src/main.go", false},
		{"README.md", false},
	}

	for _, tt := range tests {
		result := shouldIgnore(tt.path, patterns)
		if result != tt.expected {
			t.Errorf("shouldIgnore(%s) = %v, expected %v", tt.path, result, tt.expected)
		}
	}
}

func TestScanLocalDir(t *testing.T) {
	// Create temp directory with test files
	tempDir, err := os.MkdirTemp("", "bb-stream-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("hello"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	subDir := filepath.Join(tempDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	subFile := filepath.Join(subDir, "nested.txt")
	if err := os.WriteFile(subFile, []byte("world"), 0644); err != nil {
		t.Fatalf("Failed to create nested file: %v", err)
	}

	// Scan directory
	files, err := ScanLocalDir(tempDir, false)
	if err != nil {
		t.Fatalf("ScanLocalDir failed: %v", err)
	}

	// Should have 3 entries: test.txt, subdir, subdir/nested.txt
	if len(files) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(files))
	}

	// Check paths are relative
	for _, f := range files {
		if filepath.IsAbs(f.Path) {
			t.Errorf("Path should be relative, got %s", f.Path)
		}
	}
}

func TestDiffSummary(t *testing.T) {
	result := &DiffResult{
		ToUpload:   []FileInfo{{Path: "a.txt", Size: 100}, {Path: "b.txt", Size: 200}},
		ToDownload: []FileInfo{{Path: "c.txt", Size: 50}},
		ToDelete:   []FileInfo{{Path: "d.txt", Size: 25}},
		Unchanged:  []FileInfo{{Path: "e.txt", Size: 75}},
	}

	summary := result.Summary()

	if summary.ToUploadCount != 2 {
		t.Errorf("Expected ToUploadCount=2, got %d", summary.ToUploadCount)
	}
	if summary.ToUploadSize != 300 {
		t.Errorf("Expected ToUploadSize=300, got %d", summary.ToUploadSize)
	}
	if summary.ToDownloadCount != 1 {
		t.Errorf("Expected ToDownloadCount=1, got %d", summary.ToDownloadCount)
	}
	if summary.ToDeleteCount != 1 {
		t.Errorf("Expected ToDeleteCount=1, got %d", summary.ToDeleteCount)
	}
	if summary.UnchangedCount != 1 {
		t.Errorf("Expected UnchangedCount=1, got %d", summary.UnchangedCount)
	}
}
