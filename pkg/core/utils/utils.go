package coreutil

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// GenerateRandomString generates a random hex string of the specified length.
// The length parameter refers to the number of bytes, which will result in a hex string twice as long.
func GenerateRandomString(length int) (string, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// ContainsString checks if a slice of strings contains a specific string.
func ContainsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
