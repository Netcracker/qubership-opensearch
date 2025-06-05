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
    log_handler = RotatingFileHandler(filename='/opt/elasticsearch-monitoring/exec-scripts/health_metric.log',
                                      maxBytes=50 * 1024,
                                      backupCount=5)
    log_handler.setFormatter(formatter)
    log_handler.setLevel(logging.DEBUG if os.getenv('ELASTICSEARCH_MONITORING_SCRIPT_DEBUG') else logging.INFO)
    log.addHandler(log_handler)
    err_handler = RotatingFileHandler(filename='/opt/elasticsearch-monitoring/exec-scripts/health_metric.err',
                                      maxBytes=50 * 1024,
                                      backupCount=5)
    err_handler.setFormatter(formatter)
    err_handler.setLevel(logging.ERROR)
    log.addHandler(err_handler)


def _get_health_code(health: str):
    if health == 'green':
        return 0
    elif health == 'yellow':
        return 6
    elif health == 'red':
        return 10
    else:
        return -1


def _get_status_code(active_nodes: str, health_code: int, elasticsearch_total_nodes_count: int):
    status_code = 0
    if not active_nodes:
        status_code = 10
    else:
        failed_nodes = elasticsearch_total_nodes_count - int(active_nodes)
        if health_code == 10:
            status_code = 10
        elif health_code == 6 or failed_nodes > 0:
            status_code = 6
    return status_code


def _get_health_data(elasticsearch_url: str, parameter: str):
    # If request fails, it means that Elasticsearch is not available. In this case, we should interrupt all further
    # calculations.
    try:
        verify = ROOT_CA_CERTIFICATE if ROOT_CA_CERTIFICATE else None
        response = requests.get(
            f'{elasticsearch_url}/_cat/health?{parameter}',
            auth=(ELASTICSEARCH_USERNAME, ELASTICSEARCH_PASSWORD),
            timeout=(3, REQUEST_TIMEOUT),
            verify=verify)
        return str(response.content.strip(), 'utf-8') if response.status_code == 200 else ''
    except Exception:
        logger.exception('Elasticsearch is not available.')
        return ''


def _collect_metrics(elasticsearch_url: str, elasticsearch_total_nodes_count: int):
    logger.info('Start to collect metrics')

    health_data = _get_health_data(elasticsearch_url, 'h=status,node.total')
    status, active_nodes = health_data.split(maxsplit=1) if health_data else ['', '']
    health_code = _get_health_code(status)
    status_code = _get_status_code(active_nodes, health_code, elasticsearch_total_nodes_count)

    message = f'elasticsearch_cluster_health status_code={status_code},health_code={health_code},' \
              f'total_number_of_nodes={elasticsearch_total_nodes_count}'
    if not active_nodes:
        message = f'{message},number_of_nodes=0i'
    logger.info(f'The message is {message}')
    return message


def run():
    try:
        logger.info('Start script execution...')
        elasticsearch_host = os.getenv('ELASTICSEARCH_HOST')
        elasticsearch_port = os.getenv('ELASTICSEARCH_PORT')
        elasticsearch_protocol = os.getenv('ELASTICSEARCH_PROTOCOL')
        elasticsearch_url = f'{elasticsearch_protocol}://{elasticsearch_host}:{elasticsearch_port}'
        elasticsearch_total_nodes_count = int(os.getenv('ELASTICSEARCH_TOTAL_NODES_COUNT'))
        message = _collect_metrics(elasticsearch_url, elasticsearch_total_nodes_count)
        print(message)
        logger.info(f'Collected Elasticsearch health metrics are {message}')
    except Exception:
        logger.exception('Exception occurred during script execution:')
        raise


if __name__ == "__main__":
    __configure_logging(logger)
    run()
