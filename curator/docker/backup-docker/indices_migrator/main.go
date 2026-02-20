package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/Netcracker/dbaas-opensearch-adapter/backup"
	cl "github.com/Netcracker/dbaas-opensearch-adapter/client"
	"github.com/Netcracker/dbaas-opensearch-adapter/cluster"
	"github.com/Netcracker/dbaas-opensearch-adapter/common"
	"github.com/opensearch-project/opensearch-go/opensearchapi"
	"io"
	"k8s.io/apimachinery/pkg/util/wait"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

const mask uint64 = 0x08000000

var (
	opensearchHost               = common.GetEnv("ES_HOST", "opensearch-internal:9200")
	TLSHTTPEnabled               = strings.EqualFold(common.GetEnv("TLS_HTTP_ENABLED", "false"), "true")
	opensearchUsername           = common.GetEnv("ES_USERNAME", "opensearch")
	opensearchPassword           = common.GetEnv("ES_PASSWORD", "change")
	adapterUsername              = common.GetEnv("DBAAS_ADAPTER_USERNAME", "dbaas-aggregator")
	adapterPassword              = common.GetEnv("DBAAS_ADAPTER_PASSWORD", "dbaas-aggregator")
	adapterAddress               = common.GetEnv("DBAAS_ADAPTER_ADDRESS", "")
	snapshotRepoName             = common.GetEnv("SNAPSHOT_REPOSITORY_NAME", "")
	opensearchNamespace          = common.GetEnv("OPENSEARCH_NAMESPACE", "default")
	opensearchConfigSecretName   = common.GetEnv("OPENSEARCH_CONFIG_SECRET_NAME", "")
	opensearchSts                = common.GetEnv("OPENSEARCH_STATEFULSET_NAMES", "")
	opensearchDeployments        = common.GetEnv("OPENSEARCH_DEPLOYMENT_NAMES", "")
	opensearchOperatorDeployment = common.GetEnv("OPENSEARCH_OPERATOR_DEPLOYMENT_NAME", "")
	opensearchClientServiceName  = common.GetEnv("OPENSEARCH_CLIENT_SERVICE_NAME", "")
	log                          = common.GetLogger()
)

const (
	migrationSuffix      = "-migration"
	securityIndex        = ".opendistro_security"
	securityDisabled     = "plugins.security.disabled: true"
	clusterReadyTimeout  = 1200
	clusterReadyInterval = 15
	adapterReadyTimeout  = 240
	adapterReadyInterval = 10

	dbaasAPIVersion      = "v2"
	RecoveryIdleState    = "idle"
	RecoveryRunningState = "running"
	RecoveryFailedState  = "failed"
	RecoveryDoneState    = "done"
)

type Migrator struct {
	osCluster           *cluster.Opensearch
	backupProvider      *backup.BackupProvider
	adapterClient       *AdapterClient
	backupID            string
	securityNeedsReInit bool
}

type AdapterClient struct {
	endpoint string
	username string
	password string

	httpClient *http.Client
}

type createdResp map[string]struct {
	Settings struct {
		Index struct {
			Version struct {
				Created json.Number `json:"created"`
			} `json:"version"`
		} `json:"index"`
	} `json:"settings"`
}

type nodesFSResp struct {
	Nodes map[string]struct {
		FS struct {
			Total struct {
				Available json.Number `json:"available_in_bytes"`
			} `json:"total"`
		} `json:"fs"`
	} `json:"nodes"`
}

type catIndexRow struct {
	Index     string      `json:"index"`
	StoreSize json.Number `json:"store.size"`
}

type indexSettingsResp map[string]struct {
	Settings struct {
		Index map[string]any `json:"index"`
	} `json:"settings"`
}

type indexMappingResp map[string]struct {
	Mappings json.RawMessage `json:"mappings"`
}

type countResp struct {
	Count json.Number `json:"count"`
}

type reindexAsyncResp struct {
	Task string `json:"task"`
}

type taskGetResp struct {
	Completed bool `json:"completed"`
	Error     any  `json:"error,omitempty"`
	Response  any  `json:"response,omitempty"`
}

type PerfSnapshot struct {
	NumberOfReplicas *string
	RefreshInterval  *string
}

type clusterHealthResp struct {
	Status string `json:"status"`
}

