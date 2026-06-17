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
from json import dumps
from unittest import mock

from requests import Response

from replication_metric import _collect_metrics

ELASTICSEARCH_URL = 'http://elasticsearch:9200'


class TestReplicationMetric(unittest.TestCase):

    @mock.patch('replication_metric._get_request', mock.Mock(
        side_effect=lambda url: get_response(url)))
    @mock.patch('replication_metric._is_replication_fetched', mock.Mock(
        side_effect=lambda url, index: True))
    def test_metrics_when_replication_running(self):
        expected_message = 'elasticsearch_replication_metric status=1,syncing_indices=1,bootstrapping_indices=1,paused_indices=1,failed_indices=0\n' \
                           'elasticsearch_replication_metric,index=test index_lag=1,index_operations_written=0,index_status=1\n' \
                           'elasticsearch_replication_metric,index=test_bootstrapping index_status=2\n' \
                           'elasticsearch_replication_metric,index=test_paused index_status=3'
        actual_message = _collect_metrics(ELASTICSEARCH_URL)
        self.assertEqual(expected_message, actual_message)

    @mock.patch('replication_metric._get_request', mock.Mock(
        side_effect=lambda url: get_response(url, 'failed_replication')))
    @mock.patch('replication_metric._is_replication_fetched', mock.Mock(
        side_effect=lambda url, index: True))
    def test_metrics_when_replication_failed(self):
        expected_message = 'elasticsearch_replication_metric status=4,syncing_indices=0,bootstrapping_indices=0,paused_indices=0,failed_indices=1\n' \
                           'elasticsearch_replication_metric,index=test_failed index_status=4'
        actual_message = _collect_metrics(ELASTICSEARCH_URL)
        self.assertEqual(expected_message, actual_message)

    @mock.patch('replication_metric._get_request', mock.Mock(
        side_effect=lambda url: get_response(url)))
    @mock.patch('replication_metric._is_replication_fetched', mock.Mock(
        side_effect=lambda url, index: False))
    def test_metrics_when_fetching_failed(self):
        expected_message = 'elasticsearch_replication_metric status=-1\n' \
                           'elasticsearch_replication_metric,index=test index_status=-1'
        actual_message = _collect_metrics(ELASTICSEARCH_URL)
        self.assertEqual(expected_message, actual_message)

    @mock.patch('replication_metric._get_request', mock.Mock(
        side_effect=lambda url: get_response(url, 'without_replication')))
    def test_metrics_when_replication_not_in_progress(self):
        expected_message = 'elasticsearch_replication_metric status=3,syncing_indices=0,bootstrapping_indices=0,paused_indices=0,failed_indices=0'
        actual_message = _collect_metrics(ELASTICSEARCH_URL)
        self.assertEqual(expected_message, actual_message)

    @mock.patch('replication_metric._get_request', mock.Mock(
        side_effect=lambda url: get_response(url, 'elasticsearch_unavailable')))
    def test_metrics_when_elasticsearch_unavailable(self):
        expected_message = 'elasticsearch_replication_metric status=-2'
        actual_message = _collect_metrics(ELASTICSEARCH_URL)
        self.assertEqual(expected_message, actual_message)


def get_response(parameter: str, additional=''):
    response = Response()
    if additional == 'elasticsearch_unavailable':
        response.status_code = 503
    else:
        response.status_code = 200
    response._content = dumps(get_response_content(parameter, additional)).encode()
    return response


