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
import re
from datetime import datetime, timedelta
from logging.handlers import RotatingFileHandler
from tempfile import TemporaryFile
from typing import Optional

from kubernetes import client as client, config as k8s_config
from kubernetes.client import CoreV1Api, ApiClient
from kubernetes.stream import stream

logger = logging.getLogger(__name__)

SLOW_LOGS_DESTINATION_PATH = '/opt/elasticsearch-monitoring/exec-scripts/tmp_slow_logs.log'
SLOW_LOGS_SOURCE_PATH = '/usr/share/opensearch/logs/slow_logs.log'


def __configure_logging(log):
    log.setLevel(logging.DEBUG)
    formatter = logging.Formatter(fmt='[%(asctime)s,%(msecs)03d][%(levelname)s] %(message)s',
                                  datefmt='%Y-%m-%dT%H:%M:%S')
    log_handler = RotatingFileHandler(filename='/opt/elasticsearch-monitoring/exec-scripts/slow_queries_metric.log',
                                      maxBytes=50 * 1024,
                                      backupCount=5)
    log_handler.setFormatter(formatter)
    log_handler.setLevel(logging.DEBUG if os.getenv('ELASTICSEARCH_MONITORING_SCRIPT_DEBUG') else logging.INFO)
    log.addHandler(log_handler)
    err_handler = RotatingFileHandler(filename='/opt/elasticsearch-monitoring/exec-scripts/slow_queries_metric.err',
                                      maxBytes=50 * 1024,
                                      backupCount=5)
    err_handler.setFormatter(formatter)
    err_handler.setLevel(logging.ERROR)
    log.addHandler(err_handler)


"""
########################################################################################################################
######################################## SLOW LOGS PARSER SECTION ######################################################
########################################################################################################################
"""

"""
Slow log message example:
[WARN ], 2023-07-27T08:03:43, [opensearch-2], [my-supper-index][0] took[12.4ms], took_millis[12], total_hits[0
hits], stats[], search_type[QUERY_THEN_FETCH], total_shards[1], source[{"query":{ "match":{"phrase":{
"query":"heuristic","operator":"OR","prefix_length":0,"max_expansions":50, "fuzzy_transpositions":true,
"lenient":false,"zero_terms_query":"NONE","auto_generate_synonyms_phrase_query":true,"boost":1.0}}}}], id[],

Converted SlowLogRecord object:
    log_level = "WARN"
    time = "2019-10-24T19:48:51,012"
    node = "opensearch-2"
    index = "my-supper-index"
    shard = "0"
    took = "12.4ms"
    took_millis = "12"
    total_hits = "0"
    stats = ""
    search_type = "QUERY_THEN_FETCH"
    total_shards[1]
    source = "{"query":{"match":{"phrase":{"query":"heuristic","operator":"OR","prefix_length":0,"max_expansions":50,
    "fuzzy_transpositions":true,"lenient":false,"zero_terms_query":"NONE","auto_generate_synonyms_phrase_query":true,
    "boost":1.0}}}}"
    id = ""
"""


class SlowLogRecord:
    def __init__(self, data: []):
        if len(data) != 13:
            raise ValueError
        (
            self.log_level,
            self.time,
            self.node,
            self.index,
            self.shard,
            self.took,
            self.took_millis,
            self.total_hits,
            self.stats,
            self.search_type,
            self.total_shards,
            self.source,
            self.id_
        ) = data

    def __str__(self):
        return (f'[{self.log_level}], {self.time}, [{self.node}], [{self.index}][{self.shard}] took[{self.took}], '
                f'took_millis[{self.took_millis}], total_hits[{self.total_hits} hits], stats[{self.stats}], '
                f'search_type[{self.search_type}], total_shards[{self.total_shards}], source[{self.source}], '
                f'id[{self.id_}]')

    def __eq__(self, other):
        if isinstance(other, SlowLogRecord):
            return (self.log_level == other.log_level
                    and self.time == other.time
                    and self.node == other.node
                    and self.index == other.index
                    and self.shard == other.shard
                    and self.took == other.took
                    and self.took_millis == other.took_millis
                    and self.total_hits == other.total_hits
                    and self.stats == other.stats
                    and self.search_type == other.search_type
                    and self.total_shards == other.total_shards
                    and self.source == other.source
                    and self.id_ == other.id_)
        return False

    def convert_to_metric_line(self) -> str:
        modified_source = self.source.replace(',', ';')
        return (f'opensearch_slow_query,node={self.node},index={self.index},shard={self.shard},query={modified_source},'
                f'total_hits={self.total_hits},registered_time={self.time} took_millis={self.took_millis}')


def _extract_values_from_brackets(element: str) -> list:
    index = element.find('[', 0)
    if index == -1:
        return [element]
    values = []
    while index != -1:
        end_index = element.find(']', index)
        value = element[index + 1:end_index]
        if value.endswith(' hits'):
            value = value[:-5]
        values.append(value)
        index = element.find('[', end_index)
    return values


def convert_slow_log_record(record: str) -> Optional[SlowLogRecord]:
    elements = record.strip(', ').split(', ')
    if len(elements) != 11:
        logger.warning(f'Unable to parse the record, because it has incorrect format: {record}')
        return None

    data = []
    for element in elements:
        data += _extract_values_from_brackets(element)
    return SlowLogRecord(data)


def filter_slow_logs_records(records: [SlowLogRecord], top_number) -> list:
    """
    Filters slow log records. There are two filters:
    1) Leaves one record with the longest execution time (`took_millis`) for each index and query.
    2) Leaves `top_number` of the records with the highest `took_millis` value.
    :param records: the list of SlowLogRecord
    :param top_number: the maximum number of records with the highest `took_millis` value
    :return: the list of SlowLogRecord after filters
    """
    records_dict = {}
    for record in records:
        name = f'{record.index}_{record.source}'
        if name not in records_dict or records_dict[name].took_millis < record.took_millis:
            records_dict[name] = record
    return sorted(records_dict.values(), key=lambda record: int(record.took_millis), reverse=True)[:top_number]