func main() {
	log.Info("Starting migration 1.x -> 2.x(created) for upgrade 2.x -> 3.x")

	var dryRun bool
	flag.BoolVar(&dryRun, "dry-run", false, "Run in dry mode (no changes applied)")
	flag.Parse()
	if dryRun {
		log.Info("===================================")
		log.Info(" DRY RUN MODE ENABLED")
		log.Info(" No changes will be applied")
		log.Info("===================================")
	}

	osCluster, err := newOpenSearchClient()
	if err != nil {
		log.Error("OpenSearch client creation failed")
		os.Exit(2)
	}
	if osCluster == nil || osCluster.Client == nil {
		log.Error("OpenSearch client is nil")
		os.Exit(2)
	}

	migrator := &Migrator{osCluster: osCluster}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Minute)
	defer cancel()

	migrator.backupProvider = backup.NewBackupProvider(osCluster.Client, cl.ConfigureCuratorClient(), snapshotRepoName)

	oneXAll, oneXBackup, err := migrator.Step1Select1xIndicesAndPrecheck(ctx)
	if err != nil {
		log.Error(fmt.Sprintf("OpenSearch 1.x index migration preparation failed: %v", err))
		os.Exit(2)
	}
	log.Info(fmt.Sprintf("Indices need migration: %#v (count=%d)", oneXAll, len(oneXAll)))
	log.Info(fmt.Sprintf("Security needs reinit: %v", migrator.securityNeedsReInit))

	if dryRun {
		return
	}

	if len(oneXAll) == 0 {
		log.Info("Nothing to migrate")
	} else {
		if backupID, berr := migrator.CollectAndWaitBackup(ctx, oneXBackup); berr != nil {
			log.Error(fmt.Sprintf("OpenSearch 1.x index migration failed: %v", berr))
			os.Exit(2)
		} else {
			migrator.backupID = backupID
		}
		if err = migrator.Step2MigrateAll1x(ctx, oneXAll); err != nil {
			log.Error(fmt.Sprintf("Step2 failed: %v", err))
			os.Exit(2)
		}
	}

	if migrator.securityNeedsReInit {
		if err = migrator.ReinitSecurity(ctx); err != nil {
			log.Error(fmt.Sprintf("Security reinitialization failed: %v", err))
			os.Exit(2)
		}
	}

	migrator.adapterClient = NewAdapterClient()
	if err = migrator.RestoreUsers(ctx); err != nil {
		log.Error(fmt.Sprintf("User recovery failed: %v", err))
		os.Exit(2)
	}

	if err = RestartOperator(ctx); err != nil {
		log.Error(fmt.Sprintf("Operator restart failed: %v", err))
		os.Exit(2)
	}

	log.Info("All done. Step3 security later.")
}

func (m *Migrator) Step1Select1xIndicesAndPrecheck(ctx context.Context) ([]string, []string, error) {
	createdMap, err := m.fetchAllCreatedVersions(ctx)
	if err != nil {
		return nil, nil, err
	}
	sizesMap, err := m.fetchAllIndexSizesBytes(ctx)
	if err != nil {
		return nil, nil, err
	}

	log.Info(fmt.Sprintf("Indices from OS: %#v (count=%d)", createdMap, len(createdMap)))

	var oneXAll []string
	var oneXBackup []string

	for idx, raw := range createdMap {
		if idx == ".opendistro_security" || idx == ".opensearch-security" {
			m.securityNeedsReInit = true
			continue
		}
		dec := decodeCreated(raw)
		if majorOf(dec) != 1 {
			continue
		}
		oneXAll = append(oneXAll, idx)
		if !strings.HasPrefix(idx, ".") {
			oneXBackup = append(oneXBackup, idx)
		}
	}

	if len(oneXAll) == 0 {
		log.Info("No indices created on OpenSearch 1.x found (excluding security index)")
		return nil, nil, nil
	}

	sort.Strings(oneXAll)
	sort.Strings(oneXBackup)

	log.Info(fmt.Sprintf("1.x indices (all): %#v (count=%d)", oneXAll, len(oneXAll)))
	log.Info(fmt.Sprintf("1.x indices for backup (no dot-prefixed): %#v (count=%d)", oneXBackup, len(oneXBackup)))

	// heaviest 1.x (по тем, что реально будем мигрировать)
	var heaviestName string
	var heaviestSize uint64
	for _, name := range oneXAll {
		size := sizesMap[name] // если нет — будет 0
		if heaviestName == "" || size > heaviestSize {
			heaviestName = name
			heaviestSize = size
		}
	}

	minAvail, err := m.getClusterMinAvailableBytes(ctx)
	if err != nil {
		return nil, nil, err
	}

	need := heaviestSize * 2
	if minAvail < need {
		return nil, nil, errors.New(
			"Disk precheck FAILED. " +
				"Min node available=" + strconv.FormatUint(minAvail, 10) +
				" bytes, need=" + strconv.FormatUint(need, 10) +
				" bytes (2x heaviest 1.x index " + heaviestName +
				", size=" + strconv.FormatUint(heaviestSize, 10) + ").",
		)
	}

	return oneXAll, oneXBackup, nil
}

func (m *Migrator) Step2MigrateAll1x(ctx context.Context, indices []string) error {
	for i, name := range indices {
		log.Info(fmt.Sprintf("---- Migrating index: %s", name))
		errMain := m.migrateOneIndex(ctx, name)
		if errMain == nil {
			log.Info(fmt.Sprintf("Migration DONE: %s", name))
			continue
		}

		if err := m.cleanupIndices(ctx, name); err != nil {
			return err
		}
		if strings.HasPrefix(name, ".") {
			log.Error(fmt.Sprintf("Index migration failed, will delete and continue. index=%s", name))
			continue
		}
		_, err := m.backupProvider.RestoreBackup(m.backupID, indices[i:], "", false, ctx)
		if err != nil {
			return err
		}
		return errMain
	}
	log.Info("Step2 DONE: migrated all 1.x indices")
	return nil
}

