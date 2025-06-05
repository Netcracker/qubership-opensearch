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
	"bytes"
	"fmt"
	"github.com/opensearch-project/opensearch-go/opensearchtransport"
	"io"
	"net/http"
	"strings"
)

type Client interface {
	Perform(req *http.Request) (*http.Response, error)
	Metrics() (opensearchtransport.Metrics, error)
	DiscoverNodes() error
}

type ClientStub struct {
}

type TransportStub struct{}

func NewClient() *ClientStub {
	return &ClientStub{}
}

func (cs *ClientStub) Perform(req *http.Request) (*http.Response, error) {
	method := req.Method
	path := req.URL.Path
	statusCode := http.StatusOK
	body := ""
	switch {
	case strings.HasPrefix(path, "/dbaas_opensearch_metadata/_doc"):
		index := strings.ReplaceAll(path, "/dbaas_opensearch_metadata/_doc", "")
		body = cs.metadataManipulations(index, method)
	case strings.HasPrefix(path, "/_plugins/_security/api/roles/"):
		role := strings.ReplaceAll(path, "/_plugins/_security/api/roles/", "")
		body = cs.roleManipulations(role, method)
	case strings.HasPrefix(path, "/_plugins/_security/api/rolesmapping"):
		role := strings.ReplaceAll(path, "/_plugins/_security/api/rolesmapping", "")
		body = cs.roleMappingManipulations(role, method)
	case strings.HasPrefix(path, "/_plugins/_security/api/internalusers"):
		username := strings.ReplaceAll(path, "/_plugins/_security/api/internalusers/", "")
		body = cs.userManipulations(username, method)
	case strings.HasPrefix(path, "/_index_template/"):
		template := strings.ReplaceAll(path, "/_index_template/", "")
		body = cs.templateManipulations(template, method)
	case strings.HasPrefix(path, "/_nodes/reload_secure_settings"):
		body = `{"_nodes":{"total":3,"successful":3,"failed":0},"cluster_name":"opensearch","nodes":{"ddfIN7-sT3avYl4DFZfKeg":{"name":"opensearch-1"},"jxL6tjiZTIiSjxmh6wTGvw":{"name":"opensearch-0"},"jxL6tjKlshIiSjLmh6wTGvw":{"name":"opensearch-2"}}}`
	case strings.HasPrefix(path, "/_alias/"):
		alias := strings.ReplaceAll(path, "/_alias/", "")
		body = cs.aliasManipulations(alias, method)
	case strings.Contains(path, "/_aliases/"):
		alias := path[strings.LastIndex(path, "/")+1:]
		body = cs.aliasManipulations(alias, method)
	case strings.Contains(path, "/_snapshot/snapshots/_verify"):
		body = "{\"status\": 200}"
	case strings.HasPrefix(path, "/_cat/indices"):
		body = `dbaas_metadata
dbaas_opensearch_metadata
testmine
.opendistro_security
test-new
.kibana_1
testme`
	case strings.HasPrefix(path, "/"):
		index := strings.Replace(path, "/", "", 1)
		body = cs.indexManipulations(index, method)
	default:
		return nil, fmt.Errorf("there is no option for '%s' path", path)
	}

	response := &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}
	return response, nil
}

func (cs *ClientStub) metadataManipulations(index string, method string) string {
	switch method {
	case http.MethodGet:
		return `{"found":true,"_source":{"text": "check"}}`
	case http.MethodDelete:
		return `{"result":"deleted"}`
	case http.MethodPut:
		return fmt.Sprintf(`{"_index":"dbaas_opensearch_metadata","_type":"_doc","_id":"%s","_version":1,"result":"created","_shards":{"total":2,"successful":2,"failed":0},"_seq_no":30,"_primary_term":13}`, index)
	case http.MethodPost:
		return fmt.Sprintf(`{"_index":"dbaas_opensearch_metadata","_type":"_doc","_id":"%s","_version":5,"result":"updated","_shards":{"total":2,"successful":2,"failed":0},"_seq_no":8,"_primary_term":15}`, index)
	default:
		logger.Error(fmt.Sprintf("Metadata operations do not include '%s' method", method))
		return ""
	}
}

