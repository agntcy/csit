# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

"""CSIT SLIMRPC echo server (Python / slima2a)."""

from __future__ import annotations

import argparse
import asyncio
import logging
import os

import slim_bindings
from a2a.client import minimal_agent_card
from a2a.server.request_handlers import DefaultRequestHandler
from a2a.server.tasks import InMemoryTaskStore

from csit_echo_executor import CsitEchoExecutor

READY = "CSIT_SLIM_SERVER_READY"


def _split_identity(s: str) -> tuple[str, str, str]:
    p = [x for x in s.strip("/").split("/") if x]
    if len(p) != 3:
        raise SystemExit(f"identity must be ns/group/name, got {s!r}")
    return p[0], p[1], p[2]


async def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--slim-url",
        default=os.environ.get("SLIM_SERVER", "http://127.0.0.1:46357"),
    )
    parser.add_argument(
        "--identity",
        default="agntcy/a2a_csit_slim/server_python",
        help="Full SLIM name ns/group/name",
    )
    parser.add_argument(
        "--secret",
        default=os.environ.get(
            "SLIM_SHARED_SECRET", "my_shared_secret_for_testing_purposes_only"
        ),
    )
    parser.add_argument("--log-level", default="ERROR")
    args = parser.parse_args()

    logging.basicConfig(level=getattr(logging, args.log_level.upper(), logging.ERROR))

    ns, grp, nm = _split_identity(args.identity)

    from slima2a import setup_slim_client

    _service, local_app, local_name, conn_id = await setup_slim_client(
        namespace=ns,
        group=grp,
        name=nm,
        slim_url=args.slim_url,
        secret=args.secret,
        log_level="error",
    )

    agent_card = minimal_agent_card(args.identity, ["slimrpc"])
    # Advertise streaming so SendStreamingMessage is accepted (needed by the streaming
    # and task-cancel scenarios, which drive the task via the streaming RPC).
    agent_card.capabilities.streaming = True
    agent_executor = CsitEchoExecutor()
    task_store = InMemoryTaskStore()
    handler = DefaultRequestHandler(
        agent_executor=agent_executor,
        task_store=task_store,
    )

    server = slim_bindings.Server.new_with_connection(local_app, local_name, conn_id)

    from slima2a.handler import SRPCHandler
    from slima2a.types.v1.a2a_pb2_slimrpc import (
        add_A2AServiceServicer_to_server as add_v1,
    )

    slim_handler = SRPCHandler(agent_card, handler)
    add_v1(slim_handler, server)

    print(READY, flush=True)
    await server.serve_async()


if __name__ == "__main__":
    asyncio.run(main())
