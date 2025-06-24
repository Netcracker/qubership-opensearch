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

package backup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Netcracker/dbaas-opensearch-adapter/api"
	"github.com/Netcracker/dbaas-opensearch-adapter/basic"
	core "github.com/Netcracker/qubership-dbaas-adapter-core/pkg/utils"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Netcracker/dbaas-opensearch-adapter/common"
	"github.com/gorilla/mux"
	"github.com/opensearch-project/opensearch-go/opensearchapi"
)

var (
	logger                      = common.GetLogger()
	resourcePrefixAttributeName = "resource_prefix"
)

type Repository struct {
	Status int `json:"status"`
}

type ActionTrack struct {
	Action        string            `json:"action"`
	Details       TrackDetails      `json:"details"`
	Status        string            `json:"status"`
	TrackID       string            `json:"trackId"`
	ChangedNameDb map[string]string `json:"changedNameDb"`
	TrackPath     *string           `json:"trackPath"` // would be nil in case if names regeneration not requested
}

type JobStatus struct {
	State   string `json:"status"`
	Message string `json:"details,omitempty"`
	Vault   string `json:"vault"`
	Type    string `json:"type"`
	Error   string `json:"err,omitempty"`
	TaskId  string `json:"trackPath"`
}

type Database struct {
	Namespace    string `json:"namespace"`
	Microservice string `json:"microservice"`
	Name         string `json:"name"`
	Prefix       string `json:"prefix,omitempty"`
}

type RestorationRequest struct {
	Databases       []Database `json:"databases"`
	RegenerateNames bool       `json:"regenerateNames,omitempty"`
}

type TrackDetails struct {
	LocalId string `json:"localId"`
}

type Snapshots struct {
	Snapshots []SnapshotStatus
}

type SnapshotStatus struct {
	State    string
	Snapshot string
	Indices  map[string]interface{}
}

type RecoverySourceInfo struct {
	Snapshot   string
	Repository string
	Index      string
}

type ShardRecoveryInfo struct {
	Type   string
	Stage  string
	Source RecoverySourceInfo
}

type IndexRecoveryInfo struct {
	Shards []ShardRecoveryInfo
}

type RecoveryInfo map[string]IndexRecoveryInfo

type Curator struct {
	url      string
	username string
	password string
	client   *http.Client
}

var ErrBackupNotFound = errors.New("backup not found")
var ErrCuratorUnavailable = errors.New("curator return internal server error")

type BackupProvider struct {
	client     common.Client
	indexNames *common.IndexAdapter
	repoRoot   string
	Curator    *Curator
}

func NewBackupProvider(opensearchClient common.Client, curatorClient *http.Client, repoRoot string) *BackupProvider {
	logger.Info(fmt.Sprintf("Creating new backup provider, repository root is '%s'", repoRoot))
	if !strings.HasSuffix(repoRoot, "/") {
		repoRoot = repoRoot + "/"
	}
	curator := &Curator{
		url:      common.GetEnv("CURATOR_ADDRESS", ""),
		username: common.GetEnv("CURATOR_USERNAME", ""),
		password: common.GetEnv("CURATOR_PASSWORD", ""),
		client:   curatorClient,
	}
	backupService := &BackupProvider{
		client:     opensearchClient,
		indexNames: common.NewIndexAdapter(),
		repoRoot:   repoRoot,
		Curator:    curator,
	}
	return backupService
}

