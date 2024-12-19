import yaml


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


values_file = "./charts/helm/opensearch-service/values.yaml"
override_file = "./charts/helm/opensearch-service/values.override.yaml"

base_values = load_yaml(values_file)
override_values = load_yaml(override_file)

merge_yaml(base_values, override_values)

# Save back to values.yaml
save_yaml(base_values, values_file)

# print(f"Merged {override_file} into {values_file}")
