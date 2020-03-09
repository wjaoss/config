package config

import (
	"context"
	"crypto/md5"
	"fmt"
)

// Snapshot contains point of time loaded configuration
type Snapshot struct {
	Data     []byte
	checksum string
}

// Checksum return md5 checksum of snapshot data
func (s *Snapshot) Checksum() string {
	if s.checksum == "" {
		s.checksum = checksum(s.Data)
	}

	return s.checksum
}

// Loader is config source loader
type Loader interface {
	Load() (*Snapshot, error)

	// assign stream decoder for config loader
	// Loader implementation must aware and use
	// decoder before unmarshalling
	SetDecoder(Decoder)
}

// Decoder decode config stream before loaded
// which can be used to decrypt encoded stream
type Decoder interface {
	Decode([]byte) []byte
}

// Watchable indicate source is support watch changes
type Watchable interface {
	// Watch run watcher with given context
	// implementer should listen to context cancellation
	// to stop watching process
	Watch(context.Context)
}

func checksum(b []byte) string {
	return fmt.Sprintf("%x", md5.Sum(b))
}