func (m *Migrator) migrateOneIndex(ctx context.Context, indexName string) error {
	tmp := indexName + migrationSuffix

	log.Info(fmt.Sprintf("[migrateOneIndex] start index=%s tmp=%s", indexName, tmp))

	log.Info(fmt.Sprintf("[migrateOneIndex] get settings index=%s", indexName))
	settings, err := m.getIndexSettingsIndexObject(ctx, indexName)
	if err != nil {
		return err
	}

	log.Info(fmt.Sprintf("[migrateOneIndex] get mappings index=%s", indexName))
	mappings, err := m.getIndexMappings(ctx, indexName)
	if err != nil {
		return err
	}

	log.Info(fmt.Sprintf("[migrateOneIndex] sanitize settings + apply perf tweaks index=%s", indexName))
	sanitized := sanitizeIndexSettings(settings)
	snap := applyReindexPerfTweaksWithSnapshot(sanitized)

	log.Info(fmt.Sprintf("[migrateOneIndex] set write block ON index=%s", indexName))
	if err := m.setWriteBlock(ctx, indexName, true); err != nil {
		return err
	}
	defer func() {
		log.Info(fmt.Sprintf("[migrateOneIndex] set write block OFF index=%s", indexName))
		_ = m.setWriteBlock(context.Background(), indexName, false)
	}()

	log.Info(fmt.Sprintf("[migrateOneIndex] delete tmp index if exists tmp=%s", tmp))
	if err := m.deleteIndex(ctx, tmp); err != nil {
		return err
	}

	log.Info(fmt.Sprintf("[migrateOneIndex] create tmp index tmp=%s", tmp))
	if err := m.createIndex(ctx, tmp, sanitized, mappings); err != nil {
		return err
	}

	log.Info(fmt.Sprintf("[migrateOneIndex] reindex src=%s -> dst=%s", indexName, tmp))
	if err := m.reindexWait(ctx, indexName, tmp); err != nil {
		return err
	}

	if err := m.refreshIndex(ctx, tmp); err != nil {
		return err
	}

	log.Info(fmt.Sprintf("[migrateOneIndex] count check after first reindex src=%s tmp=%s", indexName, tmp))
	srcCount, err := m.getCount(ctx, indexName)
	if err != nil {
		return err
	}
	tmpCount, err := m.getCount(ctx, tmp)
	if err != nil {
		return err
	}
	log.Info(fmt.Sprintf("[migrateOneIndex] counts after first reindex src=%s(%d) tmp=%s(%d)", indexName, srcCount, tmp, tmpCount))
	if srcCount != tmpCount {
		return errors.New(
			"count mismatch after first reindex for " + indexName +
				": src=" + strconv.FormatUint(srcCount, 10) +
				" tmp=" + strconv.FormatUint(tmpCount, 10),
		)
	}

	log.Info(fmt.Sprintf("[migrateOneIndex] delete original index index=%s", indexName))
	if err := m.deleteIndex(ctx, indexName); err != nil {
		return err
	}

	log.Info(fmt.Sprintf("[migrateOneIndex] create original index again index=%s", indexName))
	if err := m.createIndex(ctx, indexName, sanitized, mappings); err != nil {
		return err
	}

	log.Info(fmt.Sprintf("[migrateOneIndex] reindex back src=%s -> dst=%s", tmp, indexName))
	if err := m.reindexWait(ctx, tmp, indexName); err != nil {
		return err
	}

	if err := m.refreshIndex(ctx, indexName); err != nil {
		return err
	}

	log.Info(fmt.Sprintf("[migrateOneIndex] count check after second reindex index=%s tmp=%s", indexName, tmp))
	finalCount, err := m.getCount(ctx, indexName)
	if err != nil {
		return err
	}
	log.Info(fmt.Sprintf("[migrateOneIndex] counts after second reindex final=%s(%d) tmp=%s(%d)", indexName, finalCount, tmp, tmpCount))
	if finalCount != tmpCount {
		return errors.New(
			"count mismatch after second reindex for " + indexName +
				": final=" + strconv.FormatUint(finalCount, 10) +
				" tmp=" + strconv.FormatUint(tmpCount, 10),
		)
	}

	log.Info(fmt.Sprintf("[migrateOneIndex] delete tmp index tmp=%s", tmp))
	if err := m.deleteIndex(ctx, tmp); err != nil {
		return err
	}

	log.Info(fmt.Sprintf("[migrateOneIndex] restore perf settings index=%s", indexName))
	if err := m.restorePerfSettings(ctx, indexName, snap); err != nil {
		return err
	}

	log.Info(fmt.Sprintf("[migrateOneIndex] done index=%s", indexName))
	return nil
}

func (m *Migrator) fetchAllCreatedVersions(ctx context.Context) (map[string]uint64, error) {
	q := url.Values{}
	q.Set("filter_path", "*.settings.index.version.created")
	q.Set("expand_wildcards", "all")

	path := "/_all/_settings?" + q.Encode()

	resp, err := m.Do(ctx, http.MethodGet, path, nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, errors.New("GET " + path + " failed: " + strings.TrimSpace(string(raw)))
	}

	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()

	var parsed createdResp
	if err := dec.Decode(&parsed); err != nil {
		return nil, err
	}

	out := make(map[string]uint64, len(parsed))
	for indexName, v := range parsed {
		num := v.Settings.Index.Version.Created
		if num == "" {
			continue
		}
		n, err := strconv.ParseUint(num.String(), 10, 64)
		if err != nil {
			continue
		}
		out[indexName] = n
	}

	return out, nil
}

