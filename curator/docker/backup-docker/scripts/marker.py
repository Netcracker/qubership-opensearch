#!/usr/bin/python
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
import logging
from datetime import datetime, timezone

from opensearchpy import NotFoundError, RequestError

import utils

# Dedicated index that keeps the single data-validation marker.
#Since a marker records which backup was
# validated, it must travel with the backup/restore lifecycle itself so that
# restoring a backup restores the marker that was current at that time.
MARKER_INDEX_NAME = 'system.backup-restore-markers'

# The marker is a singleton: a fixed document id guarantees that a new marker
# overwrites the previous one, so only the most recent value is ever stored.
MARKER_DOCUMENT_ID = 'current'

MARKER_INDEX_BODY = {
    'settings': {
        'index': {
            'number_of_shards': 1,
        }
    },
    'mappings': {
        'properties': {
            'marker': {'type': 'keyword'},
            'written_at': {'type': 'date'},
        }
    }
}

INDEX_ALREADY_EXISTS_ERROR = 'resource_already_exists_exception'

loggingLevel = logging.INFO
logging.basicConfig(level=loggingLevel,
                    format='[%(asctime)s,%(msecs)03d][%(levelname)s][category=Marker] %(message)s',
                    datefmt='%Y-%m-%dT%H:%M:%S')


class Marker:

  def __init__(self):
    self._client = utils.prepare_elasticsearch_client()

  def set_marker(self, marker: str):
    if not marker:
      raise ValueError('Marker value must not be empty')
    self._ensure_index()
    self._client.index(index=MARKER_INDEX_NAME,
                       id=MARKER_DOCUMENT_ID,
                       body={'marker': marker,
                             'written_at': _now_rfc3339()},
                       refresh=True)
    logging.info('Data validation marker is set to "%s"', marker)

  def get_marker(self) -> str:
    # The marker index may legitimately not exist yet (no marker has ever been
    # set). In that case there is simply no marker to return.
    if not self._client.indices.exists(index=MARKER_INDEX_NAME):
      logging.info('Marker index "%s" does not exist, no marker is set',
                   MARKER_INDEX_NAME)
      return ''
    try:
      response = self._client.get(index=MARKER_INDEX_NAME,
                                  id=MARKER_DOCUMENT_ID)
    except NotFoundError:
      logging.info('Marker index exists but no marker document is stored yet')
      return ''
    return response.get('_source', {}).get('marker', '')

  def _ensure_index(self):
    if self._client.indices.exists(index=MARKER_INDEX_NAME):
      return
    try:
      self._client.indices.create(index=MARKER_INDEX_NAME,
                                  body=MARKER_INDEX_BODY)
      logging.info('Marker index "%s" created', MARKER_INDEX_NAME)
    except RequestError as error:
      # The index could have been created concurrently between the existence
      # check and the create call; treat that as success.
      if _is_already_exists_error(error):
        logging.info('Marker index "%s" already exists', MARKER_INDEX_NAME)
      else:
        raise


def _now_rfc3339() -> str:
  return datetime.now(timezone.utc).strftime('%Y-%m-%dT%H:%M:%SZ')


def _is_already_exists_error(error: RequestError) -> bool:
  return getattr(error, 'error', '') == INDEX_ALREADY_EXISTS_ERROR \
      or INDEX_ALREADY_EXISTS_ERROR in str(error)


if __name__ == "__main__":
  parser = argparse.ArgumentParser()
  parser.add_argument('--action', required=True, choices=['set', 'get'])
  parser.add_argument('--marker', default='')
  args = parser.parse_args()

  marker_instance = Marker()

  if args.action == 'set':
    marker_instance.set_marker(args.marker)
  else:
    value = marker_instance.get_marker()
    if value:
      print(value, end='')
