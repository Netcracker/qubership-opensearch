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

package server

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"github.com/Netcracker/dbaas-opensearch-adapter/backup"
	"github.com/Netcracker/dbaas-opensearch-adapter/basic"
	cl "github.com/Netcracker/dbaas-opensearch-adapter/client"
	"github.com/Netcracker/dbaas-opensearch-adapter/cluster"
	"github.com/Netcracker/dbaas-opensearch-adapter/common"
	"github.com/Netcracker/dbaas-opensearch-adapter/health"
	"github.com/Netcracker/dbaas-opensearch-adapter/physical"
	"github.com/Netcracker/qubership-dbaas-adapter-core/pkg/dao"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	dbaasAggregatorRegistrationAddress    = common.GetEnv("DBAAS_AGGREGATOR_REGISTRATION_ADDRESS", "http://dbaas-aggregator.dbaas:8080")
	dbaasAggregatorRegistrationUsername   = common.GetEnv("DBAAS_AGGREGATOR_REGISTRATION_USERNAME", "cluster-dba")
	dbaasAggregatorRegistrationPassword   = common.GetEnv("DBAAS_AGGREGATOR_REGISTRATION_PASSWORD", "")
	dbaasAggregatorRegistrationFixedDelay = common.GetIntEnv("DBAAS_AGGREGATOR_REGISTRATION_FIXED_DELAY_MS", 150000)
	dbaasAggregatorRegistrationRetryTime  = common.GetIntEnv("DBAAS_AGGREGATOR_REGISTRATION_RETRY_TIME_MS", 60000)
	dbaasAggregatorRegistrationRetryDelay = common.GetIntEnv("DBAAS_AGGREGATOR_REGISTRATION_RETRY_DELAY_MS", 5000)
	dbaasAggregatorPhysicalDatabaseId     = common.GetEnv("DBAAS_AGGREGATOR_PHYSICAL_DATABASE_IDENTIFIER", "unknown_opensearch")

	opensearchHost     = common.GetEnv("OPENSEARCH_HOST", "localhost")
	opensearchPort     = common.GetIntEnv("OPENSEARCH_PORT", 9200)
	opensearchProtocol = common.GetEnv("OPENSEARCH_PROTOCOL", common.Http)
	opensearchUsername = common.GetEnv("OPENSEARCH_USERNAME", "opensearch")
	opensearchPassword = common.GetEnv("OPENSEARCH_PASSWORD", "change")
	opensearchRepo     = common.GetEnv("OPENSEARCH_REPO", "dbaas-backups-repository")
	opensearchRepoRoot = common.GetEnv("OPENSEARCH_REPO_ROOT", "/usr/share/opensearch/")
	//nolint:errcheck
	enhancedSecurityPluginEnabled, _ = strconv.ParseBool(common.GetEnv("ENHANCED_SECURITY_PLUGIN_ENABLED", "false"))

	labelsFilename    = common.GetEnv("LABELS_FILE_LOCATION_NAME", "dbaas.physical_databases.registration.labels.json")
	labelsLocationDir = common.GetEnv("LABELS_FILE_LOCATION_DIR", "/app/config/")
	//nolint:errcheck
	registrationEnabled, _ = strconv.ParseBool(common.GetEnv("REGISTRATION_ENABLED", "false"))
)

const certificatesFolder = "/tls"

func Server(ctx context.Context, adapterAddress string, adapterUsername string, adapterPassword string) {
	adapter := common.Component{
		Address: adapterAddress,
		Credentials: dao.BasicAuth{
			Username: adapterUsername,
			Password: adapterPassword,
		},
	}

	hnd := Handlers(ctx, adapter)
	if hnd == nil {
		return
	}

	server := &http.Server{
		Addr:    ":8080",
		Handler: hnd,
	}

	isTlsEnabled := strings.Contains(adapterAddress, common.Https)
	logger := common.GetLogger()

	go func() {
		var err error
		if !isTlsEnabled {
			err = server.ListenAndServe()
		} else {
			err = server.ListenAndServeTLS(fmt.Sprintf("%s/tls.crt", certificatesFolder),
				fmt.Sprintf("%s/tls.key", certificatesFolder))
		}

		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.ErrorContext(ctx, "server crashed with error", slog.String("error", err.Error()))
		}
	}()

	<-ctx.Done()
	deadlineCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := server.Shutdown(deadlineCtx)
	if err != nil {
		logger.Error("failed to shutdown server")
	}
	logger.Info("server is down gracefully")
}

