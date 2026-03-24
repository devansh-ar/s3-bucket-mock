package s3mock

import (
	"bytes"
	"testing"
)

func TestMultipartUploadFullFlow(t *testing.T) {
	m := New()
	m.CreateBucket(ctx, "b1")

	uploadID, err := m.CreateMultipartUpload(ctx, "b1", "big.zip", "application/zip")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	chunk1 := []byte("hello ")
	chunk2 := []byte("world")
	chunk3 := []byte("!")

	p1, _ := m.UploadPart(ctx, "b1", "big.zip", uploadID, 1, chunk1)
	p2, _ := m.UploadPart(ctx, "b1", "big.zip", uploadID, 2, chunk2)
	p3, _ := m.UploadPart(ctx, "b1", "big.zip", uploadID, 3, chunk3)

	parts := []MultipartPart{
		{PartNumber: p1.PartNumber, ETag: p1.ETag},
		{PartNumber: p2.PartNumber, ETag: p2.ETag},
		{PartNumber: p3.PartNumber, ETag: p3.ETag},
	}

	if err := m.CompleteMultipartUpload(ctx, "b1", "big.zip", uploadID, parts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	obj, err := m.GetObject(ctx, "b1", "big.zip")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []byte("hello world!")
	if !bytes.Equal(obj.Data, expected) {
		t.Fatalf("assembled data mismatch: got %q, want %q", obj.Data, expected)
	}
	if obj.ContentType != "application/zip" {
		t.Fatalf("content type mismatch: got %q", obj.ContentType)
	}
}

func TestMultipartPartsOutOfOrder(t *testing.T) {
	m := New()
	m.CreateBucket(ctx, "b1")

	uploadID, _ := m.CreateMultipartUpload(ctx, "b1", "f.bin", "application/octet-stream")

	// Upload parts out of order
	p3, _ := m.UploadPart(ctx, "b1", "f.bin", uploadID, 3, []byte("C"))
	p1, _ := m.UploadPart(ctx, "b1", "f.bin", uploadID, 1, []byte("A"))
	p2, _ := m.UploadPart(ctx, "b1", "f.bin", uploadID, 2, []byte("B"))

	// Complete with parts listed out of order too
	parts := []MultipartPart{
		{PartNumber: p3.PartNumber, ETag: p3.ETag},
		{PartNumber: p1.PartNumber, ETag: p1.ETag},
		{PartNumber: p2.PartNumber, ETag: p2.ETag},
	}

	m.CompleteMultipartUpload(ctx, "b1", "f.bin", uploadID, parts)

	obj, _ := m.GetObject(ctx, "b1", "f.bin")
	if string(obj.Data) != "ABC" {
		t.Fatalf("expected ABC, got %q", string(obj.Data))
	}
}

func TestMultipartAbort(t *testing.T) {
	m := New()
	m.CreateBucket(ctx, "b1")

	uploadID, _ := m.CreateMultipartUpload(ctx, "b1", "f.bin", "application/octet-stream")
	m.UploadPart(ctx, "b1", "f.bin", uploadID, 1, []byte("data"))

	if err := m.AbortMultipartUpload(ctx, "b1", "f.bin", uploadID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Upload should be gone
	_, err := m.UploadPart(ctx, "b1", "f.bin", uploadID, 2, []byte("more"))
	if !IsInvalidUploadID(err) {
		t.Fatalf("expected InvalidUploadID, got: %v", err)
	}
}

func TestMultipartInvalidUploadID(t *testing.T) {
	m := New()
	m.CreateBucket(ctx, "b1")

	_, err := m.UploadPart(ctx, "b1", "f.bin", "bogus", 1, []byte("data"))
	if !IsInvalidUploadID(err) {
		t.Fatalf("expected InvalidUploadID, got: %v", err)
	}

	err = m.CompleteMultipartUpload(ctx, "b1", "f.bin", "bogus", nil)
	if !IsInvalidUploadID(err) {
		t.Fatalf("expected InvalidUploadID, got: %v", err)
	}
}

func TestMultipartBucketNotFound(t *testing.T) {
	m := New()
	_, err := m.CreateMultipartUpload(ctx, "nope", "f.bin", "text/plain")
	if !IsNotFound(err) {
		t.Fatalf("expected NotFound, got: %v", err)
	}
}

func TestListMultipartUploads(t *testing.T) {
	m := New()
	m.CreateBucket(ctx, "b1")

	m.CreateMultipartUpload(ctx, "b1", "a.bin", "application/octet-stream")
	m.CreateMultipartUpload(ctx, "b1", "b.bin", "application/octet-stream")

	uploads, err := m.ListMultipartUploads(ctx, "b1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(uploads) != 2 {
		t.Fatalf("expected 2 uploads, got %d", len(uploads))
	}
	if uploads[0].Key != "a.bin" || uploads[1].Key != "b.bin" {
		t.Fatalf("uploads not sorted: %v", uploads)
	}
}
