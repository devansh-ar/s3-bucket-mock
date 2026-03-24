package s3mock

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"sync"
	"time"
)

type uploadPart struct {
	partNumber int
	data       []byte
	etag       string
}

type multipartUpload struct {
	bucket      string
	key         string
	contentType string
	uploadID    string
	parts       map[int]*uploadPart
	createdAt   time.Time
	mu          sync.Mutex
}

func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// CreateMultipartUpload starts a new multipart upload. Returns an upload ID.
func (m *Mock) CreateMultipartUpload(_ context.Context, bucketName, key, contentType string) (string, error) {
	if bucketName == "" || key == "" {
		return "", newError(ErrInvalidArgument, "bucket name and key cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.buckets[bucketName]; !ok {
		return "", newError(ErrNotFound, fmt.Sprintf("bucket %q not found", bucketName))
	}

	id := generateID()
	m.multiparts[id] = &multipartUpload{
		bucket:      bucketName,
		key:         key,
		contentType: contentType,
		uploadID:    id,
		parts:       make(map[int]*uploadPart),
		createdAt:   time.Now(),
	}
	return id, nil
}

// UploadPart uploads a single part for a multipart upload.
func (m *Mock) UploadPart(_ context.Context, bucketName, key, uploadID string, partNumber int, data []byte) (*MultipartPart, error) {
	m.mu.RLock()
	mp, ok := m.multiparts[uploadID]
	m.mu.RUnlock()

	if !ok {
		return nil, newError(ErrInvalidUploadID, fmt.Sprintf("upload ID %q not found", uploadID))
	}

	mp.mu.Lock()
	defer mp.mu.Unlock()

	if mp.bucket != bucketName || mp.key != key {
		return nil, newError(ErrInvalidArgument, "bucket/key does not match upload")
	}
	if partNumber < 1 {
		return nil, newError(ErrInvalidArgument, "part number must be >= 1")
	}

	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)

	etag := fmt.Sprintf("%x", sha256.Sum256(data))
	mp.parts[partNumber] = &uploadPart{
		partNumber: partNumber,
		data:       dataCopy,
		etag:       etag,
	}

	return &MultipartPart{
		PartNumber: partNumber,
		ETag:       etag,
		Size:       len(data),
	}, nil
}

// CompleteMultipartUpload assembles all parts into a single object.
// Parts are concatenated in order by part number.
func (m *Mock) CompleteMultipartUpload(ctx context.Context, bucketName, key, uploadID string, parts []MultipartPart) error {
	m.mu.Lock()
	mp, ok := m.multiparts[uploadID]
	if !ok {
		m.mu.Unlock()
		return newError(ErrInvalidUploadID, fmt.Sprintf("upload ID %q not found", uploadID))
	}
	delete(m.multiparts, uploadID)
	m.mu.Unlock()

	mp.mu.Lock()
	defer mp.mu.Unlock()

	if mp.bucket != bucketName || mp.key != key {
		return newError(ErrInvalidArgument, "bucket/key does not match upload")
	}

	// Sort provided parts by part number
	sorted := make([]MultipartPart, len(parts))
	copy(sorted, parts)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].PartNumber < sorted[j].PartNumber
	})

	// Concatenate part data in order
	var assembled []byte
	for _, p := range sorted {
		up, ok := mp.parts[p.PartNumber]
		if !ok {
			return newError(ErrInvalidArgument, fmt.Sprintf("part %d not found", p.PartNumber))
		}
		assembled = append(assembled, up.data...)
	}

	return m.PutObject(ctx, bucketName, key, assembled, mp.contentType, nil)
}

// AbortMultipartUpload cancels an in-progress multipart upload.
func (m *Mock) AbortMultipartUpload(_ context.Context, bucketName, key, uploadID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	mp, ok := m.multiparts[uploadID]
	if !ok {
		return newError(ErrInvalidUploadID, fmt.Sprintf("upload ID %q not found", uploadID))
	}
	if mp.bucket != bucketName || mp.key != key {
		return newError(ErrInvalidArgument, "bucket/key does not match upload")
	}

	delete(m.multiparts, uploadID)
	return nil
}

// ListMultipartUploads returns all in-progress multipart uploads for a bucket.
func (m *Mock) ListMultipartUploads(_ context.Context, bucketName string) ([]MultipartUploadInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if _, ok := m.buckets[bucketName]; !ok {
		return nil, newError(ErrNotFound, fmt.Sprintf("bucket %q not found", bucketName))
	}

	var result []MultipartUploadInfo
	for _, mp := range m.multiparts {
		if mp.bucket == bucketName {
			result = append(result, MultipartUploadInfo{
				Bucket:    mp.bucket,
				Key:       mp.key,
				UploadID:  mp.uploadID,
				CreatedAt: mp.createdAt,
			})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Key < result[j].Key
	})
	return result, nil
}
