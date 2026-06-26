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

import requests

sys.path.append('./tests/shared/lib')
from PlatformLibrary import PlatformLibrary

ROOT_CA_CERT_PATH = '/certs/opensearch/root-ca.pem'
environ = os.environ
protocol = environ.get("OPENSEARCH_PROTOCOL", "http")
host = environ.get("OPENSEARCH_HOST", "opensearch")
port = environ.get("OPENSEARCH_PORT", "9200")
namespace = environ.get("OPENSEARCH_NAMESPACE")
_SECRETS_DIR = environ.get(
    "INTEGRATION_TESTS_SECRETS_DIR",
    "/etc/secrets/opensearch-integration-tests-pod-secrets",
)


def _get_secret_value(key: str) -> str:
    path = os.path.join(_SECRETS_DIR, key)
    if os.path.isfile(path):
        with open(path, encoding="utf-8") as handle:
            value = handle.read().strip()
            if value:
                return value
    return environ.get(key, "")


username = _get_secret_value("OPENSEARCH_USERNAME")
password = _get_secret_value("OPENSEARCH_PASSWORD")
external = environ.get("EXTERNAL_OPENSEARCH", False)
timeout = 300

if __name__ == '__main__':
    try:
        platform_library = PlatformLibrary(managed_by_operator="true")
    except Exception:
        exit(1)
    start_time = time.time()
    url = f'{protocol}://{host}:{port}/_cat/health?v&h=status&format=json'
    auth = None
    if username and password:
        auth = (username, password)
    while timeout > time.time() - start_time:
        time.sleep(10)
        try:
            wait_for_replicas_readiness = False
            if not external:
                stateful_set_names = platform_library.get_stateful_set_names_by_label(namespace, host, 'app')
                for stateful_set_name in stateful_set_names:
                    stateful_set = platform_library.get_stateful_set(stateful_set_name, namespace)
                    if not stateful_set.status.replicas \
                            or stateful_set.status.replicas != stateful_set.status.ready_replicas \
                            or stateful_set.status.replicas != stateful_set.status.updated_replicas:
                        print(f'{stateful_set_name} is not ready yet')
                        wait_for_replicas_readiness = True
                        break
            if wait_for_replicas_readiness:
                continue
            verify = ROOT_CA_CERT_PATH if protocol == 'https' and os.path.exists(ROOT_CA_CERT_PATH) else None
            response = requests.get(url, auth=auth, verify=verify)
            if response.status_code == 200:
                status = json.loads(response.content.decode('utf-8'))[0]['status']
                if status == 'green' or (external and status == 'yellow'):
                    print('OpenSearch is ready. Waiting for subsidiary components for 30 seconds')
                    time.sleep(30)
                    exit(0)
        except Exception as e:
            print(f'Connection with OpenSearch has not established yet: {e}')
    exit(1)
