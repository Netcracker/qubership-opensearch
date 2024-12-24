import yaml
import argparse

def load_yaml(file_path):
  with open(file_path, "r") as file:
    return yaml.safe_load(file)


def save_yaml(data, file_path):
  with open(file_path, "w") as file:
    yaml.dump(data, file, default_flow_style=False, sort_keys=False)


def merge_yaml(base, override):
  for key, value in override.items():
    if isinstance(value, dict) and key in base and isinstance(base[key], dict):
      merge_yaml(base[key], value)
    else:
      base[key] = value


parser = argparse.ArgumentParser()
parser.add_argument(
  "main",
  type=str
)
parser.add_argument(
  "override",
  type=str
)

args = parser.parse_args()
values_file = args.main
override_file = args.override

base_values = load_yaml(values_file)
override_values = load_yaml(override_file)

merge_yaml(base_values, override_values)

save_yaml(base_values, values_file)