func (cs *ClientStub) roleManipulations(name string, method string) string {
	switch method {
	case http.MethodGet:
		switch {
		case strings.Contains(name, "dml"):
			return `{"dbaas_dml_role":{"reserved":false,"hidden":false,"cluster_permissions":["cluster_composite_ops","CLUSTER_COMPOSITE_OPS","cluster:monitor/state"],"index_permissions":[{"index_patterns":["${attr.internal.resource_prefix}*"],"fls":[],"masked_fields":[],"allowed_actions":["indices:data/*","INDICES:DATA/*", "indices:admin/mapping/put"]}],"tenant_permissions":[],"static":false}}`
		case strings.Contains(name, "readonly"):
			return `{"dbaas_readonly_role":{"reserved":false,"hidden":false,"cluster_permissions":["cluster:monitor/state"],"index_permissions":[{"index_patterns":["${attr.internal.resource_prefix}*"],"fls":[],"masked_fields":[],"allowed_actions":["indices:data/read/*","INDICES:DATA/READ/*"]}],"tenant_permissions":[],"static":false}}`
		case strings.Contains(name, "ism_with_plugin"):
			return `{"dbaas_ism_role":{"reserved":false,"hidden":false,"cluster_permissions":["cluster:admin/opendistro/ism/*","cluster:monitor/state"],"index_permissions":[{"index_patterns":["*"],"fls":[],"masked_fields":[],"allowed_actions":["indices:admin/opensearch/ism/managedindex"]}],"tenant_permissions":[],"static":false}}`
		case strings.Contains(name, "ism"):
			return `{"dbaas_ism_role":{"reserved":false,"hidden":false,"cluster_permissions":["cluster:admin/opendistro/ism/*","cluster:monitor/state"],"index_permissions":[{"index_patterns":["*"],"fls":[],"masked_fields":[],"allowed_actions":["indices:admin/opensearch/ism/managedindex","indices:admin/delete","indices:admin/rollover","indices:monitor/stats"]}],"tenant_permissions":[],"static":false}}`
		default:
			return `{"dbaas_admin_role":{"reserved":false,"hidden":false,"cluster_permissions":["cluster_composite_ops","CLUSTER_COMPOSITE_OPS","cluster_manage_index_templates","indices:admin/template/*","indices:admin/index_template/*","cluster:monitor/state"],"index_permissions":[{"index_patterns":["${attr.internal.resource_prefix}*"],"fls":[],"masked_fields":[],"allowed_actions":["indices_all","INDICES_ALL"]},{"index_patterns":["*"],"fls":[],"masked_fields":[],"allowed_actions":["indices:admin/index_template/*","indices:admin/aliases/get","indices:admin/resize"]}],"tenant_permissions":[],"static":false}}`
		}
	case http.MethodDelete:
		return fmt.Sprintf(`{"status":"OK","message":"'%s' deleted."}`, name)
	case http.MethodPut:
		return fmt.Sprintf(`{"status":"CREATED","message":"'%s' created."}`, name)
	default:
		logger.Error(fmt.Sprintf("Role operations do not include '%s' method", method))
		return ""
	}
}

func (cs *ClientStub) roleMappingManipulations(name string, method string) string {
	switch method {
	case http.MethodGet:
		switch {
		case strings.Contains(name, "dml"):
			return `{"dbaas_dml_role":{"hosts":[],"users":[],"reserved":false,"hidden":false,"backend_roles":[],"and_backend_roles":[]}}`
		case strings.Contains(name, "readonly"):
			return `{"dbaas_readonly_role":{"hosts":[],"users":["41f47f9c-205d-40b3-aac2-aa1facb8bd6c_28e4236ed84147c38451874d4f86425c"],"reserved":false,"hidden":false,"backend_roles":[],"and_backend_roles":[]}}`
		case strings.Contains(name, "admin"):
			return `{"dbaas_admin_role":{"hosts":[],"users":["testuser","dbaas_b5bb036119f14412963cc0979631c99f","dbaas_2097cf024d8b41b195c0f80858506ad3","dbaas_2c73c340e5d4474e8164f4b84d3e0050","dbaas_4393bb29f5ae4ac08d5c4ad42e834c14","dbaas_25225bd58e834046ac3d3c9bf5d64ebb","dbaas_125fb5e92a9041129c52a87066b5a30d","dbaas_031e6d181e9043408ff0014d989191f6","dbaas_d62fc944e0774b11bf8dc94b618c22cb","dbaas_0f5425688e244cdcaa0f259cb702e9dc"],"reserved":false,"hidden":false,"backend_roles":[],"and_backend_roles":[]}}`
		default:
			return `{"sreqh_dbaas-index_role":{"hosts":[],"users":["testuser"],"reserved":false,"hidden":false,"backend_roles":[],"and_backend_roles":[]},"dbaas_dml_role":{"hosts":[],"users":[],"reserved":false,"hidden":false,"backend_roles":[],"and_backend_roles":[]},"dbaas_admin_role":{"hosts":[],"users":["testuser","dbaas_b5bb036119f14412963cc0979631c99f","dbaas_2097cf024d8b41b195c0f80858506ad3","dbaas_2c73c340e5d4474e8164f4b84d3e0050","dbaas_4393bb29f5ae4ac08d5c4ad42e834c14","dbaas_25225bd58e834046ac3d3c9bf5d64ebb","dbaas_125fb5e92a9041129c52a87066b5a30d","dbaas_031e6d181e9043408ff0014d989191f6","dbaas_d62fc944e0774b11bf8dc94b618c22cb","dbaas_0f5425688e244cdcaa0f259cb702e9dc"],"reserved":false,"hidden":false,"backend_roles":[],"and_backend_roles":[]},"all_access":{"hosts":[],"users":[],"reserved":false,"hidden":false,"backend_roles":[],"and_backend_roles":[],"description":"Maps admin to all_access"},"dbaas_readonly_role":{"hosts":[],"users":["41f47f9c-205d-40b3-aac2-aa1facb8bd6c_28e4236ed84147c38451874d4f86425c"],"reserved":false,"hidden":false,"backend_roles":[],"and_backend_roles":[]},"opensearch_security_anonymous":{"hosts":[],"users":[],"reserved":false,"hidden":false,"backend_roles":["opensearch_security_anonymous_backendrole"],"and_backend_roles":[]}}`
		}
	case http.MethodPut:
		return fmt.Sprintf(`{"status":"OK","message":"'%s' updated."}`, name)
	default:
		logger.Error(fmt.Sprintf("Role operations do not include '%s' method", method))
		return ""
	}
}

