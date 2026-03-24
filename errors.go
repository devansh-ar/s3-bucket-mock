package s3mock

import "errors"

// ErrorCode represents the type of S3 error.
type ErrorCode int

const (
	ErrNotFound ErrorCode = iota
	ErrAlreadyExists
	ErrBucketNotEmpty
	ErrInvalidArgument
	ErrExpired
	ErrInvalidUploadID
)

// S3Error is a typed error returned by the mock.
type S3Error struct {
	Code    ErrorCode
	Message string
}

func (e *S3Error) Error() string {
	return e.Message
}

func newError(code ErrorCode, msg string) *S3Error {
	return &S3Error{Code: code, Message: msg}
}

// IsNotFound returns true if the error is a not-found error.
func IsNotFound(err error) bool {
	var s3err *S3Error
	return errors.As(err, &s3err) && s3err.Code == ErrNotFound
}

// IsAlreadyExists returns true if the error is an already-exists error.
func IsAlreadyExists(err error) bool {
	var s3err *S3Error
	return errors.As(err, &s3err) && s3err.Code == ErrAlreadyExists
}

// IsBucketNotEmpty returns true if the error is a bucket-not-empty error.
func IsBucketNotEmpty(err error) bool {
	var s3err *S3Error
	return errors.As(err, &s3err) && s3err.Code == ErrBucketNotEmpty
}

// IsInvalidArgument returns true if the error is an invalid-argument error.
func IsInvalidArgument(err error) bool {
	var s3err *S3Error
	return errors.As(err, &s3err) && s3err.Code == ErrInvalidArgument
}

// IsExpired returns true if the error is an expired presigned URL error.
func IsExpired(err error) bool {
	var s3err *S3Error
	return errors.As(err, &s3err) && s3err.Code == ErrExpired
}

// IsInvalidUploadID returns true if the error is an invalid upload ID error.
func IsInvalidUploadID(err error) bool {
	var s3err *S3Error
	return errors.As(err, &s3err) && s3err.Code == ErrInvalidUploadID
}
