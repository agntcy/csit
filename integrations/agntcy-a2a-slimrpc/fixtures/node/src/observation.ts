// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

import {
  type Artifact,
  type Message,
  type StreamResponse,
  type Task,
  type TaskState,
  taskStateToJSON,
} from '@a2a-js/sdk';

/**
 * Parseable view of a SendMessage result, mirroring the Go/Python/.NET/Java probes.
 * Emitted as CSIT_SLIM_* KEY=value lines consumed by matrix_test.go.
 */
export interface Observation {
  kind: string; // "task" | "message" | "unknown"
  state: string; // TASK_STATE_* token; empty for a bare message
  artifactPresent: boolean;
  text: string;
  streamEvents: number;
}

/** Normalizes a proto TaskState (enum number/undefined) to its TASK_STATE_* token, "" if unset. */
export function stateToken(state: TaskState | undefined): string {
  if (state === undefined || state === 0) {
    return '';
  }
  return taskStateToJSON(state);
}

function aggregateArtifacts(artifacts: Artifact[] | undefined): [string, boolean] {
  let text = '';
  let present = false;
  for (const artifact of artifacts ?? []) {
    for (const part of artifact.parts ?? []) {
      if (part.content?.$case === 'text') {
        text += part.content.value;
        present = true;
      }
    }
  }
  return [text, present];
}

function messageText(message: Message): string {
  let text = '';
  for (const part of message.parts ?? []) {
    if (part.content?.$case === 'text') {
      text += part.content.value;
    }
  }
  return text;
}

/** A non-streaming SendMessage result is a Message or a Task (SendMessageResult). */
export function observeResult(result: Message | Task): Observation {
  if ('messageId' in result && typeof result.messageId === 'string') {
    return {
      kind: 'message',
      state: '',
      artifactPresent: false,
      text: messageText(result),
      streamEvents: 0,
    };
  }
  const task = result as Task;
  const [text, present] = aggregateArtifacts(task.artifacts);
  return {
    kind: 'task',
    state: stateToken(task.status?.state),
    artifactPresent: present,
    text,
    streamEvents: 0,
  };
}

/** Observe a Task returned directly (e.g. from CancelTask). */
export function observeTask(task: Task): Observation {
  const [text, present] = aggregateArtifacts(task.artifacts);
  return {
    kind: 'task',
    state: stateToken(task.status?.state),
    artifactPresent: present,
    text,
    streamEvents: 0,
  };
}

/**
 * Aggregate a streaming SendStreamingMessage response into one observation, counting
 * events (proves the stream was multi-event) and concatenating streamed artifact text
 * in order.
 */
export async function observeStream(
  stream: AsyncGenerator<StreamResponse, void, undefined>,
): Promise<Observation> {
  const obs: Observation = {
    kind: 'task',
    state: '',
    artifactPresent: false,
    text: '',
    streamEvents: 0,
  };
  let buffer = '';
  for await (const response of stream) {
    obs.streamEvents++;
    const payload = response.payload;
    if (!payload) {
      continue;
    }
    switch (payload.$case) {
      case 'task': {
        const task = payload.value;
        obs.state = stateToken(task.status?.state);
        const [text, present] = aggregateArtifacts(task.artifacts);
        if (present) {
          buffer += text;
          obs.artifactPresent = true;
        }
        break;
      }
      case 'statusUpdate':
        obs.state = stateToken(payload.value.status?.state);
        break;
      case 'artifactUpdate': {
        const [text, present] = aggregateArtifacts(
          payload.value.artifact ? [payload.value.artifact] : [],
        );
        if (present) {
          buffer += text;
          obs.artifactPresent = true;
        }
        break;
      }
      case 'message':
        obs.kind = 'message';
        buffer += messageText(payload.value);
        break;
    }
  }
  obs.text = buffer;
  return obs;
}

/** Prints the parseable lifecycle block followed by the raw text (echo substring check). */
export function emit(obs: Observation): void {
  console.log('CSIT_SLIM_RESULT_KIND=' + obs.kind);
  console.log('CSIT_SLIM_TASK_STATE=' + obs.state);
  console.log('CSIT_SLIM_ARTIFACT_PRESENT=' + (obs.artifactPresent ? 'true' : 'false'));
  console.log('CSIT_SLIM_STREAM_EVENTS=' + obs.streamEvents);
  console.log('CSIT_SLIM_ARTIFACT_TEXT=' + obs.text);
  console.log(obs.text);
}
