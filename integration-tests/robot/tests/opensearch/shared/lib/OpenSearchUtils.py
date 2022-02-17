import os


class OpenSearchUtils:

    def __init__(self, *args):
        self.template = "for file in /usr/share/opensearch/data/nodes/0/indices/{0}/{1}/index/*; do echo 123 > ${{file}}; done "
        self.create_keyword = 'create'
        self.update_keyword = 'update'
        self.setting_keyword = 'settings'
        self.json_keyword = 'json'
        self.index_keyword = '"_index":"'
        self.quote = '"'
        self.separators = [' ', '_', '-']

    @staticmethod
    def get_primary_shard_description_to_corrupt(index_information, master_node):
        result = {}
        for row in index_information:
            if row['node'] == master_node and row['prirep'] == 'p':
                result = row
        shard = result['shard']
        for row in index_information:
            if row['shard'] == shard and row['prirep'] == 'r':
                result['replica_service'] = row['node']
        return result

    @staticmethod
    def get_replica_shard_description_to_corrupt(index_information):
        for row in index_information:
            if row['prirep'] == 'r':
                return row

    @staticmethod
    def replica_shard_become_primary(index_info: list, shard_number: str, expected_service: str) -> bool:
        """
        Returns True if replica shard become a primary one.
        :param index_info: list of dicts which describes shards distribution
        :param shard_number: number of shard which replica method looks up
        :param expected_service: node name for primary shard which has been corrupted
        :return: bool
        """
        for row in index_info:
            if row['shard'] == shard_number and row['prirep'] == 'p' and row['node'] == expected_service:
                return True
        return False

    @staticmethod
    def is_replica_shard_relocated(index_info: list, shard_number: str, expected_service: str) -> bool:
        """
        Returns True if replica shard is relocated to another node.
        :param index_info: list of dicts which describes shards distribution
        :param shard_number: number of shard which replica method looks up
        :param expected_service: previous node which contained replica of current shard
        :return: bool
        """
        for row in index_info:
            if row['shard'] == shard_number and row['prirep'] == 'r' and row['node'] != expected_service:
                return True
        return False

    def get_command_to_corrupt_shard(self, uuid, shard_number):
        return self.template.format(uuid, shard_number)

    def get_generated_test_case_data(self, test_case_name: str, index_name: str,
                                     resources_folder="tests/opensearch/ha/test-data-resources", separator=" "):
        """
        The method looks up test case folder in the resource directory. There is an ability to use different separators
        for test case's folder name.
        :param test_case_name: the name of the test case
        :param index_name: OpenSearch index name
        :param resources_folder: the relative path to common resource directory where all test cases folders are
        :param separator: one of the possible separators for test case names
        :return dictionary with three keys: index_create_file, index_update_file, index_settings_file. The values
        are appropriate file names.

        The method raises an exception if resource directory has incorrect relative path.

        Example:
        | Get Generated Test Case Data | Data Files Corrupted On Primary Shard |
        """
        processed_name = self._transform_name_to_unified_format(test_case_name, separator)
        test_case_folder = None
        for directory in os.listdir(resources_folder):
            if self._transform_name_to_unified_format(directory) == processed_name:
                test_case_folder = directory
                break
        if not test_case_folder:
            raise Exception("Incorrect directory for test case data")
        return self._find_generated_data_files(f'{resources_folder}/{test_case_folder}', index_name)

    def _transform_name_to_unified_format(self, name: str, separator=None):
        if not separator:
            separator = self._find_separator(name)
        return name.replace(separator, "").lower()

    def _find_separator(self, name: str):
        for separator in self.separators:
            if name.find(separator) != -1:
                return separator
        return None

    def _find_generated_data_files(self, directory, index_name):
        data_files = {}
        for file in os.listdir(directory):
            file_path = f'{directory}/{file}'
            if self.json_keyword not in file:
                self._set_index_in_file(file_path, index_name)
                if self.create_keyword in file:
                    data_files['index_create_file'] = file_path
                if self.update_keyword in file:
                    data_files['index_update_file'] = file_path
            if self.setting_keyword in file:
                data_files['index_settings_file'] = file_path
        return data_files

    def _set_index_in_file(self, file_path, index_name):
        """
        The method receives name of file which contains config or data for OpenSearch index.
        For example, it is a binary file for POST request to OpenSearch to update index.
        :param file_path: relative path to file
        :param index_name: OpenSearch index name. For example, `cats`
        """
        f = open(file_path, 'r')
        content = f.read()
        f.close()
        old = self._find_current_index(content)
        new = self.index_keyword + index_name + self.quote
        if not old or old == new:
            return
        new_content = content.replace(old, new)
        f = open(file_path, 'w')
        f.write(new_content)
        f.close()

    def _find_current_index(self, content):
        start = content.find(self.index_keyword)
        if start == -1:
            return None
        end = content.find(self.quote, start + len(self.index_keyword))
        index_name = content[start: end + 1]
        return index_name
