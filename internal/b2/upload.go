package b2

import (
	"context"
	"fmt"
	"io"

	"github.com/Backblaze/blazer/b2"
	"github.com/ryanoboyle/bb-stream/pkg/progress"
)

// UploadOptions configures an upload operation
type UploadOptions struct {
	ContentType       string
	ConcurrentUploads int
	LiveRead          bool
	ProgressCallback  progress.Callback
}

// DefaultUploadOptions returns sensible defaults
func DefaultUploadOptions() *UploadOptions {
	return &UploadOptions{
		ContentType:       "application/octet-stream",
		ConcurrentUploads: 4,
		LiveRead:          false,
	}
}

// Upload uploads data from a reader to B2
func (c *Client) Upload(ctx context.Context, bucketName, objectName string, reader io.Reader, size int64, opts *UploadOptions) error {
	if opts == nil {
		opts = DefaultUploadOptions()
	}

	bucket, err := c.Bucket(ctx, bucketName)
	if err != nil {
		return err
	}

	obj := bucket.Object(objectName)

	// Create writer with attributes for content type
	writerOpts := []b2.WriterOption{}
	if opts.ContentType != "" {
		writerOpts = append(writerOpts, b2.WithAttrsOption(&b2.Attrs{
			ContentType: opts.ContentType,
		}))
	}

	writer := obj.NewWriter(ctx, writerOpts...)

	// Configure upload options
	if opts.ConcurrentUploads > 0 {
		writer.ConcurrentUploads = opts.ConcurrentUploads
	}

	// Wrap reader with progress tracking if callback provided
	var src io.Reader = reader
	if opts.ProgressCallback != nil && size > 0 {
		src = progress.NewReader(reader, size, opts.ProgressCallback)
	}

	// Copy data to writer
	_, err = io.Copy(writer, src)
	if err != nil {
		writer.Close() // Attempt to close on error
		return fmt.Errorf("failed to upload: %w", err)
	}

	// Close the writer to finalize the upload
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to finalize upload: %w", err)
	}

	return nil
}

// UploadReader is a simplified upload from an io.Reader
func (c *Client) UploadReader(ctx context.Context, bucketName, objectName string, reader io.Reader) error {
	return c.Upload(ctx, bucketName, objectName, reader, -1, nil)
}

// UploadWithProgress uploads with progress reporting
func (c *Client) UploadWithProgress(ctx context.Context, bucketName, objectName string, reader io.Reader, size int64, callback progress.Callback) error {
	opts := DefaultUploadOptions()
	opts.ProgressCallback = callback
	return c.Upload(ctx, bucketName, objectName, reader, size, opts)
}

// StreamUpload handles streaming uploads from stdin or other unbounded readers
func (c *Client) StreamUpload(ctx context.Context, bucketName, objectName string, reader io.Reader, opts *UploadOptions) error {
	if opts == nil {
		opts = DefaultUploadOptions()
	}

	bucket, err := c.Bucket(ctx, bucketName)
	if err != nil {
		return err
	}

	obj := bucket.Object(objectName)

	// Create writer with attributes for content type
	writerOpts := []b2.WriterOption{}
	if opts.ContentType != "" {
		writerOpts = append(writerOpts, b2.WithAttrsOption(&b2.Attrs{
			ContentType: opts.ContentType,
		}))
	}

	writer := obj.NewWriter(ctx, writerOpts...)

	// Configure for streaming - Blazer handles chunking automatically
	if opts.ConcurrentUploads > 0 {
		writer.ConcurrentUploads = opts.ConcurrentUploads
	}

	// For streaming, we don't know the size upfront
	// Blazer's writer handles this by buffering and using multipart upload
	_, err = io.Copy(writer, reader)
	if err != nil {
		writer.Close()
		return fmt.Errorf("failed to stream upload: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to finalize stream upload: %w", err)
	}

	return nil
}

// UploadResult contains information about a completed upload
type UploadResult struct {
	Name        string
	Size        int64
	ContentType string
}

// UploadWithResult uploads and returns information about the uploaded object
func (c *Client) UploadWithResult(ctx context.Context, bucketName, objectName string, reader io.Reader, size int64, opts *UploadOptions) (*UploadResult, error) {
	if opts == nil {
		opts = DefaultUploadOptions()
	}

	bucket, err := c.Bucket(ctx, bucketName)
	if err != nil {
		return nil, err
	}

	obj := bucket.Object(objectName)

	// Create writer with attributes for content type
	writerOpts := []b2.WriterOption{}
	if opts.ContentType != "" {
		writerOpts = append(writerOpts, b2.WithAttrsOption(&b2.Attrs{
			ContentType: opts.ContentType,
		}))
	}

	writer := obj.NewWriter(ctx, writerOpts...)

	if opts.ConcurrentUploads > 0 {
		writer.ConcurrentUploads = opts.ConcurrentUploads
	}

	var src io.Reader = reader
	if opts.ProgressCallback != nil && size > 0 {
		src = progress.NewReader(reader, size, opts.ProgressCallback)
	}

	written, err := io.Copy(writer, src)
	if err != nil {
		writer.Close()
		return nil, fmt.Errorf("failed to upload: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize upload: %w", err)
	}

	return &UploadResult{
		Name:        objectName,
		Size:        written,
		ContentType: opts.ContentType,
	}, nil
}

// GetUploadWriter returns a writer for manual upload control
func (c *Client) GetUploadWriter(ctx context.Context, bucketName, objectName string) (*b2.Writer, error) {
	bucket, err := c.Bucket(ctx, bucketName)
	if err != nil {
		return nil, err
	}

	obj := bucket.Object(objectName)
	return obj.NewWriter(ctx), nil
}
