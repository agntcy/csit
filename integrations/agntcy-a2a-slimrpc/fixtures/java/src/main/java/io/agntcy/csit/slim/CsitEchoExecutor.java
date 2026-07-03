// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package io.agntcy.csit.slim;

import java.util.List;
import java.util.Map;

import org.a2aproject.sdk.server.agentexecution.AgentExecutor;
import org.a2aproject.sdk.server.agentexecution.RequestContext;
import org.a2aproject.sdk.server.tasks.AgentEmitter;
import org.a2aproject.sdk.spec.A2AError;
import org.a2aproject.sdk.spec.Part;
import org.a2aproject.sdk.spec.TextPart;

/**
 * CSIT echo agent. Mirrors the Go echoExecutor (fixtures/go/cmd/server) and the .NET
 * CsitEchoHandler: the default behavior echoes the request text back as a completed
 * task's artifact; the scenario sentinels drive the specific task lifecycle states the
 * harness asserts.
 */
final class CsitEchoExecutor implements AgentExecutor {

    private static List<Part<?>> textParts(String text) {
        return List.<Part<?>>of(new TextPart(text, null));
    }

    @Override
    public void execute(RequestContext context, AgentEmitter emitter) throws A2AError {
        String text = context.getUserInput() == null ? "" : context.getUserInput();
        switch (text) {
            case Scenarios.SENTINEL_MESSAGE_ONLY ->
                // Bare message response: no task is created.
                emitter.sendMessage(textParts(Scenarios.MESSAGE_ONLY_TEXT));

            case Scenarios.SENTINEL_TASK_FAILURE -> {
                emitter.submit();
                emitter.fail();
            }

            case Scenarios.SENTINEL_INPUT_REQUIRED -> {
                emitter.submit();
                emitter.requiresInput();
            }

            case Scenarios.SENTINEL_MULTI_TURN -> {
                // Turn 1: pause for input. The probe continues this same task with the
                // continue sentinel (referencing task/context IDs).
                emitter.submit();
                emitter.requiresInput();
            }

            case Scenarios.SENTINEL_MULTI_TURN_CONTINUE -> {
                // Turn 2: task already exists (continuation) — emit the completion artifact.
                emitter.addArtifact(textParts(Scenarios.MULTI_TURN_COMPLETE_TEXT),
                        "multi-turn", "text/plain", Map.of());
                emitter.complete();
            }

            case Scenarios.SENTINEL_STREAMING -> {
                emitter.submit();
                emitter.startWork();
                emitter.addArtifact(textParts("streaming chunk 1 "), "chunk1", "text/plain", Map.of());
                emitter.addArtifact(textParts("streaming chunk 2"), "chunk2", "text/plain", Map.of());
                emitter.complete();
            }

            case Scenarios.SENTINEL_CANCEL -> {
                // Leave the task working (non-terminal) so CancelTask can cancel it.
                emitter.submit();
                emitter.startWork();
            }

            default -> {
                // Default echo: submit, work, emit the input text as an artifact, complete.
                emitter.submit();
                emitter.startWork();
                emitter.addArtifact(textParts(text), "echo", "text/plain", Map.of());
                emitter.complete();
            }
        }
    }

    @Override
    public void cancel(RequestContext context, AgentEmitter emitter) throws A2AError {
        emitter.cancel();
    }
}
