package pkg

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

func CalculateSHA256[T []byte | string](bs T) string {
	hash := sha256.Sum256([]byte(bs))
	return hex.EncodeToString(hash[:])
}

func CalculateFileSHA256(f string) (string, error) {
	file, err := os.Open(f)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}
