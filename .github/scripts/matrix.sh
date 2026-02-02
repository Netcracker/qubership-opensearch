#! /bin/bash

# Usage: ./matrix.sh .github/dev-build-config.cfg .github/outputs/all_changed_files.json

components_file="$1"
files_file="$2"

# Use jq to filter components where any file starts with any changeset entry in the list
# shellcheck disable=2086
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
