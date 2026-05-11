package session

import (
	"crypto/rand"
	"encoding/hex"
	"sync/atomic"
)

const (
	DefaultHash = "--------"
)

var processHash atomic.Pointer[string]

func Hash() string {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return DefaultHash
	}
	return hex.EncodeToString(b[:])
}

func SetHash(h string) {
	if len(h) != 8 {
		h = DefaultHash
	}
	processHash.Store(&h)
}

func GetHash() string {
	if p := processHash.Load(); p != nil {
		return *p
	}
	return DefaultHash
}