func (bp BackupProvider) CollectBackupHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := common.PrepareContext(r)
		logger.InfoContext(ctx, fmt.Sprintf("Request to collect new backup in '%s' is received", r.URL.Path))
		keys, ok := r.URL.Query()["allowEviction"]
		if ok {
			// Actually we do nothing in this case because OpenSearch stores snapshots as long as possible
			logger.InfoContext(ctx, fmt.Sprintf("'allowEviction' property is set to '%s'", keys[0]))
		}
		decoder := json.NewDecoder(r.Body)
		var databases []string
		err := decoder.Decode(&databases)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to decode request from JSON", slog.String("error", err.Error()))
			common.ProcessResponseBody(ctx, w, []byte(err.Error()), http.StatusInternalServerError)
			return
		}

		defer func(Body io.ReadCloser) {
			err = Body.Close()
			if err != nil {
				logger.ErrorContext(ctx, "failed to close http response body", slog.String("error", err.Error()))
			}
		}(r.Body)

		backupID, err := bp.CollectBackup(databases, ctx)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to create snapshot", slog.String("error", err.Error()))
			common.ProcessResponseBody(ctx, w, []byte(err.Error()), http.StatusInternalServerError)
			return
		}

		response, err := bp.TrackBackup(backupID, ctx)
		if err != nil {
			logger.ErrorContext(ctx, "failed to create snapshot, curator return an error", slog.String("error", err.Error()))
			msg := fmt.Sprintf("failed to create snapshot, curator return an error: %s", err.Error())
			common.ProcessResponseBody(ctx, w, []byte(msg), http.StatusInternalServerError)
			return
		}

		responseBody, err := json.Marshal(response)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to marshal response to JSON", slog.String("error", err.Error()))
			common.ProcessResponseBody(ctx, w, []byte(err.Error()), http.StatusInternalServerError)
			return
		}

		common.ProcessResponseBody(ctx, w, responseBody, http.StatusAccepted)
	}
}

func (bp BackupProvider) DeleteBackupHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := common.PrepareContext(r)
		logger.InfoContext(ctx, fmt.Sprintf("Request to delete backup in '%s' is received", r.URL.Path))
		vars := mux.Vars(r)
		backupID := vars["backupID"]

		responseBody, status, err := bp.DeleteBackup(backupID, ctx)
		if err != nil {
			logger.ErrorContext(ctx, "failed to delete backup", slog.String("error", err.Error()))
			statusCode := http.StatusInternalServerError
			if errors.Is(err, ErrBackupNotFound) {
				statusCode = http.StatusNotFound
			}

			w.WriteHeader(statusCode)
			_, _ = w.Write([]byte(err.Error()))

			return
		}

		if status > 200 {
			w.WriteHeader(status)
			_, _ = w.Write(responseBody)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte{})
	}
}

func (bp BackupProvider) TrackBackupHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := common.PrepareContext(r)
		logger.InfoContext(ctx, fmt.Sprintf("Request to track backup in '%s' is received", r.URL.Path))
		vars := mux.Vars(r)
		trackID := vars["backupID"]
		response, err := bp.TrackBackup(trackID, ctx)
		if err != nil {
			if errors.Is(err, ErrBackupNotFound) {
				w.WriteHeader(http.StatusNotFound)
			} else {
				logger.ErrorContext(ctx, "Failed to marshal response to JSON", slog.Any("error", err))
				common.ProcessResponseBody(ctx, w, []byte(err.Error()), http.StatusInternalServerError)
				return
			}
		}

		responseBody, err := json.Marshal(response)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to marshal response to JSON", slog.Any("error", err))
			common.ProcessResponseBody(ctx, w, []byte(err.Error()), http.StatusInternalServerError)
			return
		}

		common.ProcessResponseBody(ctx, w, responseBody, 0)
	}
}