func (m *Migrator) fetchAllIndexSizesBytes(ctx context.Context) (map[string]uint64, error) {
	q := url.Values{}
	q.Set("expand_wildcards", "all")
	q.Set("bytes", "b")
	q.Set("h", "index,store.size")
	q.Set("s", "store.size:desc")
	q.Set("format", "json")

	path := "/_cat/indices?" + q.Encode()

	resp, err := m.Do(ctx, http.MethodGet, path, nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, errors.New("GET " + path + " failed: " + strings.TrimSpace(string(data)))
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()

	var rows []catIndexRow
	if err := dec.Decode(&rows); err != nil {
		return nil, err
	}

	out := make(map[string]uint64, len(rows))
	for _, r := range rows {
		if r.Index == "" || r.StoreSize == "" {
			continue
		}
		n, err := strconv.ParseUint(r.StoreSize.String(), 10, 64)
		if err != nil {
			continue
		}
		out[r.Index] = n
	}
	return out, nil
}

func (m *Migrator) getClusterMinAvailableBytes(ctx context.Context) (uint64, error) {
	q := url.Values{}
	q.Set("filter_path", "nodes.*.fs.total.available_in_bytes")
	path := "/_nodes/stats/fs?" + q.Encode()
	resp, err := m.Do(ctx, http.MethodGet, path, nil, "")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return 0, errors.New("GET " + path + " failed: " + strings.TrimSpace(string(raw)))
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var parsed nodesFSResp
	if err := dec.Decode(&parsed); err != nil {
		return 0, err
	}
	var (
		minSet bool
		minVal uint64
	)
	for _, node := range parsed.Nodes {
		num := node.FS.Total.Available
		if num == "" {
			continue
		}
		n, err := strconv.ParseUint(num.String(), 10, 64)
		if err != nil {
			continue
		}
		if !minSet || n < minVal {
			minVal = n
			minSet = true
		}
	}
	if !minSet {
		return 0, errors.New("cannot determine min available bytes: no nodes data")
	}
	return minVal, nil
}

func decodeCreated(raw uint64) uint64 {
	decoded := raw ^ mask
	major := decoded / 1_000_000
	if major == 1 || major == 2 {
		return decoded
	}
	return raw
}

func majorOf(versionID uint64) uint64 {
	return versionID / 1_000_000
}

func (m *Migrator) getIndexSettingsIndexObject(ctx context.Context, index string) (map[string]any, error) {
	path := "/" + url.PathEscape(index) + "/_settings"
	resp, err := m.Do(ctx, http.MethodGet, path, nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, errors.New("GET " + path + " failed: " + strings.TrimSpace(string(raw)))
	}

	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()

	var parsed indexSettingsResp
	if err := dec.Decode(&parsed); err != nil {
		return nil, err
	}

	v, ok := parsed[index]
	if !ok {
		for _, vv := range parsed {
			return vv.Settings.Index, nil
		}
		return nil, errors.New("settings not found for " + index)
	}
	return v.Settings.Index, nil
}

func (m *Migrator) getIndexMappings(ctx context.Context, index string) (json.RawMessage, error) {
	path := "/" + url.PathEscape(index) + "/_mapping"
	resp, err := m.Do(ctx, http.MethodGet, path, nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, errors.New("GET " + path + " failed: " + strings.TrimSpace(string(raw)))
	}

	var parsed indexMappingResp
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}

	if v, ok := parsed[index]; ok && len(v.Mappings) > 0 {
		return v.Mappings, nil
	}
	for _, vv := range parsed {
		if len(vv.Mappings) > 0 {
			return vv.Mappings, nil
		}
	}
	return nil, errors.New("mapping not found for " + index)
}

func (m *Migrator) createIndex(ctx context.Context, index string, settings map[string]any, mappings json.RawMessage) error {
	body := map[string]any{
		"settings": settings,
	}
	if len(mappings) > 0 {
		var mm any
		if err := json.Unmarshal(mappings, &mm); err == nil {
			body["mappings"] = mm
		} else {
			return errors.New("invalid mappings json for " + index)
		}
	}
	b, _ := json.Marshal(body)
	req := opensearchapi.IndicesCreateRequest{
		Index: index,
		Body:  bytes.NewReader(b),
	}
	resp, err := req.Do(ctx, m.osCluster.Client)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return errors.New("create index " + index + " failed: " + strings.TrimSpace(string(raw)))
	}
	return nil
}

func (m *Migrator) deleteIndex(ctx context.Context, index string) error {
	req := opensearchapi.IndicesDeleteRequest{Index: []string{index}}
	resp, err := req.Do(ctx, m.osCluster.Client)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return nil
	}
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return errors.New("delete index " + index + " failed: " + strings.TrimSpace(string(raw)))
	}
	return nil
}

