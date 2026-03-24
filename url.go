package s3mock

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"
)

type presignedEntry struct {
	bucket    string
	key       string
	operation string // "PUT" or "GET"
	expiresAt time.Time
}

// URL returns a data URI of the stored object's actual bytes.
// Paste in a browser to see the real uploaded file.
// Returns empty string if the object doesn't exist.
func (m *Mock) URL(bucket, key string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	bkt, ok := m.buckets[bucket]
	if !ok {
		return ""
	}
	obj, ok := bkt.objects[key]
	if !ok {
		return ""
	}

	encoded := base64.StdEncoding.EncodeToString(obj.data)
	return fmt.Sprintf("data:%s;base64,%s", obj.contentType, encoded)
}

// PresignedPutURL generates a mock presigned URL for uploading an object.
// The URL is valid for the given duration.
func (m *Mock) PresignedPutURL(_ context.Context, bucket, key string, expiry time.Duration) (string, error) {
	return m.createPresigned(bucket, key, "PUT", expiry)
}

// PresignedGetURL generates a mock presigned URL for downloading an object.
// The URL is valid for the given duration.
func (m *Mock) PresignedGetURL(_ context.Context, bucket, key string, expiry time.Duration) (string, error) {
	return m.createPresigned(bucket, key, "GET", expiry)
}

func (m *Mock) createPresigned(bucket, key, operation string, expiry time.Duration) (string, error) {
	if bucket == "" || key == "" {
		return "", newError(ErrInvalidArgument, "bucket and key cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.buckets[bucket]; !ok {
		return "", newError(ErrNotFound, fmt.Sprintf("bucket %q not found", bucket))
	}

	token := generateID()
	m.presigned[token] = &presignedEntry{
		bucket:    bucket,
		key:       key,
		operation: operation,
		expiresAt: time.Now().Add(expiry),
	}

	return fmt.Sprintf("s3mock://presigned/%s", token), nil
}

// ValidatePresignedURL checks if a presigned URL is valid and not expired.
// Returns the bucket, key, and operation ("PUT" or "GET") if valid.
func (m *Mock) ValidatePresignedURL(url string) (bucket, key, operation string, err error) {
	const prefix = "s3mock://presigned/"
	if !strings.HasPrefix(url, prefix) {
		return "", "", "", newError(ErrInvalidArgument, "not a valid presigned URL")
	}
	token := strings.TrimPrefix(url, prefix)

	m.mu.RLock()
	entry, ok := m.presigned[token]
	m.mu.RUnlock()

	if !ok {
		return "", "", "", newError(ErrNotFound, "presigned URL not found")
	}

	if time.Now().After(entry.expiresAt) {
		m.mu.Lock()
		delete(m.presigned, token)
		m.mu.Unlock()
		return "", "", "", newError(ErrExpired, "presigned URL has expired")
	}

	return entry.bucket, entry.key, entry.operation, nil
}
