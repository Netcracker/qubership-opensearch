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

package common

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Netcracker/dbaas-opensearch-adapter/api"
	"io"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/Netcracker/qubership-dbaas-adapter-core/pkg/dao"
	"github.com/opensearch-project/opensearch-go/opensearchapi"
	uuid "github.com/satori/go.uuid"
)

const (
	RoleNamePattern    = "dbaas_%s_role"
	AliasKind          = "alias"
	IndexKind          = "index"
	MetadataKind       = "metadataDocument"
	ResourcePrefixKind = "resourcePrefix"
	TemplateKind       = "template"
	IndexTemplateKind  = "indexTemplate"
	UserKind           = "user"
	Down               = "DOWN"
	OutOfService       = "OUT_OF_SERVICE"
	Problem            = "PROBLEM"
	Warning            = "WARNING"
	Unknown            = "UNKNOWN"
	Up                 = "UP"
	ApiV1              = "v1"
	ApiV2              = "v2"
	Http               = "http"
	Https              = "https"
	RequestIdKey       = "X-Request-Id"
)

type CorrelationID string

var logger = GetLogger()
var BasePath = GetBasePath()
var resourcePrefixAttributeName = "resource_prefix"

type Component struct {
	Address     string        `json:"address"`
	Credentials dao.BasicAuth `json:"credentials"`
}

type ComponentHealth struct {
	Status string `json:"status"`
}

type ConnectionProperties struct {
	DbName         string `json:"dbName"`
	Host           string `json:"host"`
	Port           int    `json:"port"`
	Url            string `json:"url"`
	Username       string `json:"username,omitempty"`
	Password       string `json:"password,omitempty"`
	ResourcePrefix string `json:"resourcePrefix,omitempty"`
	Role           string `json:"role,omitempty"`
	Tls            bool   `json:"tls,omitempty"`
}

type Supports struct {
	Users             bool `json:"users"`
	Settings          bool `json:"settings"`
	DescribeDatabases bool `json:"describeDatabases"`
}

type CustomLogHandler struct {
	slog.Handler
	l *log.Logger
}

type User struct {
	Attributes map[string]string `json:"attributes,omitempty"`
	Hash       string            `json:"hash"`
	Roles      []string          `json:"backend_roles"`
}

func GetBasePath() string {
	return fmt.Sprintf("/api/%s/dbaas/adapter/opensearch", GetEnv("API_VERSION", ApiV2))
}

func NewCustomLogHandler(out io.Writer) *CustomLogHandler {
	handlerOptions := &slog.HandlerOptions{}
	if _, ok := os.LookupEnv("DEBUG"); ok {
		handlerOptions.Level = slog.LevelDebug
	}

	return &CustomLogHandler{
		Handler: slog.NewTextHandler(out, handlerOptions),
		l:       log.New(out, "", 0),
	}
}

func GetLogger() *slog.Logger {
	handler := NewCustomLogHandler(os.Stdout)
	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}

func (h *CustomLogHandler) Handle(ctx context.Context, record slog.Record) error {
	level := fmt.Sprintf("[%v]", record.Level.String())
	timeStr := record.Time.Format("[2006-01-02T15:04:05.999]")
	msg := record.Message
	requestId := ctx.Value(RequestIdKey)
	if requestId == nil {
		requestId = " "
	}

	h.l.Println(timeStr, level, fmt.Sprintf("[request_id=%s] [tenant_id= ] [thread= ] [class= ]", requestId), msg)

	return nil
}

func GetCtxStringValue(ctx context.Context, key string) string {
	value := ctx.Value(key)
	return ConvertAnyToString(value)
}

func DoRequest(request opensearchapi.Request, client Client, result interface{}, ctx context.Context) error {
	response, err := request.Do(ctx, client)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	logger.DebugContext(ctx, fmt.Sprintf("Status code of request is %d", response.StatusCode))
	return ProcessBody(response.Body, result)
}

func ProcessBody(body io.ReadCloser, result interface{}) error {
	readBody, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	if len(readBody) == 0 {
		return nil
	}
	logger.Debug(fmt.Sprintf("Response body is %s", readBody))
	return json.Unmarshal(readBody, result)
}

func ProcessResponseBody(ctx context.Context, w http.ResponseWriter, responseBody []byte, status int) {
	if status > 0 {
		w.WriteHeader(status)
	}
	_, err := w.Write(responseBody)
	if err != nil {
		logger.ErrorContext(ctx, "failed to write bytes to http response", slog.String("error", err.Error()))
	}
}

func GenerateUUID() string {
	return strings.ReplaceAll(GetUUID(), "-", "")
}

func IsNotDir(info fs.DirEntry) bool {
	return !info.IsDir() && !strings.HasPrefix(info.Name(), "..")
}

func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func GetIntEnv(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return fallback
}

func ConvertStructToMap(structure interface{}) (map[string]interface{}, error) {
	var result map[string]interface{}
	body, err := json.Marshal(structure)
	if err != nil {
		return result, err
	}
	err = json.Unmarshal(body, &result)
	return result, err
}

func ConvertAnyToString(value interface{}) string {
	result, ok := value.(string)
	if !ok {
		return ""
	}
	return result
}

func Max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func GetUUID() string {
	uuidValue, err := uuid.NewV4()
	if err != nil {
		logger.Error("Failed to generate UUID", slog.Any("error", err))
		return ""
	}
	return uuidValue.String()
}

func PrepareContext(r *http.Request) context.Context {
	requestId := r.Header.Get(RequestIdKey)
	if requestId == "" {
		return context.WithValue(r.Context(), RequestIdKey, GenerateUUID())
	}
	return context.WithValue(r.Context(), RequestIdKey, requestId)
}

func CheckPrefixUniqueness(prefix string, ctx context.Context, opensearchcli Client) (bool, error) {
	logger.InfoContext(ctx, "Checking user prefix uniqueness during restoration with renaming")
	getUsersRequest := api.GetUsersRequest{}
	response, err := getUsersRequest.Do(ctx, opensearchcli)
	if err != nil {
		return false, fmt.Errorf("failed to receive users: %+v", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		var users map[string]User
		err = ProcessBody(response.Body, &users)
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
