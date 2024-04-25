import argparse
import json
import os
import re

"""
https://git.netcracker.com/Personal.Streaming.Platform/values-schema-generator

########################################## GLOBAL CONSTANTS ############################################################
"""

SERVICE_NAME_ENV = "SERVICE_NAME"
LEVELING_ENV = "LEVELING_ENABLED"
SKIP_TYPE_MISMATCH_ENV = "SKIP_TYPE_MISMATCH"
OFF_ARRAY_LIST_MISMATCH_LOG_ENV = "OFF_ARRAY_LIST_MISMATCH_LOG"
PARAMETER_FILTER_ENV = "PARAMETER_FILTER_ENV"

# Will be initialized in Main
SKIP_TYPE_MISMATCH = None
OFF_ARRAY_LIST_MISMATCH_LOG = None

SUPPORTED_SCHEMA_TYPES = ["string", "boolean", "integer", "object", "array", "list"]

SCHEMA_FIELD = "$schema"
ID_FIELD = "$id"
TYPE_FIELD = "type"
TITLE_FIELD = "title"
DESCRIPTION_FIELD = "description"
DEFAULT_FIELD = "default"
PROPERTIES_FIELD = "properties"

# values.rules.yaml
COMMENT_FIELD = "comment"
FLAGS_FIELD = "flags"
SWAP_FIELD = "swap"
REMOVE_FIELD = "remove"
ADD_FIELD = "add"

DOCUMENTATION_FILE = "installation.md"
VALUES_FILE = "values.yaml"
VALUES_RULES_FILE = "values.rules.json"
SKELETON_SCHEMA_FILE = "skeleton.values.schema.json"
RESULT_SCHEMA_FILE = "values.schema.json"

# Example: | Parameter | Type | Mandatory | Default value | Description |
TABLE_HEADER = r'^\|\s*Parameter\s*\|\s*Type\s*\|\s*Mandatory\s*\|\s*Default value\s*\|\s*Description\s*\|\s*$'
# 5 columns with "----" symbols and any number of spaces after it
TABLE_HEADER_DELIMITER = r'^\|-*\|-*\|-*\|-*\|-*\|\s*$'
# 5 columns and any number of spaces after it
TABLE_RECORD = r'^\|.*\|.*\|.*\|.*\|.*\|\s*$'
# First level header: '#' at the beginning, one space and any other symbols after it
FIRST_LEVEL_HEADER = r'^#\s.*$'
# Parameters header: '# Parameters' at the beginning any other symbols after it
PARAMETERS_HEADER = r'^# Parameters\s*$'

"""
########################################## FILE FUNCTIONS ##############################################################
"""


def get_installation_doc(_args):
    print("Create documentation reader...")
    return InstallationMDReader(get_file(_args.documentation, DOCUMENTATION_FILE, 'r', False))


def get_skeleton_file(_args):
    print("Generate the skeleton file...")
    if _args.values is None:
        os.system(f"helm schema-gen {VALUES_FILE} > {SKELETON_SCHEMA_FILE}")
    else:
        os.system(f"helm schema-gen {_args.values} > {SKELETON_SCHEMA_FILE}")
    return get_file(SKELETON_SCHEMA_FILE, "", 'r', False)


def get_output_schema_file(_args):
    print("Getting output schema file...")
    return get_file(_args.output, RESULT_SCHEMA_FILE, 'w', False)


def get_values_schema_rules(_args):
    print("Getting values schema rules...")
    file = get_file(_args.rules, VALUES_RULES_FILE, 'r', True)
    if file is None:
        return dict()
    return json.loads(file.read())


def get_file(_path: str, _default: str, _mode: str, optional: bool):
    try:
        return open(_path if _path else _default, _mode)
    except FileNotFoundError:
        if optional:
            print(f"Optional file {_path} not found")
            return None
        print(f"File {_path} not found :(")
        exit(1)


"""
########################################## InstallationMDReader ########################################################
"""


