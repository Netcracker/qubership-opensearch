package util

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

const (
	SwitchoverAnnotationKey = "switchoverRetry"
	RetryFailedComment      = "retry failed"
)

// Hash returns hash SHA-256 of object
func Hash(o interface{}) (string, error) {
	cr, err := json.Marshal(o)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	hash.Write(cr)
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func Min(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}
