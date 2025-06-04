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

import unittest
from unittest import mock

from backup_metric import _collect_metrics, _get_repositories, _get_repository_storage_type, _get_current_status
from requests import Response

ELASTICSEARCH_URL = 'http://elasticsearch:9200'


class TestBackupMetric(unittest.TestCase):

    @mock.patch('backup_metric._get_request_with_parameter', mock.Mock(
        side_effect=lambda url, parameter: get_request_result(parameter)))
    def test_receiving_repositories(self):
        expected_repositories = ['snapshots']
        actual_repositories = _get_repositories(ELASTICSEARCH_URL)
        self.assertEqual(expected_repositories, actual_repositories)

    @mock.patch('backup_metric._get_request_with_parameter', mock.Mock(
        side_effect=lambda url, parameter: get_request_result(parameter)))
    def test_receiving_repository_storage_type(self):
        expected_storage_type = 'fs'
        actual_storage_type = _get_repository_storage_type(ELASTICSEARCH_URL, 'snapshots')
        self.assertEqual(expected_storage_type, actual_storage_type)

    @mock.patch('backup_metric._get_request_with_parameter', mock.Mock(
        side_effect=lambda url, parameter: get_request_result(parameter)))
    def test_receiving_current_status(self):
        expected_status = 'SUCCESS'
        actual_status = _get_current_status(ELASTICSEARCH_URL)
        self.assertEqual(expected_status, actual_status)

    def test_metrics_when_elasticsearch_is_not_available(self):
        expected_message = 'elasticsearch_backups_metric current_status=-2'
        actual_message = _collect_metrics(ELASTICSEARCH_URL)
        self.assertEqual(expected_message, actual_message)

    @mock.patch('requests.get', mock.Mock(
        side_effect=lambda url, auth, timeout, verify: get_health_response()))
    @mock.patch('backup_metric._get_request_with_parameter', mock.Mock(
        side_effect=lambda url, parameter: get_request_result(parameter)))
    def test_metrics_when_elasticsearch_is_available(self):
        expected_message = 'elasticsearch_backups_metric count_of_snapshots=3,last_snapshot_time=1588935962159,' \
                           'count_of_successful_snapshots=3,last_successful_snapshot_time=1588935962159,storage_type=1,' \
                           'last_backup_size=1548,last_backup_time_spent=212,last_backup_status=1,current_status=1'
        actual_message = _collect_metrics(ELASTICSEARCH_URL)
        self.assertEqual(expected_message, actual_message)

    @mock.patch('requests.get', mock.Mock(
        side_effect=lambda url, auth, timeout, verify: get_health_response()))
    @mock.patch('backup_metric._get_request_with_parameter', mock.Mock(
        side_effect=lambda url, parameter: get_request_result(parameter, 'without_snapshots')))
    def test_metrics_when_there_are_no_snapshots(self):
        expected_message = 'elasticsearch_backups_metric count_of_snapshots=0,last_snapshot_time=-1,' \
                           'count_of_successful_snapshots=0,last_successful_snapshot_time=-1,storage_type=1,' \
                           'last_backup_status=-1,current_status=1'
        actual_message = _collect_metrics(ELASTICSEARCH_URL)
        self.assertEqual(expected_message, actual_message)

    @mock.patch('requests.get', mock.Mock(
        side_effect=lambda url, auth, timeout, verify: get_health_response()))
    @mock.patch('backup_metric._get_request_with_parameter', mock.Mock(
        side_effect=lambda url, parameter: get_request_result(parameter, 'without_repository')))
    def test_metrics_when_there_are_no_snapshots_and_repositories(self):
        expected_message = 'elasticsearch_backups_metric count_of_snapshots=0,last_snapshot_time=-1,' \
                           'count_of_successful_snapshots=0,last_successful_snapshot_time=-1,last_backup_status=-1,' \
                           'current_status=1'
        actual_message = _collect_metrics(ELASTICSEARCH_URL)
        self.assertEqual(expected_message, actual_message)

    @mock.patch('requests.get', mock.Mock(
        side_effect=lambda url, auth, timeout, verify: get_health_response()))
    @mock.patch('backup_metric._get_request_with_parameter', mock.Mock(
        side_effect=lambda url, parameter: get_request_result(parameter, 'last_snapshot_is_deleted')))
    def test_metrics_when_last_snapshot_is_deleted(self):
        expected_message = 'elasticsearch_backups_metric count_of_snapshots=3,last_snapshot_time=1588935962159,' \
                           'count_of_successful_snapshots=3,last_successful_snapshot_time=1588935962159,storage_type=1,' \
                           'last_backup_time_spent=357,last_backup_status=1,current_status=1'
        actual_message = _collect_metrics(ELASTICSEARCH_URL)
        self.assertEqual(expected_message, actual_message)

    @mock.patch('requests.get', mock.Mock(
        side_effect=lambda url, auth, timeout, verify: get_health_response()))
    @mock.patch('backup_metric._get_request_with_parameter', mock.Mock(
        side_effect=lambda url, parameter: get_request_result(parameter, 'without_status')))
    def test_metrics_when_gathering_backups_are_not_active(self):
        expected_message = 'elasticsearch_backups_metric count_of_snapshots=3,last_snapshot_time=1588935962159,' \
                           'count_of_successful_snapshots=3,last_successful_snapshot_time=1588935962159,storage_type=1,' \
                           'last_backup_size=1548,last_backup_time_spent=212,last_backup_status=1,current_status=-1'
        actual_message = _collect_metrics(ELASTICSEARCH_URL)
        self.assertEqual(expected_message, actual_message)