func Handlers(ctx context.Context, adapter common.Component) http.Handler {
	opensearch := cluster.NewOpensearch(opensearchHost, opensearchPort,
		opensearchProtocol, opensearchUsername, opensearchPassword)
	baseProvider := basic.NewBaseProvider(opensearch)
	err := baseProvider.EnsureAggregationIndex(ctx)
	if err != nil {
		return nil
	}
	registrationProvider := startRegistration(adapter.Address, adapter.Credentials.Username,
		adapter.Credentials.Password, baseProvider)
	createBasicRoles(baseProvider)
	curatorBaseClient := cl.ConfigureCuratorClient()
	backupProvider := backup.NewBackupProvider(opensearch.Client, curatorBaseClient, opensearchRepoRoot)
	basePath := fmt.Sprintf("/api/%s/dbaas/adapter/opensearch", registrationProvider.ApiVersion)

	healthService := health.Health{
		Status:                common.Up,
		OpensearchHealth:      opensearch.Health,
		DbaasAggregatorHealth: &registrationProvider.Health,
		Opensearch:            opensearch,
	}

	r := mux.NewRouter()
	authorizer := BasicAuthorizer(adapter.Credentials.Username, adapter.Credentials.Password,
		"This API is for using by DBaaS aggregator only")

	r.HandleFunc("/health", healthService.HealthHandler()).Methods(http.MethodGet)

	r.HandleFunc(fmt.Sprintf("%s/supports", basePath), baseProvider.SupportsHandler()).Methods(http.MethodGet)

	r.Handle(fmt.Sprintf("%s/databases", basePath),
		handlers.LoggingHandler(os.Stdout, authorizer(baseProvider.CreateDatabaseHandler())),
	).Methods(http.MethodPost)

	r.Handle(fmt.Sprintf("%s/databases", basePath),
		handlers.LoggingHandler(os.Stdout, authorizer(baseProvider.ListDatabasesHandler())),
	).Methods(http.MethodGet)

	r.Handle(fmt.Sprintf("%s/resources/bulk-drop", basePath),
		handlers.LoggingHandler(os.Stdout, authorizer(baseProvider.BulkDropResourceHandler())),
	).Methods(http.MethodPost)

	r.Handle(fmt.Sprintf("%s/databases/{dbName}/metadata", basePath),
		handlers.LoggingHandler(os.Stdout, authorizer(baseProvider.UpdateMetadataHandler())),
	).Methods(http.MethodPut)

	r.Handle(fmt.Sprintf("%s/backups/collect", basePath),
		handlers.LoggingHandler(os.Stdout, authorizer(backupProvider.CollectBackupHandler())),
	).Methods(http.MethodPost)

	r.Handle(fmt.Sprintf("%s/backups/{backupID}/restore", basePath),
		handlers.LoggingHandler(os.Stdout, authorizer(backupProvider.RestoreBackupHandler(opensearchRepo, basePath))),
	).Methods(http.MethodPost)

	r.Handle(fmt.Sprintf("%s/backups/{backupID}/restoration", basePath),
		handlers.LoggingHandler(os.Stdout, authorizer(backupProvider.RestorationBackupHandler(opensearchRepo, basePath))),
	).Methods(http.MethodPost)

	r.Handle(fmt.Sprintf("%s/backups/track/backup/{backupID}", basePath),
		handlers.LoggingHandler(os.Stdout, authorizer(backupProvider.TrackBackupHandler())),
	).Methods(http.MethodGet)

	r.Handle(fmt.Sprintf("%s/backups/track/restore/{backupID}", basePath),
		handlers.LoggingHandler(os.Stdout, authorizer(backupProvider.TrackRestoreFromTrackIdHandler(opensearchRepo))),
	).Methods(http.MethodGet)

	r.Handle(fmt.Sprintf("%s/backups/track/restoring/backups/{backupID}/indices/{indices}", basePath),
		handlers.LoggingHandler(os.Stdout, authorizer(backupProvider.TrackRestoreFromIndicesHandler(opensearchRepo))),
	).Methods(http.MethodGet)

	r.Handle(fmt.Sprintf("%s/backups/{backupID}", basePath),
		handlers.LoggingHandler(os.Stdout, authorizer(backupProvider.DeleteBackupHandler())),
	).Methods(http.MethodDelete)

	r.Handle(fmt.Sprintf("%s/physical_database", basePath),
		handlers.LoggingHandler(os.Stdout, authorizer(registrationProvider.GetPhysicalDatabaseHandler())),
	).Methods(http.MethodGet)

	r.Handle(fmt.Sprintf("/api/%s/dbaas/adapter/physical_database/force_registration", registrationProvider.ApiVersion),
		handlers.LoggingHandler(os.Stdout, authorizer(registrationProvider.ForceRegistrationHandler())),
	).Methods(http.MethodGet)

	r.Handle(fmt.Sprintf("%s/users", basePath),
		handlers.LoggingHandler(os.Stdout, authorizer(baseProvider.CreateUserHandler())),
	).Methods(http.MethodPut)

	r.Handle(fmt.Sprintf("%s/users/{name}", basePath),
		handlers.LoggingHandler(os.Stdout, authorizer(baseProvider.CreateUserHandler())),
	).Methods(http.MethodPut)

	if registrationProvider.ApiVersion == common.ApiV2 {
		r.Handle(fmt.Sprintf("%s/users/restore-password", basePath),
			handlers.LoggingHandler(os.Stdout, authorizer(baseProvider.RecoverUsersHandler())),
		).Methods(http.MethodPost)

		r.Handle(fmt.Sprintf("%s/users/restore-password/state", basePath),
			handlers.LoggingHandler(os.Stdout, authorizer(baseProvider.GetRecoveryStateHandler())),
		).Methods(http.MethodGet)
	}

	return JsonContentType(handlers.CompressHandler(r))
}

