#! /bin/bash

# Usage: ./matrix.sh .github/dev-build-config.cfg .github/outputs/all_changed_files.json

components_file="$1"
files_file="$2"

# files=$(jq -c . "$files_file")

# Use jq to filter components where any file starts with any changeset entry in the list

jq -c --argjson files "$(cat $files_file)" '
  .components |
  [ .[] | select(
      . as $component |
      any($files[];
        [ $component.changeset[] as $ch | startswith($ch) ] | any
      )
    )
  ]
' "$components_file"
