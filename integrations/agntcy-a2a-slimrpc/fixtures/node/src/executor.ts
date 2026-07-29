// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

import { randomUUID } from 'node:crypto';
import {
  Message,
  Task,
  TaskArtifactUpdateEvent,
  TaskStatusUpdateEvent,
} from '@a2a-js/sdk';
import {
  AgentEvent,
  type AgentExecutor,
  type ExecutionEventBus,
  type RequestContext,
} from '@a2a-js/sdk/server';
import * as S from './scenarios.js';

/**
 * CSIT echo agent. Mirrors the Go echoExecutor (fixtures/go/cmd/server), the .NET
 * CsitEchoHandler, and the Java CsitEchoExecutor: the default behavior echoes the
 * request text back as a completed task's artifact; the scenario sentinels drive the
 * specific task lifecycle states the harness asserts.
 *
 * The A2A JS SDK's executor contract (from @a2a-js/sdk/server): execute() MUST
 * publish a `task` or `message` event first, then any `statusUpdate`/`artifactUpdate`
 * events. The DefaultRequestHandler settles the event bus once execute() returns —
 * terminal/message states close it; interrupted states (INPUT_REQUIRED) keep it alive
 * for the multi-turn continuation — so this executor never calls finished() itself.
 */
export class CsitEchoExecutor implements AgentExecutor {
  async execute(context: RequestContext, eventBus: ExecutionEventBus): Promise<void> {
    const taskId = context.taskId;
    const contextId = context.contextId;
    const text = userText(context);

    switch (text) {
      case S.SENTINEL_MESSAGE_ONLY:
        // Bare message response: no task is created.
        eventBus.publish(
          AgentEvent.message(agentMessage(S.MESSAGE_ONLY_TEXT, contextId, taskId)),
        );
        break;

      case S.SENTINEL_TASK_FAILURE:
        eventBus.publish(AgentEvent.task(task(taskId, contextId, 'TASK_STATE_SUBMITTED')));
        eventBus.publish(
          AgentEvent.statusUpdate(statusUpdate(taskId, contextId, 'TASK_STATE_FAILED')),
        );
        break;

      case S.SENTINEL_INPUT_REQUIRED:
      case S.SENTINEL_MULTI_TURN:
        // Turn 1 (or the input-required scenario): pause for input. For multi-turn the
        // probe continues this same task with the continue sentinel (task/context IDs).
        eventBus.publish(AgentEvent.task(task(taskId, contextId, 'TASK_STATE_SUBMITTED')));
        eventBus.publish(
          AgentEvent.statusUpdate(
            statusUpdate(taskId, contextId, 'TASK_STATE_INPUT_REQUIRED'),
          ),
        );
        break;

      case S.SENTINEL_MULTI_TURN_CONTINUE:
        // Turn 2: the task already exists (continuation) — re-announce it, then emit the
        // completion artifact. The SDK requires a task/message event first even on
        // follow-up turns.
        eventBus.publish(AgentEvent.task(task(taskId, contextId, 'TASK_STATE_WORKING')));
        eventBus.publish(
          AgentEvent.artifactUpdate(
            artifactUpdate(taskId, contextId, 'multi-turn', S.MULTI_TURN_COMPLETE_TEXT),
          ),
        );
        eventBus.publish(
          AgentEvent.statusUpdate(statusUpdate(taskId, contextId, 'TASK_STATE_COMPLETED')),
        );
        break;

      case S.SENTINEL_STREAMING:
        eventBus.publish(AgentEvent.task(task(taskId, contextId, 'TASK_STATE_WORKING')));
        eventBus.publish(
          AgentEvent.artifactUpdate(
            artifactUpdate(taskId, contextId, 'chunk1', S.STREAMING_CHUNK_1),
          ),
        );
        eventBus.publish(
          AgentEvent.artifactUpdate(
            artifactUpdate(taskId, contextId, 'chunk2', S.STREAMING_CHUNK_2),
          ),
        );
        eventBus.publish(
          AgentEvent.statusUpdate(statusUpdate(taskId, contextId, 'TASK_STATE_COMPLETED')),
        );
        break;

      case S.SENTINEL_CANCEL:
        // Leave the task working (non-terminal) so CancelTask can cancel it. The handler
        // settles (finishes) the bus after execute() returns; the client then cancels.
        eventBus.publish(AgentEvent.task(task(taskId, contextId, 'TASK_STATE_SUBMITTED')));
        eventBus.publish(
          AgentEvent.statusUpdate(statusUpdate(taskId, contextId, 'TASK_STATE_WORKING')),
        );
        break;

      default:
        // Default echo: working, emit the input text as an artifact, complete.
        eventBus.publish(AgentEvent.task(task(taskId, contextId, 'TASK_STATE_WORKING')));
        eventBus.publish(
          AgentEvent.artifactUpdate(artifactUpdate(taskId, contextId, 'echo', text)),
        );
        eventBus.publish(
          AgentEvent.statusUpdate(statusUpdate(taskId, contextId, 'TASK_STATE_COMPLETED')),
        );
        break;
    }
  }

  async cancelTask(taskId: string, eventBus: ExecutionEventBus): Promise<void> {
    // Reached only if the event bus is still alive when CancelTask arrives; otherwise the
    // DefaultRequestHandler cancels the stored (working) task directly. Publish a terminal
    // canceled status so either path lands on TASK_STATE_CANCELED.
    eventBus.publish(AgentEvent.statusUpdate(statusUpdate(taskId, '', 'TASK_STATE_CANCELED')));
  }
}

// ---- domain builders (ts-proto fromJSON, proto3 JSON shape) ------------------------

function userText(context: RequestContext): string {
  const part = context.userMessage.parts[0];
  return part?.content?.$case === 'text' ? part.content.value : '';
}

function task(id: string, contextId: string, state: string): Task {
  return Task.fromJSON({ id, contextId, status: { state } });
}

function statusUpdate(
  taskId: string,
  contextId: string,
  state: string,
): TaskStatusUpdateEvent {
  return TaskStatusUpdateEvent.fromJSON({ taskId, contextId, status: { state } });
}

function artifactUpdate(
  taskId: string,
  contextId: string,
  artifactId: string,
  text: string,
): TaskArtifactUpdateEvent {
  return TaskArtifactUpdateEvent.fromJSON({
    taskId,
    contextId,
    artifact: { artifactId, name: artifactId, parts: [{ text }] },
    lastChunk: true,
  });
}

function agentMessage(text: string, contextId: string, taskId: string): Message {
  return Message.fromJSON({
    messageId: randomUUID(),
    contextId,
    taskId,
    role: 'ROLE_AGENT',
    parts: [{ text }],
  });
}
