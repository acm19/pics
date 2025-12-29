package pics

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

// testCtx is a shared context for all integration tests
var testCtx = context.Background()

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

func TestBackup_BackupDirectories(t *testing.T) {
	// Create test directory structure with multiple subdirectories
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")

	// Create first directory
	dir1 := filepath.Join(sourceDir, "2023 06 June 15 vacation")
	if err := os.MkdirAll(dir1, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	createTempTestFile(t, dir1, "photo1.jpg")
	createTempTestFile(t, dir1, "photo2.heic")

	// Create second directory
	dir2 := filepath.Join(sourceDir, "2023 12 December 25 christmas")
	if err := os.MkdirAll(dir2, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	createTempTestFile(t, dir2, "family.jpg")

	// Create videos in first directory
	videosDir := filepath.Join(dir1, "videos")
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

	// Backup all directories
	bucket := "test-bucket"
	err := backup.BackupDirectories(testCtx, sourceDir, bucket, 2, nil)

	if err != nil {
		t.Fatalf("BackupDirectories failed: %v", err)
	}

	// Verify both objects were created in S3
	if client.GetObjectCount(bucket) != 2 {
		t.Errorf("Expected 2 objects in bucket, got: %d", client.GetObjectCount(bucket))
	}

	// Verify specific keys exist
	expectedKey1 := "2023 06 June 15 vacation (2 images, 1 videos).tar.gz"
	expectedKey2 := "2023 12 December 25 christmas (1 images, 0 videos).tar.gz"

	if _, err := client.GetObjectData(bucket, expectedKey1); err != nil {
		t.Errorf("Expected to find %s in bucket", expectedKey1)
	}

	if _, err := client.GetObjectData(bucket, expectedKey2); err != nil {
		t.Errorf("Expected to find %s in bucket", expectedKey2)
	}
}

func TestBackup_RestoreDirectories(t *testing.T) {
	// Create backup with in-memory client
	client := NewInMemoryS3Client()
	backup := &s3Backup{
		client:     client,
		extensions: NewExtensions(),
	}

	bucket := "test-bucket"

	// Create and backup test directories
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")
	targetDir := filepath.Join(tmpDir, "restored")

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("Failed to create target directory: %v", err)
	}

	// Create test directory with files
	dir1 := filepath.Join(sourceDir, "2023 06 June 15 vacation")
	if err := os.MkdirAll(dir1, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	createTempTestFile(t, dir1, "photo1.jpg")
	createTempTestFile(t, dir1, "photo2.heic")

	// Backup the directory
	if err := backup.BackupDirectories(testCtx, sourceDir, bucket, 1, nil); err != nil {
		t.Fatalf("BackupDirectories failed: %v", err)
	}

	// Restore directories
	err := backup.RestoreDirectories(testCtx, bucket, targetDir, RestoreFilter{}, 1, nil)

	if err != nil {
		t.Fatalf("RestoreDirectories failed: %v", err)
	}

	// Verify restored directory exists
	restoredDir := filepath.Join(targetDir, "2023 06 June 15 vacation")
	if _, err := os.Stat(restoredDir); os.IsNotExist(err) {
		t.Errorf("Expected restored directory to exist at %s", restoredDir)
	}

	// Verify files were restored
	if _, err := os.Stat(filepath.Join(restoredDir, "photo1.jpg")); os.IsNotExist(err) {
		t.Error("Expected photo1.jpg to be restored")
	}

	if _, err := os.Stat(filepath.Join(restoredDir, "photo2.heic")); os.IsNotExist(err) {
		t.Error("Expected photo2.heic to be restored")
	}
}

func TestBackup_RestoreDirectories_WithFilter(t *testing.T) {
	// Create backup with in-memory client
	client := NewInMemoryS3Client()
	backup := &s3Backup{
		client:     client,
		extensions: NewExtensions(),
	}

	bucket := "test-bucket"
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")
	targetDir := filepath.Join(tmpDir, "restored")

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("Failed to create target directory: %v", err)
	}

	// Create directories from different dates
	dir1 := filepath.Join(sourceDir, "2023 06 June 15 vacation")
	dir2 := filepath.Join(sourceDir, "2023 12 December 25 christmas")
	dir3 := filepath.Join(sourceDir, "2024 01 January 01 newyear")

	for _, dir := range []string{dir1, dir2, dir3} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		createTempTestFile(t, dir, "photo.jpg")
	}

	// Backup all directories
	if err := backup.BackupDirectories(testCtx, sourceDir, bucket, 2, nil); err != nil {
		t.Fatalf("BackupDirectories failed: %v", err)
	}

	// Restore only 2023 directories
	filter := RestoreFilter{
		FromYear: 2023,
		ToYear:   2023,
	}
	err := backup.RestoreDirectories(testCtx, bucket, targetDir, filter, 1, nil)

	if err != nil {
		t.Fatalf("RestoreDirectories failed: %v", err)
	}

	// Verify only 2023 directories were restored
	if _, err := os.Stat(filepath.Join(targetDir, "2023 06 June 15 vacation")); os.IsNotExist(err) {
		t.Error("Expected 2023 06 June 15 vacation to be restored")
	}

	if _, err := os.Stat(filepath.Join(targetDir, "2023 12 December 25 christmas")); os.IsNotExist(err) {
		t.Error("Expected 2023 12 December 25 christmas to be restored")
	}

	if _, err := os.Stat(filepath.Join(targetDir, "2024 01 January 01 newyear")); !os.IsNotExist(err) {
		t.Error("Expected 2024 01 January 01 newyear NOT to be restored")
	}
}

