package utils

import (
	"crypto/rand"
	"fmt"
)

const (
	charset    = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	charsetLen = byte(len(charset))

	// avoid modulo bias
	maxUnbiased = 255 - (256 % len(charset))
)

func GeneratePublicID(length int) (string, error) {
	result := make([]byte, 0, length)
	// account for about 3% rejection rate due to modulo bias (256 = 62 * 4 + 8 and 8/256 = 3.125%), adding 10% more than triple so should be more than enough to not request extra bytes from crypto
	buffer := make([]byte, length+(length*10/100))

	for len(result) < length {
		if _, err := rand.Reader.Read(buffer); err != nil {
			return "", fmt.Errorf("could not generate random bytes: %w", err)
		}
		for _, randByte := range buffer {

			if int(randByte) > maxUnbiased {
				continue
			}

			result = append(result, charset[randByte%charsetLen])

			if len(result) == length {
				break
			}
		}
	}
	return string(result), nil
}
