package objstore

import (
	"errors"
	"time"
)

var ErrCorrupted = errors.New("local file changed while reading")

type ObjectMeta struct {
	ETag         string
	ContentLen   int64
	MetaMD5      string
	LastModified time.Time
}

type Store interface {
	Put(key, absPath, contentType string) error
	Head(key string) (ObjectMeta, bool, error)
}
