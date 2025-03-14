# SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
# SPDX-License-Identifier: Apache-2.0

import os
import sys

import requests
from langchain_openai import AzureChatOpenAI

model = None
output_string = None


def main():
    if len(sys.argv) != 2:
        print("Usage: python script.py <input_string>")
        sys.exit(1)

    input_string = sys.argv[1]

    key = os.environ.get("AZURE_OPENAI_API_KEY")
    if key is not None and len(key) > 0:
        print("Using Azure OpenAI")

        model = os.environ.get("AZURE_MODEL", "gpt-4o-mini")
        api_key = os.environ.get("AZURE_OPENAI_API_KEY")
        api_version = os.environ.get("AZURE_OPENAI_API_VERSION", "2025-02-01-preview")
        endpoint = os.environ.get("AZURE_OPENAI_ENDPOINT")
        deployment = os.environ.get("AZURE_OPENAI_DEPLOYMENT_NAME", "gpt-4o-mini")

        # Make request to Azure OpenAI
        llm = AzureChatOpenAI(
            model=model,
            api_key=api_key,
            api_version=api_version,
            azure_endpoint=endpoint,
            azure_deployment=deployment,
        )
        llm_response = llm.invoke(input_string)

        # Parse response
        output_string = llm_response.content

    else:
        print("Using Local OpenAI")

        model = os.environ.get("LOCAL_MODEL_NAME")
        url = os.environ.get("LOCAL_MODEL_BASE_URL")

        # Replace localhost with host.docker.internal
        url = url.replace("localhost", "host.docker.internal")

        # Make request to local model
        response = requests.post(
            url + "/api/generate",
            json={"model": model, "prompt": input_string, "stream": False},
        )

        # Parse response
        output_string = (
            response.json()["response"]
            if response.status_code == 200
            else f"Error: {response.text}"
        )

    output_string = f"Output: {output_string}"
    print(output_string)


if __name__ == "__main__":
    main()