func (m *Migrator) getCount(ctx context.Context, index string) (uint64, error) {
	path := "/" + url.PathEscape(index) + "/_count"
	resp, err := m.Do(ctx, http.MethodGet, path, nil, "")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return 0, errors.New("GET " + path + " failed: " + strings.TrimSpace(string(raw)))
	}

	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var parsed countResp
	if err := dec.Decode(&parsed); err != nil {
		return 0, err
	}
	n, err := strconv.ParseUint(parsed.Count.String(), 10, 64)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func (m *Migrator) reindexWait(ctx context.Context, src, dst string) error {
	body := map[string]any{
		"source": map[string]any{"index": src},
		"dest":   map[string]any{"index": dst},
	}
	b, _ := json.Marshal(body)

	q := url.Values{}
	q.Set("wait_for_completion", "false")
	path := "/_reindex?" + q.Encode()

	resp, err := m.Do(ctx, http.MethodPost, path, bytes.NewReader(b), "application/json")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return errors.New("reindex " + src + " -> " + dst + " failed: " + strings.TrimSpace(string(raw)))
	}

	var r reindexAsyncResp
	if err := json.Unmarshal(raw, &r); err != nil || r.Task == "" {
		return errors.New("reindex did not return task id for " + src + " -> " + dst)
	}

	log.Info(fmt.Sprintf("Reindex task started: %s (%s -> %s)", r.Task, src, dst))
	return m.waitTask(ctx, r.Task)
}

func (m *Migrator) waitTask(ctx context.Context, taskID string) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			path := "/_tasks/" + url.PathEscape(taskID)
			resp, err := m.Do(ctx, http.MethodGet, path, nil, "")
			if err != nil {
				return err
			}
			raw, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()

			if resp.StatusCode >= 400 {
				return errors.New("task get failed: " + strings.TrimSpace(string(raw)))
			}

			var tr taskGetResp
			if err := json.Unmarshal(raw, &tr); err != nil {
				return err
			}

			if tr.Completed {
				if tr.Error != nil {
					return errors.New("task failed: " + taskID)
				}
				log.Info(fmt.Sprintf("Task completed: %s", taskID))
				return nil
			}
		}
	}
}

func (m *Migrator) setWriteBlock(ctx context.Context, index string, on bool) error {
	body := map[string]any{
		"index": map[string]any{
			"blocks": map[string]any{
				"write": on,
			},
		},
	}
	b, _ := json.Marshal(body)

	path := "/" + url.PathEscape(index) + "/_settings"
	resp, err := m.Do(ctx, http.MethodPut, path, bytes.NewReader(b), "application/json")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return errors.New("set write block failed for " + index + ": " + strings.TrimSpace(string(raw)))
	}
	return nil
}

func sanitizeIndexSettings(idx map[string]any) map[string]any {
	out := make(map[string]any, len(idx))
	for k, v := range idx {
		out[k] = v
	}
	delete(out, "uuid")
	delete(out, "provided_name")
	delete(out, "creation_date")
	delete(out, "version")
	delete(out, "history_uuid")
	delete(out, "routing")

	return out
}

func (m *Migrator) Do(ctx context.Context, method, path string, body io.Reader, contentType string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, path, body)
	if err != nil {
		return nil, err
	}
	if body != nil && contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return m.osCluster.Client.Perform(req)
}

func getStringSetting(idx map[string]any, key string) *string {
	v, ok := idx[key]
	if !ok || v == nil {
		return nil
	}
	switch t := v.(type) {
	case string:
		s := t
		return &s
	case json.Number:
		s := t.String()
		return &s
	case float64:
		s := strconv.FormatUint(uint64(t), 10)
		return &s
	default:
		return nil
	}
}

func applyReindexPerfTweaksWithSnapshot(idx map[string]any) PerfSnapshot {
	snap := PerfSnapshot{
		NumberOfReplicas: getStringSetting(idx, "number_of_replicas"),
	}
	idx["number_of_replicas"] = "0"
	return snap
}

func (m *Migrator) restorePerfSettings(ctx context.Context, index string, snap PerfSnapshot) error {
	settings := map[string]any{}
	changed := false
	if snap.NumberOfReplicas != nil {
		settings["number_of_replicas"] = *snap.NumberOfReplicas
		changed = true
	}
	if !changed {
		return nil
	}
	body := map[string]any{
		"index": settings,
	}
	b, _ := json.Marshal(body)
	path := "/" + url.PathEscape(index) + "/_settings"
	resp, err := m.Do(ctx, http.MethodPut, path, bytes.NewReader(b), "application/json")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return errors.New("restore perf settings failed for " + index + ": " + strings.TrimSpace(string(raw)))
	}

	return nil
}

func newOpenSearchClient() (*cluster.Opensearch, error) {
	scheme := "http"
	if TLSHTTPEnabled {
		scheme = "https"
	}
	host, port, err := net.SplitHostPort(opensearchHost)
	if err != nil {
		panic(fmt.Errorf("invalid ES_HOST format: %w", err))
	}
	portConverted, err := strconv.Atoi(port)
	if err != nil {
		return nil, fmt.Errorf("invalid port value: %w", err)
	}
	return cluster.NewOpensearch(host, portConverted, scheme, opensearchUsername, opensearchPassword), nil
}

