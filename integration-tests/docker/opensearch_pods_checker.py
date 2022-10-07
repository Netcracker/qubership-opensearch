import json
import os
import sys
import time

import requests

sys.path.append('./tests/shared/lib')
from PlatformLibrary import PlatformLibrary

environ = os.environ
protocol = environ.get("OPENSEARCH_PROTOCOL", "http")
host = environ.get("OPENSEARCH_HOST", "opensearch")
port = environ.get("OPENSEARCH_PORT", "9200")
namespace = environ.get("OPENSEARCH_NAMESPACE")
username = environ.get("OPENSEARCH_USERNAME")
password = environ.get("OPENSEARCH_PASSWORD")
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
            verify = '/certs/opensearch/root-ca.pem' if protocol == 'https' else None
            response = requests.get(url, auth=auth, verify=verify)
            if response.status_code == 200:
                status = json.loads(response.content.decode('utf-8'))[0]['status']
                if status == 'green':
                    print('OpenSearch is in "green" status. Waiting for subsidiary components for 30 seconds')
                    time.sleep(30)
                    exit(0)
        except Exception as e:
            print(f'Connection with OpenSearch has not established yet: {e}')
    exit(1)