class InstallationMDReader:
    def __init__(self, installation_md_file):
        self.file = installation_md_file
        self.file_iterator = self.file.__iter__()

    def __iter__(self):
        self.sentinel = object()
        self.fit()
        return self

    def close(self):
        return

    def fit(self):
        if not self.find_parameters_section():
            print("Parameters section was not found")
            raise RuntimeError
        if not self.find_next_table():
            print("No tables were found")
            raise RuntimeError

    def find_parameters_section(self) -> bool:
        table_record = None
        while table_record is not self.sentinel:
            table_record = next(self.file_iterator, self.sentinel).rstrip()
            match = re.search(PARAMETERS_HEADER, table_record)
            if match:
                return True
        return False

    def find_next_table(self) -> bool:
        doc_line = next(self.file_iterator, self.sentinel)
        while doc_line is not self.sentinel:
            if re.search(TABLE_HEADER, doc_line):
                doc_line = next(self.file_iterator, self.sentinel)
                if re.search(TABLE_HEADER_DELIMITER, doc_line):
                    return True
            # New section after Parameters
            if re.search(FIRST_LEVEL_HEADER, doc_line):
                return False
            doc_line = next(self.file_iterator, self.sentinel)
        return False

    def __next__(self):
        next_record = self.find_next_parameter_record()
        if next_record is None:
            if not self.find_next_table():
                raise StopIteration
            return self.find_next_parameter_record()
        else:
            return next_record

    def find_next_parameter_record(self):
        doc_line = next(self.file_iterator, self.sentinel)
        if (doc_line is not self.sentinel
            and (re.search(TABLE_RECORD, doc_line))):
            return doc_line
        else:
            return None


"""
########################################## DOCUMENT FUNCTIONS ##########################################################
"""


def str2bool(value: str, _default: bool) -> bool:
    if value == "true":
        return True
    elif value == "false":
        return False
    else:
        print(f"Can't convert {value} to boolean, so return default - {_default}")
        return _default


def extract_description(line: str) -> str:
    """
    Function returns first sentence from the description line.
    :param line: the description line.
    :return: first sentence.
    """
    return line.strip().split(".")[0]


def extract_schema_flags(line: str):
    cleaned = clean_data(line)
    if len(cleaned) == 0:
        return None
    else:
        return json.loads(cleaned)


def clean_data(line: str):
    """
    :param line: line to clean
    :return: line without spaces and "`" symbols at the beginning and the end.
    """
    return line.strip().strip('`')


def extract_parameters_from_doc_record(record):
    """
    The method works with records that contains information about parameters from values.yaml
    :param record: the line from one of the tables from `Parameters` section from `installation.md`.
    :return: None if string is empty or line doesn't contain 5 columns, otherwise the function return list with
        the following information:
        1. Parameter path. E.g.: `service.resources.limits.cpu`
        2. Type. E.g.: boolean, integer, string, object, list, array
        3. Default value
        4. First sentence from the description.
    """
    record = record.rstrip()
    if record == "":
        return None
    record_parts = record.split('|')
    # 5 columns + 2 empty strings
    if len(record_parts) != 7:
        return None
    parameters = [clean_data(record_parts[1]),
                  clean_data(record_parts[2]),
                  clean_data(record_parts[4]),
                  extract_description(record_parts[5])]
    return parameters


"""
########################################## SCHEMA FUNCTIONS ############################################################
"""


def set_id(_schema, _path: str):
    _schema[ID_FIELD] = "#/properties/" + _path.replace(".", "/properties/")


def process_type_mismatch(_schema, _type: str, _path: str):
    schema_type = _schema[TYPE_FIELD]
    if schema_type == "array" and _type == "list":
        if not OFF_ARRAY_LIST_MISMATCH_LOG:
            print(f'Schema {_path} has different type in skeleton ({schema_type}) and '
                  f'documentation ({_type})! Will use type from skeleton - {schema_type}')
        return
    if SKIP_TYPE_MISMATCH:
        print(f'WARNING: {_path} schema has different type in skeleton ({schema_type}) and '
              f'documentation ({_type})! Will use type from documentation - {_type}')
    else:
        print(f'ERROR: {_path} schema has different type in skeleton ({schema_type}) and '
              f'documentation ({_type})')
        exit(1)


