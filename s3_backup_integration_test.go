package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// InMemoryS3Client is an in-memory S3 implementation for integration testing
type InMemoryS3Client struct {
	mu      sync.RWMutex
	buckets map[string]map[string]*s3Object
}

type s3Object struct {
	data []byte
	etag string
}

// NewInMemoryS3Client creates a new in-memory S3 client
func NewInMemoryS3Client() *InMemoryS3Client {
	return &InMemoryS3Client{
		buckets: make(map[string]map[string]*s3Object),
	}
}

// PutObject stores an object in memory
func (c *InMemoryS3Client) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	if params.Bucket == nil || params.Key == nil {
		return nil, fmt.Errorf("bucket and key are required")
	}

	bucket := *params.Bucket
	key := *params.Key

	// Read body data
	data, err := io.ReadAll(params.Body)
	if err != nil {
		return nil, err
	}

	// Calculate ETag (MD5 hash)
	hash := md5.Sum(data)
	etag := hex.EncodeToString(hash[:])

	c.mu.Lock()
	defer c.mu.Unlock()

	// Create bucket if it doesn't exist
	if c.buckets[bucket] == nil {
		c.buckets[bucket] = make(map[string]*s3Object)
	}

	// Store object
	c.buckets[bucket][key] = &s3Object{
		data: data,
		etag: etag,
	}

	etagWithQuotes := fmt.Sprintf("\"%s\"", etag)
	return &s3.PutObjectOutput{
		ETag: &etagWithQuotes,
	}, nil
}

// GetObject retrieves an object from memory
func (c *InMemoryS3Client) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	if params.Bucket == nil || params.Key == nil {
		return nil, fmt.Errorf("bucket and key are required")
	}

	bucket := *params.Bucket
	key := *params.Key

	c.mu.RLock()
	defer c.mu.RUnlock()

	bucketData, exists := c.buckets[bucket]
	if !exists {
		return nil, &types.NoSuchBucket{
			Message: stringPtr("bucket does not exist"),
		}
	}

	obj, exists := bucketData[key]
	if !exists {
		return nil, &types.NotFound{
			Message: stringPtr("key does not exist"),
		}
	}

	// Return a copy of the data
	dataCopy := make([]byte, len(obj.data))
	copy(dataCopy, obj.data)

	etagWithQuotes := fmt.Sprintf("\"%s\"", obj.etag)
	return &s3.GetObjectOutput{
		Body: io.NopCloser(bytes.NewReader(dataCopy)),
		ETag: &etagWithQuotes,
	}, nil
}

// HeadObject retrieves object metadata
func (c *InMemoryS3Client) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	if params.Bucket == nil || params.Key == nil {
		return nil, fmt.Errorf("bucket and key are required")
	}

	bucket := *params.Bucket
	key := *params.Key

	c.mu.RLock()
	defer c.mu.RUnlock()

	bucketData, exists := c.buckets[bucket]
	if !exists {
		// For testing purposes, treat missing bucket same as missing object
		return nil, &types.NotFound{
			Message: stringPtr("key does not exist"),
		}
	}

	obj, exists := bucketData[key]
	if !exists {
		return nil, &types.NotFound{
			Message: stringPtr("key does not exist"),
		}
	}

	contentLength := int64(len(obj.data))
	etagWithQuotes := fmt.Sprintf("\"%s\"", obj.etag)
	return &s3.HeadObjectOutput{
		ContentLength: &contentLength,
		ETag:          &etagWithQuotes,
	}, nil
}

