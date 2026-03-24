package s3mock

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"testing"
)

var ctx = context.Background()

func TestCreateBucket(t *testing.T) {
	m := New()

	if err := m.CreateBucket(ctx, "my-bucket"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Duplicate
	err := m.CreateBucket(ctx, "my-bucket")
	if !IsAlreadyExists(err) {
		t.Fatalf("expected AlreadyExists, got: %v", err)
	}

	// Empty name
	err = m.CreateBucket(ctx, "")
	if !IsInvalidArgument(err) {
		t.Fatalf("expected InvalidArgument, got: %v", err)
	}
}

func TestListBuckets(t *testing.T) {
	m := New()
	m.CreateBucket(ctx, "charlie")
	m.CreateBucket(ctx, "alpha")
	m.CreateBucket(ctx, "bravo")

	buckets := m.ListBuckets(ctx)
	if len(buckets) != 3 {
		t.Fatalf("expected 3 buckets, got %d", len(buckets))
	}
	if buckets[0].Name != "alpha" || buckets[1].Name != "bravo" || buckets[2].Name != "charlie" {
		t.Fatalf("buckets not sorted: %v", buckets)
	}
}

func TestDeleteBucket(t *testing.T) {
	m := New()
	m.CreateBucket(ctx, "b1")

	// Delete non-existent
	err := m.DeleteBucket(ctx, "nope")
	if !IsNotFound(err) {
		t.Fatalf("expected NotFound, got: %v", err)
	}

	// Put object, try delete non-empty
	m.PutObject(ctx, "b1", "file.txt", []byte("hello"), "text/plain", nil)
	err = m.DeleteBucket(ctx, "b1")
	if !IsBucketNotEmpty(err) {
		t.Fatalf("expected BucketNotEmpty, got: %v", err)
	}

	// Delete object, then bucket
	m.DeleteObject(ctx, "b1", "file.txt")
	if err := m.DeleteBucket(ctx, "b1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	buckets := m.ListBuckets(ctx)
	if len(buckets) != 0 {
		t.Fatalf("expected 0 buckets, got %d", len(buckets))
	}
}

func TestPutAndGetObject(t *testing.T) {
	m := New()
	m.CreateBucket(ctx, "b1")

	data := []byte("hello world")
	if err := m.PutObject(ctx, "b1", "greeting.txt", data, "text/plain", map[string]string{"author": "test"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	obj, err := m.GetObject(ctx, "b1", "greeting.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(obj.Data, data) {
		t.Fatalf("data mismatch")
	}
	if obj.ContentType != "text/plain" {
		t.Fatalf("content type mismatch: got %q", obj.ContentType)
	}
	if obj.Size != len(data) {
		t.Fatalf("size mismatch: got %d, want %d", obj.Size, len(data))
	}
	if obj.ETag == "" {
		t.Fatal("etag should not be empty")
	}
	if obj.Metadata["author"] != "test" {
		t.Fatalf("metadata mismatch: got %v", obj.Metadata)
	}

	// Deep copy: mutating returned data shouldn't affect stored data
	obj.Data[0] = 'X'
	obj2, _ := m.GetObject(ctx, "b1", "greeting.txt")
	if obj2.Data[0] != 'h' {
		t.Fatal("deep copy violation: stored data was mutated via returned slice")
	}
}

func TestGetNonExistent(t *testing.T) {
	m := New()
	m.CreateBucket(ctx, "b1")

	_, err := m.GetObject(ctx, "b1", "nope")
	if !IsNotFound(err) {
		t.Fatalf("expected NotFound, got: %v", err)
	}

	_, err = m.GetObject(ctx, "no-bucket", "nope")
	if !IsNotFound(err) {
		t.Fatalf("expected NotFound, got: %v", err)
	}
}

func TestDeleteObject(t *testing.T) {
	m := New()
	m.CreateBucket(ctx, "b1")
	m.PutObject(ctx, "b1", "f.txt", []byte("data"), "text/plain", nil)

	if err := m.DeleteObject(ctx, "b1", "f.txt"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err := m.GetObject(ctx, "b1", "f.txt")
	if !IsNotFound(err) {
		t.Fatalf("expected NotFound after delete, got: %v", err)
	}

	// Delete non-existent
	err = m.DeleteObject(ctx, "b1", "f.txt")
	if !IsNotFound(err) {
		t.Fatalf("expected NotFound, got: %v", err)
	}
}

func TestHeadObject(t *testing.T) {
	m := New()
	m.CreateBucket(ctx, "b1")
	m.PutObject(ctx, "b1", "img.png", []byte{0x89, 0x50, 0x4E, 0x47}, "image/png", nil)

	info, err := m.HeadObject(ctx, "b1", "img.png")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.ContentType != "image/png" {
		t.Fatalf("content type mismatch: got %q", info.ContentType)
	}
	if info.Size != 4 {
		t.Fatalf("size mismatch: got %d", info.Size)
	}
}

func TestCopyObject(t *testing.T) {
	m := New()
	m.CreateBucket(ctx, "src")
	m.CreateBucket(ctx, "dst")

	data := []byte("copy me")
	m.PutObject(ctx, "src", "original.txt", data, "text/plain", map[string]string{"k": "v"})

	if err := m.CopyObject(ctx, "src", "original.txt", "dst", "copied.txt"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	obj, err := m.GetObject(ctx, "dst", "copied.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(obj.Data, data) {
		t.Fatal("copied data mismatch")
	}
	if obj.Metadata["k"] != "v" {
		t.Fatal("metadata not preserved in copy")
	}
}

func TestListObjectsWithPrefix(t *testing.T) {
	m := New()
	m.CreateBucket(ctx, "b1")
	m.PutObject(ctx, "b1", "photos/a.png", []byte("a"), "image/png", nil)
	m.PutObject(ctx, "b1", "photos/b.png", []byte("b"), "image/png", nil)
	m.PutObject(ctx, "b1", "docs/c.pdf", []byte("c"), "application/pdf", nil)

	// List with prefix
	result, err := m.ListObjects(ctx, "b1", "photos/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 objects, got %d", len(result))
	}
	if result[0].Key != "photos/a.png" || result[1].Key != "photos/b.png" {
		t.Fatalf("wrong objects or not sorted: %v", result)
	}

	// List all
	all, _ := m.ListObjects(ctx, "b1", "")
	if len(all) != 3 {
		t.Fatalf("expected 3 objects, got %d", len(all))
	}
}

func TestPutToNonExistentBucket(t *testing.T) {
	m := New()
	err := m.PutObject(ctx, "nope", "f.txt", []byte("data"), "text/plain", nil)
	if !IsNotFound(err) {
		t.Fatalf("expected NotFound, got: %v", err)
	}
}

func TestConcurrentAccess(t *testing.T) {
	m := New()
	m.CreateBucket(ctx, "b1")

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("obj-%d", i)
			data := []byte(fmt.Sprintf("data-%d", i))
			m.PutObject(ctx, "b1", key, data, "text/plain", nil)
			m.GetObject(ctx, "b1", key)
			m.HeadObject(ctx, "b1", key)
			m.ListObjects(ctx, "b1", "")
		}(i)
	}
	wg.Wait()

	if m.ObjectCount("b1") != 50 {
		t.Fatalf("expected 50 objects, got %d", m.ObjectCount("b1"))
	}
}
