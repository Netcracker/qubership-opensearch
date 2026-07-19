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
from marker import Marker, MARKER_INDEX_NAME, MARKER_DOCUMENT_ID


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
              'number_of_shards': '2'}
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
              'number_of_shards': '2'}
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


class MarkerTests(unittest.TestCase):

  @patch('utils.prepare_elasticsearch_client')
  def test_set_marker_creates_index_when_absent(self, mock_prepare_client):
    mock_client = mock_prepare_client.return_value
    mock_client.indices.exists.return_value = False
    marker = Marker()

    marker.set_marker('my-backup/2024-01-15T12:00:00Z')

    mock_client.indices.create.assert_called_once()
    _, create_kwargs = mock_client.indices.create.call_args
    self.assertEqual(create_kwargs['index'], MARKER_INDEX_NAME)
    index_settings = create_kwargs['body']['settings']['index']
    self.assertEqual(index_settings['number_of_shards'], 1)

    mock_client.index.assert_called_once()
    _, index_kwargs = mock_client.index.call_args
    self.assertEqual(index_kwargs['index'], MARKER_INDEX_NAME)
    self.assertEqual(index_kwargs['id'], MARKER_DOCUMENT_ID)
    self.assertEqual(index_kwargs['body']['marker'],
                     'my-backup/2024-01-15T12:00:00Z')
    self.assertTrue(index_kwargs['refresh'])

  @patch('utils.prepare_elasticsearch_client')
  def test_set_marker_reuses_existing_index(self, mock_prepare_client):
    mock_client = mock_prepare_client.return_value
    mock_client.indices.exists.return_value = True
    marker = Marker()

    marker.set_marker('my-backup/2024-01-15T12:00:00Z')

    mock_client.indices.create.assert_not_called()
    mock_client.index.assert_called_once()

  @patch('utils.prepare_elasticsearch_client')
  def test_set_marker_empty_value_raises(self, mock_prepare_client):
    marker = Marker()
    with self.assertRaises(ValueError):
      marker.set_marker('')
    mock_prepare_client.return_value.index.assert_not_called()

  @patch('utils.prepare_elasticsearch_client')
  def test_get_marker_returns_value(self, mock_prepare_client):
    mock_client = mock_prepare_client.return_value
    mock_client.indices.exists.return_value = True
    mock_client.get.return_value = {
      '_source': {'marker': 'my-backup/2024-01-15T12:00:00Z'}}
    marker = Marker()

    self.assertEqual(marker.get_marker(),
                     'my-backup/2024-01-15T12:00:00Z')

  @patch('utils.prepare_elasticsearch_client')
  def test_get_marker_missing_index_returns_empty(self, mock_prepare_client):
    mock_client = mock_prepare_client.return_value
    mock_client.indices.exists.return_value = False
    marker = Marker()

    self.assertEqual(marker.get_marker(), '')
    mock_client.get.assert_not_called()

  @patch('utils.prepare_elasticsearch_client')
  def test_get_marker_missing_document_returns_empty(self, mock_prepare_client):
    from opensearchpy import NotFoundError
    mock_client = mock_prepare_client.return_value
    mock_client.indices.exists.return_value = True
    mock_client.get.side_effect = NotFoundError(404, 'not_found', {})
    marker = Marker()

    self.assertEqual(marker.get_marker(), '')