func (cs *ClientStub) userManipulations(name string, method string) string {
	switch method {
	case http.MethodGet:
		if strings.HasPrefix(name, "dbaas_") {
			return fmt.Sprintf(`{"%s":{"hash":"","reserved":false,"hidden":false,"backend_roles":["%s"],"attributes":{},"opendistro_security_roles":[],"static":false}}`, name, name)
		}
		prefix := strings.Split(name, "_")[0]
		return fmt.Sprintf(`{"%s":{"hash":"","reserved":false,"hidden":false,"backend_roles":["%s"],"attributes":{"resource_prefix": "%s"},"opendistro_security_roles":[],"static":false}}`, name, name, prefix)
	case http.MethodDelete:
		return fmt.Sprintf(`{"status":"OK","message":"'%s' deleted."}`, name)
	case http.MethodPut:
		return fmt.Sprintf(`{"status":"CREATED","message":"'%s' created."}`, name)
	case http.MethodPatch:
		return `{"status":"OK","message":"Resource updated."}`
	default:
		logger.Error(fmt.Sprintf("User operations do not include '%s' method", method))
		return ""
	}
}

func (cs *ClientStub) templateManipulations(name string, method string) string {
	switch method {
	case http.MethodGet:
		return fmt.Sprintf(`{"index_templates":[{"name":"%s","index_template":{"index_patterns":["test*"],"template":{"settings":{"index":{"number_of_shards":"3","number_of_replicas":"1"}}},"composed_of":[]}}]}`, name)
	case http.MethodDelete:
		return `{"acknowledged":true}`
	default:
		logger.Error(fmt.Sprintf("Template operations do not include '%s' method", method))
		return ""
	}
}

func (cs *ClientStub) aliasManipulations(name string, method string) string {
	logger.Info(fmt.Sprintf("Name is %s, method is %s", name, method))
	switch method {
	case http.MethodGet:
		return fmt.Sprintf(`{"test-news":{"aliases":{"%s":{}}}}`, name)
	case http.MethodDelete:
		return `{"acknowledged":true}`
	default:
		logger.Error(fmt.Sprintf("Alias operations do not include '%s' method", method))
		return ""
	}
}

func (cs *ClientStub) indexManipulations(name string, method string) string {
	switch method {
	case http.MethodGet:
		return fmt.Sprintf(`{"%s":{"aliases":{"fewe":{}},"mappings":{},"settings":{"index":{"creation_date":"1649406295426","number_of_shards":"3","number_of_replicas":"1","uuid":"qYw1NVdlShSfPB9dFs2qIg","version":{"created":"135238127"},"provided_name":"%s"}}}}`, name, name)
	case http.MethodDelete:
		return `{"acknowledged":true}`
	case http.MethodPut:
		return fmt.Sprintf(`{"acknowledged":true,"shards_acknowledged":true,"index":"%s"}`, name)
	default:
		logger.Error(fmt.Sprintf("[%s] index operations do not include '%s' method", name, method))
		return ""
	}
}

func (cs *ClientStub) Metrics() (opensearchtransport.Metrics, error) {
	return opensearchtransport.Metrics{}, nil
}

func (cs *ClientStub) DiscoverNodes() error {
	return nil
}

func (c *TransportStub) RoundTrip(req *http.Request) (*http.Response, error) {
	path := req.URL.Path
	statusCode := http.StatusOK
	var body string
	switch {
	case strings.HasSuffix(path, "backup"):
		body = "20240322T091826"
	case strings.HasSuffix(path, "restore"):
		body = "200"
	default:
		return nil, fmt.Errorf("there is no option for '%s' path", path)
	}

	response := &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}
	return response, nil
}