"""
########################################################################################################################
######################################## COPYING SLOW LOGS SECTION #####################################################
########################################################################################################################
"""


def _get_k8s_client() -> ApiClient:
    try:
        k8s_config.load_incluster_config()
        return client.ApiClient()
    except k8s_config.ConfigException:
        return k8s_config.new_client_from_config()


def _get_core_v1_api() -> CoreV1Api:
    k8s_client = _get_k8s_client()

    if k8s_client is None:
        import sys
        logger.exception("Can't load any kubernetes config.")
        sys.exit(1)

    return client.CoreV1Api(k8s_client)


def _get_search_engine_pod_names(core_v1_api: CoreV1Api) -> []:
    namespace = _get_search_engine_namespace()
    service_name = re.sub('-monitoring.*', '-data-svc', os.getenv('HOSTNAME'))
    service = core_v1_api.read_namespaced_service(service_name, namespace)
    labels = ','.join([f'{key}={value}' for key, value in service.spec.selector.items()])
    pod_list = core_v1_api.list_namespaced_pod(
        namespace=namespace,
        label_selector=labels
    )
    names = []
    for pod in pod_list.items:
        names.append(pod.metadata.name)
    return names


def _copy_slow_log_file_from_opensearch_pod(namespace, pod_name, core_v1_api: CoreV1Api):
    logger.info(f'Copy logs from {pod_name} pod')
    exec_command = ['/bin/cat', SLOW_LOGS_SOURCE_PATH]

    with TemporaryFile() as tmp_buffer:
        # Load file from pod to buffer
        file_stream = stream(core_v1_api.connect_get_namespaced_pod_exec,
                             pod_name,
                             namespace,
                             command=exec_command,
                             stderr=True,
                             stdin=True,
                             stdout=True,
                             tty=False,
                             _preload_content=False)

        while file_stream.is_open():
            file_stream.update(timeout=1)
            if file_stream.peek_stdout():
                out = file_stream.read_stdout()
                tmp_buffer.write(out.encode('utf-8'))
            if file_stream.peek_stderr():
                print("STDERR: %s" % file_stream.read_stderr())

        file_stream.close()
        tmp_buffer.seek(0)

        # Load data from buffer to file
        with open(SLOW_LOGS_DESTINATION_PATH, mode='w') as dest_file:
            for line in tmp_buffer:
                dest_file.write(str(line.decode('utf-8')))


def _clear_local_log_file():
    logger.info("Clear copied slow-log file")
    open(SLOW_LOGS_DESTINATION_PATH, 'w').close()


"""
########################################################################################################################
########################################### PARAMETERS SECTION #########################################################
########################################################################################################################
"""


def _get_search_engine_namespace() -> str:
    return os.getenv('OS_PROJECT')


def _get_slow_queries_top_number() -> int:
    return int(os.getenv('SLOW_QUERIES_TOP_NUMBER', '10'))


def _get_processing_interval_minutes() -> int:
    period = int(os.getenv('PROCESSING_INTERVAL_MINUTES', '0'))
    if period < 0:
        raise Exception(f'Incorrect value of PROCESSING_INTERVAL_MINUTES parameter is used: {period}')
    return period


"""
########################################################################################################################
######################################## MAIN FUNCTION SECTION #########################################################
########################################################################################################################
"""


def _process_log_records(filename: str, time_to_stop: datetime):
    """
    Reads and converts log records to SlowLogRecord format in specified time interval.
    :param time_to_stop: time point before which log records are not processed
    :return: list of SlowLogRecord
    """
    logger.info('Process log records...')
    slow_logs = []
    for line in reversed(open(filename, 'r').readlines()):
        slow_log = convert_slow_log_record(line.rstrip())
        if slow_log is None:
            logger.info(f'Skip "{line}" record, because it is not correct')
            continue
        if datetime.fromisoformat(slow_log.time) < time_to_stop:
            logger.info('Stop processing records, because all next records are out of the processing interval')
            break
        slow_logs.append(slow_log)
    return slow_logs


def _send_slow_logs_metrics(slow_log_records: [SlowLogRecord]):
    logger.info(f'Sending {len(slow_log_records)} slow-query metrics')
    for record in slow_log_records:
        print(record.convert_to_metric_line())


def run():
    logger.info('Start script execution...')
    namespace = _get_search_engine_namespace()
    slow_queries_top_number = _get_slow_queries_top_number()
    processing_interval_minutes = _get_processing_interval_minutes()
    core_v1_api = _get_core_v1_api()

    pods = _get_search_engine_pod_names(core_v1_api)
    if len(pods) == 0:
        logger.info('There are no available pods, so skip slow-log metric collection')
        return

    slow_log_objects = []
    time_to_stop = datetime.now() - timedelta(minutes=processing_interval_minutes)
    for pod in pods:
        try:
            logger.info(f'Process slow-log from {pod} pod')
            _copy_slow_log_file_from_opensearch_pod(namespace, pod, core_v1_api)
            log_objects = _process_log_records(SLOW_LOGS_DESTINATION_PATH, time_to_stop)
            slow_log_objects += log_objects
        except Exception:
            logger.exception(f'Error occurred while process logs from {pod} pod.')
    _clear_local_log_file()

    result = filter_slow_logs_records(slow_log_objects, slow_queries_top_number)
    _send_slow_logs_metrics(result)


if __name__ == '__main__':
    __configure_logging(logger)
    run()
