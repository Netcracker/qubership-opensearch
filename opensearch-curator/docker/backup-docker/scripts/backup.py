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
import ast
import json
import logging
import os
import curator
import utils

SNAPSHOT_REPOSITORY_NAME = os.environ.get('SNAPSHOT_REPOSITORY_NAME')

loggingLevel = logging.INFO
logging.basicConfig(level=loggingLevel,
                    format='[%(asctime)s,%(msecs)03d][%(levelname)s][category=Backup] %(message)s',
                    datefmt='%Y-%m-%dT%H:%M:%S')


class Backup:

  def __init__(self, args):
    self._client = utils.prepare_elasticsearch_client()
    self._storage_folder = args.folder
    self._dbs = ast.literal_eval(args.dbs) if args.dbs else None

  def granular_backup(self):
    self.backup_templates()

    indices = curator.IndexList(self._client)
    aliases = self._client.indices.get_alias()
    indices.filter_by_regex(kind="prefix", value="|".join(self._dbs))

    aliases = {name: aliases[name] for name in indices.indices if
               name in aliases}

    if len(indices.indices) == 0:
      with open(f"{self._storage_folder}/databases.txt", "w") as f:
        for db in self._dbs:
          f.write("%s\n" % db)
      return

    if bool(aliases):
      with open(f"{self._storage_folder}/aliases.json", "w") as f:
        json.dump(aliases, f)

    with open(f"{self._storage_folder}/indices.txt", "w") as f:
      for index in indices.indices:
        f.write("%s\n" % index)

    with open(f"{self._storage_folder}/databases.txt", "w") as f:
      for db in self._dbs:
        f.write("%s\n" % db)

    snapshot_name = utils.extract_snapshot_name(self._storage_folder)
    snapshot = curator.Snapshot(ilo=indices,
                                repository=SNAPSHOT_REPOSITORY_NAME,
                                name=snapshot_name, include_global_state=False,
                                skip_repo_fs_check=True)
    snapshot.do_action()

  def full_backup(self):
    self.backup_templates()

    indices = curator.IndexList(self._client)
    indices.filter_by_regex(kind="prefix", value="\\.", exclude=True)

    snapshot = curator.Snapshot(ilo=indices,
                                repository=SNAPSHOT_REPOSITORY_NAME,
                                name=utils.extract_snapshot_name(
                                  self._storage_folder),
                                include_global_state=False,
                                skip_repo_fs_check=True)
    snapshot.do_action()

  def backup_templates(self):
    logging.info('Backing up templates')
    templates = self._client.indices.get_index_template()['index_templates']
    obsolete_templates = self._client.indices.get_template()
    component_templates = self._client.cluster.get_component_template()[
      'component_templates']
    if self._dbs:
      templates = [template for template in templates if any(
          template['name'].startswith(prefix) for prefix in self._dbs)]
      obsolete_templates = dict(
        filter(lambda x: any(x[0].startswith(prefix) for prefix in self._dbs),
               obsolete_templates.items()))
      component_templates = [template for template in component_templates
                             if any(template['name'].startswith(prefix)
                                    for prefix in self._dbs)]

    if len(templates) > 0:
      with open(f"{self._storage_folder}/templates.json", "w") as f:
        json.dump(templates, f)
    if len(obsolete_templates) > 0:
      if "tenant_template" in obsolete_templates:
        del obsolete_templates["tenant_template"]
      with open(f"{self._storage_folder}/obsolete_templates.json", "w") as f:
        json.dump(obsolete_templates, f)
    if len(component_templates) > 0:
      with open(f"{self._storage_folder}/component_templates.json", "w") as f:
        json.dump(component_templates, f)
    logging.info('All templates are backed up')


if __name__ == "__main__":
  parser = argparse.ArgumentParser()
  parser.add_argument('folder')
  parser.add_argument('-d', '--dbs')
  args = parser.parse_args()

  logging.info('Backup has started.')

  backup_instance = Backup(args=args)

  if args.dbs:
    backup_instance.granular_backup()
  else:
    backup_instance.full_backup()

  logging.info('Backup is completed successfully.')
