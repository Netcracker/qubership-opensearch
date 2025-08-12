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

import logging
import os

import opensearchpy


def extract_snapshot_name(folder):
  return os.path.basename(folder).lower()


def str2bool(v: str) -> bool:
  return v.lower() in ("yes", "true", "t", "1")


def create_elasticsearch_url():
  host = os.environ.get('ES_HOST', "")
  if not host:
    logging.error('Elasticsearch host or port is not specified')
    raise RuntimeError(
        'Elasticsearch url is not specified correctly, host is empty')
  prefix = 'https' if str2bool(os.environ.get('TLS_HTTP_ENABLED')) else 'http'
  return f'{prefix}://{host}'


def get_credentials():
  user = os.environ.get('ES_USERNAME')
  password = os.environ.get('ES_PASSWORD')
  return user, password if user and password else None


def get_certificate() -> str:
  root_ca_certificate = os.environ.get('ROOT_CA_CERTIFICATE')
  return root_ca_certificate if root_ca_certificate else None


def prepare_elasticsearch_client():
  url = create_elasticsearch_url()
  credentials = get_credentials()
  ca_certs = get_certificate()
  return opensearchpy.OpenSearch([url], http_auth=credentials,
                                 ca_certs=ca_certs, timeout=60)