def set_type(_schema, _type: str, _path: str):
    if TYPE_FIELD in _schema and _schema[TYPE_FIELD] != _type:
        process_type_mismatch(_schema, _type, _path)

    if _type == "list" or _type == "array":
        _schema[TYPE_FIELD] = "array"
    elif _type not in ["string", "boolean", "integer", "object"]:
        print(f'ERROR: not supported type \'{_type}\', the supported types: {SUPPORTED_SCHEMA_TYPES}')
        exit(1)
    else:
        _schema[TYPE_FIELD] = _type


def set_title(_schema, _path: str):
    _schema[TITLE_FIELD] = f'The {_path} schema'


def set_description(_schema, _description: str):
    _schema[DESCRIPTION_FIELD] = _description


def set_default(_schema, _type: str, _default: str, _path: str):
    if len(_default) == 0:
        print(f"Received empty default for {_path}, so skip it")
        return
    elif _type == "string":
        _schema[DEFAULT_FIELD] = _default.strip().strip("\"")
    elif _type == "boolean":
        _schema[DEFAULT_FIELD] = bool(_default)
    elif _type == "integer":
        _schema[DEFAULT_FIELD] = int(_default)
    elif _type == "list":
        if _default == "[]":
            _schema[DEFAULT_FIELD] = []
        else:
            _schema[DEFAULT_FIELD] = _default[1:-1].split(', ')
    elif _type == "object":
        set_default_to_object(_schema, _default, _path)
    else:
        print(f"ERROR: {_path} has unsupported type {_type}")
        exit(1)


def set_default_to_object(_schema, _default: str, _path: str):
    if len(_default) == 0 or _default == "{}":
        _schema[DEFAULT_FIELD] = dict()
        return
    try:
        _schema[DEFAULT_FIELD] = json.loads(_default)
    except:
        print(f"WARNING: {_path} default = \"{_default}\" can't be converted as object, so use empty object as default")
        _schema[DEFAULT_FIELD] = dict()


def set_schema_flags(_schema, flags):
    for flag in list(flags.keys()):
        _schema[flag] = flags[flag]


def remove_schema_flags(_schema, flags):
    for flag in flags:
        del _schema[flag]


def swap_flags(_schema, swap_descriptor):
    # TODO: move checks to rules extraction from JSON
    if REMOVE_FIELD not in swap_descriptor or ADD_FIELD not in swap_descriptor:
        return
    to_remove = swap_descriptor[REMOVE_FIELD]
    to_add = swap_descriptor[ADD_FIELD]
    if to_remove in _schema:
        del _schema[to_remove]
    for key in to_add.keys():
        _schema[key] = to_add[key]


def process_schema_rules(_schema, _rules):
    if _rules is None:
        return
    if FLAGS_FIELD in _rules:
        set_schema_flags(_schema, _rules[FLAGS_FIELD])
    if SWAP_FIELD in _rules:
        for _swap_descriptor in _rules[SWAP_FIELD]:
            swap_flags(_schema, _swap_descriptor)


def set_properties(_schema):
    _schema[PROPERTIES_FIELD] = dict()


def fill_basic_information(_schema, _path: str, _type: str, _description: str, _schema_rules):
    set_id(_schema, _path)
    set_type(_schema, _type, _path)
    set_title(_schema, _path)
    set_description(_schema, _description)
    process_schema_rules(_schema, _schema_rules)


