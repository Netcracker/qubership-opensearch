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
    log_handler = RotatingFileHandler(filename='/opt/elasticsearch-monitoring/exec-scripts/backup_metric.log',
                                      maxBytes=50 * 1024,
                                      backupCount=5)
    log_handler.setFormatter(formatter)
    log_handler.setLevel(logging.DEBUG if os.getenv('ELASTICSEARCH_MONITORING_SCRIPT_DEBUG') else logging.INFO)
    log.addHandler(log_handler)
    err_handler = RotatingFileHandler(filename='/opt/elasticsearch-monitoring/exec-scripts/backup_metric.err',
                                      maxBytes=50 * 1024,
                                      backupCount=5)
    err_handler.setFormatter(formatter)
    err_handler.setLevel(logging.ERROR)
    log.addHandler(err_handler)


def _is_elasticsearch_available(elasticsearch_url: str):
    # If request fails, it means that Elasticsearch is not available. In this case, we should interrupt all further
    # calculations.
    try:
        verify = ROOT_CA_CERTIFICATE if ROOT_CA_CERTIFICATE else None
        response = requests.get(
            f'{elasticsearch_url}/_cat/health',
            auth=(ELASTICSEARCH_USERNAME, ELASTICSEARCH_PASSWORD),
            timeout=(3, REQUEST_TIMEOUT),
            verify=verify)
    except Exception:
        logger.exception('Elasticsearch is not available.')
        return False
    logger.debug(f'Health response is {response}')
    return response.status_code == 200


def _get_request_with_parameter(elasticsearch_url: str, parameter):
    try:
        verify = ROOT_CA_CERTIFICATE if ROOT_CA_CERTIFICATE else None
        response = requests.get(
            f'{elasticsearch_url}/_snapshot/{parameter}',
            auth=(ELASTICSEARCH_USERNAME, ELASTICSEARCH_PASSWORD),
            timeout=(3, REQUEST_TIMEOUT),
            verify=verify)
        logger.debug(f'Response is {response.json()}')
        return response.json()
    except Exception:
        logger.exception('Elasticsearch is not available. There is no ability to perform request.')
        return ''


def _get_repositories(elasticsearch_url: str):
    logger.info(f'Receiving repositories...')
    repositories = _get_request_with_parameter(elasticsearch_url, '')
    return list(repositories.keys())


def _get_repository_storage_type(elasticsearch_url: str, repository_name: str):
    logger.info(f'Receiving repository snapshot type...')
    repository = _get_request_with_parameter(elasticsearch_url, repository_name)
    return repository[repository_name]['type']


def _get_current_status(elasticsearch_url: str):
    logger.info(f'Receiving current status...')
    snapshots_status = _get_request_with_parameter(elasticsearch_url, '_status')
    snapshots = snapshots_status['snapshots']
    return snapshots[-1]['state'] if len(snapshots) else ''


def _get_state_code(state: str):
    if state == 'SUCCESS':
        return 1
    elif state == 'STARTED':
        return 2
    elif state == 'IN_PROGRESS':
        return 3
    elif state == 'PARTIAL':
        return 4
    elif state == 'INCOMPATIBLE':
        return 5
    elif state == 'FAILED':
        return 6
    else:
        return -1


def _get_storage_code(storage: str):
    if storage == 'fs':
        return 1
    elif storage == 's3':
        return 2
    else:
        return -1


def _get_snapshots_info(elasticsearch_url: str):
    logger.info(f'Receiving snapshots info...')
    repository_names = _get_repositories(elasticsearch_url)
    logger.debug(f'Repository names are {repository_names}')
    snapshots_count = 0
    successful_snapshots_count = 0
    last_snapshot = ''
    last_repository_name = ''
    last_snapshot_time = -1
    last_successful_snapshot_time = -1
    if len(repository_names):
        for repository_name in repository_names:
            snapshots = _get_request_with_parameter(elasticsearch_url, f'{repository_name}/*')
            if 'error' not in snapshots:
                for snapshot in snapshots['snapshots']:
                    snapshots_count += 1
                    if snapshot['end_time_in_millis'] > last_snapshot_time:
                        last_repository_name = repository_name
                        last_snapshot = snapshot
                        last_snapshot_time = snapshot['end_time_in_millis']
                    if snapshot['state'] == 'SUCCESS':
                        successful_snapshots_count += 1
                        if snapshot['end_time_in_millis'] > last_successful_snapshot_time:
                            last_successful_snapshot_time = snapshot['end_time_in_millis']
        if not last_repository_name:
            last_repository_name = repository_names[-1]
    return last_snapshot, last_repository_name, snapshots_count, successful_snapshots_count, last_snapshot_time, last_successful_snapshot_time


def _collect_metrics(elasticsearch_url: str):
    logger.debug('Start to collect metrics...')

    if not _is_elasticsearch_available(elasticsearch_url):
        return 'elasticsearch_backups_metric current_status=-2'

    last_snapshot, last_repository_name, snapshots_count, successful_snapshots_count, last_snapshot_time, last_successful_snapshot_time = \
        _get_snapshots_info(elasticsearch_url)
    message = f'elasticsearch_backups_metric count_of_snapshots={snapshots_count},last_snapshot_time={last_snapshot_time},' \
              f'count_of_successful_snapshots={successful_snapshots_count},last_successful_snapshot_time={last_successful_snapshot_time}'

    if last_repository_name:
        storage_type = _get_repository_storage_type(elasticsearch_url, last_repository_name)
        message = f'{message},storage_type={_get_storage_code(storage_type)}'

    current_status_code = -1
    current_status = _get_current_status(elasticsearch_url)
    if current_status:
        current_status_code = _get_state_code(current_status)

    last_backup_status_code = -1
    if last_snapshot:
        last_snapshot_name = last_snapshot['snapshot']
        last_snapshot_status = _get_request_with_parameter(elasticsearch_url,
                                                           f'{last_repository_name}/{last_snapshot_name}/_status')
        if last_snapshot_status and 'error' not in last_snapshot_status:
            last_backup_status_code = _get_state_code(last_snapshot_status['snapshots'][-1]['state'])
            message = f'{message},last_backup_size={last_snapshot_status["snapshots"][-1]["stats"]["total"]["size_in_bytes"]},' \
                      f'last_backup_time_spent={last_snapshot_status["snapshots"][-1]["stats"]["time_in_millis"]}'
        else:
            last_backup_status_code = _get_state_code(last_snapshot['state'])
            message = f'{message},last_backup_time_spent={last_snapshot["duration_in_millis"]}'

    message = f'{message},last_backup_status={last_backup_status_code},current_status={current_status_code}'
    logger.debug(f'Final message is {message}.')
    return message


def run():
    logger.info('Start script execution...')
    try:
        elasticsearch_host = os.getenv('ELASTICSEARCH_HOST')
        elasticsearch_port = os.getenv('ELASTICSEARCH_PORT')
        elasticsearch_protocol = os.getenv('ELASTICSEARCH_PROTOCOL')
        if elasticsearch_host:
            elasticsearch_url = f'{elasticsearch_protocol}://{elasticsearch_host}:{elasticsearch_port}'
            message = _collect_metrics(elasticsearch_url)
            print(message)
            logger.info(f'Collected backup metrics are {message}')
    except Exception:
        logger.exception('Exception occurred during script execution:')
        raise


if __name__ == "__main__":
    __configure_logging(logger)
    run()
