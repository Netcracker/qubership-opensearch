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

import requests

DBASS_CERTS_PATHFILE = '/certs/dbaas-adapter/ca.crt'

request_headers = {
  'Content-type': 'application/json',
}


class DbaasAggregatorClient:

  def __init__(self):
    self._url = self.get_url()
    self._auth = self.get_credentials()
    self._verify = f'{DBASS_CERTS_PATHFILE}' if os.path.isfile(
        DBASS_CERTS_PATHFILE) else None

  def get_url(self):
    host = os.environ.get('DBAAS_AGGREGATOR_REGISTRATION_ADDRESS', "")
    if not host:
      logging.info('DBaaS Aggregator host does not specified')
      return None
    return host

  def get_credentials(self):
    user = os.environ.get('DBAAS_AGGREGATOR_REGISTRATION_USERNAME')
    password = os.environ.get('DBAAS_AGGREGATOR_REGISTRATION_PASSWORD')
    return (user, password) if user and password else None

  def send_request(self, path, method, data):
    return requests.request(url=self._url + path, method=method, data=data,
                     verify=self._verify, auth=self._auth,
                     headers=request_headers)

  @property
  def url(self):
    return self._url


class DbaasAdapterClient:

  def __init__(self):
    self._url = self.get_url()
    self._auth = self.get_credentials()
    self._verify = f'{DBASS_CERTS_PATHFILE}' if os.path.isfile(
        DBASS_CERTS_PATHFILE) else None

  def get_url(self):
    host = os.environ.get('DBAAS_ADAPTER_ADDRESS', "")
    if not host:
      logging.info('DBaaS Adapter host does not specified')
      return None
    return host

  def get_credentials(self):
    user = os.environ.get('DBAAS_ADAPTER_USERNAME')
    password = os.environ.get('DBAAS_ADAPTER_PASSWORD')
    return (user, password) if user and password else None

  def send_request(self, path, method, data=None):
    return requests.request(url=self._url + path, method=method, data=data,
                            verify=self._verify, auth=self._auth,
                            headers=request_headers)

  @property
  def url(self):
    return self._url