func (bp BackupProvider) RestoreBackupHandler(repo string, basePath string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := common.PrepareContext(r)
		vars := mux.Vars(r)
		backupID := vars["backupID"]
		logger.InfoContext(ctx, fmt.Sprintf("Request to restore '%s' backup is received", backupID))
		decoder := json.NewDecoder(r.Body)
		var databases []string
		err := decoder.Decode(&databases)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to decode request from JSON", slog.Any("error", err))
			common.ProcessResponseBody(ctx, w, []byte(err.Error()), http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		regenerateNames := r.URL.Query().Get("regenerateNames") == "true"
		changedNameDb, err := bp.RestoreBackup(backupID, databases, repo, regenerateNames, ctx)
		if err != nil {
			logMsg := "failed to restore backup, internal server error occur"
			statusCode := http.StatusInternalServerError
			if errors.Is(err, ErrBackupNotFound) {
				logMsg = "failed to restore backup, the backup is not found"
				statusCode = http.StatusNotFound
			}
			logger.ErrorContext(ctx, logMsg, slog.String("error", err.Error()))
			common.ProcessResponseBody(ctx, w, []byte(err.Error()), statusCode)
			return
		}

		response, err := bp.TrackRestore(backupID, ctx, changedNameDb)
		if err != nil {
			logger.ErrorContext(ctx, "restore backup is failed", slog.String("error", err.Error()))
			w.WriteHeader(http.StatusInternalServerError)
			_, err = w.Write([]byte(err.Error()))
			if err != nil {
				logger.ErrorContext(ctx, "failed to write bytes to the http body response")
			}
		}

		if regenerateNames {
			var indices []string
			indices, err = bp.getActualIndices(backupID, repo, changedNameDb, ctx)
			if err != nil {
				logger.ErrorContext(ctx, "Failed to receive indices from snapshot", slog.String("error", err.Error()))
				common.ProcessResponseBody(ctx, w, []byte(err.Error()), http.StatusInternalServerError)
				return
			}
			trackPath := fmt.Sprintf("%s/backups/track/restoring/backups/%s/indices/%s",
				basePath,
				backupID,
				strings.Join(indices, ","),
			)
			response.TrackPath = &trackPath
		}

		responseBody, err := json.Marshal(response)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to marshal response to JSON", slog.String("error", err.Error()))
			common.ProcessResponseBody(ctx, w, []byte(err.Error()), http.StatusInternalServerError)
			return
		}
		common.ProcessResponseBody(ctx, w, responseBody, 0)
	}
}

func (bp BackupProvider) RestorationBackupHandler(repo string, basePath string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := common.PrepareContext(r)
		vars := mux.Vars(r)
		backupID := vars["backupID"]
		logger.InfoContext(ctx, fmt.Sprintf("Request to restore '%s' backup is received", backupID))

		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				logger.Error("failed to close http body response")
			}
		}(r.Body)

		body, err := io.ReadAll(r.Body)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to decode request body", slog.String("error", err.Error()))
			common.ProcessResponseBody(ctx, w, []byte(err.Error()), http.StatusInternalServerError)
			return
		}

		var req RestorationRequest
		err = json.Unmarshal(body, &req)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to unmarshal request from JSON", slog.String("error", err.Error()))
			common.ProcessResponseBody(ctx, w, []byte(err.Error()), http.StatusInternalServerError)
			return
		}

		changedNameDb, err, trackId := bp.ProcessRestorationRequest(backupID, req, ctx)
		if err != nil {
			logger.ErrorContext(ctx, "failed to process restoration", slog.String("error", err.Error()))
			common.ProcessResponseBody(ctx, w, []byte(err.Error()), http.StatusInternalServerError)
			return
		}

		response, err := bp.TrackRestore(trackId, ctx, changedNameDb)
		if err != nil {
			errStatusCode := http.StatusInternalServerError
			logMsg := "an internal server error occurred while attempting to retrieve the recovery"
			if errors.Is(err, ErrBackupNotFound) {
				logMsg = "track restore not found"
				errStatusCode = http.StatusNotFound
			}
			logger.ErrorContext(ctx, logMsg, slog.String("error", err.Error()))
			common.ProcessResponseBody(ctx, w, []byte(err.Error()), errStatusCode)
			return
		}
		responseBody, err := json.Marshal(response)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to marshal response to JSON", slog.String("error", err.Error()))
			common.ProcessResponseBody(ctx, w, []byte(err.Error()), http.StatusInternalServerError)
			return
		}
		common.ProcessResponseBody(ctx, w, responseBody, 0)
	}
}

func (bp BackupProvider) TrackRestoreFromTrackIdHandler(fromRepo string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := common.PrepareContext(r)
		logger.InfoContext(ctx, fmt.Sprintf("Request to track restore in '%s' in '%s' repository is received", r.URL.Path, fromRepo))
		vars := mux.Vars(r)
		backupID := vars["backupID"]
		response, err := bp.TrackRestore(backupID, ctx, nil)
		if err != nil {
			errStatusCode := http.StatusInternalServerError
			logMsg := "an internal server error occurred while attempting to retrieve the recovery"
			if errors.Is(err, ErrBackupNotFound) {
				logMsg = "track restore not found"
				errStatusCode = http.StatusNotFound
			}
			logger.ErrorContext(ctx, logMsg, slog.String("error", err.Error()))
			w.WriteHeader(errStatusCode)
		}
		var responseBody []byte
		responseBody, err = json.Marshal(response)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to marshal response to JSON", slog.Any("error", err))
			common.ProcessResponseBody(ctx, w, responseBody, http.StatusInternalServerError)
			return
		}

		common.ProcessResponseBody(ctx, w, responseBody, 0)
	}
}

