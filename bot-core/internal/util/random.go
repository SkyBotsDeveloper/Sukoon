package util

import (
	"crypto/rand"
	"encoding/hex"
)

func RandomID(length int) string {
	if length <= 0 {
		length = 16
	}

	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "random-fallback"
	}
	return hex.EncodeToString(buf)
}
