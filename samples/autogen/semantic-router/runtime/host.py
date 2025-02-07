# SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
# SPDX-License-Identifier: Apache-2.0

import asyncio
import logging

from autogen_core.application import WorkerAgentRuntimeHost
from autogen_core.application.logging import TRACE_LOGGER_NAME


async def run_host():
    host = WorkerAgentRuntimeHost(address="0.0.0.0:50051")
    host.start()  # Start a host service in the background.
    await host.stop_when_signal()


if __name__ == "__main__":
    logging.basicConfig(level=logging.DEBUG)
    logger = logging.getLogger(f"{TRACE_LOGGER_NAME}.host")
    asyncio.run(run_host())