func (m *Migrator) CollectAndWaitBackup(ctx context.Context, dbs []string) (string, error) {
	log.Info("Collecting Backup")
	backupID, err := m.backupProvider.CollectBackup(dbs, ctx)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(backupID) == "" {
		return "", errors.New("backup id is empty")
	}
	log.Info(fmt.Sprintf("Backup started. TrackID: %s", backupID))

	interval := 10 * time.Second
	err = wait.PollUntilContextCancel(ctx, interval, true, func(ctx context.Context) (bool, error) {
		track, trErr := m.backupProvider.TrackBackup(backupID, ctx)
		if trErr != nil {
			log.Error(fmt.Sprintf("TrackBackup error (will retry): %v", trErr))
			return false, nil
		}
		st := strings.ToUpper(strings.TrimSpace(track.Status))
		log.Info(fmt.Sprintf("Backup status: %s trackId=%s", st, track.TrackID))
		log.Info(fmt.Sprintf("Backup started. TrackID: %s", backupID))
		switch st {
		case "SUCCESS":
			return true, nil
		case "FAIL", "FAILED", "ERROR":
			return false, errors.New("backup failed with status=" + st)
		default:
			return false, nil
		}
	})
	if err != nil {
		return "", err
	}
	log.Info(fmt.Sprintf("Backup completed successfully. TrackID: %s", backupID))
	return backupID, nil
}

func (m *Migrator) indexExists(ctx context.Context, index string) (bool, error) {
	req := opensearchapi.IndicesExistsRequest{Index: []string{index}}
	resp, err := req.Do(ctx, m.osCluster.Client)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		return true, nil
	}
	if resp.StatusCode == 404 {
		return false, nil
	}
	raw, _ := io.ReadAll(resp.Body)
	return false, errors.New("indices exists unexpected status for " + index + ": " + strings.TrimSpace(string(raw)))
}

func (m *Migrator) cleanupIndices(ctx context.Context, name string) error {
	dErr := m.deleteIndex(ctx, name)
	if dErr != nil {
		return dErr
	}
	dErr = m.deleteIndex(ctx, name+migrationSuffix)
	if dErr != nil {
		return dErr
	}
	return nil
}

func runKubectl(ctx context.Context, args ...string) (string, error) {
	fullArgs := append([]string{"-n", opensearchNamespace}, args...)
	log.Info(fmt.Sprintf("Executing kubectl %s", strings.Join(fullArgs, " ")))
	cmd := exec.CommandContext(ctx, "kubectl", fullArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Error(fmt.Sprintf("kubectl failed: %s", string(out)))
		return "", fmt.Errorf("kubectl error: %w", err)
	}

	return strings.TrimSpace(string(out)), nil
}

