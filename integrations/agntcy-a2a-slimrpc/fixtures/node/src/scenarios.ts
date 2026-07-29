// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

/**
 * Scenario sentinels: the probe sends one of these as the request text to drive a
 * non-echo server response. These travel over the wire, so they MUST match the
 * sentinels in fixtures/{go,python,dotnet,java} byte-for-byte.
 */
export const ECHO = 'echo';

export const SENTINEL_MESSAGE_ONLY = 'csit-scenario:message-only';
export const SENTINEL_TASK_FAILURE = 'csit-scenario:task-failure';
export const SENTINEL_INPUT_REQUIRED = 'csit-scenario:input-required';
export const SENTINEL_STREAMING = 'csit-scenario:streaming';
export const SENTINEL_CANCEL = 'csit-scenario:cancel';
export const SENTINEL_MULTI_TURN = 'csit-scenario:multi-turn';
export const SENTINEL_MULTI_TURN_CONTINUE = 'csit-scenario:multi-turn-continue';

// Artifact emitted on the multi-turn continuation turn (matches the other fixtures
// and the harness's multiTurnCompleteMarker).
export const MULTI_TURN_COMPLETE_TEXT = 'multi-turn complete';

// Bare-message response text for the message-only scenario (only the kind is asserted).
export const MESSAGE_ONLY_TEXT = 'node server message-only response';

// Streaming artifact chunks (the harness asserts both substrings appear, aggregated in order).
export const STREAMING_CHUNK_1 = 'streaming chunk 1 ';
export const STREAMING_CHUNK_2 = 'streaming chunk 2';

// Probe transport modes selected per scenario.
export const MODE_UNARY = 'unary';
export const MODE_STREAMING = 'streaming';
export const MODE_CANCEL = 'cancel';
export const MODE_MULTI_TURN = 'multi-turn';

/** Outbound request text, whether the response must echo it, and the transport mode. */
export interface ScenarioRequest {
  want: string;
  enforceEcho: boolean;
  mode: string;
}

export function request(scenario: string, text: string): ScenarioRequest {
  switch (scenario || '') {
    case ECHO:
    case '':
      return { want: text, enforceEcho: true, mode: MODE_UNARY };
    case 'message-only':
      return { want: SENTINEL_MESSAGE_ONLY, enforceEcho: false, mode: MODE_UNARY };
    case 'task-failure':
      return { want: SENTINEL_TASK_FAILURE, enforceEcho: false, mode: MODE_UNARY };
    case 'input-required':
      return { want: SENTINEL_INPUT_REQUIRED, enforceEcho: false, mode: MODE_UNARY };
    case 'streaming':
      return { want: SENTINEL_STREAMING, enforceEcho: false, mode: MODE_STREAMING };
    case 'task-cancel':
      return { want: SENTINEL_CANCEL, enforceEcho: false, mode: MODE_CANCEL };
    case 'multi-turn':
      // The probe drives both turns; the start turn sends SENTINEL_MULTI_TURN.
      return { want: SENTINEL_MULTI_TURN, enforceEcho: false, mode: MODE_MULTI_TURN };
    default:
      throw new Error(`unknown scenario ${scenario}`);
  }
}
