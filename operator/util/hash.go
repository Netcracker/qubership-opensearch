// Copyright 2024-2025 NetCracker Technology Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
