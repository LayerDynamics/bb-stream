package sync

import (
	"bytes"
	"io"
	"testing"
)

func TestDefaultSyncOptions(t *testing.T) {
	opts := DefaultSyncOptions()

	if opts == nil {
		t.Fatal("DefaultSyncOptions should not return nil")
	}

	if opts.Direction != ToRemote {
		t.Errorf("Expected Direction=ToRemote, got %d", opts.Direction)
	}

	if opts.DryRun != false {
		t.Error("Expected DryRun=false by default")
	}

	if opts.Delete != false {
		t.Error("Expected Delete=false by default")
	}

	if opts.Checksum != false {
		t.Error("Expected Checksum=false by default")
	}

	if opts.Concurrent != 4 {
		t.Errorf("Expected Concurrent=4, got %d", opts.Concurrent)
	}

	// Check default ignore patterns
	expectedPatterns := []string{".git", ".DS_Store", "node_modules", "__pycache__"}
	if len(opts.IgnorePatterns) != len(expectedPatterns) {
		t.Errorf("Expected %d ignore patterns, got %d", len(expectedPatterns), len(opts.IgnorePatterns))
	}

	for i, pattern := range expectedPatterns {
		if i < len(opts.IgnorePatterns) && opts.IgnorePatterns[i] != pattern {
			t.Errorf("Expected pattern %s at index %d, got %s", pattern, i, opts.IgnorePatterns[i])
		}
	}
}

func TestSyncDirection(t *testing.T) {
	tests := []struct {
		name      string
		direction Direction
		expected  string
	}{
		{"ToRemote", ToRemote, "ToRemote"},
		{"ToLocal", ToLocal, "ToLocal"},
		{"Bidirectional", Bidirectional, "Bidirectional"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify constants have expected values
			switch tt.direction {
			case ToRemote:
				if tt.direction != 0 {
					t.Errorf("ToRemote should be 0, got %d", tt.direction)
				}
			case ToLocal:
				if tt.direction != 1 {
					t.Errorf("ToLocal should be 1, got %d", tt.direction)
				}
			case Bidirectional:
				if tt.direction != 2 {
					t.Errorf("Bidirectional should be 2, got %d", tt.direction)
				}
			}
		})
	}
}

func TestSyncStatus(t *testing.T) {
	status := SyncStatus{
		Phase:            "Uploading",
		CurrentFile:      "test.txt",
		FilesTotal:       10,
		FilesCompleted:   5,
		BytesTotal:       1024,
		BytesTransferred: 512,
		Errors:           []string{"error1", "error2"},
	}

	if status.Phase != "Uploading" {
		t.Errorf("Expected Phase='Uploading', got '%s'", status.Phase)
	}

	if status.CurrentFile != "test.txt" {
		t.Errorf("Expected CurrentFile='test.txt', got '%s'", status.CurrentFile)
	}

	if status.FilesTotal != 10 {
		t.Errorf("Expected FilesTotal=10, got %d", status.FilesTotal)
	}

	if status.FilesCompleted != 5 {
		t.Errorf("Expected FilesCompleted=5, got %d", status.FilesCompleted)
	}

	if status.BytesTotal != 1024 {
		t.Errorf("Expected BytesTotal=1024, got %d", status.BytesTotal)
	}

	if status.BytesTransferred != 512 {
		t.Errorf("Expected BytesTransferred=512, got %d", status.BytesTransferred)
	}

	if len(status.Errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(status.Errors))
	}
}

func TestSyncResult(t *testing.T) {
	result := &SyncResult{
		Uploaded:   5,
		Downloaded: 3,
		Deleted:    2,
		Skipped:    10,
	}

	if result.Uploaded != 5 {
		t.Errorf("Expected Uploaded=5, got %d", result.Uploaded)
	}

	if result.Downloaded != 3 {
		t.Errorf("Expected Downloaded=3, got %d", result.Downloaded)
	}

	if result.Deleted != 2 {
		t.Errorf("Expected Deleted=2, got %d", result.Deleted)
	}

	if result.Skipped != 10 {
		t.Errorf("Expected Skipped=10, got %d", result.Skipped)
	}
}

func TestNewSyncer_NilOptions(t *testing.T) {
	// Should use defaults when nil options provided
	syncer := NewSyncer(nil, nil)

	if syncer == nil {
		t.Fatal("NewSyncer should not return nil")
	}

	if syncer.opts == nil {
		t.Fatal("Syncer.opts should not be nil")
	}

	// Should have default options
	if syncer.opts.Direction != ToRemote {
		t.Errorf("Expected default Direction=ToRemote")
	}
}

func TestNewSyncer_CustomOptions(t *testing.T) {
	opts := &SyncOptions{
		Direction:  ToLocal,
		DryRun:     true,
		Delete:     true,
		Concurrent: 8,
	}

	syncer := NewSyncer(nil, opts)

	if syncer.opts.Direction != ToLocal {
		t.Errorf("Expected Direction=ToLocal")
	}

	if syncer.opts.DryRun != true {
		t.Error("Expected DryRun=true")
	}

	if syncer.opts.Delete != true {
		t.Error("Expected Delete=true")
	}

	if syncer.opts.Concurrent != 8 {
		t.Errorf("Expected Concurrent=8, got %d", syncer.opts.Concurrent)
	}
}