// ListObjectsV2 lists objects in a bucket
func (c *InMemoryS3Client) ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	if params.Bucket == nil {
		return nil, fmt.Errorf("bucket is required")
	}

	bucket := *params.Bucket

	c.mu.RLock()
	defer c.mu.RUnlock()

	bucketData, exists := c.buckets[bucket]
	if !exists {
		return nil, &types.NoSuchBucket{
			Message: stringPtr("bucket does not exist"),
		}
	}

	// Collect all objects
	var objects []types.Object
	for key, obj := range bucketData {
		keyCopy := key
		etagWithQuotes := fmt.Sprintf("\"%s\"", obj.etag)
		size := int64(len(obj.data))
		objects = append(objects, types.Object{
			Key:  &keyCopy,
			ETag: &etagWithQuotes,
			Size: &size,
		})
	}

	keyCount := int32(len(objects))
	return &s3.ListObjectsV2Output{
		Contents: objects,
		KeyCount: &keyCount,
	}, nil
}

// Helper methods for tests

// GetObjectCount returns number of objects in a bucket
func (c *InMemoryS3Client) GetObjectCount(bucket string) int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if bucketData, exists := c.buckets[bucket]; exists {
		return len(bucketData)
	}
	return 0
}

// GetObjectData retrieves object data directly
func (c *InMemoryS3Client) GetObjectData(bucket, key string) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	bucketData, exists := c.buckets[bucket]
	if !exists {
		return nil, fmt.Errorf("bucket does not exist")
	}

	obj, exists := bucketData[key]
	if !exists {
		return nil, fmt.Errorf("key does not exist")
	}

	// Return a copy
	dataCopy := make([]byte, len(obj.data))
	copy(dataCopy, obj.data)
	return dataCopy, nil
}

// Integration tests

func TestS3Backup_BackupDirectory_Integration(t *testing.T) {
	// Create test directory structure
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")
	testDir := filepath.Join(sourceDir, "2023 06 June 15 vacation")

	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create test files
	createTempTestFile(t, testDir, "photo1.jpg")
	createTempTestFile(t, testDir, "photo2.heic")

	// Create videos subdirectory
	videosDir := filepath.Join(testDir, "videos")
	if err := os.MkdirAll(videosDir, 0755); err != nil {
		t.Fatalf("Failed to create videos directory: %v", err)
	}
	createTempTestFile(t, videosDir, "video1.mov")

	// Create backup with in-memory client
	client := NewInMemoryS3Client()
	backup := &s3Backup{
		client:     client,
		extensions: NewExtensions(),
	}

	// Backup the directory
	bucket := "test-bucket"
	err := backup.backupDirectory(sourceDir, "2023 06 June 15 vacation", bucket)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify object was created in S3
	expectedKey := "2023 06 June 15 vacation (2 images, 1 videos).tar.gz"
	if client.GetObjectCount(bucket) != 1 {
		t.Errorf("Expected 1 object in bucket, got: %d", client.GetObjectCount(bucket))
	}

	// Verify we can retrieve the object
	data, err := client.GetObjectData(bucket, expectedKey)
	if err != nil {
		t.Errorf("Expected to retrieve object, got error: %v", err)
	}

	if len(data) == 0 {
		t.Error("Expected non-empty archive data")
	}
}

func TestS3Backup_BackupDirectory_Deduplication(t *testing.T) {
	// Create test directory
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")
	testDir := filepath.Join(sourceDir, "2023 06 June 15 vacation")

	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	createTempTestFile(t, testDir, "photo1.jpg")

	// Create backup with in-memory client
	client := NewInMemoryS3Client()
	backup := &s3Backup{
		client:     client,
		extensions: NewExtensions(),
	}

	bucket := "test-bucket"

	// First backup
	err := backup.backupDirectory(sourceDir, "2023 06 June 15 vacation", bucket)
	if err != nil {
		t.Fatalf("First backup failed: %v", err)
	}

	// Second backup (should be skipped due to matching hash)
	err = backup.backupDirectory(sourceDir, "2023 06 June 15 vacation", bucket)
	if err != nil {
		t.Errorf("Second backup should succeed (skipped), got error: %v", err)
	}

	// Should still have only 1 object
	if client.GetObjectCount(bucket) != 1 {
		t.Errorf("Expected 1 object after deduplication, got: %d", client.GetObjectCount(bucket))
	}
}