def get_response_content(parameter: str, additional=''):
    if parameter == f'{ELASTICSEARCH_URL}/_cat/health?format=json':
        return [{"epoch": "1588948411",
                 "timestamp": "14:33:31",
                 "cluster": "opensearch",
                 "status": "green",
                 "node.total": "2",
                 "node.data": "2",
                 "shards": "6",
                 "pri": "3",
                 "relo": "0",
                 "init": "0",
                 "unassign": "0",
                 "pending_tasks": "0",
                 "max_task_wait_time": "-",
                 "active_shards_percent": "100.0%"}]
    elif parameter == f'{ELASTICSEARCH_URL}/_plugins/_replication/follower_stats':
        if additional == "without_replication":
            return {
                "num_syncing_indices": 0,
                "num_bootstrapping_indices": 0,
                "num_paused_indices": 0,
                "num_failed_indices": 0,
                "num_shard_tasks": 0,
                "num_index_tasks": 0,
                "operations_written": 0,
                "operations_read": 0,
                "failed_read_requests": 0,
                "throttled_read_requests": 0,
                "failed_write_requests": 0,
                "throttled_write_requests": 0,
                "follower_checkpoint": 0,
                "leader_checkpoint": 0,
                "total_write_time_millis": 0,
                "index_stats": {}
            }
        if additional == "failed_replication":
            return {
                "num_syncing_indices": 0,
                "num_bootstrapping_indices": 0,
                "num_paused_indices": 0,
                "num_failed_indices": 1,
                "num_shard_tasks": 0,
                "num_index_tasks": 0,
                "operations_written": 0,
                "operations_read": 0,
                "failed_read_requests": 0,
                "throttled_read_requests": 0,
                "failed_write_requests": 0,
                "throttled_write_requests": 0,
                "follower_checkpoint": 0,
                "leader_checkpoint": 0,
                "total_write_time_millis": 0,
                "index_stats": {}
            }
        return {
            "num_syncing_indices": 1,
            "num_bootstrapping_indices": 1,
            "num_paused_indices": 1,
            "num_failed_indices": 0,
            "num_shard_tasks": 1,
            "num_index_tasks": 1,
            "operations_written": 770056,
            "operations_read": 770056,
            "failed_read_requests": 0,
            "throttled_read_requests": 0,
            "failed_write_requests": 0,
            "throttled_write_requests": 0,
            "follower_checkpoint": 14308308,
            "leader_checkpoint": 14308219,
            "total_write_time_millis": 60896846,
            "index_stats": {
                "test": {
                    "operations_written": 0,
                    "operations_read": 0,
                    "failed_read_requests": 0,
                    "throttled_read_requests": 0,
                    "failed_write_requests": 0,
                    "throttled_write_requests": 0,
                    "follower_checkpoint": -1,
                    "leader_checkpoint": 0,
                    "total_write_time_millis": 0
                }
            }
        }
    elif parameter == f'{ELASTICSEARCH_URL}/_plugins/_replication/test/_status':
        if additional == "without_replication":
            return {"status": "REPLICATION NOT IN PROGRESS"}
        return {
            "status": "SYNCING",
            "reason": "User initiated",
            "leader_alias": "leader-cluster",
            "leader_index": "test",
            "follower_index": "test",
            "syncing_details":
                {
                    "leader_checkpoint": -1,
                    "follower_checkpoint": -1,
                    "seq_no": 0
                }
        }
    elif parameter == f'{ELASTICSEARCH_URL}/_plugins/_replication/test_bootstrapping/_status':
        return {"status": "BOOTSTRAPPING"}
    elif parameter == f'{ELASTICSEARCH_URL}/_plugins/_replication/test_paused/_status':
        return {"status": "PAUSED"}
    elif parameter == f'{ELASTICSEARCH_URL}/_plugins/_replication/test_failed/_status':
        return {"status": "NOT IN PROGRESS"}
    elif parameter == f'{ELASTICSEARCH_URL}/_plugins/_replication/autofollow_stats':
        if additional == "failed_replication":
            return {
                "num_success_start_replication": 0,
                "num_failed_start_replication": 0,
                "num_failed_leader_calls": 0,
                "failed_indices": ["test_failed"],
                "autofollow_stats": [
                    {
                        "name": "dr-replication",
                        "pattern": "*",
                        "num_success_start_replication": 0,
                        "num_failed_start_replication": 0,
                        "num_failed_leader_calls": 0,
                        "failed_indices": ["test_failed"],
                        "last_execution_time": 1
                    }
                ]
            }
        else:
            return {
                "num_success_start_replication": 1,
                "num_failed_start_replication": 0,
                "num_failed_leader_calls": 0,
                "failed_indices": [],
                "autofollow_stats": [
                    {
                        "name": "dr-replication",
                        "pattern": "*",
                        "num_success_start_replication": 1,
                        "num_failed_start_replication": 0,
                        "num_failed_leader_calls": 0,
                        "failed_indices": [],
                        "last_execution_time": 1
                    }
                ]
            }
    elif parameter == f'{ELASTICSEARCH_URL}/_cat/indices/*?h=index&format=json':
        if additional == 'without_replication':
            return []
        if additional == 'failed_replication':
            return [{'index': 'test_failed'}]
        return [{'index': 'test'}, {'index': 'test_bootstrapping'}, {'index': 'test_paused'}]


if __name__ == '__main__':
    unittest.main()
