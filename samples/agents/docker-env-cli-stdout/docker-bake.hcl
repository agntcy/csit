// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

group "default" {
  targets = ["docker-env-cli-stdout"]
}

target "docker-env-cli-stdout" {
  context = "."
  dockerfile = "./Dockerfile"
  tags = ["docker-env-cli-stdout:latest"]
}
