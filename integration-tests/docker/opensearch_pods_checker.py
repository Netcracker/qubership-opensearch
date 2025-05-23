# Copyright 2024-2025 NetCracker Technology Corporation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import json
import os
import sys
import time
import traceback
import requests

sys.path.append('./tests/shared/lib')
from PlatformLibrary import PlatformLibrary

ROOT_CA_CERT_PATH = '/certs/opensearch/root-ca.pem'
environ = os.environ
protocol = environ.get("OPENSEARCH_PROTOCOL", "http")
host = environ.get("OPENSEARCH_HOST", "opensearch")
port = environ.get("OPENSEARCH_PORT", "9200")
namespace = environ.get("OPENSEARCH_NAMESPACE")
username = environ.get("OPENSEARCH_USERNAME")
password = environ.get("OPENSEARCH_PASSWORD")
external = environ.get("EXTERNAL_OPENSEARCH", "false").lower() == "true"
timeout = 300

print(f"Connecting to OpenSearch at {protocol}://{host}:{port} (external={external})")
print(f"Using namespace: {namespace}")
print(f"Auth provided: {bool(username and password)}")

def run_opensearch_diagnostics():
    print("Cluster is yellow — performing diagnostics...")

    def get_json(endpoint):
        resp = requests.get(f"{protocol}://{host}:{port}{endpoint}", auth=auth, verify=verify)
        if resp.status_code != 200:
            print(f"❌ Failed to GET {endpoint}, status {resp.status_code}")
            return []
        return json.loads(resp.content.decode('utf-8'))

    try:
        # Cluster settings
        settings = get_json("/_cluster/settings?include_defaults=true&flat_settings=true")
        routing_allocation = settings.get("persistent", {}).get("cluster.routing.allocation.enable") or \
                             settings.get("transient", {}).get("cluster.routing.allocation.enable") or \
                             settings.get("defaults", {}).get("cluster.routing.allocation.enable")
        if routing_allocation:
            print(f"Routing allocation setting: cluster.routing.allocation.enable = {routing_allocation}")

        # Cluster stats
        cluster_stats = get_json("/_cluster/stats")
        print("Cluster stats:")
        print(json.dumps(cluster_stats.get("nodes", {}), indent=2))

        # Node stats
        node_stats = get_json("/_nodes/stats")
        print("Node stats (heap, fs):")
        for node_id, stats in node_stats.get("nodes", {}).items():
            heap_used = stats.get("jvm", {}).get("mem", {}).get("heap_used_percent")
            disk_used = stats.get("fs", {}).get("total", {}).get("disk_used_percent")
            print(f"Node: {stats.get('name')} - Heap Used: {heap_used}%, Disk Used: {disk_used}%")

    except Exception as e:
        print(f"Diagnostic error: {e}")
        traceback.print_exc()

