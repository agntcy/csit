// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package io.agntcy.csit.slim;

/**
 * Scenario sentinels: the probe sends one of these as the request text to drive a
 * non-echo server response. These travel over the wire, so they MUST match the
 * sentinels in fixtures/go, fixtures/python, and fixtures/dotnet byte-for-byte.
 */
final class Scenarios {
    private Scenarios() {}

    static final String ECHO = "echo";

    static final String SENTINEL_MESSAGE_ONLY = "csit-scenario:message-only";
    static final String SENTINEL_TASK_FAILURE = "csit-scenario:task-failure";
    static final String SENTINEL_INPUT_REQUIRED = "csit-scenario:input-required";
    static final String SENTINEL_STREAMING = "csit-scenario:streaming";
    static final String SENTINEL_CANCEL = "csit-scenario:cancel";
    static final String SENTINEL_MULTI_TURN = "csit-scenario:multi-turn";
    static final String SENTINEL_MULTI_TURN_CONTINUE = "csit-scenario:multi-turn-continue";

    // Artifact emitted on the multi-turn continuation turn (matches the other fixtures
    // and the harness's multiTurnCompleteMarker).
    static final String MULTI_TURN_COMPLETE_TEXT = "multi-turn complete";

    // Bare-message response text for the message-only scenario (only the kind is asserted).
    static final String MESSAGE_ONLY_TEXT = "java server message-only response";

    // Probe transport modes.
    static final String MODE_UNARY = "unary";
    static final String MODE_STREAMING = "streaming";
    static final String MODE_CANCEL = "cancel";
    static final String MODE_MULTI_TURN = "multi-turn";

    /** Outbound request text, whether the response must echo it, and the transport mode. */
    record Request(String want, boolean enforceEcho, String mode) {}

    static Request request(String scenario, String text) {
        return switch (scenario == null ? "" : scenario) {
            case ECHO, "" -> new Request(text, true, MODE_UNARY);
            case "message-only" -> new Request(SENTINEL_MESSAGE_ONLY, false, MODE_UNARY);
            case "task-failure" -> new Request(SENTINEL_TASK_FAILURE, false, MODE_UNARY);
            case "input-required" -> new Request(SENTINEL_INPUT_REQUIRED, false, MODE_UNARY);
            case "streaming" -> new Request(SENTINEL_STREAMING, false, MODE_STREAMING);
            case "task-cancel" -> new Request(SENTINEL_CANCEL, false, MODE_CANCEL);
            case "multi-turn" -> new Request(SENTINEL_MULTI_TURN, false, MODE_MULTI_TURN);
            default -> throw new IllegalArgumentException("unknown scenario " + scenario);
        };
    }
}
