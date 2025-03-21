# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

import asyncio
import logging

from autogen_core import TRACE_LOGGER_NAME
from autogen_ext.runtimes.grpc import GrpcWorkerAgentRuntimeHost


async def run_host():
    host = GrpcWorkerAgentRuntimeHost(address="0.0.0.0:50051")
    host.start()  # Start a host service in the background.
    await host.stop_when_signal()


if __name__ == "__main__":
    logging.basicConfig(level=logging.DEBUG)
    logger = logging.getLogger(f"{TRACE_LOGGER_NAME}.host")
    asyncio.run(run_host())
