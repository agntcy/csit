# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

import asyncio
from llama_deploy import (
    WorkflowServiceConfig,
    ControlPlaneConfig,
    deploy_workflow,
)
from llama_sum.workflows import NumGenWorkflow
from llama_sum.workflows import SumWorkflow
import os

async def main():
    workflow_host = os.getenv("WORKFLOW_HOST", "127.0.0.1")
    workflow_port = os.getenv("WORKFLOW_PORT", 8002)
    workflow_internal_host = os.getenv("WORKFLOW_INTERNAL_HOST", None)
    workflow_internal_port = os.getenv("WORKFLOW_INTERNAL_PORT", None)
    workflow_name = os.getenv("WORKFLOW_NAME", "sum")

    sum = SumWorkflow()
    sum.add_workflows(gen=NumGenWorkflow())

    sum_task = asyncio.create_task(
        deploy_workflow(
            sum,
            WorkflowServiceConfig(
                host=workflow_host, 
                port=workflow_port,
                internal_host=workflow_internal_host,
                internal_port=workflow_internal_port,
                service_name=workflow_name,
            ),
            ControlPlaneConfig(),
        )
    )

    await asyncio.gather(sum_task)

if __name__ == "__main__":
    import asyncio

    asyncio.run(main())