func TestBackup_RoundTrip(t *testing.T) {
	// Full integration test: backup and restore
	client := NewInMemoryS3Client()
	backup := &s3Backup{
		client:     client,
		extensions: NewExtensions(),
	}

	bucket := "test-bucket"
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")
	targetDir := filepath.Join(tmpDir, "restored")

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("Failed to create target directory: %v", err)
	}

	// Create test directory with images and videos
	testDir := filepath.Join(sourceDir, "2023 06 June 15 vacation")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	createTempTestFile(t, testDir, "photo1.jpg")
	createTempTestFile(t, testDir, "photo2.heic")

	videosDir := filepath.Join(testDir, "videos")
	if err := os.MkdirAll(videosDir, 0755); err != nil {
		t.Fatalf("Failed to create videos directory: %v", err)
	}
	createTempTestFile(t, videosDir, "video1.mov")

	// Backup
	if err := backup.BackupDirectories(testCtx, sourceDir, bucket, 1, nil); err != nil {
		t.Fatalf("BackupDirectories failed: %v", err)
	}

	// Verify backup exists
	if client.GetObjectCount(bucket) != 1 {
		t.Fatalf("Expected 1 object in bucket, got: %d", client.GetObjectCount(bucket))
	}

	// Restore
	if err := backup.RestoreDirectories(testCtx, bucket, targetDir, RestoreFilter{}, 1, nil); err != nil {
		t.Fatalf("RestoreDirectories failed: %v", err)
	}

	// Verify all files were restored correctly
	restoredDir := filepath.Join(targetDir, "2023 06 June 15 vacation")
	restoredVideosDir := filepath.Join(restoredDir, "videos")

	files := []string{
		filepath.Join(restoredDir, "photo1.jpg"),
		filepath.Join(restoredDir, "photo2.heic"),
		filepath.Join(restoredVideosDir, "video1.mov"),
	}

	for _, file := range files {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("Expected file to be restored: %s", file)
		}
	}
}

func TestBackup_Deduplication(t *testing.T) {
	client := NewInMemoryS3Client()
	backup := &s3Backup{
		client:     client,
		extensions: NewExtensions(),
	}

	bucket := "test-bucket"
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")

	// Create test directory
	testDir := filepath.Join(sourceDir, "2023 06 June 15 vacation")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	createTempTestFile(t, testDir, "photo1.jpg")

	// First backup
	if err := backup.BackupDirectories(testCtx, sourceDir, bucket, 1, nil); err != nil {
		t.Fatalf("First backup failed: %v", err)
	}

	// Second backup (should skip due to matching hash)
	if err := backup.BackupDirectories(testCtx, sourceDir, bucket, 1, nil); err != nil {
		t.Fatalf("Second backup failed: %v", err)
	}

	// Should still have only 1 object
	if client.GetObjectCount(bucket) != 1 {
		t.Errorf("Expected 1 object after deduplication, got: %d", client.GetObjectCount(bucket))
	}
}
