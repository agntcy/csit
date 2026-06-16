# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

"""CSIT SLIMRPC probe client (Python / slima2a + a2a-sdk)."""

from __future__ import annotations

import argparse
import asyncio
import logging
import os
import sys

import httpx
from a2a.client import ClientFactory, create_text_message_object, minimal_agent_card
from a2a.types.a2a_pb2 import SendMessageRequest
from slima2a import setup_slim_client
from slima2a.client_transport import ClientConfig, SRPCTransport, slimrpc_channel_factory


def _split_identity(s: str) -> tuple[str, str, str]:
    p = [x for x in s.strip("/").split("/") if x]
    if len(p) != 3:
        raise SystemExit(f"identity must be ns/group/name, got {s!r}")
    return p[0], p[1], p[2]


def _task_state_name(task) -> str:
    """Return the canonical TASK_STATE_* token for the observed task, matching a2a-go."""
    if task is None:
        return ""
    try:
        from a2a.types.a2a_pb2 import TaskState

        return TaskState.Name(task.status.state)
    except Exception:  # pragma: no cover - defensive, fall back to a string form
        try:
            return str(task.status.state)
        except Exception:
            return ""


async def collect_response(client, text: str) -> tuple[str, object, bool]:
    """Drain send_message until the stream ends.

    Returns the accumulated echoed text, the last observed task (for the
    terminal lifecycle state) and whether a text artifact was seen. Uses the
    non-streaming ClientConfig so send_message completes like the Go unary client.
    """
    message = create_text_message_object(content=text)
    request = SendMessageRequest(message=message)
    out = ""
    last_task = None
    artifact_present = False
    async for stream_response, task in client.send_message(request=request):
        which = stream_response.WhichOneof("payload")
        if which == "message":
            for part in stream_response.message.parts:
                if part.WhichOneof("content") == "text":
                    out += part.text
        if task is not None:
            last_task = task
            for artifact in task.artifacts:
                for part in artifact.parts:
                    if part.WhichOneof("content") == "text":
                        out += part.text
                        artifact_present = True
        if which == "artifact_update":
            artifact = stream_response.artifact_update.artifact
            for part in artifact.parts:
                if part.WhichOneof("content") == "text":
                    out += part.text
                    artifact_present = True
        if text in out:
            break
    return out, last_task, artifact_present


async def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--slim-url",
        default=os.environ.get("SLIM_SERVER", "http://127.0.0.1:46357"),
    )
    parser.add_argument(
        "--local",
        default="agntcy/a2a_csit_slim/client_python",
        help="Local SLIM identity ns/group/name",
    )
    parser.add_argument(
        "--remote",
        default="agntcy/a2a_csit_slim/server_python",
        help="Remote server identity",
    )
    parser.add_argument(
        "--secret",
        default=os.environ.get(
            "SLIM_SHARED_SECRET", "my_shared_secret_for_testing_purposes_only"
        ),
    )
    parser.add_argument(
        "--text",
        default="Hello there!",
        help="Outbound text; response must contain this substring",
    )
    parser.add_argument("--log-level", default="ERROR")
    args = parser.parse_args()

    logging.basicConfig(level=getattr(logging, args.log_level.upper(), logging.ERROR))

    ns, grp, nm = _split_identity(args.local)

    _service, slim_local_app, local_name, conn_id = await setup_slim_client(
        namespace=ns,
        group=grp,
        name=nm,
        slim_url=args.slim_url,
        secret=args.secret,
        log_level="error",
    )

    httpx_client = httpx.AsyncClient()
    # Match upstream echo client default: unary SendMessage completes the async iterator.
    # streaming=True can leave async for ... send_message() waiting forever if the server
    # does not drive stream closure the way the client expects.
    client_config = ClientConfig(
        supported_protocol_bindings=["slimrpc"],
        streaming=False,
        httpx_client=httpx_client,
        slimrpc_channel_factory=slimrpc_channel_factory(slim_local_app, conn_id),
    )

    client_factory = ClientFactory(client_config)
    client_factory.register("slimrpc", SRPCTransport.create)

    agent_card = minimal_agent_card(args.remote, ["slimrpc"])
    client = client_factory.create(card=agent_card)

    probe_timeout_s = int(os.environ.get("CSIT_SLIM_PYTHON_PROBE_TIMEOUT", "180"))
    try:
        out, task, artifact_present = await asyncio.wait_for(
            collect_response(client, args.text),
            timeout=probe_timeout_s,
        )
    except TimeoutError as e:
        raise SystemExit(
            f"probe timed out after {probe_timeout_s}s "
            f"(set CSIT_SLIM_PYTHON_PROBE_TIMEOUT to override): {e}",
        ) from e
    finally:
        await httpx_client.aclose()

    # Parseable lifecycle block (keys consumed by matrix_test.go), then the raw
    # echoed text so the echo spec's substring check still holds.
    kind = "task" if task is not None else "message"
    sys.stdout.write(f"CSIT_SLIM_RESULT_KIND={kind}\n")
    sys.stdout.write(f"CSIT_SLIM_TASK_STATE={_task_state_name(task)}\n")
    sys.stdout.write(
        f"CSIT_SLIM_ARTIFACT_PRESENT={'true' if artifact_present else 'false'}\n"
    )
    sys.stdout.write(f"CSIT_SLIM_ARTIFACT_TEXT={out}\n")
    sys.stdout.write(out)
    if args.text not in out:
        raise SystemExit(
            f"response {out!r} does not contain sent text {args.text!r}",
        )


if __name__ == "__main__":
    asyncio.run(main())
