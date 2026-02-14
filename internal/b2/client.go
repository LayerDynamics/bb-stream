package b2

import (
	"context"
	"fmt"
	"sync"

	"github.com/Backblaze/blazer/b2"
	"github.com/ryanoboyle/bb-stream/internal/config"
)

// Client wraps the Blazer B2 client
type Client struct {
	client *b2.Client
	mu     sync.RWMutex
}

var (
	defaultClient *Client
	clientOnce    sync.Once
)

// New creates a new B2 client with the provided credentials
func New(ctx context.Context, keyID, appKey string) (*Client, error) {
	client, err := b2.NewClient(ctx, keyID, appKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create B2 client: %w", err)
	}

	return &Client{
		client: client,
	}, nil
}

// NewFromConfig creates a new B2 client using the stored configuration
func NewFromConfig(ctx context.Context) (*Client, error) {
	cfg := config.Get()
	if !config.IsConfigured() {
		return nil, fmt.Errorf("B2 credentials not configured. Run 'bb-stream config init' first")
	}

	return New(ctx, cfg.KeyID, cfg.ApplicationKey)
}

// GetDefault returns the default client (singleton)
func GetDefault(ctx context.Context) (*Client, error) {
	var err error
	clientOnce.Do(func() {
		defaultClient, err = NewFromConfig(ctx)
	})
	if err != nil {
		return nil, err
	}
	return defaultClient, nil
}

// ResetDefault resets the default client (useful for testing or credential changes)
func ResetDefault() {
	clientOnce = sync.Once{}
	defaultClient = nil
}

// Bucket returns a reference to a bucket by name
func (c *Client) Bucket(ctx context.Context, name string) (*b2.Bucket, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	buckets, err := c.client.ListBuckets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list buckets: %w", err)
	}

	for _, bucket := range buckets {
		if bucket.Name() == name {
			return bucket, nil
		}
	}

	return nil, fmt.Errorf("bucket %q not found", name)
}

// ListBuckets returns all buckets in the account
func (c *Client) ListBuckets(ctx context.Context) ([]*b2.Bucket, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.client.ListBuckets(ctx)
}

// BucketInfo contains information about a bucket
type BucketInfo struct {
	Name string
	Type string
}

// ListBucketInfo returns information about all buckets
func (c *Client) ListBucketInfo(ctx context.Context) ([]BucketInfo, error) {
	buckets, err := c.ListBuckets(ctx)
	if err != nil {
		return nil, err
	}

	info := make([]BucketInfo, len(buckets))
	for i, bucket := range buckets {
		// Get bucket attrs to get the type
		attrs, err := bucket.Attrs(ctx)
		bucketType := "unknown"
		if err == nil {
			bucketType = string(attrs.Type)
		}
		info[i] = BucketInfo{
			Name: bucket.Name(),
			Type: bucketType,
		}
	}

	return info, nil
}

// ObjectInfo contains information about an object in a bucket
type ObjectInfo struct {
	Name        string
	Size        int64
	ContentType string
	Timestamp   int64
}

// ListObjects lists objects in a bucket with an optional prefix
func (c *Client) ListObjects(ctx context.Context, bucketName, prefix string) ([]ObjectInfo, error) {
	bucket, err := c.Bucket(ctx, bucketName)
	if err != nil {
		return nil, err
	}

	var objects []ObjectInfo
	iter := bucket.List(ctx, b2.ListPrefix(prefix))

	for iter.Next() {
		obj := iter.Object()
		attrs, err := obj.Attrs(ctx)
		if err != nil {
			continue // Skip objects we can't get attrs for
		}
		objects = append(objects, ObjectInfo{
			Name:        obj.Name(),
			Size:        attrs.Size,
			ContentType: attrs.ContentType,
			Timestamp:   attrs.UploadTimestamp.Unix(),
		})
	}

	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}

	return objects, nil
}

// DeleteObject deletes an object from a bucket
// B2 requires deleting by file version, so we list versions and delete the latest
func (c *Client) DeleteObject(ctx context.Context, bucketName, objectName string) error {
	bucket, err := c.Bucket(ctx, bucketName)
	if err != nil {
		return err
	}

	// List file versions to get the file ID
	iter := bucket.List(ctx, b2.ListPrefix(objectName), b2.ListHidden())

	var deleted bool
	for iter.Next() {
		obj := iter.Object()
		if obj.Name() == objectName {
			if err := obj.Delete(ctx); err != nil {
				return fmt.Errorf("failed to delete %s: %w", objectName, err)
			}
			deleted = true
			// Delete all versions of this file
		}
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("failed to list file versions: %w", err)
	}

	if !deleted {
		return fmt.Errorf("file %s not found", objectName)
	}

	return nil
}

// GetClient returns the underlying Blazer client
func (c *Client) GetClient() *b2.Client {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.client
}
