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
import time

import curator
import opensearchpy
import schedule
import urllib3
import yaml

urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

loggingLevel = logging.INFO
logging.basicConfig(level=loggingLevel,
                    format='[%(asctime)s,%(msecs)03d][%(levelname)s][category=Indices_cleaner] %(message)s',
                    datefmt='%Y-%m-%dT%H:%M:%S')

CONFIG_MAP_NAME = 'es-curator-configuration'

SCHEDULER_POSSIBLE_UNITS = [func for func in dir(schedule.Job)
                            if not callable(getattr(schedule.Job, func))
                            and not func.startswith("__")]

CONFIGURATION_KEYS = ['filter_kind', 'filter_value', 'filter_direction', 'filter_unit', 'filter_unit_count']


def _str2bool(v: str) -> bool:
    return v.lower() in ("yes", "true", "t", "1")


def create_elasticsearch_url():
    host = os.environ.get('ELASTICSEARCH_HOST', "")
    if not host:
        logging.error('Elasticsearch host or port does not specified')
        raise RuntimeError('Elasticsearch url does not specified correctly, host is empty')
    prefix = 'https' if _str2bool(os.environ.get('TLS_HTTP_ENABLED')) else 'http'
    return f'{prefix}://{host}'


def get_credentials():
    es_credentials = os.environ.get('ES_AUTH').split(":")
    user = es_credentials[0]
    password = es_credentials[1]
    return user, password if user and password else None


def get_certificate() -> str:
    root_ca_certificate = os.environ.get('ROOT_CA_CERTIFICATE')
    return root_ca_certificate if root_ca_certificate else None


def prepare_elasticsearch_client():
    url = create_elasticsearch_url()
    credentials = get_credentials()
    ca_certs = get_certificate()
    return opensearchpy.OpenSearch([url], http_auth=credentials, ca_certs=ca_certs)


def delete_indices():
    configs = get_cleaner_config()
    if not configs:
        logging.info(f'Configuration in {CONFIG_MAP_NAME} config map is empty. Nothing to delete.')
        return
    check_configuration(configs)
    client = prepare_elasticsearch_client()
    for config in configs:
        _delete_indices_by_pattern(config, client)


def check_configuration(configs):
    check_configs_structure(configs)
    for config in configs:
        config_keys = list(config.keys())
        for key in CONFIGURATION_KEYS:
            if key not in config_keys:
                logging.error(f'Configuration {config["name"] if "name" in config_keys else config} '
                              f'should contain item {key}')
                raise RuntimeError(f'Invalid configuration for Elasticsearch Indices Cleaner, '
                                   f'key - {key} for configuration - {config} is unknown')
            if not isinstance(config[key], str) and not isinstance(config[key], int):
                logging.error(f'The key - {key} '
                              f'for configuration - {config["name"] if "name" in config_keys else config} '
                              f'should be string or integer')
                raise RuntimeError(f'Invalid configuration for Elasticsearch Indices Cleaner, key - {key} '
                                   f'for configuration - {config} should be string or integer, {config[key]} given')


def check_configs_structure(configs):
    if not isinstance(configs, list):
        logging.error(f'Configurations - {configs} should be array of configuration templates')
        raise RuntimeError(f'Not valid Elasticsearch Indices Cleaner configuration - {configs}, array is expected')


def _delete_indices_by_pattern(config, client):
    indices = curator.IndexList(client)
    indices.filter_by_regex(kind=config['filter_kind'], value=config['filter_value'])
    indices.filter_by_age(source='creation_date',
                          direction=config['filter_direction'],
                          unit=config['filter_unit'],
                          unit_count=config['filter_unit_count'])
    logging.info(f'indices for {config["name"]} are {indices.indices}')
    if indices.indices:
        delete_action = curator.DeleteIndices(indices)
        delete_action.do_action()


def get_cleaner_config():
    configuration_key = os.environ.get('INDICES_CLEANER_CONFIGURATION_KEY')
    if not configuration_key:
        logging.error('INDICES_CLEANER_CONFIGURATION_KEY variable can not be empty')
        raise RuntimeError('Invalid value INDICES_CLEANER_CONFIGURATION_KEY. It is empty!')

    with open("/opt/elasticsearch-indices-cleaner/cleaner.yml", "r") as fh:
        data = fh.read()

    return yaml.safe_load(data)[configuration_key]


def schedule_cleaning():
    scheduler_unit = os.environ.get('INDICES_CLEANER_SCHEDULER_UNIT')
    if scheduler_unit not in SCHEDULER_POSSIBLE_UNITS:
        raise RuntimeError(f'Scheduler unit {scheduler_unit} is not valid, '
                           f'possible values are {SCHEDULER_POSSIBLE_UNITS}')

    unit_count = os.environ.get('INDICES_CLEANER_SCHEDULER_UNIT_COUNT')

    if unit_count.find(":") != -1:
        time_list = unit_count.split(":")
        if len(time_list) > 2:
            logging.error('Invalid format for INDICES_CLEANER_SCHEDULER_UNIT_COUNT variable. Too many separators')
            raise RuntimeError(f'Invalid format for INDICES_CLEANER_SCHEDULER_UNIT_COUNT variable. '
                               f'It should contains not more than one ":" separator, {len(time_list) - 1} is given')
        if time_list[0] and not time_list[0].isdigit():
            logging.error(f'Invalid format for INDICES_CLEANER_SCHEDULER_UNIT_COUNT variable. '
                          f'Hours value set incorrect - {time_list[0]}')
            raise RuntimeError(f'Invalid hours value for Elasticsearch Indices Cleaner scheduler. '
                               f'Digit is expected, {time_list[0]} given')
        if time_list[1] and not time_list[1].isdigit():
            logging.error(f'Invalid format for INDICES_CLEANER_SCHEDULER_UNIT_COUNT variable. '
                          f'Minutes value set incorrect - {time_list[1]}')
            raise RuntimeError(f'Invalid minutes value for Elasticsearch Indices Cleaner scheduler. '
                               f'Digit is expected, {time_list[1]} given')

        schedule.every().__getattribute__(scheduler_unit).at(unit_count).do(delete_indices)
    else:
        if not unit_count.isdigit():
            logging.error(f'Invalid format for INDICES_CLEANER_SCHEDULER_UNIT_COUNT variable - {unit_count}. '
                          f'It should be integer value or two integer values separated by ":"')
            raise RuntimeError(f'Invalid format for INDICES_CLEANER_SCHEDULER_UNIT_COUNT variable - {unit_count}. '
                               f'Variable is not integer and is not separated by ":"')

        schedule.every(int(unit_count)).__getattribute__(scheduler_unit).do(delete_indices)


if __name__ == '__main__':
    schedule_cleaning()
    while True:
        schedule.run_pending()
        time.sleep(1)