func (bp BackupProvider) TrackRestoreFromIndicesHandler(fromRepo string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := common.PrepareContext(r)
		logger.InfoContext(ctx, fmt.Sprintf("Request to track restore in '%s' in '%s' repository is received", r.URL.Path, fromRepo))
		vars := mux.Vars(r)
		backupID := vars["backupID"]
		indicesLine := vars["indices"]
		indices := strings.Split(indicesLine, ",")
		response := bp.TrackRestoreIndices(ctx, backupID, indices, fromRepo, nil)
		responseBody, err := json.Marshal(response)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to marshal response to JSON", slog.Any("error", err))
			common.ProcessResponseBody(ctx, w, []byte(err.Error()), http.StatusInternalServerError)
			return
		}

		common.ProcessResponseBody(ctx, w, responseBody, http.StatusOK)
	}
}

func (bp BackupProvider) CollectBackup(dbs []string, ctx context.Context) (string, error) {
	var body *strings.Reader
	if len(dbs) != 0 {
		quotedDbs := make([]string, len(dbs))
		for i, db := range dbs {
			quotedDbs[i] = fmt.Sprintf(`"%s"`, db)
		}
		body = strings.NewReader(fmt.Sprintf(`
		{	
			"allow_eviction":"False",	
			"dbs": [%s]
		}`, strings.Join(quotedDbs, ",")))
	}
	url := fmt.Sprintf("%s/%s", bp.Curator.url, "backup")
	request, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to prepare request to collect backup", slog.Any("error", err))
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")

	request.Header.Set(common.RequestIdKey, common.GetCtxStringValue(ctx, common.RequestIdKey))
	request.SetBasicAuth(bp.Curator.username, bp.Curator.password)
	response, err := bp.Curator.client.Do(request)
	if err != nil {
		logger.ErrorContext(ctx, fmt.Sprintf("Failed to create snapshot with provided database prefixes: '%v'", body))
		return "", err
	}

	defer func() {
		err = response.Body.Close()
		if err != nil {
			logger.Error("failed to close http body", slog.String("error", err.Error()))
		}
	}()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	logger.DebugContext(ctx, fmt.Sprintf("Snapshot is created: %s", responseBody))
	return string(responseBody), nil
}

func (bp BackupProvider) TrackBackup(backupID string, ctx context.Context) (ActionTrack, error) {
	logger.DebugContext(ctx, fmt.Sprintf("Request to track '%s' backup is requested",
		backupID))
	jobStatus, err := bp.getJobStatus(backupID, ctx)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to find snapshot", slog.Any("error", err))
		return backupTrack(backupID, "FAIL"), err
	}
	logger.DebugContext(ctx, fmt.Sprintf("'%s' backup status is %s", backupID, jobStatus))
	return backupTrack(backupID, jobStatus), nil
}

func (bp BackupProvider) DeleteBackup(backupID string, ctx context.Context) ([]byte, int, error) {
	url := fmt.Sprintf("%s/%s/%s", bp.Curator.url, "evict", backupID)
	request, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to prepare request to delete backup", slog.Any("error", err))
		return nil, http.StatusInternalServerError, err
	}

	request.Header.Set(common.RequestIdKey, common.GetCtxStringValue(ctx, common.RequestIdKey))
	request.SetBasicAuth(bp.Curator.username, bp.Curator.password)
	response, err := bp.Curator.client.Do(request)

	if err != nil {
		logger.ErrorContext(ctx, "failed to delete snapshot", slog.String("error", err.Error()))
		return nil, http.StatusInternalServerError, err
	}

	defer func() {
		err = response.Body.Close()
		if err != nil {
			logger.ErrorContext(ctx, "failed to close http response body", slog.String("error", err.Error()))
		}
	}()

	if response.StatusCode == http.StatusInternalServerError {
		return nil, http.StatusInternalServerError, ErrCuratorUnavailable
	}

	if response.StatusCode == http.StatusNotFound {
		return nil, http.StatusNotFound, ErrBackupNotFound
	}

	all, err := io.ReadAll(response.Body)
	if err != nil {
		logger.ErrorContext(ctx, "failed to read bytes from http response body", slog.String("error", err.Error()))
		return nil, http.StatusInternalServerError, err
	}

	return all, response.StatusCode, nil
}

