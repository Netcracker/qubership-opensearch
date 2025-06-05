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
import re
import time

import curator
import requests
from opensearchpy import NotFoundError

import dbaas_client
import utils

SNAPSHOT_REPOSITORY_NAME = os.environ.get('SNAPSHOT_REPOSITORY_NAME')

USERS_RECOVERY_DONE_STATE = 'done'
USERS_RECOVERY_FAILED_STATE = 'failed'
USERS_RECOVERY_RUNNING_STATE = 'running'
USERS_RECOVERY_IDLE_STATE = 'idle'

INTERVAL = 10
TIMEOUT = 240

loggingLevel = logging.INFO
logging.basicConfig(level=loggingLevel,
                    format='[%(asctime)s,%(msecs)03d][%(levelname)s][category=Restore] %(message)s',
                    datefmt='%Y-%m-%dT%H:%M:%S')


class Restore:

  def __init__(self, args):
    self._client = utils.prepare_elasticsearch_client()
    self._storage_folder = args.folder
    self._dbs = ast.literal_eval(args.dbs) if args.dbs else None
    # correctness of `dbmap` is checked in backup-daemon.
    # Correct pattern for keys and values is [0-9a-zA-Z-_]
    self._renames = ast.literal_eval(args.dbmap) if args.dbmap else {}
    self._clean_recovery = utils.str2bool(args.clean) if args.clean else False

  def granular_restore(self):
    try:
      with open(f'{self._storage_folder}/databases.txt', 'r') as f:
        stored_databases = f.read().splitlines()
    except FileNotFoundError:
      stored_databases = None

    if stored_databases:
      if not all(item in stored_databases for item in self._dbs):
        raise Exception(
            f'Databases are not valid. Valid databases are {stored_databases}')

    self.clean_restore()
    self.restore_templates()

    try:
      with open(f'{self._storage_folder}/indices.txt', 'r') as f:
        indices = f.read().splitlines()
        indices_by_dbs = {
          prefix: [index for index in indices if index.startswith(prefix)]
          for prefix in self._dbs}
    except FileNotFoundError:
      indices_by_dbs = {db: [db] for db in self._dbs}

    if stored_databases and not os.path.exists(f'{self._storage_folder}/indices.txt'):
      logging.info('No indices and aliases to restore')
      return

    indices_to_rename = dict(
        filter(lambda x: x[0] in self._renames, indices_by_dbs.items()))
    for prefix, indices_list in indices_to_rename.items():
      self.restore_indices(indices=indices_list, prefix=prefix,
                           replacement=self._renames[prefix])

    indices = sum([value for key, value in indices_by_dbs.items() if
                   key not in self._renames], [])
    if indices:
      self.restore_indices(indices=indices)

    try:
      with open(f'{self._storage_folder}/aliases.json', 'r') as f:
        aliases = json.load(f)
        aliases_by_dbs = {
          prefix: {index: body for index, body in aliases.items() if
                   index.startswith(prefix)}
          for prefix in self._dbs}
        aliases = [(self.rename(index, prefix),
                    self.rename_index_aliases(index_aliases, prefix))
                   if prefix in self._renames else (index, index_aliases)
                   for prefix, aliases in aliases_by_dbs.items() for
                   index, index_aliases in aliases.items()]
        for index, index_aliases in aliases:
          for name, body in index_aliases['aliases'].items():
            self._client.indices.put_alias(index=index,
                                           name=name,
                                           body=body)
    except FileNotFoundError:
      logging.info('No aliases to restore')

  def full_restore(self):
    self.clean_restore()
    self.restore_templates()
    self.restore_indices(include_aliases=True)

  def restore_indices(self, indices: list = None, include_aliases: bool = False,
      prefix: str = None, replacement: str = None):
    rename_pattern = None
    rename_replacement = None
    slo = curator.SnapshotList(client=self._client,
                               repository=SNAPSHOT_REPOSITORY_NAME)
    snapshot_name = utils.extract_snapshot_name(self._storage_folder)

    if indices:
      if prefix and replacement:
        rename_pattern = f'{prefix}(.*)'
        rename_replacement = f'{replacement}$1'
        indices_to_stop = [self.rename(index, prefix) for index in indices]
      else:
        indices_to_stop = indices
    else:
      indices_to_stop = slo.snapshot_info[snapshot_name]['indices']
    logging.info(
        f'These indices will be stopped before restore: {indices_to_stop}')
    self._client.indices.close(index=indices_to_stop, ignore_unavailable=True)

    restore_action = curator.Restore(slo=slo,
                                     name=snapshot_name,
                                     indices=indices,
                                     include_aliases=include_aliases,
                                     rename_pattern=rename_pattern,
                                     rename_replacement=rename_replacement,
                                     skip_repo_fs_check=True)
    restore_action.do_action()

  def restore_templates(self):
    try:
      with open(f'{self._storage_folder}/component_templates.json', 'r') as f:
        logging.info('Restoring component templates')
        component_templates = json.load(f)
        if self._dbs:
          component_templates_by_dbs = {
            prefix: [template for template in component_templates
                     if template['name'].startswith(prefix)]
            for prefix in self._dbs}
          component_templates = sum(
              [self.rename_component_templates(templates, prefix)
               if prefix in self._renames else templates
               for prefix, templates in component_templates_by_dbs.items()], [])
        for template in component_templates:
          self._client.cluster.put_component_template(name=template['name'],
                                                      body=template[
                                                        'component_template'])
    except FileNotFoundError:
      logging.info('No component templates to restore')

    try:
      with open(f'{self._storage_folder}/templates.json', 'r') as f:
        logging.info('Restoring index templates')
        templates = json.load(f)
        if self._dbs:
          templates_by_dbs = {prefix: [template for template in templates
                                       if template['name'].startswith(prefix)]
                              for prefix in self._dbs}
          templates = sum([self.rename_templates(templates, prefix)
                           if prefix in self._renames else templates
                           for prefix, templates in templates_by_dbs.items()],
                          [])
        for template in templates:
          self._client.indices.put_index_template(name=template['name'],
                                                  body=template[
                                                    'index_template'])
    except FileNotFoundError:
      logging.info('No index templates to restore')

    try:
      with open(f'{self._storage_folder}/obsolete_templates.json', 'r') as f:
        logging.info('Restoring obsolete index templates')
        obsolete_templates = json.load(f).items()
        if self._dbs:
          obsolete_templates_by_dbs = {
            prefix: {name: body for name, body in obsolete_templates
                     if name.startswith(prefix)} for prefix in self._dbs}
          obsolete_templates = [(self.rename(name, prefix),
                                 self.rename_template(body, prefix))
                                if prefix in self._renames else (name, body)
                                for prefix, temps in obsolete_templates_by_dbs.items()
                                for name, body in temps.items()]
        for name, body in obsolete_templates:
          if name == "tenant_template":
            continue
          self._client.indices.put_template(name=name, body=body)
    except FileNotFoundError:
      logging.info('No obsolete index templates to restore')

  def rename(self, name: str, prefix: str):
    return re.sub(f'{prefix}(.*)', f'{self._renames[prefix]}\\1', name)

  def rename_templates(self, templates: list, prefix: str):
    return [self.rename_template(template, prefix, 'index_template')
            for template in templates]

  def rename_component_templates(self, templates: list, prefix: str):
    return [self.rename_template(template, prefix, 'component_template')
            for template in templates]

  def rename_template(self, template: dict, prefix: str, template_key=None):
    name = template.get('name')
    if name:
      template['name'] = self.rename(name, prefix)
    # Get template settings depending on template type
    inner_template = template
    if template_key and template.get(template_key):
      inner_template = template.get(template_key)
    # Rename index patterns
    index_patterns = inner_template.get('index_patterns')
    if index_patterns:
      renamed_index_patterns = []
      for index_pattern in index_patterns:
        renamed_index_patterns.append(self.rename(index_pattern, prefix))
      inner_template['index_patterns'] = renamed_index_patterns
    # Rename used component templates
    composed_of = inner_template.get('composed_of')
    if composed_of:
      templates = []
      for template_name in composed_of:
        templates.append(self.rename(template_name, prefix))
      inner_template['composed_of'] = templates
    # Rename aliases depending on template type
    if inner_template.get('template'):
      inner_template['template']['aliases'] = \
        (self.rename_index_aliases(inner_template['template'], prefix))[
          'aliases']
    else:
      inner_template['aliases'] = \
        self.rename_index_aliases(inner_template, prefix)['aliases']
    return template

  def rename_index_aliases(self, index_aliases: dict, prefix: str) -> dict:
    renamed_index_aliases = {'aliases': {}}
    aliases = index_aliases.get('aliases')
    for name, body in aliases.items() if aliases else {}:
      renamed_index_aliases['aliases'][self.rename(name, prefix)] = body
    return renamed_index_aliases

  def clean_restore(self):
    if self._clean_recovery:
      pattern = '*'
      if self._dbs:
        pattern_list = [f'{self._renames.get(db)}*' if self._renames.get(db)
                        else f'{db}*' for db in self._dbs]
        for prefix in pattern_list:
          try:
            self._client.indices.delete_index_template(prefix)
            self._client.cluster.delete_component_template(prefix)
            self._client.indices.delete_template(prefix)
          except NotFoundError:
            pass
        pattern = ','.join(pattern_list)
      else:
        self._client.indices.delete_index_template(pattern)
        self._client.cluster.delete_component_template(pattern)
        self._client.indices.delete_template(pattern)
      self._client.indices.delete(f'{pattern},-.*')


