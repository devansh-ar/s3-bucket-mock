# s3mock

A zero-dependency, in-memory AWS S3 mock for Go. Use it in your tests to verify upload/download logic without real AWS credentials, network access, or Docker.

## Features

| Feature | Description |
|---------|-------------|
| **Core S3 Ops** | CreateBucket, DeleteBucket, ListBuckets, PutObject, GetObject, DeleteObject, HeadObject, CopyObject, ListObjects |
| **Data URI URLs** | `URL()` returns a `data:<type>;base64,...` URI of the actual uploaded file — paste in a browser, see your image |
| **Presigned URLs** | Generate mock presigned PUT/GET URLs with expiry and validate them |
| **Multipart Upload** | CreateMultipartUpload → UploadPart → CompleteMultipartUpload (parts assembled in order) |
| **Event Hooks** | `OnPut` / `OnDelete` callbacks fire after every successful operation |
| **Object Tagging** | PutObjectTagging, GetObjectTagging, DeleteObjectTagging |
| **Test Helpers** | `AssertObjectExists(t, ...)`, `AssertBucketEmpty(t, ...)`, `Reset()`, `ObjectCount()` |
| **Thread Safe** | All operations protected by `sync.RWMutex` — safe for concurrent goroutines |
| **Zero Dependencies** | Only Go stdlib in production code |

## Install

```bash
go get github.com/devansh-ar/s3-bucket-mock
```

## Import

```go
import s3mock "github.com/devansh-ar/s3-bucket-mock"
```

## Quick Start

```go
package myapp_test

import (
    "context"
    "testing"

    s3mock "github.com/devansh-ar/s3-bucket-mock"
)

func TestUploadAndDownload(t *testing.T) {
    mock := s3mock.New()
    ctx := context.Background()

    // Create a bucket
    mock.CreateBucket(ctx, "avatars")

    // Upload a file
    imgBytes := []byte{0x89, 0x50, 0x4E, 0x47} // PNG header
    mock.PutObject(ctx, "avatars", "user-123.png", imgBytes, "image/png", nil)

    // Download it back — exact same bytes
    obj, err := mock.GetObject(ctx, "avatars", "user-123.png")
    if err != nil {
        t.Fatal(err)
    }
    if string(obj.Data) != string(imgBytes) {
        t.Fatal("data mismatch")
    }

    // Get a viewable URL — paste in browser to see the image
    url := mock.URL("avatars", "user-123.png")
    // url == "data:image/png;base64,iVBORw0KGg=="
    t.Log("Mock URL:", url)
}
```

## API Reference

### Creating the Mock

```go
mock := s3mock.New()
```

No config needed. Returns a `*Mock` with no buckets.

### Bucket Operations

```go
// Create a bucket
err := mock.CreateBucket(ctx, "my-bucket")

// List all buckets (sorted by name)
buckets := mock.ListBuckets(ctx)
// []BucketInfo{{Name: "my-bucket", CreatedAt: ...}}

// Delete a bucket (must be empty)
err := mock.DeleteBucket(ctx, "my-bucket")
```

### Object Operations

```go
// Upload
err := mock.PutObject(ctx, "my-bucket", "photos/cat.png", data, "image/png", map[string]string{
    "author": "alice",
})

// Download — returns deep copy of data
obj, err := mock.GetObject(ctx, "my-bucket", "photos/cat.png")
// obj.Data         — []byte (your file)
// obj.ContentType  — "image/png"
// obj.ETag         — SHA256 hash
// obj.Size         — len(data)
// obj.Metadata     — map[string]string{"author": "alice"}

// Head — metadata only, no data transfer
info, err := mock.HeadObject(ctx, "my-bucket", "photos/cat.png")
// info.Key, info.Size, info.ContentType, info.ETag, info.LastModified

// Delete
err := mock.DeleteObject(ctx, "my-bucket", "photos/cat.png")

// Copy (works across buckets)
err := mock.CopyObject(ctx, "src-bucket", "original.png", "dst-bucket", "copy.png")

// List with prefix filter (sorted by key)
objects, err := mock.ListObjects(ctx, "my-bucket", "photos/")
// []ObjectInfo — only keys starting with "photos/"
```

### Data URI URLs

```go
url := mock.URL("my-bucket", "photo.png")
// "data:image/png;base64,iVBORw0KGgo..."
```

Returns the **actual uploaded bytes** as a base64 data URI. The content type matches what you uploaded. Paste it in a browser address bar or use it in an `<img src="...">` tag — it just works.

Returns `""` if the object doesn't exist.

### Presigned URLs

```go
// Generate a presigned PUT URL (valid for 15 minutes)
putURL, err := mock.PresignedPutURL(ctx, "my-bucket", "upload.png", 15*time.Minute)
// "s3mock://presigned/a1b2c3d4..."

// Generate a presigned GET URL (valid for 1 hour)
getURL, err := mock.PresignedGetURL(ctx, "my-bucket", "photo.png", 1*time.Hour)

// Validate a presigned URL
bucket, key, operation, err := mock.ValidatePresignedURL(putURL)
// bucket="my-bucket", key="upload.png", operation="PUT", err=nil

// Expired URLs return an error
_, _, _, err := mock.ValidatePresignedURL(expiredURL)
s3mock.IsExpired(err) // true
```

### Multipart Upload

