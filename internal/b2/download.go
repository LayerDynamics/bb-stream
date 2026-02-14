package b2

import (
	"context"
	"fmt"
	"io"

	"github.com/Backblaze/blazer/b2"
	"github.com/ryanoboyle/bb-stream/pkg/progress"
)

// DownloadOptions configures a download operation
type DownloadOptions struct {
	ConcurrentDownloads int
	Range               *ByteRange
	ProgressCallback    progress.Callback
}

// ByteRange specifies a range of bytes to download
type ByteRange struct {
	Start int64
	End   int64
}

// DefaultDownloadOptions returns sensible defaults
func DefaultDownloadOptions() *DownloadOptions {
	return &DownloadOptions{
		ConcurrentDownloads: 4,
	}
}

// Download downloads an object to a writer
func (c *Client) Download(ctx context.Context, bucketName, objectName string, writer io.Writer, opts *DownloadOptions) error {
	if opts == nil {
		opts = DefaultDownloadOptions()
	}

	bucket, err := c.Bucket(ctx, bucketName)
	if err != nil {
		return err
	}

	obj := bucket.Object(objectName)

	// Get object attributes for size
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get object attributes: %w", err)
	}

	var reader *b2.Reader

	// Handle range requests using NewRangeReader
	if opts.Range != nil {
		length := opts.Range.End - opts.Range.Start
		if length <= 0 {
			length = attrs.Size - opts.Range.Start
		}
		reader = obj.NewRangeReader(ctx, opts.Range.Start, length)
	} else {
		reader = obj.NewReader(ctx)
	}
	defer reader.Close()

	// Configure download options
	if opts.ConcurrentDownloads > 0 {
		reader.ConcurrentDownloads = opts.ConcurrentDownloads
	}

	// Wrap writer with progress tracking if callback provided
	var dest io.Writer = writer
	if opts.ProgressCallback != nil {
		dest = progress.NewWriter(writer, attrs.Size, opts.ProgressCallback)
	}

	// Copy data from reader to writer
	_, err = io.Copy(dest, reader)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}

	return nil
}

// DownloadToWriter is a simplified download to an io.Writer
func (c *Client) DownloadToWriter(ctx context.Context, bucketName, objectName string, writer io.Writer) error {
	return c.Download(ctx, bucketName, objectName, writer, nil)
}

// DownloadWithProgress downloads with progress reporting
func (c *Client) DownloadWithProgress(ctx context.Context, bucketName, objectName string, writer io.Writer, callback progress.Callback) error {
	opts := DefaultDownloadOptions()
	opts.ProgressCallback = callback
	return c.Download(ctx, bucketName, objectName, writer, opts)
}

// DownloadRange downloads a specific byte range
func (c *Client) DownloadRange(ctx context.Context, bucketName, objectName string, writer io.Writer, start, end int64) error {
	opts := DefaultDownloadOptions()
	opts.Range = &ByteRange{Start: start, End: end}
	return c.Download(ctx, bucketName, objectName, writer, opts)
}

// StreamDownload handles streaming downloads to stdout or other destinations
func (c *Client) StreamDownload(ctx context.Context, bucketName, objectName string, writer io.Writer, opts *DownloadOptions) error {
	if opts == nil {
		opts = DefaultDownloadOptions()
	}

	bucket, err := c.Bucket(ctx, bucketName)
	if err != nil {
		return err
	}

	obj := bucket.Object(objectName)

	var reader *b2.Reader

	// Handle range requests using NewRangeReader
	if opts.Range != nil {
		// Get size if needed for calculating length
		attrs, err := obj.Attrs(ctx)
		if err != nil {
			return fmt.Errorf("failed to get object attributes: %w", err)
		}
		length := opts.Range.End - opts.Range.Start
		if length <= 0 {
			length = attrs.Size - opts.Range.Start
		}
		reader = obj.NewRangeReader(ctx, opts.Range.Start, length)
	} else {
		reader = obj.NewReader(ctx)
	}
	defer reader.Close()

	if opts.ConcurrentDownloads > 0 {
		reader.ConcurrentDownloads = opts.ConcurrentDownloads
	}

	_, err = io.Copy(writer, reader)
	if err != nil {
		return fmt.Errorf("failed to stream download: %w", err)
	}

	return nil
}

// GetDownloadReader returns a reader for manual download control
func (c *Client) GetDownloadReader(ctx context.Context, bucketName, objectName string) (*b2.Reader, error) {
	bucket, err := c.Bucket(ctx, bucketName)
	if err != nil {
		return nil, err
	}

	obj := bucket.Object(objectName)
	return obj.NewReader(ctx), nil
}

// GetObjectInfo returns information about an object
func (c *Client) GetObjectInfo(ctx context.Context, bucketName, objectName string) (*ObjectInfo, error) {
	bucket, err := c.Bucket(ctx, bucketName)
	if err != nil {
		return nil, err
	}

	obj := bucket.Object(objectName)
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get object attributes: %w", err)
	}

	return &ObjectInfo{
		Name:        objectName,
		Size:        attrs.Size,
		ContentType: attrs.ContentType,
		Timestamp:   attrs.UploadTimestamp.Unix(),
	}, nil
}

// ObjectExists checks if an object exists in a bucket
func (c *Client) ObjectExists(ctx context.Context, bucketName, objectName string) (bool, error) {
	_, err := c.GetObjectInfo(ctx, bucketName, objectName)
	if err != nil {
		// Check if it's a "not found" error
		return false, nil
	}
	return true, nil
}
