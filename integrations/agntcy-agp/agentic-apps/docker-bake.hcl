// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0


# Documentation available at: https://docs.docker.com/build/bake/

# Docker build args
variable "IMAGE_REPO" {default = "ghcr.io/cisco-eti/phoenix/container-images"}
variable "IMAGE_TAG" {default = "v0.0.0-dev"}
variable "POETRY_HTTP_BASIC_DEVHUBCLOUD_USERNAME" {default = ""}
variable "POETRY_HTTP_BASIC_DEVHUBCLOUD_PASSWORD" {default = ""}

function "get_tag" {
  params = [tags, name]
  result = coalescelist(tags, ["${IMAGE_REPO}/${name}:${IMAGE_TAG}"])
}

group "default" {
  targets = [
    "phoenix-test-autogen-agent",
    "phoenix-test-langchain-agent",
  ]
}

target "_common" {
  output = [
    "type=image",
  ]
  platforms = [
    "linux/arm64",
    "linux/amd64",
  ]
}

target "docker-metadata-action" {
  tags = []
}

target "phoenix-test-autogen-agent" {
  args = {
    POETRY_HTTP_BASIC_DEVHUBCLOUD_USERNAME = "${POETRY_HTTP_BASIC_DEVHUBCLOUD_USERNAME}"
    POETRY_HTTP_BASIC_DEVHUBCLOUD_PASSWORD = "${POETRY_HTTP_BASIC_DEVHUBCLOUD_PASSWORD}"
  }
  context = "./autogen_agent"
  dockerfile = "./Dockerfile"
  inherits = [
    "_common",
    "docker-metadata-action",
  ]
  tags = get_tag(target.docker-metadata-action.tags, "${target.phoenix-test-autogen-agent.name}")
}

target "phoenix-test-langchain-agent" {
  args = {
    POETRY_HTTP_BASIC_DEVHUBCLOUD_USERNAME = "${POETRY_HTTP_BASIC_DEVHUBCLOUD_USERNAME}"
    POETRY_HTTP_BASIC_DEVHUBCLOUD_PASSWORD = "${POETRY_HTTP_BASIC_DEVHUBCLOUD_PASSWORD}"
  }
  context = "./langchain_agent"
  dockerfile = "./Dockerfile"
  inherits = [
    "_common",
    "docker-metadata-action",
  ]
  tags = get_tag(target.docker-metadata-action.tags, "${target.phoenix-test-langchain-agent.name}")
}
