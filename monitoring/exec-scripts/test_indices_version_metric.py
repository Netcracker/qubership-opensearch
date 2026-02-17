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

from indices_version_metric import _parse_version_created, _get_version_num, _collect_metrics

ELASTICSEARCH_URL = 'http://opensearch:9092'


class TestIndicesVersionMetric(unittest.TestCase):

    def test_parse_version_created_opensearch_1_3_17(self):
        """Test parsing OpenSearch 1.3.17 (version.created = 135249527)"""
        # 135249527 ^ 0x08000000 = 1031799 → remove 99 → 10317 → 1.3.17
        self.assertEqual(_parse_version_created(135249527), "1.3.17")
        
    def test_parse_version_created_opensearch_2_11_1(self):
        """Test parsing OpenSearch 2.11.1 (version.created = 136327927)"""
        # 136327927 ^ 0x08000000 = 2110199 → remove 99 → 21101 → 2.11.1
        self.assertEqual(_parse_version_created(136327927), "2.11.1")
        
    def test_get_version_num_1_3_17(self):
        """Test version number for OpenSearch 1.3.17"""
        # 135249527 → 10317
        self.assertEqual(_get_version_num(135249527), 10317)
        
    def test_get_version_num_2_11_1(self):
        """Test version number for OpenSearch 2.11.1"""
        # 136327927 → 21101
        self.assertEqual(_get_version_num(136327927), 21101)
        
    def test_version_num_less_than_20000_is_1x(self):
        """Test that version numbers < 20000 correspond to 1.x versions"""
        # All 1.x versions should have version_num < 20000
        self.assertLess(_get_version_num(135249527), 20000)  # 1.3.17
        
        # All 2.x versions should have version_num >= 20000
        self.assertGreaterEqual(_get_version_num(136327927), 20000)  # 2.11.1

    @mock.patch('indices_version_metric._get_indices_settings')
    def test_collect_metrics_with_mixed_versions(self, mock_get_settings):
        """Test metric collection with indices of different versions"""
        mock_get_settings.return_value = {
            'index-1': {
                'settings': {
                    'index': {
                        'version': {
                            'created': '135249527'  # 1.3.17
                        }
                    }
                }
            },
            'index-2': {
                'settings': {
                    'index': {
                        'version': {
                            'created': '135249527'  # 1.3.17 (same version)
                        }
                    }
                }
            },
            'index-3': {
                'settings': {
                    'index': {
                        'version': {
                            'created': '136327927'  # 2.11.1
                        }
                    }
                }
            }
        }
        
        metrics = _collect_metrics(ELASTICSEARCH_URL)
        
        self.assertEqual(len(metrics), 2)
        self.assertIn('opensearch_indices_version_created,version="1.3.17",version_num=10317 count=2i', metrics)
        self.assertIn('opensearch_indices_version_created,version="2.11.1",version_num=21101 count=1i', metrics)

    @mock.patch('indices_version_metric._get_indices_settings')
    def test_collect_metrics_skips_system_indices(self, mock_get_settings):
        """Test that system indices (starting with .) are skipped"""
        mock_get_settings.return_value = {
            '.system-index': {
                'settings': {
                    'index': {
                        'version': {
                            'created': '136327927'  # 2.11.1
                        }
                    }
                }
            },
            'user-index': {
                'settings': {
                    'index': {
                        'version': {
                            'created': '136327927'  # 2.11.1
                        }
                    }
                }
            }
        }
        
        metrics = _collect_metrics(ELASTICSEARCH_URL)
        
        self.assertEqual(len(metrics), 1)
        self.assertIn('opensearch_indices_version_created,version="2.11.1",version_num=21101 count=1i', metrics)

    @mock.patch('indices_version_metric._get_indices_settings')
    def test_collect_metrics_handles_empty_response(self, mock_get_settings):
        """Test metric collection with no indices"""
        mock_get_settings.return_value = {}
        
        metrics = _collect_metrics(ELASTICSEARCH_URL)
        
        self.assertEqual(len(metrics), 0)

    @mock.patch('indices_version_metric._get_indices_settings')
    def test_collect_metrics_handles_missing_version(self, mock_get_settings):
        """Test metric collection when version.created is missing"""
        mock_get_settings.return_value = {
            'index-1': {
                'settings': {
                    'index': {}
                }
            }
        }
        
        metrics = _collect_metrics(ELASTICSEARCH_URL)
        
        self.assertEqual(len(metrics), 0)


if __name__ == '__main__':
    unittest.main()