func getOpenSearchYmlFromSecret(ctx context.Context) (string, error) {
	secretName := strings.TrimSpace(os.Getenv("OPENSEARCH_CONFIG_SECRET_NAME"))
	if secretName == "" {
		return "", errors.New("OPENSEARCH_CONFIG_SECRET_NAME must be set")
	}
	out, err := runKubectl(ctx,
		"get", "secret", secretName,
		"-o", `jsonpath={.data.opensearch\.yml}`,
	)
	if err != nil {
		log.Error(fmt.Sprintf("Cannot read opensearch.yml from secret: %s", out))
		return "", err
	}
	b64 := strings.TrimSpace(out)
	if b64 == "" {
		return "", errors.New("opensearch.yml key is empty in secret " + secretName)
	}
	decoded, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

func disableSecurityInSecret(ctx context.Context) (bool, error) {
	content, err := getOpenSearchYmlFromSecret(ctx)
	if err != nil {
		return false, err
	}
	if strings.Contains(content, securityDisabled) {
		log.Info("plugins.security.disabled: true already present — nothing to add")
		return true, nil
	}
	newContent := strings.TrimRight(content, "\n") + "\n" + securityDisabled + "\n"
	log.Info(fmt.Sprintf("Adding '%s' to opensearch.yml in secret '%s'", securityDisabled, os.Getenv("OPENSEARCH_CONFIG_SECRET_NAME")))
	if err := setOpenSearchYmlInSecret(ctx, newContent); err != nil {
		return false, err
	}
	return true, nil
}

func setOpenSearchYmlInSecret(ctx context.Context, newYml string) error {
	if opensearchConfigSecretName == "" {
		return errors.New("OPENSEARCH_CONFIG_SECRET_NAME must be set")
	}
	b64 := base64.StdEncoding.EncodeToString([]byte(newYml))
	patchObj := map[string]any{
		"data": map[string]string{
			"opensearch.yml": b64,
		},
	}
	patchBytes, err := json.Marshal(patchObj)
	if err != nil {
		return err
	}
	_, err = runKubectl(ctx,
		"patch", "secret", opensearchConfigSecretName,
		"--type=merge",
		"-p", string(patchBytes),
	)
	if err != nil {
		log.Error(fmt.Sprintf("Failed to patch secret %s", opensearchConfigSecretName))
		return err
	}
	log.Info(fmt.Sprintf("Secret updated: %s key=opensearch.yml", opensearchConfigSecretName))
	return nil
}

func enableSecurityInSecret(ctx context.Context) error {
	content, err := getOpenSearchYmlFromSecret(ctx)
	if err != nil {
		return err
	}
	lines := strings.Split(content, "\n")
	newLines := make([]string, 0, len(lines))
	removed := false
	for _, ln := range lines {
		if strings.TrimSpace(ln) == securityDisabled {
			removed = true
			continue
		}
		newLines = append(newLines, ln)
	}
	if !removed {
		log.Info("plugins.security.disabled: true not present — nothing to remove")
		return nil
	}
	newContent := strings.Join(newLines, "\n")
	if newContent != "" && !strings.HasSuffix(newContent, "\n") {
		newContent += "\n"
	}
	log.Info(fmt.Sprintf("Removing '%s' from opensearch.yml in secret '%s'", securityDisabled, os.Getenv("OPENSEARCH_CONFIG_SECRET_NAME")))
	if err := setOpenSearchYmlInSecret(ctx, newContent); err != nil {
		return err
	}
	return nil
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func restartOpenSearchWorkloads(ctx context.Context) error {
	stsNames := splitCSV(opensearchSts)
	depNames := splitCSV(opensearchDeployments)
	if len(stsNames) == 0 && len(depNames) == 0 {
		return errors.New("neither OPENSEARCH_STATEFULSET_NAMES nor OPENSEARCH_DEPLOYMENT_NAMES is set; cannot restart pods")
	}
	for _, name := range stsNames {
		target := "statefulset/" + name
		_, err := runKubectl(ctx, "rollout", "restart", target)
		if err != nil {
			log.Error(fmt.Sprintf("Failed to restart %s", target))
			return err
		} else {
			log.Info(fmt.Sprintf("✓ rollout restart triggered for %s", target))
		}
	}
	for _, name := range depNames {
		target := "deployment/" + name
		_, err := runKubectl(ctx, "rollout", "restart", target)
		if err != nil {
			log.Error(fmt.Sprintf("Failed to restart %s", target))
			return err
		} else {
			log.Info(fmt.Sprintf("✓ rollout restart triggered for %s", target))
		}
	}
	return nil
}

func (m *Migrator) waitForClusterReadyHTTP(ctx context.Context, useAuth bool) bool {
	healthURL := m.osCluster.Protocol + "://" + m.osCluster.Host + "/_cluster/health?wait_for_status=green&timeout=5s"
	interval := time.Duration(clusterReadyInterval) * time.Second
	timeout := time.Duration(clusterReadyTimeout) * time.Second
	deadlineCtx, cancel := context.WithTimeout(ctx, time.Duration(clusterReadyTimeout)*time.Second)
	defer cancel()

	httpClient := cl.ConfigureClient()
	httpClient.Timeout = 15 * time.Second

	attempt := 0
	for {
		if deadlineCtx.Err() != nil {
			log.Error(fmt.Sprintf("Timeout (%s) waiting for cluster to become ready", timeout.String()))
			return false
		}
		attempt++
		if isGreen(deadlineCtx, httpClient, healthURL, useAuth, opensearchUsername, opensearchPassword) {
			log.Info(fmt.Sprintf("✓ Cluster is green and API is reachable (attempt %d)", attempt))
			return true
		}
		time.Sleep(interval)
	}
}

func isGreen(ctx context.Context, c *http.Client, url string, useAuth bool, user, pass string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}
	if useAuth {
		req.SetBasicAuth(user, pass)
	}
	resp, err := c.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return false
	}
	var parsed clusterHealthResp
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return false
	}
	return parsed.Status == "green"
}

func (m *Migrator) ReinitSecurity(ctx context.Context) error {
	log.Info("Starting security reinitialization")

	if err := disableClientServiceKubectl(ctx); err != nil {
		return err
	}
	defer func() {
		_ = enableClientServiceKubectl(context.Background())
	}()

	ok, err := disableSecurityInSecret(ctx)
	if err != nil || !ok {
		return fmt.Errorf("disable security failed: %w", err)
	}

	if err = restartOpenSearchWorkloads(ctx); err != nil {
		return errors.New("failed to restart OpenSearch workloads after disabling security")
	}

	if !m.waitForClusterReadyHTTP(ctx, false) {
		return errors.New("cluster is not ready (green) after restart with security disabled")
	}

	if err = m.deleteIndex(ctx, securityIndex); err != nil {
		return err
	}

	if err = enableSecurityInSecret(ctx); err != nil {
		return fmt.Errorf("enable security failed: %w", err)
	}

	if err = restartOpenSearchWorkloads(ctx); err != nil {
		return errors.New("failed to restart OpenSearch workloads after enabling security")
	}

	if !m.waitForClusterReadyHTTP(ctx, true) {
		return errors.New("cluster is not ready (green) after restart with security enabled")
	}

	log.Info("Security reinitialization flow completed")
	return nil
}

func NewAdapterClient() *AdapterClient {
	return &AdapterClient{
		endpoint: adapterAddress,
		username: adapterUsername,
		password: adapterPassword,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		},
	}
}