def fill_schema(root_schema, _path: str, _type: str, _default: str, _description: str, _schema_rules):
    """
    The function search the target schema in `root_schema` by using its path.
    Example:
        Root schema:
            {
                properties: {
                    a: {
                        properties: {
                            b {
                                properties: {
                                    c {}
                                }
                            }
                        }
                    },
                    d {}
                }
            }
        v_path: "a.b.c"
    If intermediate or target schema does not exist, then function creates it and fills it.
    Example:
        Root schema:
            {
                properties: {
                    a: {
                        properties: {},
                    },
                    d {}
                }
            }
        v_path: "a.b.c"
        So, function will create and fill schema for "b" and "c" parameters.
    :param _schema_flags: additional schema flags in JSON format
    :param root_schema: schema with the upper level schemas.
    :param _path: path to value schema. E.g.: "a.b.c"
    :param _type: type from the documentation.
    :param _default: default value from the documentation.
    :param _description: first sentence from the documentation.
    :return: nothing
    """
    path = _path.split(".")
    v_schema = root_schema
    path_element_id = 0

    for parameter in path:
        if PROPERTIES_FIELD not in v_schema:
            v_schema[PROPERTIES_FIELD] = dict()
        properties = v_schema[PROPERTIES_FIELD]
        if parameter in properties:
            v_schema = properties[parameter]
            path_element_id += 1
            continue

        # The value schema was not found, so need to generate it
        v_schema = dict()
        properties[parameter] = v_schema  # backward link

        is_target_schema = path_element_id == len(path) - 1
        if is_target_schema:
            print(f'Destination schema `{parameter}` not found, so create it. Full path: {_path}')
            # the table filling will be after cycle as for already generated schema.
            continue

        full_sub_path = '.'.join(path[0:path_element_id + 1])
        print(f'Intermediate schema `{parameter}` not found, so create it. Full path: {full_sub_path}')
        fill_basic_information(v_schema, full_sub_path, "object", f'{full_sub_path} parameters', _schema_rules)
        set_properties(v_schema)
        path_element_id += 1

    # The descriptor was already found or created, so just fill it
    fill_basic_information(v_schema, _path, _type, _description, _schema_rules)
    set_default(v_schema, _type, _default, _path)


def fill_presented_object_schemas(schema, _path=""):
    """
    The function fill intermediate object schemas that were not filled using the documentation
    :param schema: some value schema
    :param _path: path to schema
    :return: nothing
    """
    if PROPERTIES_FIELD not in schema or TYPE_FIELD not in schema or schema[TYPE_FIELD] != "object":
        return

    # Avoid root descriptor filling
    if _path != "":
        if ID_FIELD not in schema:
            set_id(schema, _path)
        if TITLE_FIELD not in schema:
            set_title(schema, _path)
        if DESCRIPTION_FIELD not in schema:
            set_description(schema, f'{_path} parameters')

    properties = schema[PROPERTIES_FIELD]
    if _path == "":
        for nested_schema in list(properties.keys()):
            fill_presented_object_schemas(properties[nested_schema], nested_schema)
    else:
        for nested_schema in list(properties.keys()):
            fill_presented_object_schemas(properties[nested_schema], f'{_path}.{nested_schema}')


"""
########################################## MAIN FUNCTION ###############################################################
"""


def perform_leveling(schema, _path=""):
    """
    The function search empty schemas (there is no "id" field) and removes them.
    It runs recursive and use "traversing in depth" strategy.
    :param schema: some schema
    :param _path: path to that schema. It is used for logging
    :return: True if schema doesn't have "id" field and no one property, otherwise it returns False.
    """
    properties_exist = PROPERTIES_FIELD in schema
    if properties_exist:
        properties = schema[PROPERTIES_FIELD]
        for nested_schema in list(properties.keys()):
            nested_path = _path + "." + nested_schema
            if perform_leveling(properties[nested_schema], nested_path):
                print(f"Clean {nested_path}")
                del properties[nested_schema]

    id_not_exist = ID_FIELD not in schema
    any_properties_exist = properties_exist and len(list(schema[PROPERTIES_FIELD].keys())) != 0
    return id_not_exist and not any_properties_exist


def get_service_name(_args):
    if _args.service_name is not None:
        return _args.service_name
    else:
        return os.getenv(SERVICE_NAME_ENV, "None")


def leveling_enabled(_args):
    env_value = os.getenv(LEVELING_ENV)
    if env_value is not None:
        return str2bool(env_value, False)
    return _args.leveling


