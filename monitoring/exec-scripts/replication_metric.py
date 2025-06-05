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
    log_handler = RotatingFileHandler(filename='/opt/elasticsearch-monitoring/exec-scripts/replication_metric.log',
                                      maxBytes=50 * 1024,
                                      backupCount=5)
    log_handler.setFormatter(formatter)
    log_handler.setLevel(logging.DEBUG if os.getenv('ELASTICSEARCH_MONITORING_SCRIPT_DEBUG') else logging.INFO)
    log.addHandler(log_handler)
    err_handler = RotatingFileHandler(filename='/opt/elasticsearch-monitoring/exec-scripts/replication_metric.err',
                                      maxBytes=50 * 1024,
                                      backupCount=5)
    err_handler.setFormatter(formatter)
    err_handler.setLevel(logging.ERROR)
    log.addHandler(err_handler)


def _get_request(url: str):
    verify = ROOT_CA_CERTIFICATE if ROOT_CA_CERTIFICATE else None
    try:
        response = requests.get(
            url,
            auth=(ELASTICSEARCH_USERNAME, ELASTICSEARCH_PASSWORD),
            timeout=(3, REQUEST_TIMEOUT),
            verify=verify)
        return response
    except Exception:
        logger.exception('Elasticsearch is not available. There is no ability to perform request.')
        return ''


def _is_elasticsearch_available(elasticsearch_url: str):
    response = _get_request(f'{elasticsearch_url}/_cat/health?format=json')
    if response:
        logger.debug(f'Health response is {response.json()}')
        return response.status_code == 200
    return False


def _is_replication_fetched(elasticsearch_url: str, index: str):
    response = _get_request(f'{elasticsearch_url}/_plugins/_replication/{index}/_status')
    if response:
        logger.debug(f'Response is {response.json()}')
        return response.status_code != 500
    return False


def _get_follower_stats(elasticsearch_url: str):
    logger.info(f'Receiving follower stats...')
    follower_stats_endpoint = f'{elasticsearch_url}/_plugins/_replication/follower_stats'
    response = _get_request(follower_stats_endpoint)
    if response:
        follower_stats = response.json()
        logger.debug(f'Follower stats response is {follower_stats}')
        return follower_stats
    return ''


def _get_autofollow_stats(elasticsearch_url: str):
    logger.info(f'Receiving autofollow stats...')
    autofollow_stats_endpoint = f'{elasticsearch_url}/_plugins/_replication/autofollow_stats'
    response = _get_request(autofollow_stats_endpoint)
    if response:
        autofollow_stats = response.json()
        logger.debug(f'Follower stats response is {autofollow_stats}')
        return autofollow_stats
    return ''


def _get_indices(elasticsearch_url: str, pattern: str):
    logger.info(f'Receiving indices by replication pattern...')
    indices_endpoint = f'{elasticsearch_url}/_cat/indices/{pattern}?h=index&format=json'
    response = _get_request(indices_endpoint)
    if response:
        indices = response.json()
        logger.debug(f'Indices response is {indices}')
        return indices
    return []


def _get_index_status(elasticsearch_url: str, index: str):
    index_status_endpoint = f'{elasticsearch_url}/_plugins/_replication/{index}/_status'
    response = _get_request(index_status_endpoint)
    if response:
        status = response.json()
        logger.debug(f'Index status response is {status}')
        return status.get('status')
    return 'FAILED'


def _get_status_code(status):
    if status == 'BOOTSTRAPPING':
        return 2
    elif status == 'PAUSED':
        return 3
    else:
        return -1


def _determine_replication_status(syncing_indices, failed_indices):
    if syncing_indices > 0:
        if failed_indices == 0:
            return 1
        else:
            return 2
    elif failed_indices == 0:
        return 3
    else:
        return 4


def _collect_metrics(elasticsearch_url: str):
    logger.debug('Start to collect metrics...')

    if not _is_elasticsearch_available(elasticsearch_url):
        return 'elasticsearch_replication_metric status=-2'

    logger.info(f'Receiving replication info...')
    follower_stats = _get_follower_stats(elasticsearch_url)
    autofollow_stats = _get_autofollow_stats(elasticsearch_url)
    if not follower_stats or not autofollow_stats:
        return 'elasticsearch_replication_metric status=-2'
    num_syncing_indices = follower_stats.get('num_syncing_indices')
    num_bootstrapping_indices = follower_stats.get('num_bootstrapping_indices')
    num_paused_indices = follower_stats.get('num_paused_indices')
    num_failed_indices = follower_stats.get('num_failed_indices')
    index_stats = follower_stats.get('index_stats')
    if len(index_stats) and not _is_replication_fetched(elasticsearch_url, list(index_stats.keys())[0]):
        message = 'elasticsearch_replication_metric status=-1'
        for index in index_stats:
            message = f'{message}\nelasticsearch_replication_metric,index={index} index_status=-1'
        return message
    replication_status = _determine_replication_status(num_syncing_indices,
                                                       num_failed_indices)
    message = f'elasticsearch_replication_metric status={replication_status},' \
              f'syncing_indices={num_syncing_indices},' \
              f'bootstrapping_indices={num_bootstrapping_indices},' \
              f'paused_indices={num_paused_indices},' \
              f'failed_indices={num_failed_indices}'
    if len(index_stats):
        logger.info(f'Receiving indices metrics...')
        for index in index_stats:
            leader_checkpoint = int(index_stats[index]['leader_checkpoint'])
            follower_checkpoint = int(index_stats[index]['follower_checkpoint'])
            if follower_checkpoint > leader_checkpoint and leader_checkpoint == 0:
                lag = 0
            else:
                lag = leader_checkpoint - follower_checkpoint
            operations_written = index_stats[index]['operations_written']
            message = f'{message}\nelasticsearch_replication_metric,index={index} ' \
                      f'index_lag={lag},index_operations_written={operations_written},index_status=1'
    failed_indices = autofollow_stats.get('failed_indices')
    if len(failed_indices):
        for index in failed_indices:
            message = f'{message}\nelasticsearch_replication_metric,index={index} ' \
                      f'index_status=4'
    rules = autofollow_stats.get('autofollow_stats')
    if len(rules):
        for rule in rules:
            indices = _get_indices(elasticsearch_url, rule['pattern'])
            for index in indices:
                index_name = index['index']
                index_status = _get_index_status(elasticsearch_url, index_name)
                if index_status == "BOOTSTRAPPING" or index_status == "PAUSED":
                    message = f'{message}\nelasticsearch_replication_metric,index={index_name} ' \
                              f'index_status={_get_status_code(index_status)}'
                elif index_status == "FAILED":
                    message = f'{message}\nelasticsearch_replication_metric,index={index_name} ' \
                              f'index_status=4'
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
            logger.info(f'Collected replication metrics are {message}')
    except Exception:
        logger.exception('Exception occurred during script execution:')
        raise


if __name__ == "__main__":
    __configure_logging(logger)
    run()
