package s3mock

import "time"

// BucketInfo holds metadata about a bucket.
type BucketInfo struct {
	Name      string
	CreatedAt time.Time
}

// ObjectInfo holds metadata about an object (no data).
type ObjectInfo struct {
	Key          string
	Size         int
	ContentType  string
	ETag         string
	LastModified time.Time
}

// Object holds metadata and the actual data bytes.
type Object struct {
	ObjectInfo
	Data     []byte
	Metadata map[string]string
	Tags     map[string]string
}

// MultipartPart represents a single uploaded part in a multipart upload.
type MultipartPart struct {
	PartNumber int
	ETag       string
	Size       int
}

// MultipartUploadInfo holds metadata about an in-progress multipart upload.
type MultipartUploadInfo struct {
	Bucket    string
	Key       string
	UploadID  string
	CreatedAt time.Time
}
