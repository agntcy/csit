#!/bin/bash
# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0


# List of environment variables to save
env_vars=(
  "AZURE_MODEL"
  "AZURE_OPENAI_API_KEY"
  "AZURE_OPENAI_API_VERSION"
  "AZURE_OPENAI_ENDPOINT"
  "AZURE_OPENAI_DEPLOYMENT_NAME"
  "LOCAL_MODEL_NAME"
  "LOCAL_MODEL_BASE_URL"
)

# Path to the .env file
env_file_path=".env"

# Create or overwrite the .env file
> "$env_file_path"

# Loop through the environment variables and save them to the .env file
for var in "${env_vars[@]}"; do
  value=$(printenv "$var")
  if [ -n "$value" ]; then
    echo "$var=$value" >> "$env_file_path"
  fi
done
