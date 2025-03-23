# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0


from llama_deploy import ControlPlaneConfig, LlamaDeployClient

result = None


def run_test():
    # points to deployed control plane
    client = LlamaDeployClient(ControlPlaneConfig())

    session = client.create_session()
    result = session.run("sum", max=10)
    v = result.split()
    # expected string in the form of
    # v1 + v2 = v3
    # we check it the sum returned is correct
    sum = int(v[0]) + int(v[2])
    assert v[4] == str(sum), (
        f"Got a wrong results. Expected {str(sum)}, received {v[4]}"
    )
    print("test succeded")


run_test()
