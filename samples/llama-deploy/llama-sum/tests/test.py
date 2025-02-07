# SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
# SPDX-License-Identifier: Apache-2.0

import json
import os
import sys
import time

from llama_deploy import ControlPlaneConfig, LlamaDeployClient

sys.path.append(
    os.path.abspath(os.path.join(os.path.dirname(__file__), "../../../.."))
)
from integrations.report.template.python.report_crew import Report

init_timestamp = time.time()

# points to deployed control plane
client = LlamaDeployClient(ControlPlaneConfig())

session = client.create_session()
result = session.run("sum", max=10)
v = result.split()
# expected string in the form of
# v1 + v2 = v3
# we check it the sum returned is correct
sum = int(v[0]) + int(v[2])
assert v[4] == str(
    sum
), f"Got a wrong results. Expected {str(sum)}, received {v[4]}"
print("test succeded")

# Fill the report
report = Report(
    duration=time.time() - init_timestamp,
    timestamp=init_timestamp,
    extra_data={
        "input": {"operation": "sum", "max": 10},
        "output": result,
    },
)
report.load_metadata()
report.export()
