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

import argparse
import unittest
import os
import utils
from unittest.mock import mock_open, patch
from restore import Restore
from backup import Backup


class CuratorTests(unittest.TestCase):

  def setUp(self):
    parser = argparse.ArgumentParser()
    parser.add_argument('folder')
    parser.add_argument('-skip_users_recovery')
    parser.add_argument('-d', '--dbs')
    parser.add_argument('-m', '--dbmap')
    parser.add_argument('-clean')
    self.args = parser.parse_args(['C:/CLOUD/docker-elastic-curator/docker/backup-docker/scripts/'])

  def test_create_elasticsearch_url_is_empty(self):
    os.environ.__setitem__('ES_HOST', '')
    self.assertRaises(RuntimeError, utils.create_elasticsearch_url)

  @patch('utils.prepare_elasticsearch_client')
  @patch('builtins.open', new_callable=unittest.mock.mock_open, read_data='db1\ndb2\ndb3\n')
  def test_granular_restore_invalid_databases(self, mock_open, mock_prepare_client):
    os.environ.__setitem__('ES_HOST', 'elasticsearch:9200')
    self._restore = Restore(args=self.args)
    mock_client = mock_prepare_client.return_value
    self._restore._dbs = ["db1", "db4", "db5"]
    with self.assertRaises(Exception) as context:
      self._restore.granular_restore()
    self.assertEqual(str(context.exception), "Databases are not valid. Valid databases are ['db1', 'db2', 'db3']")

  @patch('utils.prepare_elasticsearch_client')
  def test_template_renaming(self, mock_prepare_client):
    os.environ.__setitem__('ES_HOST', 'opensearch:9200')
    self._restore = Restore(args=self.args)
    self._restore._renames = {'temporary': 'constant'}

    templates = [{
      'name': 'temporary',
      'index_template': {
        'index_patterns': ['temporary-*'],
        'template': {
          'settings': {
            'index': {
              'number_of_shards': '2',
              'number_of_replicas': '1'}
          },
          'aliases': {'temporary': {}}
        },
        'composed_of': ['temporary_123', 'temporary_321']
      }
    }]
    renamed_templates = self._restore.rename_templates(templates, 'temporary')
    expected_template = {
      'name': 'constant',
      'index_template': {
        'index_patterns': ['constant-*'],
        'template': {
          'settings': {
            'index': {
              'number_of_shards': '2',
              'number_of_replicas': '1'}
          },
          'aliases': {'constant': {}}
        },
        'composed_of': ['constant_123', 'constant_321']
      }
    }
    self.assertEqual(renamed_templates[0], expected_template)
    mock_prepare_client.assert_called_once()

  @patch('utils.prepare_elasticsearch_client')
  def test_component_template_renaming(self, mock_prepare_client):
    os.environ.__setitem__('ES_HOST', 'opensearch:9200')
    self._restore = Restore(args=self.args)
    self._restore._renames = {'temporary': 'constant'}

    templates = [{
      'name': 'temporary_123',
      'component_template': {
        'template': {
          'settings': {
            'index': {
              'number_of_shards': '4'
            }
          },
          'aliases': {
            'temporary8291': {
              'routing': 'shard-1'
            }
          }
        }
      }
    }]
    renamed_templates = self._restore.rename_component_templates(templates, 'temporary')
    expected_template = {
      'name': 'constant_123',
      'component_template': {
        'template': {
          'settings': {
            'index': {
              'number_of_shards': '4'
            }
          },
          'aliases': {
            'constant8291': {
              'routing': 'shard-1'
            }
          }
        }
      }
    }
    self.assertEqual(renamed_templates[0], expected_template)
    mock_prepare_client.assert_called_once()

  @patch('utils.prepare_elasticsearch_client')
  def test_obsolete_template_renaming(self, mock_prepare_client):
    os.environ.__setitem__('ES_HOST', 'opensearch:9200')
    self._restore = Restore(args=self.args)
    self._restore._renames = {'tests': 'prod'}

    template = {
      'order': 0,
      'index_patterns': [
        'testsdsad*'
      ],
      'settings': {
        'index': {
          'number_of_shards': '1'
        }
      },
      'mappings': {
        '_source': {
          'enabled': False
        },
        'properties': {
          'created_at': {
            'format': 'EEE MMM dd HH:mm:ss Z yyyy',
            'type': 'date'
          },
          'host_name': {
            'type': 'keyword'
          }
        }
      },
      'aliases': {
        'testsdsad21': {}
      }
    }
    renamed_template = self._restore.rename_template(template, 'tests')
    expected_template = {
      'order': 0,
      'index_patterns': [
        'proddsad*'
      ],
      'settings': {
        'index': {
          'number_of_shards': '1'
        }
      },
      'mappings': {
        '_source': {
          'enabled': False
        },
        'properties': {
          'created_at': {
            'format': 'EEE MMM dd HH:mm:ss Z yyyy',
            'type': 'date'
          },
          'host_name': {
            'type': 'keyword'
          }
        }
      },
      'aliases': {
        'proddsad21': {}
      }
    }
    self.assertEqual(renamed_template, expected_template)
    mock_prepare_client.assert_called_once()
