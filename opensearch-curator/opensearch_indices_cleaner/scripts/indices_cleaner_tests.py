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

import unittest
import indices_cleaner
import os


class SchedulerTest(unittest.TestCase):
    def test_create_elasticsearch_url_is_empty(self):
        os.environ.__setitem__('ELASTICSEARCH_HOST', '')
        self.assertRaises(RuntimeError, indices_cleaner.create_elasticsearch_url)

    def test_create_elasticsearch_url_with_tls(self):
        os.environ.__setitem__('ELASTICSEARCH_HOST', 'elasticsearch:9200')
        os.environ.__setitem__('TLS_HTTP_ENABLED', 'true')
        actual_result = indices_cleaner.create_elasticsearch_url()
        self.assertEqual('https://elasticsearch:9200', actual_result)

    def test_create_elasticsearch_url_without_tls(self):
        os.environ.__setitem__('ELASTICSEARCH_HOST', 'elasticsearch:9200')
        os.environ.__setitem__('TLS_HTTP_ENABLED', 'false')
        actual_result = indices_cleaner.create_elasticsearch_url()
        self.assertEqual('http://elasticsearch:9200', actual_result)

    def test_get_credentials(self):
        os.environ.__setitem__('ES_AUTH', 'login:password')
        self.assertEqual(('login', 'password'), indices_cleaner.get_credentials())

    def test_check_configuration_with_invalid_main_structure(self):
        configs = {'name': 'zipkin',
                   'filter_kind': 'prefix',
                   'filter_value': 'streaming',
                   'filter_direction': 'older',
                   'filter_unit': 'days',
                   'filter_unit_count': '1'}
        self.assertRaises(RuntimeError, indices_cleaner.check_configuration, configs)

    def test_check_configuration_with_absent_parameter(self):
        configs = [{'name': 'zipkin',
                    'filter_kind': 'prefix',
                    'filter_value': 'streaming',
                    'filter_unit': 'days',
                    'filter_unit_count': '1'}]
        self.assertRaises(RuntimeError, indices_cleaner.check_configuration, configs)

    def test_check_configuration_with_invalid_nested_structure(self):
        configs = {'name': 'zipkin',
                   'filter_kind': 'prefix',
                   'filter_value': 'streaming',
                   'filter_direction': ['older'],
                   'filter_unit': 'days',
                   'filter_unit_count': '1'}
        self.assertRaises(RuntimeError, indices_cleaner.check_configuration, configs)

    def test_get_cleaner_config_without_configuration_key(self):
        os.environ.__setitem__('CLOUD_NAMESPACE', 'namespace')
        os.environ.__setitem__('INDICES_CLEANER_CONFIGURATION_KEY', '')
        self.assertRaises(RuntimeError, indices_cleaner.get_cleaner_config)

    def test_schedule_cleaning_scheduler_unit_is_empty(self):
        os.environ.__setitem__('INDICES_CLEANER_SCHEDULER_UNIT', '')
        os.environ.__setitem__('INDICES_CLEANER_SCHEDULER_UNIT_COUNT', '1')
        self.assertRaises(RuntimeError, indices_cleaner.schedule_cleaning)

    def test_schedule_cleaning_invalid_scheduler_unit_structure(self):
        os.environ.__setitem__('INDICES_CLEANER_SCHEDULER_UNIT', 'day')
        os.environ.__setitem__('INDICES_CLEANER_SCHEDULER_UNIT_COUNT', '11:22:33')
        self.assertRaises(RuntimeError, indices_cleaner.schedule_cleaning)

    def test_schedule_cleaning_invalid_scheduler_hours(self):
        os.environ.__setitem__('INDICES_CLEANER_SCHEDULER_UNIT', 'day')
        os.environ.__setitem__('INDICES_CLEANER_SCHEDULER_UNIT_COUNT', 'a:22')
        self.assertRaises(RuntimeError, indices_cleaner.schedule_cleaning)

    def test_schedule_cleaning_invalid_scheduler_minutes(self):
        os.environ.__setitem__('INDICES_CLEANER_SCHEDULER_UNIT', 'day')
        os.environ.__setitem__('INDICES_CLEANER_SCHEDULER_UNIT_COUNT', ':aa')
        self.assertRaises(RuntimeError, indices_cleaner.schedule_cleaning)

    def test_schedule_cleaning_invalid_scheduler_unit_count_format(self):
        os.environ.__setitem__('INDICES_CLEANER_SCHEDULER_UNIT', 'days')
        os.environ.__setitem__('INDICES_CLEANER_SCHEDULER_UNIT_COUNT', 'hour')
        self.assertRaises(RuntimeError, indices_cleaner.schedule_cleaning)
