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

import os
import unittest
from datetime import datetime, timedelta

from slow_queries_metric import convert_slow_log_record, _process_log_records, filter_slow_logs_records

CURRENT_DATE = "2023-08-21T13:44:04"
SLOW_LOGS_FILENAME = "/opt/elasticsearch-monitoring/exec-scripts/test_resources/slow_logs.log"

log_record = ('[WARN ], 2023-07-27T08:00:43, [opensearch-1], [test-index-1][0] took[12.4ms], took_millis[12], '
              'total_hits[0 hits], stats[], search_type[QUERY_THEN_FETCH], total_shards[1], '
              'source[{"query":{"match":{"phrase":{"query":"heuristic","operator":"OR","prefix_length":0,'
              '"max_expansions":50,"fuzzy_transpositions":true,"lenient":false,"zero_terms_query":"NONE",'
              '"auto_generate_synonyms_phrase_query":true,"boost":1.0}}}}], id[],')


class TestSlowQueriesMetric(unittest.TestCase):

    def test_slow_log_conversion(self):
        record = convert_slow_log_record(log_record)
        self.assertEqual(record.log_level, 'WARN ')
        self.assertEqual(record.time, '2023-07-27T08:00:43')
        self.assertEqual(record.node, 'opensearch-1')
        self.assertEqual(record.index, 'test-index-1')
        self.assertEqual(record.shard, '0')
        self.assertEqual(record.took, '12.4ms')
        self.assertEqual(record.took_millis, '12')
        self.assertEqual(record.total_hits, '0')
        self.assertEqual(record.stats, '')
        self.assertEqual(record.search_type, 'QUERY_THEN_FETCH')
        self.assertEqual(record.total_shards, '1')
        self.assertEqual(record.source,
                         '{"query":{"match":{"phrase":{"query":"heuristic","operator":"OR","prefix_length":0,'
                         '"max_expansions":50,"fuzzy_transpositions":true,"lenient":false,"zero_terms_query":"NONE",'
                         '"auto_generate_synonyms_phrase_query":true,"boost":1.0}}}}')
        self.assertEqual(record.id_, '')

    def test_processing_log_records(self):
        time_to_stop = datetime.fromisoformat("2023-08-21T13:48:35") - timedelta(minutes=5)
        records = _process_log_records(SLOW_LOGS_FILENAME, time_to_stop)
        self.assertEqual(len(records), 2)
        self.assertEqual(str(records[0]),
                         '[INFO], 2023-08-21T13:43:55, [opensearch-0], [test_index][1] took[3.9s], took_millis[3925], total_hits[0 hits], stats[], search_type[QUERY_THEN_FETCH], total_shards[5], source[{"query":{"regexp":{"value":{"value":".*M.*","flags_value":255,"max_determinized_states":10000,"boost":1.0}}}}], id[]')
        self.assertEqual(str(records[1]),
                         '[INFO], 2023-08-21T13:43:55, [opensearch-0], [test_index][0] took[3.7s], took_millis[3772], total_hits[0 hits], stats[], search_type[QUERY_THEN_FETCH], total_shards[5], source[{"query":{"regexp":{"value":{"value":".*M.*","flags_value":255,"max_determinized_states":10000,"boost":1.0}}}}], id[]')

    def test_slow_logs_filtering_with_high_top_number(self):
        expected_records_list = [
            '[INFO], 2023-08-21T13:40:45, [opensearch-0], [test_index][0] took[1.2ms], took_millis[1], total_hits[0 hits], stats[], search_type[QUERY_THEN_FETCH], total_shards[5], source[{"size":10000,"query":{"regexp":{"value":{"value":"f2H.*","flags_value":255,"max_determinized_states":10000,"boost":1.0}}}}], id[],',
            '[INFO], 2023-08-21T13:41:16, [opensearch-0], [test_index][1] took[5.7s], took_millis[5735], total_hits[0 hits], stats[], search_type[QUERY_THEN_FETCH], total_shards[5], source[{"size":10000,"query":{"regexp":{"value":{"value":".*Nr.*","flags_value":255,"max_determinized_states":10000,"boost":1.0}}}}], id[],',
            '[INFO], 2023-08-21T13:42:04, [opensearch-0], [test_index][1] took[3.2s], took_millis[3252], total_hits[0 hits], stats[], search_type[QUERY_THEN_FETCH], total_shards[5], source[{"size":10000,"query":{"regexp":{"value":{"value":".*mM.*","flags_value":255,"max_determinized_states":10000,"boost":1.0}}}}], id[],',
            '[INFO], 2023-08-21T13:42:23, [opensearch-0], [test_index][1] took[2.4s], took_millis[2484], total_hits[0 hits], stats[], search_type[QUERY_THEN_FETCH], total_shards[5], source[{"size":10000,"query":{"regexp":{"value":{"value":".*N.*","flags_value":255,"max_determinized_states":10000,"boost":1.0}}}}], id[],',
            '[INFO], 2023-08-21T13:42:46, [opensearch-0], [test_index][0] took[69.8ms], took_millis[69], total_hits[4672 hits], stats[], search_type[QUERY_THEN_FETCH], total_shards[5], source[{"size":2048}], id[],',
            '[INFO], 2023-08-21T13:43:06, [opensearch-0], [test_index][1] took[3.7s], took_millis[3786], total_hits[4248 hits], stats[], search_type[QUERY_THEN_FETCH], total_shards[5], source[{"size":10000,"query":{"regexp":{"value":{"value":".*1.*","flags_value":255,"max_determinized_states":10000,"boost":1.0}}}}], id[],',
            '[INFO], 2023-08-21T13:43:55, [opensearch-0], [test_index][1] took[3.9s], took_millis[3925], total_hits[0 hits], stats[], search_type[QUERY_THEN_FETCH], total_shards[5], source[{"query":{"regexp":{"value":{"value":".*M.*","flags_value":255,"max_determinized_states":10000,"boost":1.0}}}}], id[],',
        ]
        file_name = 'slow_logs_expected.log'
        with open(file_name, mode='w') as file:
            file.write('\n'.join(expected_records_list))
        time_to_stop = datetime.fromisoformat(CURRENT_DATE) - timedelta(minutes=5)
        expected_records = _process_log_records(file_name, time_to_stop)
        os.remove(file_name)

        log_objs = _process_log_records(SLOW_LOGS_FILENAME, time_to_stop)
        records = filter_slow_logs_records(log_objs, 10)

        self.assertEqual(len(records), len(expected_records))
        self.assertCountEqual(records, expected_records)

    def test_slow_logs_filtering_with_low_top_number(self):
        expected_records_list = [
            '[INFO], 2023-08-21T13:41:16, [opensearch-0], [test_index][1] took[5.7s], took_millis[5735], total_hits[0 hits], stats[], search_type[QUERY_THEN_FETCH], total_shards[5], source[{"size":10000,"query":{"regexp":{"value":{"value":".*Nr.*","flags_value":255,"max_determinized_states":10000,"boost":1.0}}}}], id[],',
            '[INFO], 2023-08-21T13:42:04, [opensearch-0], [test_index][1] took[3.2s], took_millis[3252], total_hits[0 hits], stats[], search_type[QUERY_THEN_FETCH], total_shards[5], source[{"size":10000,"query":{"regexp":{"value":{"value":".*mM.*","flags_value":255,"max_determinized_states":10000,"boost":1.0}}}}], id[],',
            '[INFO], 2023-08-21T13:42:23, [opensearch-0], [test_index][1] took[2.4s], took_millis[2484], total_hits[0 hits], stats[], search_type[QUERY_THEN_FETCH], total_shards[5], source[{"size":10000,"query":{"regexp":{"value":{"value":".*N.*","flags_value":255,"max_determinized_states":10000,"boost":1.0}}}}], id[],',
            '[INFO], 2023-08-21T13:43:06, [opensearch-0], [test_index][1] took[3.7s], took_millis[3786], total_hits[4248 hits], stats[], search_type[QUERY_THEN_FETCH], total_shards[5], source[{"size":10000,"query":{"regexp":{"value":{"value":".*1.*","flags_value":255,"max_determinized_states":10000,"boost":1.0}}}}], id[],',
            '[INFO], 2023-08-21T13:43:55, [opensearch-0], [test_index][1] took[3.9s], took_millis[3925], total_hits[0 hits], stats[], search_type[QUERY_THEN_FETCH], total_shards[5], source[{"query":{"regexp":{"value":{"value":".*M.*","flags_value":255,"max_determinized_states":10000,"boost":1.0}}}}], id[],',
        ]
        file_name = 'slow_logs_expected.log'
        with open(file_name, mode='w') as file:
            file.write('\n'.join(expected_records_list))
        time_to_stop = datetime.fromisoformat(CURRENT_DATE) - timedelta(minutes=5)
        expected_records = _process_log_records(file_name, time_to_stop)
        os.remove(file_name)

        log_objs = _process_log_records(SLOW_LOGS_FILENAME, time_to_stop)
        records = filter_slow_logs_records(log_objs, 5)

        self.assertEqual(len(records), len(expected_records))
        self.assertCountEqual(records, expected_records)


if __name__ == '__main__':
    unittest.main()
