// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

group "default" {
  targets = ["docker-env-cli-stdout"]
}

target "docker-env-cli-stdout" {
  context = "./docker-env-cli-stdout"
  dockerfile = "./Dockerfile"
  tags = ["docker-env-cli-stdout:latest"]
}
