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


# Scenario sentinels: outbound request text that drives a non-echo server response.
# Must match the sentinels in the Go fixtures and csit_echo_executor.py byte-for-byte.
SCENARIO_ECHO = "echo"
SENTINEL_MESSAGE_ONLY = "csit-scenario:message-only"
SENTINEL_TASK_FAILURE = "csit-scenario:task-failure"
SENTINEL_INPUT_REQUIRED = "csit-scenario:input-required"
SENTINEL_STREAMING = "csit-scenario:streaming"
SENTINEL_CANCEL = "csit-scenario:cancel"

MODE_UNARY = "unary"
MODE_STREAMING = "streaming"
MODE_CANCEL = "cancel"


def _scenario_request(scenario: str, text: str) -> tuple[str, bool, str]:
    """Map a scenario selector to (outbound text, enforce-echo, transport mode).

    Non-echo scenarios send a fixed sentinel and only emit the observation block; the
    harness asserts the terminal state.
    """
    if scenario in (SCENARIO_ECHO, ""):
        return text, True, MODE_UNARY
    if scenario == "message-only":
        return SENTINEL_MESSAGE_ONLY, False, MODE_UNARY
    if scenario == "task-failure":
        return SENTINEL_TASK_FAILURE, False, MODE_UNARY
    if scenario == "input-required":
        return SENTINEL_INPUT_REQUIRED, False, MODE_UNARY
    if scenario == "streaming":
        return SENTINEL_STREAMING, False, MODE_STREAMING
    if scenario == "task-cancel":
        return SENTINEL_CANCEL, False, MODE_CANCEL
    raise SystemExit(f"unknown scenario {scenario!r}")


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


async def collect_response(
    client, text: str, break_on_echo: bool = True
) -> tuple[str, object, bool, int]:
    """Drain send_message until the stream ends.

    Returns the accumulated echoed text, the last observed task (for the terminal
    lifecycle state), whether a text artifact was seen, and the number of streamed
    events. For non-echo scenarios break_on_echo is False, so the iterator runs to
    completion (a failed/input-required task arrives as a normal terminal result).
    """
    message = create_text_message_object(content=text)
    request = SendMessageRequest(message=message)
    out = ""
    last_task = None
    artifact_present = False
    events = 0
    async for stream_response, task in client.send_message(request=request):
        events += 1
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
        if break_on_echo and text in out:
            break
    return out, last_task, artifact_present, events


async def run_cancel(client, text: str) -> tuple[str, object, bool, int]:
    """Create a working task, then cancel it via cancel_task.

    The high-level client only surfaces stream events after the stream ends, so the probe
    drops to the low-level transport's send_message_streaming, which yields raw events as
    they arrive. It reads just enough to learn the server-assigned task ID (the server
    generates its own ID, so a client-supplied one would not match), stops the stream
    while the task is still working, and cancels by that ID. Returns the canceled task's
    artifact text, the canceled task, artifact presence, and the events read before
    cancelling.
    """
    from a2a.types.a2a_pb2 import CancelTaskRequest

    message = create_text_message_object(content=text)
    request = SendMessageRequest(message=message)
    task_id = None
    events = 0
    stream = client._transport.send_message_streaming(request)  # noqa: SLF001
    try:
        async for item in stream:
            events += 1
            # The slimrpc transport yields StreamResponse; the group transport yields
            # (source, StreamResponse). Normalize to the StreamResponse.
            response = item[-1] if isinstance(item, tuple) else item
            which = response.WhichOneof("payload")
            if which == "task" and response.task.id:
                task_id = response.task.id
            elif which == "status_update" and response.status_update.task_id:
                task_id = response.status_update.task_id
            elif which == "artifact_update" and response.artifact_update.task_id:
                task_id = response.artifact_update.task_id
            if task_id:
                break
    finally:
        await stream.aclose()
    if not task_id:
        raise SystemExit("task-cancel scenario: no task id observed")

    canceled = await client.cancel_task(CancelTaskRequest(id=task_id))
    out = ""
    artifact_present = False
    for artifact in canceled.artifacts:
        for part in artifact.parts:
            if part.WhichOneof("content") == "text":
                out += part.text
                artifact_present = True
    return out, canceled, artifact_present, events


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
        help="Outbound text for the echo scenario; response must contain this substring",
    )
    parser.add_argument(
        "--scenario",
        default=SCENARIO_ECHO,
        help="Behavior to drive: echo, message-only, task-failure, input-required",
    )
    parser.add_argument("--log-level", default="ERROR")
    args = parser.parse_args()

    want, enforce_echo, mode = _scenario_request(args.scenario, args.text)

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
    # Unary scenarios use streaming=False so send_message completes like the Go unary
    # client. Streaming and task-cancel use streaming=True: the server drives stream
    # closure (execute returns), so the iterator terminates cleanly. task-cancel needs
    # this because a non-terminal working task would otherwise block a unary send.
    client_config = ClientConfig(
        supported_protocol_bindings=["slimrpc"],
        streaming=(mode in (MODE_STREAMING, MODE_CANCEL)),
        httpx_client=httpx_client,
        slimrpc_channel_factory=slimrpc_channel_factory(slim_local_app, conn_id),
    )

    client_factory = ClientFactory(client_config)
    client_factory.register("slimrpc", SRPCTransport.create)

    agent_card = minimal_agent_card(args.remote, ["slimrpc"])
    client = client_factory.create(card=agent_card)

    probe_timeout_s = int(os.environ.get("CSIT_SLIM_PYTHON_PROBE_TIMEOUT", "180"))
    if mode == MODE_CANCEL:
        worker = run_cancel(client, want)
    else:
        worker = collect_response(client, want, break_on_echo=enforce_echo)
    try:
        out, task, artifact_present, stream_events = await asyncio.wait_for(
            worker,
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
    sys.stdout.write(f"CSIT_SLIM_STREAM_EVENTS={stream_events}\n")
    sys.stdout.write(f"CSIT_SLIM_ARTIFACT_TEXT={out}\n")
    sys.stdout.write(out)
    if enforce_echo and want not in out:
        raise SystemExit(
            f"response {out!r} does not contain sent text {want!r}",
        )


if __name__ == "__main__":
    asyncio.run(main())