func JsonContentType(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() { // error handler, when error occurred it sends request with http status 400 and body with error message
			if err := recover(); err != nil {
				strErr, ok := err.(string)
				if ok {
					http.Error(w, strErr, http.StatusBadRequest)
				} else {
					http.Error(w, "unrecognized error", http.StatusInternalServerError)
				}

				return
			}
		}()
		w.Header().Set("Content-Type", "application/json")
		h.ServeHTTP(w, r)
	})
}

func startRegistration(adapterAddress string, adapterUsername string, adapterPassword string,
	baseProvider *basic.BaseProvider) *physical.RegistrationProvider {
	dbaasAggregatorCredentials := dao.BasicAuth{
		Username: dbaasAggregatorRegistrationUsername,
		Password: dbaasAggregatorRegistrationPassword,
	}
	adapterCredentials := dao.BasicAuth{
		Username: adapterUsername,
		Password: adapterPassword,
	}
	registrationService := physical.NewRegistrationProvider(
		dbaasAggregatorRegistrationAddress,
		dbaasAggregatorCredentials,
		labelsLocationDir+labelsFilename,
		nil,
		dbaasAggregatorRegistrationFixedDelay,
		dbaasAggregatorRegistrationRetryTime,
		dbaasAggregatorRegistrationRetryDelay,
		dbaasAggregatorPhysicalDatabaseId,
		adapterAddress,
		adapterCredentials,
		baseProvider,
	)
	if registrationEnabled {
		registrationService.StartRegistration()
	}
	return registrationService
}

func createBasicRoles(baseProvider *basic.BaseProvider) {
	// Migration is tracked by the role mapping, because it is created at the end of the initialization
	mapping, err := baseProvider.GetRoleMapping(fmt.Sprintf(common.RoleNamePattern, basic.AdminRoleType))
	if err != nil {
		panic(err)
	}
	if err = baseProvider.CreateRoleWithISMPermissions(enhancedSecurityPluginEnabled); err != nil {
		panic(err)
	}
	if err = baseProvider.CreateRoleWithAdminPermissions(); err != nil {
		panic(err)
	}
	if err = baseProvider.CreateRoleWithDMLPermissions(); err != nil {
		panic(err)
	}
	if err = baseProvider.CreateRoleWithReadOnlyPermissions(); err != nil {
		panic(err)
	}
	// migration is necessary if specific roles mapping does not exist
	if mapping == nil {
		if err := performMigration(baseProvider); err != nil {
			panic(err)
		}
	}
	for _, roleType := range baseProvider.GetSupportedRoleTypes() {
		if err = baseProvider.CreateOrUpdateRoleMapping(roleType); err != nil {
			panic(err)
		}
	}
}

func performMigration(baseProvider *basic.BaseProvider) error {
	rolesMapping, err := baseProvider.GetRolesMapping()
	if err != nil {
		return err
	}
	for role, mapping := range rolesMapping {
		if relevantForMigration(role, mapping) {
			if err = updateUserConfiguration(mapping.Users[0], role, baseProvider); err != nil {
				return err
			}
		}
	}
	return nil
}

func relevantForMigration(roleName string, roleMapping basic.RoleMapping) bool {
	return (strings.HasSuffix(roleName, "_role") || strings.HasSuffix(roleName, "_dml") ||
		strings.HasSuffix(roleName, "_readonly") || strings.HasSuffix(roleName, "_admin") ||
		strings.HasSuffix(roleName, "_ism")) && !roleMapping.Reserved && len(roleMapping.Users) == 1
}

func updateUserConfiguration(username string, roleName string, baseProvider *basic.BaseProvider) error {
	user, err := baseProvider.GetUser(username)
	if err != nil || user == nil {
		return err
	}
	role, err := baseProvider.GetRole(roleName)
	if err != nil || role == nil {
		return err
	}
	var pattern string
	for _, permission := range role.IndexPermissions {
		if len(permission.IndexPatterns) == 1 && permission.IndexPatterns[0] != basic.AllIndices {
			pattern = permission.IndexPatterns[0]
		}
	}
	if pattern == "" {
		return nil
	}
	roleType := baseProvider.DefineRoleType(roleName)
	return baseProvider.PatchUser(username, "", pattern, roleType, context.Background())
}

func BasicAuthorizer(username string, password string, realm string) func(func(w http.ResponseWriter, r *http.Request)) http.Handler {
	return func(f func(w http.ResponseWriter, r *http.Request)) http.Handler {
		h := http.HandlerFunc(f)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			user, pass, ok := r.BasicAuth()

			if !ok || subtle.ConstantTimeCompare([]byte(user), []byte(username)) != 1 || subtle.ConstantTimeCompare([]byte(pass), []byte(password)) != 1 {
				w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
				common.ProcessResponseBody(ctx, w, []byte("Not authorized to use this API, only DBaaS aggregator can use it.\n"), http.StatusUnauthorized)
				return
			}
			h.ServeHTTP(w, r)
		})
	}
}
