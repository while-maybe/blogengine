package sqlite

import (
	"crypto/rand"
	"encoding/base64"
)

func gen60CharString() string {
	hashBytes := make([]byte, 45)
	_, _ = rand.Read(hashBytes)
	return base64.RawURLEncoding.EncodeToString(hashBytes)
}
