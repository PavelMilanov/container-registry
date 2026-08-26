package storage

import "errors"

var (
	ErrUploadNotFound = errors.New("upload not found")
	ErrBlobNotFound   = errors.New("blob not found")
	ErrInvalidOffset  = errors.New("invalid upload offset")
	ErrInvalidLength  = errors.New("invalid upload length")
	ErrInvalidDigest  = errors.New("invalid digest")
	ErrDigestMismatch = errors.New("digest mismatch")
	ErrBlobCorrupted  = errors.New("blob corrupted")
	ErrCrossDevice    = errors.New("upload and blob directories are on different filesystems")
)