func (bp BackupProvider) RestoreBackup(backupId string, dbs []string, fromRepo string, regenerateNames bool, ctx context.Context) (map[string]string, error) {
	if len(dbs) == 0 {
		logger.ErrorContext(ctx, "Database prefixes to restore are not specified")
		return nil, errors.New("database prefixes to restore are not specified")
	}
	var indices []string
	var err error
	maxLen := 0

	if regenerateNames {
		indices, err = bp.getActualIndices(backupId, fromRepo, map[string]string{}, ctx)
		if err != nil {
			return nil, err
		}
		logger.InfoContext(ctx, fmt.Sprintf("%d indices is received to restore from '%s' backup in '%s' repository: %v",
			len(indices), backupId, fromRepo, indices))
		for _, index := range indices {
			maxLen = common.Max(len(index), maxLen) // need to find the longest name to determine if bulk restore is available
			// no need to close target index, as it should not exist
		}
	}

	if regenerateNames {
		var changedNameDb = make(map[string]string)
		logger.DebugContext(ctx, fmt.Sprintf("Maximum length of restoring indices is %d", maxLen))
		prefix := bp.indexNames.NameIndex() + "_"
		if /*prefix */ len(prefix)+maxLen >= 255 /*max in OpenSearch*/ {
			logger.InfoContext(ctx, "Cannot perform bulk restoration")
			logger.WarnContext(ctx, "In names regeneration mode when restoring names are too long, restore request could take more time than expected and even time out, because restoration cannot be executed in parallel")
			// TODO can speed up overall process if use subsequent mode only for overflowing names
			for _, index := range indices {
				newName := bp.indexNames.NameIndex()
				err := bp.requestRestore(ctx, []string{index}, backupId, index, newName)
				if err != nil {
					return nil, err
				}

				changedNameDb[index] = newName

				limit := 120 //TODO configure limit, return after timeout and proceed in background
				tries := 0
				var status string
			TrackLoop:
				for tries < limit {
					tries++
					logger.DebugContext(ctx, fmt.Sprintf("Wait for %s->%s index to be restored, one second period try: %d/%d",
						index, newName, tries, limit))
					track := bp.TrackRestoreIndices(ctx, backupId, []string{newName}, fromRepo, nil)
					switch status = track.Status; status {
					case "PROCEEDING":
						logger.DebugContext(ctx, fmt.Sprintf("Wait for %s->%s index to be restored, status: %s",
							index, newName, status))
						time.Sleep(1 * time.Second)
					default:
						logger.DebugContext(ctx, fmt.Sprintf("Status is %s", status))
						break TrackLoop
					}
				}

				if status != "SUCCESS" {
					return nil, fmt.Errorf("failed to restore %s->%s, status is '%s' after %d retries",
						index, newName, status, tries)
				}
			}
		} else {
			logger.InfoContext(ctx, "Maximum index name allows to perform bulk restoration")
			err := bp.requestRestore(
				ctx,
				indices,
				backupId,
				".+",        /*any index*/
				prefix+"$0", /*renamed with new unique prefix, $0 is a whole match*/
			)
			if err != nil {
				return nil, err
			}

			for _, indexName := range indices {
				newName := prefix + indexName
				changedNameDb[indexName] = newName
			}
		}
		return changedNameDb, nil
	}

	err = bp.requestRestore(ctx, dbs, backupId, "", "")
	return nil, err
}