```go
// Start
uploadID, err := mock.CreateMultipartUpload(ctx, "my-bucket", "big-file.zip", "application/zip")

// Upload parts (any order)
p1, _ := mock.UploadPart(ctx, "my-bucket", "big-file.zip", uploadID, 1, chunk1)
p2, _ := mock.UploadPart(ctx, "my-bucket", "big-file.zip", uploadID, 2, chunk2)
p3, _ := mock.UploadPart(ctx, "my-bucket", "big-file.zip", uploadID, 3, chunk3)

// Complete — parts assembled in order by part number
parts := []s3mock.MultipartPart{
    {PartNumber: p1.PartNumber, ETag: p1.ETag},
    {PartNumber: p2.PartNumber, ETag: p2.ETag},
    {PartNumber: p3.PartNumber, ETag: p3.ETag},
}
err := mock.CompleteMultipartUpload(ctx, "my-bucket", "big-file.zip", uploadID, parts)

// The object is now available via GetObject
obj, _ := mock.GetObject(ctx, "my-bucket", "big-file.zip")
// obj.Data == chunk1 + chunk2 + chunk3

// Or abort to cancel
err := mock.AbortMultipartUpload(ctx, "my-bucket", "big-file.zip", uploadID)

// List in-progress uploads
uploads, err := mock.ListMultipartUploads(ctx, "my-bucket")
```

### Event Hooks

```go
// Fire after every successful PutObject
mock.OnPut(func(bucket, key string) {
    fmt.Printf("uploaded: %s/%s\n", bucket, key)
})

// Fire after every successful DeleteObject
mock.OnDelete(func(bucket, key string) {
    fmt.Printf("deleted: %s/%s\n", bucket, key)
})

// Multiple hooks can be registered — all fire in order
```

### Object Tagging

```go
// Set tags
err := mock.PutObjectTagging(ctx, "my-bucket", "photo.png", map[string]string{
    "env":  "production",
    "team": "backend",
})

// Get tags
tags, err := mock.GetObjectTagging(ctx, "my-bucket", "photo.png")
// map[string]string{"env": "production", "team": "backend"}

// Remove all tags
err := mock.DeleteObjectTagging(ctx, "my-bucket", "photo.png")
```

### Test Helpers

```go
// Assertions — fail the test immediately if condition not met
mock.AssertObjectExists(t, "my-bucket", "photo.png")
mock.AssertObjectNotExists(t, "my-bucket", "deleted.png")
mock.AssertBucketExists(t, "my-bucket")
mock.AssertBucketEmpty(t, "empty-bucket")
mock.AssertObjectContent(t, "my-bucket", "photo.png", expectedBytes)

// Utilities
count := mock.ObjectCount("my-bucket") // number of objects

// Reset everything — clean slate for next test
mock.Reset()
```

### Error Handling

All errors are `*S3Error` with a code. Use the predicate functions:

```go
_, err := mock.GetObject(ctx, "bucket", "nope")
if s3mock.IsNotFound(err) {
    // object doesn't exist
}

err := mock.CreateBucket(ctx, "existing")
if s3mock.IsAlreadyExists(err) {
    // bucket already exists
}

err := mock.DeleteBucket(ctx, "non-empty")
if s3mock.IsBucketNotEmpty(err) {
    // bucket has objects
}

_, _, _, err := mock.ValidatePresignedURL(url)
if s3mock.IsExpired(err) {
    // presigned URL expired
}

if s3mock.IsInvalidArgument(err) {
    // empty bucket name, empty key, etc.
}

if s3mock.IsInvalidUploadID(err) {
    // bad multipart upload ID
}
```

## Types

```go
type BucketInfo struct {
    Name      string
    CreatedAt time.Time
}

type ObjectInfo struct {
    Key          string
    Size         int
    ContentType  string
    ETag         string
    LastModified time.Time
}

type Object struct {
    ObjectInfo
    Data     []byte
    Metadata map[string]string
    Tags     map[string]string
}

type MultipartPart struct {
    PartNumber int
    ETag       string
    Size       int
}
```

## Example: Testing a Real Service

```go
package user_test

import (
    "context"
    "os"
    "testing"

    s3mock "github.com/devansh-ar/s3-bucket-mock"
)

// Imagine your app has a UserService that uploads avatars to S3.
// In production it talks to real AWS. In tests, swap in the mock.

type AvatarUploader interface {
    PutObject(ctx context.Context, bucket, key string, data []byte, contentType string, metadata map[string]string) error
    GetObject(ctx context.Context, bucket, key string) (*s3mock.Object, error)
}

type UserService struct {
    storage AvatarUploader
    bucket  string
}

func (s *UserService) UploadAvatar(ctx context.Context, userID string, img []byte) error {
    return s.storage.PutObject(ctx, s.bucket, "avatars/"+userID+".png", img, "image/png", nil)
}

func (s *UserService) GetAvatar(ctx context.Context, userID string) ([]byte, error) {
    obj, err := s.storage.GetObject(ctx, s.bucket, "avatars/"+userID+".png")
    if err != nil {
        return nil, err
    }
    return obj.Data, nil
}

func TestUserAvatarUpload(t *testing.T) {
    mock := s3mock.New()
    mock.CreateBucket(context.Background(), "app-data")

    svc := &UserService{storage: mock, bucket: "app-data"}
    ctx := context.Background()

    // Load a real test image
    img, _ := os.ReadFile("testdata/avatar.png")

    // Upload
    if err := svc.UploadAvatar(ctx, "user-42", img); err != nil {
        t.Fatal(err)
    }

    // Verify
    mock.AssertObjectExists(t, "app-data", "avatars/user-42.png")
    mock.AssertObjectContent(t, "app-data", "avatars/user-42.png", img)

    // Download back
    got, err := svc.GetAvatar(ctx, "user-42")
    if err != nil {
        t.Fatal(err)
    }
    if string(got) != string(img) {
        t.Fatal("avatar data mismatch")
    }

    // Get a browsable URL for debugging
    url := mock.URL("app-data", "avatars/user-42.png")
    t.Log("Preview:", url) // paste in browser to see the avatar
}
```

## Running Tests

```bash
go test -race -v ./...
```

## License

MIT