func TestNewConcurrentSyncer(t *testing.T) {
	// Test with nil options
	cs := NewConcurrentSyncer(nil, nil)
	if cs.workers != 4 {
		t.Errorf("Expected default workers=4, got %d", cs.workers)
	}

	// Test with custom concurrent value
	opts := &SyncOptions{Concurrent: 10}
	cs = NewConcurrentSyncer(nil, opts)
	if cs.workers != 10 {
		t.Errorf("Expected workers=10, got %d", cs.workers)
	}

	// Test with zero concurrent (should use default)
	opts = &SyncOptions{Concurrent: 0}
	cs = NewConcurrentSyncer(nil, opts)
	if cs.workers != 4 {
		t.Errorf("Expected default workers=4 when Concurrent=0, got %d", cs.workers)
	}
}

func TestProgressWriter(t *testing.T) {
	var buf bytes.Buffer
	var calledBytes int64
	var calledTotal int64

	callback := func(written, total int64) {
		calledBytes = written
		calledTotal = total
	}

	pw := &ProgressWriter{
		writer:   &buf,
		total:    100,
		callback: callback,
	}

	// Write some data
	data := []byte("hello")
	n, err := pw.Write(data)

	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if n != 5 {
		t.Errorf("Expected to write 5 bytes, wrote %d", n)
	}

	if pw.written != 5 {
		t.Errorf("Expected written=5, got %d", pw.written)
	}

	// Check callback was called
	if calledBytes != 5 {
		t.Errorf("Expected callback with written=5, got %d", calledBytes)
	}

	if calledTotal != 100 {
		t.Errorf("Expected callback with total=100, got %d", calledTotal)
	}

	// Write more data
	data = []byte(" world")
	n, err = pw.Write(data)

	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if n != 6 {
		t.Errorf("Expected to write 6 bytes, wrote %d", n)
	}

	if pw.written != 11 {
		t.Errorf("Expected written=11, got %d", pw.written)
	}

	// Verify buffer contents
	if buf.String() != "hello world" {
		t.Errorf("Expected buffer='hello world', got '%s'", buf.String())
	}
}

func TestProgressWriter_NilCallback(t *testing.T) {
	var buf bytes.Buffer

	pw := &ProgressWriter{
		writer:   &buf,
		total:    100,
		callback: nil,
	}

	// Should not panic with nil callback
	data := []byte("test")
	n, err := pw.Write(data)

	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if n != 4 {
		t.Errorf("Expected to write 4 bytes, wrote %d", n)
	}
}

func TestSyncerReportStatus(t *testing.T) {
	var receivedStatus *SyncStatus

	opts := &SyncOptions{
		ProgressCallback: func(status SyncStatus) {
			receivedStatus = &status
		},
	}

	syncer := NewSyncer(nil, opts)

	status := SyncStatus{
		Phase:       "Testing",
		CurrentFile: "file.txt",
	}

	syncer.reportStatus(status)

	if receivedStatus == nil {
		t.Fatal("Progress callback should have been called")
	}

	if receivedStatus.Phase != "Testing" {
		t.Errorf("Expected Phase='Testing', got '%s'", receivedStatus.Phase)
	}

	if receivedStatus.CurrentFile != "file.txt" {
		t.Errorf("Expected CurrentFile='file.txt', got '%s'", receivedStatus.CurrentFile)
	}
}

func TestSyncerReportStatus_NilCallback(t *testing.T) {
	opts := &SyncOptions{
		ProgressCallback: nil,
	}

	syncer := NewSyncer(nil, opts)

	// Should not panic with nil callback
	status := SyncStatus{Phase: "Testing"}
	syncer.reportStatus(status)
}

// Mock writer that fails after N bytes
type failingWriter struct {
	failAfter int
	written   int
}

func (fw *failingWriter) Write(p []byte) (int, error) {
	if fw.written+len(p) > fw.failAfter {
		remaining := fw.failAfter - fw.written
		if remaining > 0 {
			fw.written += remaining
			return remaining, io.ErrShortWrite
		}
		return 0, io.ErrShortWrite
	}
	fw.written += len(p)
	return len(p), nil
}

func TestProgressWriter_PartialWrite(t *testing.T) {
	fw := &failingWriter{failAfter: 3}

	pw := &ProgressWriter{
		writer: fw,
		total:  10,
	}

	// Try to write more than will succeed
	data := []byte("hello")
	n, err := pw.Write(data)

	// Should return partial write
	if n != 3 {
		t.Errorf("Expected partial write of 3 bytes, got %d", n)
	}

	if err == nil {
		t.Error("Expected error for partial write")
	}
}

func TestSyncOptions_WithProgressCallback(t *testing.T) {
	callCount := 0

	opts := &SyncOptions{
		Direction: ToRemote,
		ProgressCallback: func(status SyncStatus) {
			callCount++
		},
	}

	// Simulate calling the callback
	opts.ProgressCallback(SyncStatus{Phase: "Test1"})
	opts.ProgressCallback(SyncStatus{Phase: "Test2"})

	if callCount != 2 {
		t.Errorf("Expected callback to be called 2 times, got %d", callCount)
	}
}

func TestSyncResultWithErrors(t *testing.T) {
	result := &SyncResult{
		Uploaded:   3,
		Downloaded: 0,
		Deleted:    0,
		Skipped:    5,
		Errors: []error{
			io.EOF,
			io.ErrShortWrite,
		},
	}

	if len(result.Errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(result.Errors))
	}

	if result.Errors[0] != io.EOF {
		t.Error("Expected first error to be io.EOF")
	}

	if result.Errors[1] != io.ErrShortWrite {
		t.Error("Expected second error to be io.ErrShortWrite")
	}
}
