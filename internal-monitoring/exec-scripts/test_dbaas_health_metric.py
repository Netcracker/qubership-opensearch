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

from dbaas_health_metric import _collect_metrics

ELASTICSEARCH_DBAAS_ADAPTER_URL = "http://elasticsearch-dbaas-adapter:8080"


class TestDbaasHealthMetric(unittest.TestCase):

    def test_metrics_if_dbaas_adapter_is_not_available(self):
        expected_result = 'elasticsearch_dbaas_health status=1,elastic_cluster_status=3'
        actual_result = _collect_metrics(ELASTICSEARCH_DBAAS_ADAPTER_URL)
        self.assertEqual(expected_result, actual_result)

    @mock.patch('dbaas_health_metric._get_health_data', mock.Mock(side_effect=lambda url:
        {"status": "UP", "elasticCluster": {"status": "UP"}, "physicalDatabaseRegistration": {"status": "OK"}}
        if url == ELASTICSEARCH_DBAAS_ADAPTER_URL else ''))
    def test_metrics_if_elasticsearch_dbaas_adapter_is_available(self):
        expected_result = 'elasticsearch_dbaas_health status=5,elastic_cluster_status=5'
        actual_result = _collect_metrics(ELASTICSEARCH_DBAAS_ADAPTER_URL)
        self.assertEqual(expected_result, actual_result)

    @mock.patch('dbaas_health_metric._get_health_data', mock.Mock(side_effect=lambda url:
        {"status": "UP", "opensearchHealth": {"status": "UP"}, "dbaasAggregatorHealth": {"status": "OK"}}
        if url == ELASTICSEARCH_DBAAS_ADAPTER_URL else ''))
    def test_metrics_if_opensearch_dbaas_adapter_is_available(self):
        expected_result = 'elasticsearch_dbaas_health status=5,elastic_cluster_status=5'
        actual_result = _collect_metrics(ELASTICSEARCH_DBAAS_ADAPTER_URL)
        self.assertEqual(expected_result, actual_result)


if __name__ == '__main__':
    unittest.main()
