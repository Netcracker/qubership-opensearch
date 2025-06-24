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
import curator
import utils

SNAPSHOT_REPOSITORY_NAME = os.environ.get('SNAPSHOT_REPOSITORY_NAME')
loggingLevel = logging.INFO
logging.basicConfig(level=loggingLevel,
                    format='[%(asctime)s,%(msecs)03d][%(levelname)s][category=List_Instances] %(message)s',
                    datefmt='%Y-%m-%dT%H:%M:%S')

if __name__ == "__main__":
  parser = argparse.ArgumentParser()
  parser.add_argument('folder')
  args = parser.parse_args()
  try:
    with open(f'{args.folder}/databases.txt', 'r') as f:
      dbs_list = f.read().splitlines()
    for db in dbs_list:
      print(db)
  except FileNotFoundError:
    logging.info('Databases list not found, showing stored indices')
    client = utils.prepare_elasticsearch_client()
    snapshots = curator.SnapshotList(client=client,
                                     repository=SNAPSHOT_REPOSITORY_NAME).all_snapshots
    filtered_snapshots = [snapshot for snapshot in snapshots if
                          snapshot['snapshot'] == utils.extract_snapshot_name(args.folder)]
    for snapshot in filtered_snapshots:
      for index in snapshot['indices']:
        print(index)
