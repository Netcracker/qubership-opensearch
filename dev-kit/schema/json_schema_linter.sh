#!/usr/bin/env bash

# Refer to https://git.netcracker.com/Personal.Streaming.Platform/values-schema-generator#user-guide
# to install nc-schema-gen

ROOT_PATH=../..
nc-schema-gen --config ${ROOT_PATH}/.jsonschema-gen.yaml --linter