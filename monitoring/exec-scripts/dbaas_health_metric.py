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


def __configure_logging(log):
    log.setLevel(logging.DEBUG)
    formatter = logging.Formatter(fmt='[%(asctime)s,%(msecs)03d][%(levelname)s] %(message)s',
                                  datefmt='%Y-%m-%dT%H:%M:%S')
    log_handler = RotatingFileHandler(filename='/opt/elasticsearch-monitoring/exec-scripts/dbaas_health_metric.log',
                                      maxBytes=50 * 1024,
                                      backupCount=5)
    log_handler.setFormatter(formatter)
    log_handler.setLevel(logging.DEBUG if os.getenv('ELASTICSEARCH_MONITORING_SCRIPT_DEBUG') else logging.INFO)
    log.addHandler(log_handler)
    err_handler = RotatingFileHandler(filename='/opt/elasticsearch-monitoring/exec-scripts/dbaas_health_metric.err',
                                      maxBytes=50 * 1024,
                                      backupCount=5)
    err_handler.setFormatter(formatter)
    err_handler.setLevel(logging.ERROR)
    log.addHandler(err_handler)


def _get_health_data(dbaas_adapter_url):
    # If request fails, it means that Elasticsearch DBaaS Adapter is not available. In this case, we should interrupt
    # all further calculations.
    try:
        response = requests.get(
            f'{dbaas_adapter_url}/health',
            timeout=(3, REQUEST_TIMEOUT))
        return response.json()
    except Exception:
        logger.exception('Elasticsearch DBaaS Adapter is not available.')
        return ''


def _get_status_code(status: str):
    if status == 'UP':
        return 5
    elif status == 'WARNING':
        return 4
    elif status == 'UNKNOWN':
        return 3
    elif status == 'PROBLEM':
        return 2
    else:
        return 1


def _collect_metrics(dbaas_adapter_url: str):
    logger.debug('Start to collect metrics')

    health = _get_health_data(dbaas_adapter_url)
    if not health:
        return 'elasticsearch_dbaas_health status=1,elastic_cluster_status=3'

    dbaas_status = health['status']
    dbaas_status_code = _get_status_code(dbaas_status)
    elasticsearch_health = health.get('opensearchHealth')
    if not elasticsearch_health:
        elasticsearch_health = health.get('elasticCluster')
    elasticsearch_status = elasticsearch_health['status']
    elasticsearch_status_code = _get_status_code(elasticsearch_status)
    return f'elasticsearch_dbaas_health status={dbaas_status_code},elastic_cluster_status={elasticsearch_status_code}'


def run():
    try:
        logger.info('Start script execution...')
        elasticsearch_dbaas_adapter_host = os.getenv('ELASTICSEARCH_DBAAS_ADAPTER_HOST')
        elasticsearch_dbaas_adapter_port = os.getenv('ELASTICSEARCH_DBAAS_ADAPTER_PORT')
        if elasticsearch_dbaas_adapter_host and elasticsearch_dbaas_adapter_port:
            elasticsearch_dbaas_adapter_url = f'http://{elasticsearch_dbaas_adapter_host}:{elasticsearch_dbaas_adapter_port}'
            message = _collect_metrics(elasticsearch_dbaas_adapter_url)
            print(message)
            logger.info(f'Collected DBaaS health metrics are {message}')
        else:
            logger.info('To calculate DBaaS metrics specify ELASTICSEARCH_DBAAS_ADAPTER_HOST and ' +
                        'ELASTICSEARCH_DBAAS_ADAPTER_PORT variables.')
    except Exception:
        logger.exception('Exception occurred during script execution:')
        raise


if __name__ == "__main__":
    __configure_logging(logger)
    run()