func (bp BackupProvider) ProcessRestorationRequest(backupId string, restorationRequest RestorationRequest, ctx context.Context) (map[string]string, error, string) {
	if len(restorationRequest.Databases) == 0 {
		logger.ErrorContext(ctx, "Databases to restore are not specified")
		return nil, errors.New("database to restore are not specified"), ""
	}
	var renames, dbs []string
	var changedDbNames map[string]string
	prefixes := make(map[string]struct{})
	for _, dabatase := range restorationRequest.Databases {
		dbs = append(dbs, fmt.Sprintf(`"%s"`, dabatase.Name))
		if restorationRequest.RegenerateNames {
			if dabatase.Prefix != "" {
				if ok, err := bp.checkPrefixUniqueness(dabatase.Prefix, ctx); ok {
					if err != nil {
						return nil, err, ""
					}
					renames = append(renames, fmt.Sprintf("%s:%s", dabatase.Name, dabatase.Prefix))
				}
			} else {
				prefix, err := core.PrepareDatabaseName(dabatase.Namespace, dabatase.Microservice, 64)
				if _, ok := prefixes[prefix]; ok {
					// Make an artificial delay for prefix creation, since it happens too fast
					// currently we can't include nanoseconds into pattern for prefix creation
					time.Sleep(1 * time.Millisecond)
					prefix, err = core.PrepareDatabaseName(dabatase.Namespace, dabatase.Microservice, 64)
				}
				if err != nil {
					logger.ErrorContext(ctx, fmt.Sprintf("Failed to regenerate name for provided database: %v", dabatase), slog.Any("error", err))
					return nil, err, ""
				}
				renames = append(renames, fmt.Sprintf("%s:%s", dabatase.Name, prefix))
				prefixes[prefix] = struct{}{}
			}
		}
	}
	if len(renames) != 0 {
		changedDbNames = make(map[string]string)
		for _, pair := range renames {
			parts := strings.Split(pair, ":")
			changedDbNames[parts[0]] = parts[1]
		}
	}
	err, trackId := bp.requestRestoration(ctx, dbs, backupId, renames)
	if err != nil {
		return nil, err, trackId
	}
	return changedDbNames, err, trackId
}

func (bp BackupProvider) TrackRestore(trackId string, ctx context.Context, changedNameDb map[string]string) (ActionTrack, error) {
	logger.InfoContext(ctx, fmt.Sprintf("Request to track '%s' restoration is received", trackId))
	jobStatus, err := bp.getJobStatus(trackId, ctx)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to find snapshot", slog.String("error", err.Error()))
		return backupTrack(trackId, "FAIL"), err
	}
	logger.DebugContext(ctx, fmt.Sprintf("'%s' backup status is %s", trackId, jobStatus))
	return restoreTrack(trackId, jobStatus, changedNameDb), nil
}

func (bp BackupProvider) checkPrefixUniqueness(prefix string, ctx context.Context) (bool, error) {
	logger.InfoContext(ctx, "Checking user prefix uniqueness during restoration with renaming")
	getUsersRequest := api.GetUsersRequest{}
	response, err := getUsersRequest.Do(ctx, bp.client)
	if err != nil {
		return false, fmt.Errorf("failed to receive users: %+v", err)
	}

	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			logger.Error("failed to close http body", slog.String("error", err.Error()))
		}
	}(response.Body)

	if response.StatusCode == http.StatusOK {
		var users map[string]basic.User
		err = common.ProcessBody(response.Body, &users)
		if err != nil {
			return false, err
		}
		for element, user := range users {
			if strings.HasPrefix(element, prefix) {
				logger.ErrorContext(ctx, fmt.Sprintf("provided prefix already exists or a part of another prefix: %+v", prefix))
				return false, fmt.Errorf("provided prefix already exists or a part of another prefix: %+v", prefix)
			}
			if user.Attributes[resourcePrefixAttributeName] != "" && strings.HasPrefix(user.Attributes[resourcePrefixAttributeName], prefix) {
				logger.ErrorContext(ctx, fmt.Sprintf("provided prefix already exists or a part of another prefix: %+v", prefix))
				return false, fmt.Errorf("provided prefix already exists or a part of another prefix: %+v", prefix)
			}
		}
	} else if response.StatusCode == http.StatusNotFound {
		return true, nil
	}
	return true, nil
}

