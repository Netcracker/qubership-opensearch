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
import os
import shutil
import curator
import utils

SNAPSHOT_REPOSITORY_NAME = os.environ.get('SNAPSHOT_REPOSITORY_NAME')


loggingLevel = logging.INFO
logging.basicConfig(level=loggingLevel,
                    format='[%(asctime)s,%(msecs)03d][%(levelname)s][category=Evict] %(message)s',
                    datefmt='%Y-%m-%dT%H:%M:%S')

class Evict:

  def __init__(self):
    self._storage_folder = args.folder

  def evict(self):
    client = utils.prepare_elasticsearch_client()
    snapshot_list = curator.SnapshotList(client=client, repository=SNAPSHOT_REPOSITORY_NAME)
    snapshot_list.filter_by_regex(kind="regex", value=utils.extract_snapshot_name(self._storage_folder), exclude=False)
    try:
      shutil.rmtree(self._storage_folder)
    except FileNotFoundError:
      logging.info('Directory does not exists or already deleted')

    delete_action = curator.DeleteSnapshots(slo=snapshot_list)

    delete_action.do_action()


if __name__ == "__main__":
  parser = argparse.ArgumentParser()
  parser.add_argument('folder')
  args = parser.parse_args()

  logging.info('Backup eviction has started.')

  evict_instance = Evict()

  evict_instance.evict()

  logging.info('Backup eviction completed successfully.')