if __name__ == '__main__':
    try:
        platform_library = PlatformLibrary(managed_by_operator="true")
    except Exception as e:
        print(f"Failed to initialize PlatformLibrary: {e}")
        traceback.print_exc()
        exit(1)

    start_time = time.time()
    url = f'{protocol}://{host}:{port}/_cat/health?v&h=status&format=json'
    auth = (username, password) if username and password else None

    while timeout > time.time() - start_time:
        print(f"Checking OpenSearch readiness... Elapsed: {int(time.time() - start_time)}s")
        time.sleep(10)
        try:
            wait_for_replicas_readiness = False
            if not external:
                print(f"Checking statefulsets readiness for host={host}")
                stateful_set_names = platform_library.get_stateful_set_names_by_label(namespace, host, 'app')
                print(f"Found statefulsets: {stateful_set_names}")
                for stateful_set_name in stateful_set_names:
                    stateful_set = platform_library.get_stateful_set(stateful_set_name, namespace)
                    replicas = stateful_set.status.replicas
                    ready = stateful_set.status.ready_replicas
                    updated = stateful_set.status.updated_replicas
                    print(f"{stateful_set_name} replicas: {replicas}, ready: {ready}, updated: {updated}")
                    if not replicas or replicas != ready or replicas != updated:
                        print(f'{stateful_set_name} is not ready yet')
                        wait_for_replicas_readiness = True
                        break
            if wait_for_replicas_readiness:
                continue

            verify = ROOT_CA_CERT_PATH if protocol == 'https' and os.path.exists(ROOT_CA_CERT_PATH) else None
            print(f"Sending request to {url}")
            response = requests.get(url, auth=auth, verify=verify)
            print(f"Response code: {response.status_code}, content: {response.text}")
            if response.status_code == 200:
                try:
                    status = json.loads(response.content.decode('utf-8'))[0]['status']
                    print(f"Cluster status: {status}")
                    if status == 'green':
                        print('OpenSearch is ready. Waiting for subsidiary components for 30 seconds')
                        time.sleep(30)
                        exit(0)
                    elif status == 'yellow':
                        print("Cluster is yellow — performing diagnostics...")
                        indices_url = f'{protocol}://{host}:{port}/_cat/indices?v&format=json'
                        nodes_url = f'{protocol}://{host}:{port}/_cat/nodes?v&format=json'
                        shards_url = f'{protocol}://{host}:{port}/_cat/shards?v&format=json'

                        indices_resp = requests.get(indices_url, auth=auth, verify=verify)
                        nodes_resp = requests.get(nodes_url, auth=auth, verify=verify)
                        shards_resp = requests.get(shards_url, auth=auth, verify=verify)

                        print("Indices:")
                        print(indices_resp.text)

                        print("Nodes:")
                        print(nodes_resp.text)

                        print("Shards:")
                        print(shards_resp.text)

                    def get_json(endpoint):
                        resp = requests.get(f"{protocol}://{host}:{port}{endpoint}", auth=auth, verify=verify)
                        if resp.status_code != 200:
                            print(f"❌ Failed to GET {endpoint}, status {resp.status_code}")
                            return []
                        return json.loads(resp.content.decode('utf-8'))

                    try:
                        # Cluster settings
                        settings = get_json("/_cluster/settings?include_defaults=true&flat_settings=true")
                        routing_allocation = settings.get("persistent", {}).get("cluster.routing.allocation.enable") or \
                                             settings.get("transient", {}).get("cluster.routing.allocation.enable") or \
                                             settings.get("defaults", {}).get("cluster.routing.allocation.enable")
                        if routing_allocation:
                            print(f"Routing allocation setting: cluster.routing.allocation.enable = {routing_allocation}")

                        # Cluster stats
                        cluster_stats = get_json("/_cluster/stats")
                        print("Cluster stats:")
                        print(json.dumps(cluster_stats.get("nodes", {}), indent=2))

                        # Node stats
                        node_stats = get_json("/_nodes/stats")
                        print("Node stats (heap, fs):")
                        for node_id, stats in node_stats.get("nodes", {}).items():
                            heap_used = stats.get("jvm", {}).get("mem", {}).get("heap_used_percent")
                            disk_used = stats.get("fs", {}).get("total", {}).get("disk_used_percent")
                            print(f"Node: {stats.get('name')} - Heap Used: {heap_used}%, Disk Used: {disk_used}%")

                        # Shards
                        shards = get_json("/_cat/shards?format=json")
                        for shard in shards:
                            if shard.get("state") == "UNASSIGNED" and shard.get("prirep") == "r":
                                explain_body = {
                                    "index": shard["index"],
                                    "shard": int(shard["shard"]),
                                    "primary": False
                                }
                                headers = {'Content-Type': 'application/json'}
                                explain_resp = requests.post(
                                    f"{protocol}://{host}:{port}/_cluster/allocation/explain",
                                    auth=auth, verify=verify,
                                    headers=headers,
                                    data=json.dumps(explain_body)
                                )
                                if explain_resp.status_code == 200:
                                    explanation = explain_resp.json()
                                    print("Shard allocation explanation:")
                                    print(json.dumps(explanation, indent=2))
                                else:
                                    print("Failed to get allocation explanation")
                                break

                    except Exception as e:
                        print(f"Diagnostic error: {e}")
                        traceback.print_exc()

                except (json.JSONDecodeError, KeyError, IndexError) as e:
                    print(f"Failed to parse JSON response: {e}")
                    print(f"Raw response content: {response.text}")
        except requests.exceptions.RequestException as e:
            print(f"RequestException while connecting to OpenSearch: {e}")
        except Exception as e:
            print(f"Unexpected error: {e}")
            traceback.print_exc()

    print("Timeout reached. OpenSearch is not ready.")
    exit(1)