func (m *Migrator) RestoreUsers(ctx context.Context) error {
	if m.adapterClient.endpoint == "" {
		log.Info("DBaaS adapter not configured - skipping user restoration")
		return nil
	}
	log.Info("Starting DBaaS user restoration...")

	deadlineCtx, cancel := context.WithTimeout(ctx, adapterReadyTimeout)
	defer cancel()

	state := m.getRestoreState(deadlineCtx)
	if state != RecoveryRunningState {
		state = RecoveryIdleState
	}
	restoreFailed := false

	for state != RecoveryDoneState && state != RecoveryFailedState {
		if deadlineCtx.Err() != nil {
			log.Error("Timeout reached during user restoration")
			return errors.New("timeout reached during user restoration")
		}
		if state == RecoveryIdleState {
			if !m.triggerRestore(deadlineCtx) {
				restoreFailed = true
				break
			}
		}
		time.Sleep(adapterReadyInterval)
		state = m.getRestoreState(deadlineCtx)
	}

	if state == RecoveryFailedState || restoreFailed {
		log.Error(fmt.Sprintf("User restoration failed with state: %s", state))
		return errors.New("user restoration failed")
	}
	log.Info(fmt.Sprintf("✓ User restoration completed successfully (state: %s)", state))
	return nil
}

func (m *Migrator) getRestoreState(ctx context.Context) string {
	u := m.adapterClient.endpoint + "/api/" + dbaasAPIVersion +
		"/dbaas/adapter/opensearch/users/restore-password/state"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		log.Info(fmt.Sprintf("Could not build restore state request: %v", err))
		return RecoveryIdleState
	}
	if m.adapterClient.username != "" && m.adapterClient.password != "" {
		req.SetBasicAuth(m.adapterClient.username, m.adapterClient.password)
	}
	resp, err := m.adapterClient.httpClient.Do(req)
	if err != nil {
		log.Info(fmt.Sprintf("Could not get restore state: %v", err))
		return RecoveryIdleState
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		log.Info(fmt.Sprintf("Could not get restore state: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(raw))))
		return RecoveryIdleState
	}
	return strings.TrimSpace(string(raw))
}

func (m *Migrator) triggerRestore(ctx context.Context) bool {
	u := m.adapterClient.endpoint + "/api/" + dbaasAPIVersion +
		"/dbaas/adapter/opensearch/users/restore-password"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader([]byte(`{}`)))
	if err != nil {
		log.Error(fmt.Sprintf("Failed to build trigger restore request: %v", err))
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	if m.adapterClient.username != "" && m.adapterClient.password != "" {
		req.SetBasicAuth(m.adapterClient.username, m.adapterClient.password)
	}
	resp, err := m.adapterClient.httpClient.Do(req)
	if err != nil {
		log.Error(fmt.Sprintf("Failed to trigger user restoration: %v", err))
		return false
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		log.Error(fmt.Sprintf("Failed to trigger user restoration: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(raw))))
		return false
	}
	log.Info("User restoration triggered successfully")
	return true
}

func RestartOperator(ctx context.Context) error {
	if opensearchOperatorDeployment == "" {
		log.Info("OPENSEARCH_OPERATOR_DEPLOYMENT_NAME not set - skipping operator restart")
		return nil
	}
	target := "deployment/" + opensearchOperatorDeployment
	_, err := runKubectl(ctx, "rollout", "restart", target)
	if err != nil {
		return errors.New("failed to restart operator")
	}
	log.Info(fmt.Sprintf("rollout restart triggered for %s", target))
	_, err = runKubectl(ctx, "rollout", "status", target)
	if err != nil {
		return errors.New("operator rollout did not complete successfully")
	}
	log.Info("Operator restarted successfully")
	return nil
}

func disableClientServiceKubectl(ctx context.Context) error {
	if opensearchClientServiceName == "" {
		return errors.New("opensearch client name is not set")
	}
	out, err := runKubectl(ctx,
		"patch", "service", opensearchClientServiceName,
		"--type=json",
		"-p", `[{"op":"add","path":"/spec/selector/none","value":"true"}]`,
	)
	if err != nil {
		log.Error(fmt.Sprintf("Failed to disable client service: %s out=%s", opensearchClientServiceName, out))
		return err
	}
	log.Info(fmt.Sprintf("Client service disabled: %s", opensearchClientServiceName))
	return nil
}

func enableClientServiceKubectl(ctx context.Context) error {
	if opensearchClientServiceName == "" {
		return errors.New("opensearch client name is not set")
	}
	out, err := runKubectl(ctx,
		"patch", "service", opensearchClientServiceName,
		"--type=json",
		"-p", `[{"op":"remove","path":"/spec/selector/none"}]`,
	)
	if err != nil {
		low := strings.ToLower(out + " " + err.Error())
		if strings.Contains(low, "missing path") ||
			strings.Contains(low, "not found") ||
			strings.Contains(low, "does not apply") {
			log.Info(fmt.Sprintf("Client service already enabled (selector none absent): %s", opensearchClientServiceName))
			return nil
		}

		log.Error(fmt.Sprintf("Failed to enable client service: %s out=%s", opensearchClientServiceName, out))
		return err
	}
	log.Info(fmt.Sprintf("✓ Client service enabled: %s", opensearchClientServiceName))
	return nil
}

func (m *Migrator) refreshIndex(ctx context.Context, index string) error {
	req := opensearchapi.IndicesRefreshRequest{
		Index: []string{index},
	}
	resp, err := req.Do(ctx, m.osCluster.Client)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("refresh failed for %s: %s", index, strings.TrimSpace(string(body)))
	}
	log.Info("Index refreshed: ", index)
	return nil
}
