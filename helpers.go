package s3mock

import (
	"bytes"
	"context"
	"testing"
)

// AssertObjectExists fails the test if the object doesn't exist.
func (m *Mock) AssertObjectExists(t *testing.T, bucket, key string) {
	t.Helper()
	_, err := m.HeadObject(context.Background(), bucket, key)
	if err != nil {
		t.Fatalf("expected object %q to exist in bucket %q, but got error: %v", key, bucket, err)
	}
}

// AssertObjectNotExists fails the test if the object exists.
func (m *Mock) AssertObjectNotExists(t *testing.T, bucket, key string) {
	t.Helper()
	_, err := m.HeadObject(context.Background(), bucket, key)
	if err == nil {
		t.Fatalf("expected object %q to not exist in bucket %q, but it does", key, bucket)
	}
}

// AssertBucketExists fails the test if the bucket doesn't exist.
func (m *Mock) AssertBucketExists(t *testing.T, bucket string) {
	t.Helper()
	m.mu.RLock()
	_, ok := m.buckets[bucket]
	m.mu.RUnlock()
	if !ok {
		t.Fatalf("expected bucket %q to exist, but it does not", bucket)
	}
}

// AssertBucketEmpty fails the test if the bucket has any objects.
func (m *Mock) AssertBucketEmpty(t *testing.T, bucket string) {
	t.Helper()
	m.mu.RLock()
	bkt, ok := m.buckets[bucket]
	if !ok {
		m.mu.RUnlock()
		t.Fatalf("bucket %q does not exist", bucket)
		return
	}
	count := len(bkt.objects)
	m.mu.RUnlock()
	if count > 0 {
		t.Fatalf("expected bucket %q to be empty, but it has %d objects", bucket, count)
	}
}

// AssertObjectContent fails the test if the object data doesn't match expected.
func (m *Mock) AssertObjectContent(t *testing.T, bucket, key string, expected []byte) {
	t.Helper()
	obj, err := m.GetObject(context.Background(), bucket, key)
	if err != nil {
		t.Fatalf("failed to get object %q from bucket %q: %v", key, bucket, err)
	}
	if !bytes.Equal(obj.Data, expected) {
		t.Fatalf("object %q content mismatch: got %d bytes, want %d bytes", key, len(obj.Data), len(expected))
	}
}

// ObjectCount returns the number of objects in a bucket. Returns 0 if bucket doesn't exist.
func (m *Mock) ObjectCount(bucket string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	bkt, ok := m.buckets[bucket]
	if !ok {
		return 0
	}
	return len(bkt.objects)
}

// Reset clears all buckets, objects, multipart uploads, and presigned URLs.
func (m *Mock) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.buckets = make(map[string]*bucket)
	m.multiparts = make(map[string]*multipartUpload)
	m.presigned = make(map[string]*presignedEntry)
}
