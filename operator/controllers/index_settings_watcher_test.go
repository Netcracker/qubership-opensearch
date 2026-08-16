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

//go:build unit

package controllers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	opensearchservice "github.com/Netcracker/qubership-opensearch/operator/api/v1"
	"github.com/Netcracker/qubership-opensearch/operator/util"
	"github.com/go-logr/logr"
)

// capturedRequest records one incoming PUT request for assertion.
type capturedRequest struct {
	path string
	body []byte
}

// newTestHelper builds an IndexSettingsHelper pointing at the given test server.
func newTestHelper(server *httptest.Server) IndexSettingsHelper {
	return IndexSettingsHelper{
		logger:     logr.Discard(),
		restClient: util.NewRestClient(server.URL, http.Client{}, util.Credentials{}),
	}
}

func TestNewIndexSettingsWatcher_InitialStateIsStopped(t *testing.T) {
	var mu sync.Mutex
	w := NewIndexSettingsWatcher(&mu)
	if *w.State != stoppedWatcherState {
		t.Errorf("expected initial state %q, got %q", stoppedWatcherState, *w.State)
	}
}

func TestApplyAllSettings_SingleEntry_CorrectURLAndBody(t *testing.T) {
	var captured capturedRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		captured = capturedRequest{path: r.URL.RequestURI(), body: body}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	entries := []opensearchservice.IndexSettingEntry{
		{
			Pattern: "*",
			Settings: map[string]interface{}{
				"index.translog.durability": "async",
			},
		},
	}

	var mu sync.Mutex
	w := NewIndexSettingsWatcher(&mu)
	w.applyAllSettings(newTestHelper(server), entries)

	expectedPath := "/*,-.*/_settings?allow_no_indices=true"
	if captured.path != expectedPath {
		t.Errorf("expected path %q, got %q", expectedPath, captured.path)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(captured.body, &body); err != nil {
		t.Fatalf("could not parse request body as JSON: %v", err)
	}
	if body["index.translog.durability"] != "async" {
		t.Errorf("unexpected body: %s", captured.body)
	}
}

func TestApplyAllSettings_MultipleEntries_AppliedInOrder(t *testing.T) {
	var captured []capturedRequest
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		captured = append(captured, capturedRequest{path: r.URL.RequestURI(), body: body})
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	entries := []opensearchservice.IndexSettingEntry{
		{
			Pattern:  "*",
			Settings: map[string]interface{}{"index.translog.sync_interval": "30s"},
		},
		{
			Pattern:  "*bss*",
			Settings: map[string]interface{}{"index.translog.sync_interval": "15s"},
		},
	}

	var wMu sync.Mutex
	w := NewIndexSettingsWatcher(&wMu)
	w.applyAllSettings(newTestHelper(server), entries)

	if len(captured) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(captured))
	}
	if captured[0].path != "/*,-.*/_settings?allow_no_indices=true" {
		t.Errorf("first request path: got %q", captured[0].path)
	}
	if captured[1].path != "/*bss*,-.*/_settings?allow_no_indices=true" {
		t.Errorf("second request path: got %q", captured[1].path)
	}
}

func TestApplyAllSettings_NullValueSerializedAsJSONNull(t *testing.T) {
	var captured capturedRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		captured = capturedRequest{path: r.URL.RequestURI(), body: body}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	entries := []opensearchservice.IndexSettingEntry{
		{
			Pattern: "*",
			Settings: map[string]interface{}{
				"index.translog.sync_interval": nil,
			},
		},
	}

	var mu sync.Mutex
	w := NewIndexSettingsWatcher(&mu)
	w.applyAllSettings(newTestHelper(server), entries)

	var body map[string]interface{}
	if err := json.Unmarshal(captured.body, &body); err != nil {
		t.Fatalf("could not parse request body: %v", err)
	}
	val, ok := body["index.translog.sync_interval"]
	if !ok {
		t.Fatal("key index.translog.sync_interval missing from body")
	}
	if val != nil {
		t.Errorf("expected null, got %v (%T)", val, val)
	}
}

func TestApplyAllSettings_SystemIndicesExcludedViaPattern(t *testing.T) {
	var paths []string
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.RequestURI())
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	entries := []opensearchservice.IndexSettingEntry{
		{
			Pattern:  "myindex*",
			Settings: map[string]interface{}{"index.translog.durability": "async"},
		},
	}

	var wMu sync.Mutex
	w := NewIndexSettingsWatcher(&wMu)
	w.applyAllSettings(newTestHelper(server), entries)

	if len(paths) != 1 {
		t.Fatalf("expected 1 request, got %d", len(paths))
	}
	// System indices must be excluded via ,-.*
	expected := "/myindex*,-.*/_settings?allow_no_indices=true"
	if paths[0] != expected {
		t.Errorf("expected path %q, got %q", expected, paths[0])
	}
}

func TestApplyAllSettings_HTTPError_ContinuesToNextEntry(t *testing.T) {
	var paths []string
	var mu sync.Mutex
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.RequestURI())
		callCount++
		mu.Unlock()
		// Fail the first request, succeed the rest.
		if callCount == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	entries := []opensearchservice.IndexSettingEntry{
		{Pattern: "first*", Settings: map[string]interface{}{"k": "v1"}},
		{Pattern: "second*", Settings: map[string]interface{}{"k": "v2"}},
	}

	var wMu sync.Mutex
	w := NewIndexSettingsWatcher(&wMu)
	w.applyAllSettings(newTestHelper(server), entries)

	if len(paths) != 2 {
		t.Fatalf("expected both entries to be attempted, got %d requests", len(paths))
	}
}

func TestWatcher_StartSetsRunningState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	var mu sync.Mutex
	w := NewIndexSettingsWatcher(&mu)

	entries := []opensearchservice.IndexSettingEntry{
		{Pattern: "*", Settings: map[string]interface{}{"index.translog.durability": "async"}},
	}
	w.start(newTestHelper(server), entries)

	// Give the goroutine time to transition to running before we check.
	time.Sleep(20 * time.Millisecond)
	if *w.State != runningWatcherState {
		t.Errorf("expected state %q after start, got %q", runningWatcherState, *w.State)
	}

	w.stop()
}

func TestWatcher_StopSetsStoppedState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	var mu sync.Mutex
	w := NewIndexSettingsWatcher(&mu)

	entries := []opensearchservice.IndexSettingEntry{
		{Pattern: "*", Settings: map[string]interface{}{"index.translog.durability": "async"}},
	}
	w.start(newTestHelper(server), entries)
	time.Sleep(20 * time.Millisecond)

	w.stop()

	// After stop() the state flag is set immediately; the goroutine exits when it
	// next wakes and checks the flag (within watchInterval). We only assert the flag.
	if *w.State != stoppedWatcherState {
		t.Errorf("expected state %q after stop, got %q", stoppedWatcherState, *w.State)
	}
}

func TestWatcher_NoEntriesNoRequests(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var mu sync.Mutex
	w := NewIndexSettingsWatcher(&mu)
	w.applyAllSettings(newTestHelper(server), nil)

	if requestCount != 0 {
		t.Errorf("expected 0 requests for empty entries, got %d", requestCount)
	}
}
