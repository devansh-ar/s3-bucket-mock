package s3mock

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

func TestURLReturnsDataURI(t *testing.T) {
	m := New()
	m.CreateBucket(ctx, "photos")

	data := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A} // PNG header
	m.PutObject(ctx, "photos", "cat.png", data, "image/png", nil)

	url := m.URL("photos", "cat.png")

	// Verify prefix
	if !strings.HasPrefix(url, "data:image/png;base64,") {
		t.Fatalf("expected data URI prefix, got: %s", url[:50])
	}

	// Decode base64 and verify round-trip
	encoded := strings.TrimPrefix(url, "data:image/png;base64,")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("failed to decode base64: %v", err)
	}
	if len(decoded) != len(data) {
		t.Fatalf("decoded length mismatch: got %d, want %d", len(decoded), len(data))
	}
	for i := range data {
		if decoded[i] != data[i] {
			t.Fatalf("byte mismatch at index %d", i)
		}
	}
}

func TestURLMatchesContentType(t *testing.T) {
	m := New()
	m.CreateBucket(ctx, "files")

	tests := []struct {
		key         string
		contentType string
	}{
		{"pic.jpg", "image/jpeg"},
		{"doc.pdf", "application/pdf"},
		{"page.html", "text/html"},
	}

	for _, tt := range tests {
		m.PutObject(ctx, "files", tt.key, []byte("content"), tt.contentType, nil)
		url := m.URL("files", tt.key)
		prefix := "data:" + tt.contentType + ";base64,"
		if !strings.HasPrefix(url, prefix) {
			t.Errorf("URL for %s: expected prefix %q, got %q", tt.key, prefix, url[:len(prefix)+5])
		}
	}
}

func TestURLNonExistent(t *testing.T) {
	m := New()
	m.CreateBucket(ctx, "b1")

	if url := m.URL("b1", "nope"); url != "" {
		t.Fatalf("expected empty string, got: %s", url)
	}
	if url := m.URL("no-bucket", "nope"); url != "" {
		t.Fatalf("expected empty string, got: %s", url)
	}
}

func TestPresignedPutURL(t *testing.T) {
	m := New()
	m.CreateBucket(ctx, "b1")

	url, err := m.PresignedPutURL(ctx, "b1", "upload.txt", 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(url, "s3mock://presigned/") {
		t.Fatalf("unexpected URL format: %s", url)
	}

	bucket, key, op, err := m.ValidatePresignedURL(url)
	if err != nil {
		t.Fatalf("validation failed: %v", err)
	}
	if bucket != "b1" || key != "upload.txt" || op != "PUT" {
		t.Fatalf("unexpected values: bucket=%s key=%s op=%s", bucket, key, op)
	}
}

func TestPresignedGetURL(t *testing.T) {
	m := New()
	m.CreateBucket(ctx, "b1")

	url, err := m.PresignedGetURL(ctx, "b1", "file.txt", 1*time.Hour)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	bucket, key, op, err := m.ValidatePresignedURL(url)
	if err != nil {
		t.Fatalf("validation failed: %v", err)
	}
	if bucket != "b1" || key != "file.txt" || op != "GET" {
		t.Fatalf("unexpected values: bucket=%s key=%s op=%s", bucket, key, op)
	}
}

func TestPresignedURLExpiry(t *testing.T) {
	m := New()
	m.CreateBucket(ctx, "b1")

	// Create with 0 duration (already expired)
	url, err := m.PresignedPutURL(ctx, "b1", "f.txt", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Small sleep to ensure we're past expiry
	time.Sleep(1 * time.Millisecond)

	_, _, _, err = m.ValidatePresignedURL(url)
	if !IsExpired(err) {
		t.Fatalf("expected Expired error, got: %v", err)
	}
}

func TestPresignedURLInvalid(t *testing.T) {
	m := New()

	_, _, _, err := m.ValidatePresignedURL("https://example.com/not-valid")
	if !IsInvalidArgument(err) {
		t.Fatalf("expected InvalidArgument, got: %v", err)
	}

	_, _, _, err = m.ValidatePresignedURL("s3mock://presigned/bogus-token")
	if !IsNotFound(err) {
		t.Fatalf("expected NotFound, got: %v", err)
	}
}

func TestPresignedURLBucketNotFound(t *testing.T) {
	m := New()
	_, err := m.PresignedPutURL(ctx, "nope", "f.txt", time.Minute)
	if !IsNotFound(err) {
		t.Fatalf("expected NotFound, got: %v", err)
	}
}