def recover_users():
  adapter_client = dbaas_client.DbaasAdapterClient()
  aggregator_client = dbaas_client.DbaasAggregatorClient()
  db_identifier = os.environ.get(
      'DBAAS_AGGREGATOR_PHYSICAL_DATABASE_IDENTIFIER', "")

  if adapter_client.url == "" or aggregator_client.url == "" or db_identifier == "":
    logging.info('Users recovery is disabled')
    return

  logging.info('Users recovery has started')

  data = {
    'physicalDbId': db_identifier,
    'type': 'opensearch',
    'settings': {}
  }

  resp = adapter_client.send_request(
      path='/api/v2/dbaas/adapter/opensearch/users/restore-password/state',
      method='GET')

  state = resp.text

  if state != USERS_RECOVERY_RUNNING_STATE:
    state = USERS_RECOVERY_IDLE_STATE
  restore_failed = False
  while state != USERS_RECOVERY_DONE_STATE and state != USERS_RECOVERY_FAILED_STATE:
    if state == USERS_RECOVERY_IDLE_STATE:
      start_time = time.time()
      while True:
        if time.time() - start_time > TIMEOUT:
          logging.info("Timeout reached during users passwords recovery")
          restore_failed = True
          break
        restore_failed = False
        try:
          resp = aggregator_client.send_request(
              path='/api/v3/dbaas/internal/physical_databases/users/restore-password',
              method='POST',
              data=json.dumps(data))
        except requests.ConnectionError or requests.HTTPError:
          logging.info(
              "Unable to restore user passwords via DBaaS aggregator")
          restore_failed = True
        if resp.status_code == 200 and not restore_failed:
          break
        else:
          time.sleep(INTERVAL)

      if restore_failed:
        state = USERS_RECOVERY_FAILED_STATE
        continue
    time.sleep(5)
    try:
      resp = adapter_client.send_request(
          path='/api/v2/dbaas/adapter/opensearch/users/restore-password/state',
          method='GET')
    except requests.ConnectionError or requests.HTTPError:
      logging.info("Unable to restore user passwords via DBaaS aggregator")
      continue
    if resp.status_code != 200:
      logging.info("Unable to restore user passwords via DBaaS aggregator")
      continue
    state = resp.text

  if state == USERS_RECOVERY_FAILED_STATE or restore_failed:
    raise Exception(
        "User recovery has failed with following state: {}".format(state))
  logging.info("Users recovery is finished with %s state", state)


if __name__ == "__main__":
  parser = argparse.ArgumentParser()
  parser.add_argument('folder')
  parser.add_argument('-skip_users_recovery')
  parser.add_argument('-d', '--dbs')
  parser.add_argument('-m', '--dbmap')
  parser.add_argument('-clean')
  args = parser.parse_args()

  logging.info('Restore has started.')

  restore_instance = Restore(args=args)

  if args.dbs:
    restore_instance.granular_restore()
  elif 'granular' in args.folder:
    raise Exception('Attempt to restore granular backup without databases')
  else:
    restore_instance.full_restore()

  skip_users_recovery = utils.str2bool(
    args.skip_users_recovery) if args.skip_users_recovery else False
  if not skip_users_recovery:
    recover_users()

  logging.info('Restore is completed successfully.')
