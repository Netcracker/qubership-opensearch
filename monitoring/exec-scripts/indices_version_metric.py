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

import logging
import os
from logging.handlers import RotatingFileHandler
from collections import defaultdict

import requests

logger = logging.getLogger(__name__)

REQUEST_TIMEOUT = 7
ELASTICSEARCH_USERNAME = os.getenv('ELASTICSEARCH_USERNAME')
ELASTICSEARCH_PASSWORD = os.getenv('ELASTICSEARCH_PASSWORD')
ROOT_CA_CERTIFICATE = os.getenv('ROOT_CA_CERTIFICATE')


def __configure_logging(log):
    log.setLevel(logging.DEBUG)
    formatter = logging.Formatter(fmt='[%(asctime)s,%(msecs)03d][%(levelname)s] %(message)s',
                                  datefmt='%Y-%m-%dT%H:%M:%S')
    log_handler = RotatingFileHandler(filename='/opt/elasticsearch-monitoring/exec-scripts/indices_version_metric.log',
                                      maxBytes=50 * 1024,
                                      backupCount=5)
    log_handler.setFormatter(formatter)
    log_handler.setLevel(logging.DEBUG if os.getenv('ELASTICSEARCH_MONITORING_SCRIPT_DEBUG') else logging.INFO)
    log.addHandler(log_handler)
    err_handler = RotatingFileHandler(filename='/opt/elasticsearch-monitoring/exec-scripts/indices_version_metric.err',
                                      maxBytes=50 * 1024,
                                      backupCount=5)
    err_handler.setFormatter(formatter)
    err_handler.setLevel(logging.ERROR)
    log.addHandler(err_handler)


def _parse_version_created(version_created_num: int) -> str:
    """
    Parse OpenSearch version from version.created number.
    
    The version.created field is encoded and needs to be XOR'd with 0x08000000
    to get the real version number.
    
    After XOR, the format is: XXYYZZ99 where:
    - XX = major version
    - YY = minor version (zero-padded)
    - ZZ = patch version (zero-padded)
    - 99 = suffix to remove
    
    Real examples:
      135249527 ^ 0x08000000 = 1031799 → remove 99 → 10317 → version 1.3.17
      136327927 ^ 0x08000000 = 2110199 → remove 99 → 21101 → version 2.11.1
    """
    # XOR with 0x08000000 to decode
    decoded = version_created_num ^ 0x08000000
    
    # Remove last 2 digits (99 suffix)
    version_num = decoded // 100
    
    # Extract components
    major = version_num // 10000
    minor = (version_num // 100) % 100
    patch = version_num % 100
    
    return f"{major}.{minor}.{patch}"


def _get_version_num(version_created_num: int) -> int:
    """
    Convert version.created to a version number for comparison and alerting.
    
    Decodes the version using XOR with 0x08000000 and returns XXYYZZ format:
    - XX = major version * 10000
    - YY = minor version * 100
    - ZZ = patch version
    
    This allows alerts to trigger when version_num < 20000 (i.e., 1.x indices exist)
    
    Examples:
      135249527 → 10317 (1.3.17)
      136327927 → 21101 (2.11.1)
    """
    # XOR with 0x08000000 to decode
    decoded = version_created_num ^ 0x08000000
    
    # Remove last 2 digits (99 suffix)
    version_num = decoded // 100
    
    return version_num


def _get_indices_settings(elasticsearch_url: str):
    """
    Get all indices settings from OpenSearch.
    Returns dict with index names as keys and version.created as values.
    """
    try:
        verify = ROOT_CA_CERTIFICATE if ROOT_CA_CERTIFICATE else None
        response = requests.get(
            f'{elasticsearch_url}/*/_settings',
            auth=(ELASTICSEARCH_USERNAME, ELASTICSEARCH_PASSWORD),
            timeout=(3, REQUEST_TIMEOUT),
            verify=verify)
        
        if response.status_code == 200:
            return response.json()
        else:
            logger.warning(f'Failed to get indices settings. Status code: {response.status_code}')
            return {}
    except Exception:
        logger.exception('Failed to retrieve indices settings from OpenSearch.')
        return {}


def _collect_metrics(elasticsearch_url: str):
    """
    Collect metrics about indices grouped by their creation version.
    """
    logger.info('Start to collect indices version metrics')
    
    indices_settings = _get_indices_settings(elasticsearch_url)
    
    if not indices_settings:
        logger.warning('No indices settings retrieved')
        return []
    
    # Group indices by version
    version_counts = defaultdict(int)
    
    for index_name, index_data in indices_settings.items():
        try:
            # Skip system indices if needed
            if index_name.startswith('.'):
                logger.debug(f'Skipping system index: {index_name}')
                continue
            
            version_created = int(index_data.get('settings', {}).get('index', {}).get('version', {}).get('created', 0))
            
            if version_created == 0:
                logger.warning(f'Index {index_name} has no version.created field')
                continue
            
            version_str = _parse_version_created(version_created)
            version_num = _get_version_num(version_created)
            
            # Use tuple as key: (version_string, version_num)
            version_counts[(version_str, version_num)] += 1
            
        except Exception as e:
            logger.warning(f'Failed to process index {index_name}: {e}')
            continue
    
    # Generate metrics in InfluxDB line protocol format
    metrics = []
    for (version_str, version_num), count in version_counts.items():
        metric_line = f'opensearch_indices_version_created,version="{version_str}",version_num={version_num} count={count}i'
        metrics.append(metric_line)
        logger.info(f'Metric: {metric_line}')
    
    return metrics


def run():
    try:
        logger.info('Start script execution...')
        elasticsearch_host = os.getenv('ELASTICSEARCH_HOST')
        elasticsearch_port = os.getenv('ELASTICSEARCH_PORT')
        elasticsearch_protocol = os.getenv('ELASTICSEARCH_PROTOCOL')
        elasticsearch_url = f'{elasticsearch_protocol}://{elasticsearch_host}:{elasticsearch_port}'
        
        metrics = _collect_metrics(elasticsearch_url)
        
        if metrics:
            for metric in metrics:
                print(metric)
            logger.info(f'Collected {len(metrics)} indices version metrics')
        else:
            logger.warning('No metrics collected')
            
    except Exception:
        logger.exception('Exception occurred during script execution:')
        raise


if __name__ == "__main__":
    __configure_logging(logger)
    run()