def get_health_response():
    response = Response()
    response.status_code = 200
    response._content = '1588948411 14:33:31 left-elasticsearch green 2 2 12 6 0 0 0 0 - 100.0%'
    return response


def get_request_result(parameter: str, additional=''):
    if parameter == '' or parameter == 'snapshots':
        if additional == "without_repository":
            return {}
        return {'snapshots': {'type': 'fs', 'settings': {'compress': 'true',
                                                         'location': '/usr/share/elasticsearch/snapshots'}}}
    elif parameter == '_status':
        if additional == 'without_status':
            return {'snapshots': []}
        return {'snapshots': [{'snapshot': '20200508t100600', 'repository': 'snapshots',
                               'uuid': 'noQDcGXsSbetpPeDPTUTlw', 'state': 'SUCCESS', 'include_global_state': True,
                               'shards_stats': {'initializing': 0, 'started': 0, 'finalizing': 0, 'done': 6,
                                                'failed': 0, 'total': 6},
                               'stats': {'incremental': {'file_count': 1, 'size_in_bytes': 243},
                                         'total': {'file_count': 6, 'size_in_bytes': 1548},
                                         'start_time_in_millis': 1588932362528, 'time_in_millis': 310},
                               'indices': {'dbaas_metadata': {}}}]}
    elif parameter == 'snapshots/*':
        if additional == 'without_snapshots':
            return {'snapshots': []}
        return {'snapshots': [
            {'snapshot': '20200508t100000', 'uuid': 'bDAqDLCETf22xiUpa_X09A', 'version_id': 6050099, 'version': '6.5.0',
             'indices': ['dbaas_metadata'], 'include_global_state': True, 'state': 'SUCCESS',
             'start_time': '2020-05-08T10:00:03.324Z', 'start_time_in_millis': 1588932003324,
             'end_time': '2020-05-08T10:00:04.530Z', 'end_time_in_millis': 1588932004530, 'duration_in_millis': 1206,
             'failures': [], 'shards': {'total': 6, 'failed': 0, 'successful': 6}},
            {'snapshot': '20200508t100600', 'uuid': 'noQDcGXsSbetpPeDPTUTlw', 'version_id': 6050099, 'version': '6.5.0',
             'indices': ['dbaas_metadata'], 'include_global_state': True, 'state': 'SUCCESS',
             'start_time': '2020-05-08T10:06:02.469Z', 'start_time_in_millis': 1588932362469,
             'end_time': '2020-05-08T10:06:02.836Z', 'end_time_in_millis': 1588932362836, 'duration_in_millis': 367,
             'failures': [], 'shards': {'total': 6, 'failed': 0, 'successful': 6}},
            {'snapshot': '20200508t110600', 'uuid': 'Y6gvM_XgRhCLL8lN38MYBg', 'version_id': 6050099, 'version': '6.5.0',
             'indices': ['dbaas_metadata'], 'include_global_state': True, 'state': 'SUCCESS',
             'start_time': '2020-05-08T11:06:01.802Z', 'start_time_in_millis': 1588935961802,
             'end_time': '2020-05-08T11:06:02.159Z', 'end_time_in_millis': 1588935962159, 'duration_in_millis': 357,
             'failures': [], 'shards': {'total': 6, 'failed': 0, 'successful': 6}}]}
    elif parameter == 'snapshots/20200508t110600/_status':
        if additional == 'last_snapshot_is_deleted':
            return {'error': {
                'root_cause': [
                    {'type': 'snapshot_missing_exception', 'reason': '[snapshots:20200508t110600] is missing'}
                ],
                'type': 'snapshot_missing_exception', 'reason': '[snapshots:20200508t110600] is missing'},
                'status': 404}
        return {'snapshots': [
            {'snapshot': '20200508t110600', 'repository': 'snapshots', 'uuid': 'Y6gvM_XgRhCLL8lN38MYBg',
             'state': 'SUCCESS', 'include_global_state': True,
             'shards_stats': {'initializing': 0, 'started': 0, 'finalizing': 0, 'done': 6, 'failed': 0, 'total': 6},
             'stats': {'incremental': {'file_count': 1, 'size_in_bytes': 243},
                       'total': {'file_count': 6, 'size_in_bytes': 1548},
                       'start_time_in_millis': 1588935961895,
                       'time_in_millis': 212},
             'indices': {'dbaas_metadata': {}}}
        ]}


if __name__ == '__main__':
    unittest.main()
