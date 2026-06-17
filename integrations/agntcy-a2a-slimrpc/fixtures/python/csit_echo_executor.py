# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

"""Minimal echo agent executor for CSIT SLIMRPC fixtures (mirrors Go echo semantics)."""

from __future__ import annotations

import logging

from a2a.server.agent_execution import AgentExecutor, RequestContext
from a2a.server.events import EventQueue
from a2a.server.tasks.task_updater import TaskUpdater
from a2a.types import Message, Part, Role

logger = logging.getLogger(__name__)

# Scenario sentinels: request text that drives a non-echo response. Must match the
# sentinels in the Go fixtures (cmd/{server,probe}/main.go) byte-for-byte.
SENTINEL_MESSAGE_ONLY = "csit-scenario:message-only"
SENTINEL_TASK_FAILURE = "csit-scenario:task-failure"
SENTINEL_INPUT_REQUIRED = "csit-scenario:input-required"
SENTINEL_STREAMING = "csit-scenario:streaming"
SENTINEL_CANCEL = "csit-scenario:cancel"


class CsitEchoExecutor(AgentExecutor):
    """Echo user text back as a task artifact, or drive a scenario response (A2A v1)."""

    async def execute(self, context: RequestContext, event_queue: EventQueue) -> None:
        if not context.message:
            raise RuntimeError("invalid message: missing message")
        if not context.message.parts:
            raise RuntimeError("invalid message: no parts")
        if context.message.parts[0].WhichOneof("content") != "text":
            raise RuntimeError("only text parts are supported")

        logger.debug("received message: %s", context.message)
        text = context.message.parts[0].text

        # message-only returns a bare agent message; no task is created.
        if text == SENTINEL_MESSAGE_ONLY:
            await event_queue.enqueue_event(
                Message(
                    role=Role.ROLE_AGENT,
                    message_id=context.message.message_id,
                    parts=[Part(text="python server message-only response")],
                )
            )
            return

        # Task-based scenarios require task/context identifiers.
        if not context.message.task_id or not context.message.context_id:
            raise RuntimeError("invalid message: missing task_id or context_id")

        task_updater = TaskUpdater(
            event_queue=event_queue,
            task_id=context.message.task_id,
            context_id=context.message.context_id,
        )
        await task_updater.submit(message=context.message)

        if text == SENTINEL_TASK_FAILURE:
            await task_updater.failed()
            return
        if text == SENTINEL_INPUT_REQUIRED:
            await task_updater.requires_input()
            return
        if text == SENTINEL_STREAMING:
            # Multiple status + artifact events so a streaming client observes a stream.
            await task_updater.start_work()
            await task_updater.add_artifact(
                parts=[Part(text="streaming chunk 1 ")], name="result"
            )
            await task_updater.add_artifact(
                parts=[Part(text="streaming chunk 2")], name="result"
            )
            await task_updater.complete()
            return
        if text == SENTINEL_CANCEL:
            # Leave the task working (non-terminal) so cancel_task can cancel it.
            await task_updater.start_work()
            return

        response = Message(
            role=Role.ROLE_AGENT,
            message_id=context.message.message_id,
            parts=[Part(text=text)],
        )
        await task_updater.add_artifact(
            parts=list(response.parts),
            name="result",
        )
        await task_updater.complete(message=response)

    async def cancel(self, context: RequestContext, event_queue: EventQueue) -> None:
        task_id = context.task_id
        context_id = context.context_id
        if not task_id and context.current_task is not None:
            task_id = context.current_task.id
            context_id = context.current_task.context_id
        if not task_id or not context_id:
            raise RuntimeError("cancel: missing task_id or context_id")

        task_updater = TaskUpdater(
            event_queue=event_queue,
            task_id=task_id,
            context_id=context_id,
        )
        await task_updater.cancel()