// TrackRestoreIndices We keep this logic, but first we need to fix the problem with users and regenerate names for indexes, until then it will not work incorrectly.
func (bp BackupProvider) TrackRestoreIndices(ctx context.Context, backupId string, indices []string, repoName string, changedNameDb map[string]string) ActionTrack {
	// TODO should investigate this behavior and try to fix - elastic never return recovery in progress
	logger.InfoContext(ctx, fmt.Sprintf("Request to track indices restoration from '%s' snapshot in '%s' is received: %v",
		backupId, repoName, indices))
	if repoName == "" {
		repoName = backupId
	}

	indicesRecoveryRequest := opensearchapi.IndicesRecoveryRequest{
		Index: indices,
	}
	var info RecoveryInfo
	err := common.DoRequest(indicesRecoveryRequest, bp.client, &info, ctx)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to parse recovery info", slog.Any("error", err))
		return restoreTrack(backupId, "PROCEEDING", changedNameDb)
	}
	logger.DebugContext(ctx, fmt.Sprintf("Info on %d indices restoration from '%s' backup is received: %v",
		len(info), backupId, info))

	foundOneDone := false
	for _, indexRecInfo := range info {
		for _, shardInfo := range indexRecInfo.Shards {
			if shardInfo.Source.Snapshot == backupId && shardInfo.Source.Repository == repoName {
				if shardInfo.Stage != "DONE" && shardInfo.Stage != "done" {
					return restoreTrack(backupId, "PROCEEDING", changedNameDb)
				}
				foundOneDone = true
			}

		}
	}
	if foundOneDone {
		return restoreTrack(backupId, "SUCCESS", changedNameDb)
	}
	return restoreTrack(backupId, "PROCEEDING", changedNameDb)
}

func (bp BackupProvider) requestRestore(ctx context.Context, dbs []string, backupId string, pattern, replacement string) error {
	body := strings.NewReader(fmt.Sprintf(`
		{
			"vault": "%s",
			"skip_users_recovery": "true",		
			"dbs": ["%s"]
		%s
		}		
		`, backupId, strings.Join(dbs, ","), namesRegenerateRequestPart(pattern, replacement)))
	url := fmt.Sprintf("%s/%s", bp.Curator.url, "restore")
	request := bp.prepareRestoreRequest(ctx, url, body)
	logger.DebugContext(ctx, fmt.Sprintf("Request body built to restore '%s' backup: %v", backupId, body))
	response, err := bp.Curator.client.Do(request)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			logger.Error("failed to close http body", slog.String("error", err.Error()))
		}
	}(response.Body)

	if response.StatusCode == 404 {
		return ErrBackupNotFound
	}

	if response.StatusCode >= 500 {
		return ErrCuratorUnavailable
	}
	logger.InfoContext(ctx, fmt.Sprintf("'%s' snapshot restoration is started: %s", backupId, response.Body))
	return nil
}

func (bp BackupProvider) requestRestoration(ctx context.Context, dbs []string, backupId string, replacement []string) (error, string) {
	body := strings.NewReader(fmt.Sprintf(`
		{
			"vault": "%s",
			"skip_users_recovery": "true",
			"dbs": [%s]
		%s
		}		
		`, backupId, strings.Join(dbs, ","), prepareChangeNameRequestPart(replacement)))
	url := fmt.Sprintf("%s/%s", bp.Curator.url, "restore")
	request := bp.prepareRestoreRequest(ctx, url, body)
	logger.DebugContext(ctx, fmt.Sprintf("Request body built to restore '%s' backup: %v", backupId, body))
	response, err := bp.Curator.client.Do(request)
	if err != nil {
		return err, ""
	}

	defer func() {
		err = response.Body.Close()
		if err != nil {
			logger.Error("failed to close http body", slog.String("error", err.Error()))
		}
	}()

	trackId, err := io.ReadAll(response.Body)
	if err != nil {
		logger.ErrorContext(ctx, "Error reading body", "error", err)
		return err, ""
	}

	logger.InfoContext(ctx, fmt.Sprintf("'%s' snapshot restoration is started: %s", backupId, trackId))
	return nil, string(trackId)
}

