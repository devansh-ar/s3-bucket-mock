package s3mock

import (
	"context"
	"fmt"
)

// PutObjectTagging sets tags on an existing object.
func (m *Mock) PutObjectTagging(_ context.Context, bucketName, key string, tags map[string]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	bkt, ok := m.buckets[bucketName]
	if !ok {
		return newError(ErrNotFound, fmt.Sprintf("bucket %q not found", bucketName))
	}
	obj, ok := bkt.objects[key]
	if !ok {
		return newError(ErrNotFound, fmt.Sprintf("object %q not found in bucket %q", key, bucketName))
	}

	obj.tags = copyMap(tags)
	return nil
}

// GetObjectTagging returns the tags on an object.
func (m *Mock) GetObjectTagging(_ context.Context, bucketName, key string) (map[string]string, error) {
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

	return copyMap(obj.tags), nil
}

// DeleteObjectTagging removes all tags from an object.
func (m *Mock) DeleteObjectTagging(_ context.Context, bucketName, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	bkt, ok := m.buckets[bucketName]
	if !ok {
		return newError(ErrNotFound, fmt.Sprintf("bucket %q not found", bucketName))
	}
	obj, ok := bkt.objects[key]
	if !ok {
		return newError(ErrNotFound, fmt.Sprintf("object %q not found in bucket %q", key, bucketName))
	}

	obj.tags = make(map[string]string)
	return nil
}