def skip_type_mismatch_enabled(_args):
    env_value = os.getenv(SKIP_TYPE_MISMATCH_ENV)
    if env_value is not None:
        return str2bool(env_value, False)
    return _args.skip_type_mismatch


def array_list_mismatch_log_disabled(_args):
    env_value = os.getenv(OFF_ARRAY_LIST_MISMATCH_LOG_ENV)
    if env_value is not None:
        return str2bool(env_value, False)
    return _args.off_array_list_mismatch_log


def get_schema_rules(_path, _rules):
    for pattern in _rules.keys():
        if re.compile(pattern).match(_path):
            return _rules[pattern]
    return None


def get_path_filter(_args):
    env_value = os.getenv(PARAMETER_FILTER_ENV)
    if env_value is None:
        return re.compile(_args.filter)
    return re.compile(env_value)


def get_args():
    parser = argparse.ArgumentParser()

    # Generator configuration options
    parser.add_argument("-n", "--name", dest="service_name", help="The name of the service.", type=str)
    parser.add_argument("-l", "--leveling", action="store_true", dest="leveling", help="Enables `leveling` feature "
                                                                                       "when filler removes not "
                                                                                       "filled schemas.")
    parser.add_argument("--skip-type-mismatch", action="store_true", dest="skip_type_mismatch", help="Disable error "
                                                                                                     "throwing if "
                                                                                                     "schema type "
                                                                                                     "form skeleton "
                                                                                                     "and "
                                                                                                     "documentation "
                                                                                                     "are different")
    parser.add_argument("--off-array-list-mismatch-log", action="store_true",
                        dest="off_array_list_mismatch_log", help="Disable logging case when skeleton has \'array\'"
                                                                 "type and documentation has \'list\', because"
                                                                 "script will use \'array\' for schema.")

    # Files options
    parser.add_argument("-d", "--documentation", dest="documentation", help="Path to installation.md file", type=str)
    parser.add_argument("-v", "--values", dest="values", help="Path to values.yaml file", type=str)
    parser.add_argument("-o", "--output", dest="output", default="values.schema.json", help="Path to JSON file, which "
                                                                                            "should contain generated"
                                                                                            " schema. "
                                                                                            "`values.schema.json` is "
                                                                                            "default.", type=str)
    parser.add_argument("-r", "--rules", dest="rules", help="Path to values.rules.yaml file", type=str)
    parser.add_argument("-f", "--filter", dest="filter", help="Regular expression for determination the set "
                                                              "of parameters form the documentation "
                                                              "that should be proceed", type=str, default=".*")

    return parser.parse_args()


if __name__ == '__main__':
    print("Script running...")
    args = get_args()

    SKIP_TYPE_MISMATCH = skip_type_mismatch_enabled(args)
    OFF_ARRAY_LIST_MISMATCH_LOG = array_list_mismatch_log_disabled(args)

    skeleton_schema_file = get_skeleton_file(args)
    skeleton_schema = json.loads(skeleton_schema_file.read())
    set_title(skeleton_schema, get_service_name(args))

    installation_doc = get_installation_doc(args)
    output_file = get_output_schema_file(args)
    rules = get_values_schema_rules(args)
    _filter = get_path_filter(args)

    print("Fill skeleton using the documentation...")
    for doc_record in installation_doc:
        path, _type, default, description = extract_parameters_from_doc_record(doc_record)
        if re.search(_filter, path):
            _schema_rules = get_schema_rules(path, rules)
            fill_schema(skeleton_schema, path, _type, default, description, _schema_rules)

    if leveling_enabled(args):
        print("Clean up empty schemas...")
        perform_leveling(skeleton_schema)

    print("Fill intermediate object schemas...")
    fill_presented_object_schemas(skeleton_schema)

    json.dump(skeleton_schema, output_file, indent=4)
    print("JSON dump is node")

    if installation_doc is not None:
        installation_doc.close()
    if skeleton_schema_file is not None:
        skeleton_schema_file.close()
        os.remove(SKELETON_SCHEMA_FILE)
    if output_file is not None:
        output_file.close()
    print("Script finished")