func (bp BackupProvider) prepareRestoreRequest(ctx context.Context, url string, body io.Reader) *http.Request {
	request, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to prepare request to restore backup", slog.Any("error", err))
		panic(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(common.RequestIdKey, common.GetCtxStringValue(ctx, common.RequestIdKey))
	request.SetBasicAuth(bp.Curator.username, bp.Curator.password)
	return request
}

func (bp BackupProvider) getJobStatus(snapshotName string, ctx context.Context) (string, error) {
	url := fmt.Sprintf("%s/%s/%s", bp.Curator.url, "jobstatus", snapshotName)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to prepare request to track backup", slog.Any("error", err))
		return "FAIL", err
	}
	request.Header.Set(common.RequestIdKey, common.GetCtxStringValue(ctx, common.RequestIdKey))
	request.SetBasicAuth(bp.Curator.username, bp.Curator.password)
	response, err := bp.Curator.client.Do(request)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to process request by curator", slog.Any("error", err))
		return "FAIL", err
	}

	defer func() {
		err = response.Body.Close()
		if err != nil {
			logger.ErrorContext(ctx, "Failed to properly close the response body ")
		}
	}()

	if response.StatusCode == 404 {
		return "FAIL", ErrBackupNotFound
	}

	var jobStatus JobStatus
	err = json.NewDecoder(response.Body).Decode(&jobStatus)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to decode response from JSON", slog.Any("error", err))
		return "FAIL", fmt.Errorf("failed to decode response from JSON: %w", err)
	}

	var status string
	switch state := jobStatus.State; state {
	case "Failed":
		status = "FAIL"
	case "Successful":
		status = "SUCCESS"
	case "Queued":
		status = "PROCEEDING"
	case "Processing":
		status = "PROCEEDING"
	default:
		status = "FAIL"
	}

	return status, nil
}

func (bp BackupProvider) getSnapshotStatus(snapshotName string, repo string, ctx context.Context) (SnapshotStatus, error) {
	snapshotStatusRequest := opensearchapi.SnapshotStatusRequest{
		Repository: repo,
		Snapshot:   []string{snapshotName},
	}
	var snapshots Snapshots
	err := common.DoRequest(snapshotStatusRequest, bp.client, &snapshots, ctx)
	if err != nil {
		return SnapshotStatus{}, err
	}
	logger.DebugContext(ctx, fmt.Sprintf("Found snapshots: %v", snapshots))
	for _, snapshot := range snapshots.Snapshots {
		if snapshot.Snapshot == snapshotName {
			return snapshot, nil
		}
	}
	return SnapshotStatus{}, fmt.Errorf("failed to find '%s' snapshot in %s", snapshotName, repo)
}

func (bp BackupProvider) getActualIndices(backupId string, repoName string, changedNameDb map[string]string, ctx context.Context) ([]string, error) {
	if repoName == "" {
		repoName = backupId
	}
	snapshot, err := bp.getSnapshotStatus(backupId, repoName, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find '%s' snapshot: %v", backupId, err)
	}
	indices := snapshot.Indices

	result := make([]string, 0, len(indices))
	for name := range indices {
		newName := changedNameDb[name]
		if newName == "" {
			newName = name
		}
		result = append(result, newName)
	}
	return result, nil
}

func backupTrack(backupId string, backupStatus string) ActionTrack {
	return ActionTrack{
		Action: "BACKUP",
		Details: TrackDetails{
			LocalId: backupId,
		},
		Status:        backupStatus,
		TrackID:       backupId,
		ChangedNameDb: nil,
		TrackPath:     nil,
	}
}

func restoreTrack(backupId string, restoreStatus string, changedNameDb map[string]string) ActionTrack {
	return ActionTrack{
		Action: "RESTORE",
		Details: TrackDetails{
			LocalId: backupId,
		},
		Status:        restoreStatus,
		TrackID:       backupId,
		ChangedNameDb: changedNameDb,
		TrackPath:     nil,
	}
}

func namesRegenerateRequestPart(pattern string, replacement string) string {
	if pattern == "" {
		return ""
	}
	return fmt.Sprintf(`
		,"rename_pattern": "%s",
		"rename_replacement": "%s"
	`, pattern, replacement)
}

func prepareChangeNameRequestPart(renames []string) string {
	if len(renames) == 0 {
		return ""
	}
	entries := make([]string, len(renames))
	for i, pair := range renames {
		parts := strings.Split(pair, ":")
		entries[i] = fmt.Sprintf(`"%s":"%s"`, parts[0], parts[1])
	}
	return fmt.Sprintf(`,"changeDbNames": {%s}`, strings.Join(entries, ","))
}
