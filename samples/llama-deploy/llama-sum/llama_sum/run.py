# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

from llama_deploy import LlamaDeployClient, ControlPlaneConfig

# points to deployed control plane
client = LlamaDeployClient(ControlPlaneConfig())

session = client.create_session()
result = session.run("sum", max=10)
print(result)