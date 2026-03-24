package s3mock

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type object struct {
	key          string
	data         []byte
	contentType  string
	etag         string
	lastModified time.Time
	metadata     map[string]string
	tags         map[string]string
}

type bucket struct {
	name      string
	createdAt time.Time
	objects   map[string]*object
}

// Mock is an in-memory S3 mock. Create one with New().
type Mock struct {
	mu         sync.RWMutex
	buckets    map[string]*bucket
	multiparts map[string]*multipartUpload
	presigned  map[string]*presignedEntry
	onPut      []func(string, string)
	onDelete   []func(string, string)
}

// New creates a new S3 mock with no buckets.
func New() *Mock {
	return &Mock{
		buckets:    make(map[string]*bucket),
		multiparts: make(map[string]*multipartUpload),
		presigned:  make(map[string]*presignedEntry),
	}
}

// CreateBucket creates a new bucket. Returns error if name is empty or bucket already exists.
func (m *Mock) CreateBucket(_ context.Context, name string) error {
	if name == "" {
		return newError(ErrInvalidArgument, "bucket name cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.buckets[name]; ok {
		return newError(ErrAlreadyExists, fmt.Sprintf("bucket %q already exists", name))
	}

	m.buckets[name] = &bucket{
		name:      name,
		createdAt: time.Now(),
		objects:   make(map[string]*object),
	}
	return nil
}

// DeleteBucket deletes a bucket. Returns error if not found or not empty.
func (m *Mock) DeleteBucket(_ context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	bkt, ok := m.buckets[name]
	if !ok {
		return newError(ErrNotFound, fmt.Sprintf("bucket %q not found", name))
	}
	if len(bkt.objects) > 0 {
		return newError(ErrBucketNotEmpty, fmt.Sprintf("bucket %q is not empty", name))
	}

	delete(m.buckets, name)
	return nil
}

// ListBuckets returns all buckets sorted by name.
func (m *Mock) ListBuckets(_ context.Context) []BucketInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]BucketInfo, 0, len(m.buckets))
	for _, bkt := range m.buckets {
		result = append(result, BucketInfo{
			Name:      bkt.name,
			CreatedAt: bkt.createdAt,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// PutObject stores an object in a bucket. Deep copies the data.
// Fires OnPut hooks after successful storage.
func (m *Mock) PutObject(_ context.Context, bucketName, key string, data []byte, contentType string, metadata map[string]string) error {
	if bucketName == "" {
		return newError(ErrInvalidArgument, "bucket name cannot be empty")
	}
	if key == "" {
		return newError(ErrInvalidArgument, "object key cannot be empty")
	}

	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)

	metaCopy := copyMap(metadata)
	etag := fmt.Sprintf("%x", sha256.Sum256(data))

	m.mu.Lock()
	bkt, ok := m.buckets[bucketName]
	if !ok {
		m.mu.Unlock()
		return newError(ErrNotFound, fmt.Sprintf("bucket %q not found", bucketName))
	}

	bkt.objects[key] = &object{
		key:          key,
		data:         dataCopy,
		contentType:  contentType,
		etag:         etag,
		lastModified: time.Now(),
		metadata:     metaCopy,
		tags:         make(map[string]string),
	}

	hooks := make([]func(string, string), len(m.onPut))
	copy(hooks, m.onPut)
	m.mu.Unlock()

	for _, fn := range hooks {
		fn(bucketName, key)
	}
	return nil
}

// GetObject retrieves an object. Returns a deep copy of the data.
func (m *Mock) GetObject(_ context.Context, bucketName, key string) (*Object, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	bkt, ok := m.buckets[bucketName]
	if !ok {
		return nil, newError(ErrNotFound, fmt.Sprintf("bucket %q not found", bucketName))
	}

	obj, ok := bkt.objects[key]
	if !ok {
		return nil, newError(ErrNotFound, fmt.Sprintf("object %q not found in bucket %q", key, bucketName))
	}

	dataCopy := make([]byte, len(obj.data))
	copy(dataCopy, obj.data)

	return &Object{
		ObjectInfo: ObjectInfo{
			Key:          obj.key,
			Size:         len(obj.data),
			ContentType:  obj.contentType,
			ETag:         obj.etag,
			LastModified: obj.lastModified,
		},
		Data:     dataCopy,
		Metadata: copyMap(obj.metadata),
		Tags:     copyMap(obj.tags),
	}, nil
}

// DeleteObject removes an object from a bucket.
// Fires OnDelete hooks after successful deletion.
func (m *Mock) DeleteObject(_ context.Context, bucketName, key string) error {
	m.mu.Lock()
	bkt, ok := m.buckets[bucketName]
	if !ok {
		m.mu.Unlock()
		return newError(ErrNotFound, fmt.Sprintf("bucket %q not found", bucketName))
	}

	if _, ok := bkt.objects[key]; !ok {
		m.mu.Unlock()
		return newError(ErrNotFound, fmt.Sprintf("object %q not found in bucket %q", key, bucketName))
	}

	delete(bkt.objects, key)

	hooks := make([]func(string, string), len(m.onDelete))
	copy(hooks, m.onDelete)
	m.mu.Unlock()

	for _, fn := range hooks {
		fn(bucketName, key)
	}
	return nil
}

// HeadObject returns object metadata without the data bytes.
func (m *Mock) HeadObject(_ context.Context, bucketName, key string) (*ObjectInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	bkt, ok := m.buckets[bucketName]
	if !ok {
		return nil, newError(ErrNotFound, fmt.Sprintf("bucket %q not found", bucketName))
	}

	obj, ok := bkt.objects[key]
	if !ok {
		return nil, newError(ErrNotFound, fmt.Sprintf("object %q not found in bucket %q", key, bucketName))
	}

	return &ObjectInfo{
		Key:          obj.key,
		Size:         len(obj.data),
		ContentType:  obj.contentType,
		ETag:         obj.etag,
		LastModified: obj.lastModified,
	}, nil
}

// CopyObject copies an object from src to dst. Works across buckets.
func (m *Mock) CopyObject(_ context.Context, srcBucket, srcKey, dstBucket, dstKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	sBkt, ok := m.buckets[srcBucket]
	if !ok {
		return newError(ErrNotFound, fmt.Sprintf("source bucket %q not found", srcBucket))
	}
	srcObj, ok := sBkt.objects[srcKey]
	if !ok {
		return newError(ErrNotFound, fmt.Sprintf("source object %q not found in bucket %q", srcKey, srcBucket))
	}

	dBkt, ok := m.buckets[dstBucket]
	if !ok {
		return newError(ErrNotFound, fmt.Sprintf("destination bucket %q not found", dstBucket))
	}

	dataCopy := make([]byte, len(srcObj.data))
	copy(dataCopy, srcObj.data)

	dBkt.objects[dstKey] = &object{
		key:          dstKey,
		data:         dataCopy,
		contentType:  srcObj.contentType,
		etag:         srcObj.etag,
		lastModified: time.Now(),
		metadata:     copyMap(srcObj.metadata),
		tags:         copyMap(srcObj.tags),
	}
	return nil
}

// ListObjects returns objects in a bucket filtered by prefix, sorted by key.
func (m *Mock) ListObjects(_ context.Context, bucketName, prefix string) ([]ObjectInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	bkt, ok := m.buckets[bucketName]
	if !ok {
		return nil, newError(ErrNotFound, fmt.Sprintf("bucket %q not found", bucketName))
	}

	var result []ObjectInfo
	for _, obj := range bkt.objects {
		if prefix == "" || strings.HasPrefix(obj.key, prefix) {
			result = append(result, ObjectInfo{
				Key:          obj.key,
				Size:         len(obj.data),
				ContentType:  obj.contentType,
				ETag:         obj.etag,
				LastModified: obj.lastModified,
			})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Key < result[j].Key
	})
	return result, nil
}

func copyMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	c := make(map[string]string, len(m))
	for k, v := range m {
		c[k] = v
	}
	return c
}
