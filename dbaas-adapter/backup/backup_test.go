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
	"github.com/Netcracker/dbaas-opensearch-adapter/common"
	"github.com/stretchr/testify/assert"
	"net/http"
	"os"
	"testing"
)

var backupProvider BackupProvider
var ctx context.Context

var opensearchClient *common.ClientStub

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	shutdown()
	os.Exit(code)
}

func setup() {
	opensearchClient = common.NewClient()
	curatorClient := &http.Client{
		Transport: &common.TransportStub{},
	}
	backupProvider = *NewBackupProvider(opensearchClient, curatorClient, "snapshots")
	ctx = context.WithValue(context.Background(), common.RequestIdKey, common.GenerateUUID())
}

func shutdown() {
	backupProvider.client = nil
}

func TestCreateBackup(t *testing.T) {
	dbs := []string{"db1", "db2"}
	backupId, err := backupProvider.CollectBackup(dbs, ctx)
	assert.Contains(t, backupId, "20240322T091826")
	assert.Nil(t, err)
}

func TestRestoreBackup(t *testing.T) {
	dbs := []string{"db1", "db2"}
	restoreInfo, err := backupProvider.RestoreBackup("dbaas_1_1", dbs, "snapshots", false, ctx)
	assert.Nil(t, err)
	assert.Nil(t, restoreInfo)
}

func TestRestoreBackupWithEmptyDatabasePrefixes(t *testing.T) {
	dbs := []string{}
	restoreInfo, err := backupProvider.RestoreBackup("dbaas_1_1", dbs, "snapshots", false, context.Background())
	assert.Nil(t, restoreInfo)
	assert.NotNil(t, err)
}
