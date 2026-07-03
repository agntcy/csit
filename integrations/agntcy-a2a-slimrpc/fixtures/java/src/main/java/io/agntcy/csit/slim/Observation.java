// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package io.agntcy.csit.slim;

import io.agntcy.slim.a2a.SlimA2AClient;
import io.agntcy.slim.bindings.ResponseStreamReader;
import io.agntcy.slim.bindings.slimrpc.ClientResponseStream;

import org.a2aproject.sdk.spec.Artifact;
import org.a2aproject.sdk.spec.CancelTaskParams;
import org.a2aproject.sdk.spec.EventKind;
import org.a2aproject.sdk.spec.Message;
import org.a2aproject.sdk.spec.Part;
import org.a2aproject.sdk.spec.Task;
import org.a2aproject.sdk.spec.TextPart;

/**
 * Parseable view of a SendMessage result, mirroring the Go/Python/.NET probes. Emitted as
 * CSIT_SLIM_* KEY=value lines consumed by matrix_test.go. Unary + cancel read SDK domain
 * types (SlimA2AClient returns them); streaming reads raw proto StreamResponse events.
 */
final class Observation {
    String kind = "unknown"; // "task" | "message" | "unknown"
    String state = "";       // TASK_STATE_* token; empty for a bare message
    boolean artifactPresent;
    String text = "";
    int streamEvents;

    String text() { return text; }

    void emit() {
        System.out.println("CSIT_SLIM_RESULT_KIND=" + kind);
        System.out.println("CSIT_SLIM_TASK_STATE=" + state);
        System.out.println("CSIT_SLIM_ARTIFACT_PRESENT=" + (artifactPresent ? "true" : "false"));
        System.out.println("CSIT_SLIM_STREAM_EVENTS=" + streamEvents);
        System.out.println("CSIT_SLIM_ARTIFACT_TEXT=" + text);
        System.out.println(text);
        System.out.flush();
    }

    // ---- SDK (unary / cancel) ---------------------------------------------------------

    static Observation runUnary(SlimA2AClient client, String text) throws Exception {
        return fromEventKind(client.sendMessage(Main.sendParams(text, null, null)));
    }

    static Observation runMultiTurn(SlimA2AClient client) throws Exception {
        EventKind r1 = client.sendMessage(Main.sendParams(Scenarios.SENTINEL_MULTI_TURN, null, null));
        if (!(r1 instanceof Task task1)) {
            throw new IllegalStateException("multi-turn start: expected a task, got " + r1);
        }
        // Continue the same task/context to completion; only the final observation is emitted.
        EventKind r2 = client.sendMessage(
                Main.sendParams(Scenarios.SENTINEL_MULTI_TURN_CONTINUE, task1.id(), task1.contextId()));
        return fromEventKind(r2);
    }

    static Observation runCancel(SlimA2AClient client, String text) throws Exception {
        // Read the stream just far enough to learn the server-assigned task id, then cancel it.
        String taskId = null;
        int events = 0;
        ResponseStreamReader reader = client.sendStreamingMessage(Main.sendParams(text, null, null));
        ClientResponseStream<org.a2aproject.sdk.grpc.StreamResponse> stream =
                ClientResponseStream.create(reader, Observation::parse);
        org.a2aproject.sdk.grpc.StreamResponse ev;
        while ((ev = stream.recv()) != null) {
            events++;
            taskId = switch (ev.getPayloadCase()) {
                case TASK -> ev.getTask().getId();
                case STATUS_UPDATE -> ev.getStatusUpdate().getTaskId();
                case ARTIFACT_UPDATE -> ev.getArtifactUpdate().getTaskId();
                default -> taskId;
            };
            if (taskId != null && !taskId.isEmpty()) break;
        }
        if (taskId == null || taskId.isEmpty()) {
            throw new IllegalStateException("task-cancel scenario: no task id observed");
        }
        Task canceled = client.cancelTask(CancelTaskParams.builder().id(taskId).build());
        Observation obs = fromSdkTask(canceled);
        obs.streamEvents = events;
        return obs;
    }

    private static Observation fromEventKind(EventKind result) {
        if (result instanceof Task task) {
            return fromSdkTask(task);
        }
        if (result instanceof Message msg) {
            Observation o = new Observation();
            o.kind = "message";
            o.text = messageText(msg);
            return o;
        }
        return new Observation();
    }

    private static Observation fromSdkTask(Task task) {
        Observation o = new Observation();
        o.kind = "task";
        o.state = stateToken(task.status() == null ? null : task.status().state().name());
        if (task.artifacts() != null) {
            StringBuilder sb = new StringBuilder();
            for (Artifact a : task.artifacts()) {
                if (a.parts() == null) continue;
                for (Part<?> p : a.parts()) {
                    if (p instanceof TextPart tp) { sb.append(tp.text()); o.artifactPresent = true; }
                }
            }
            o.text = sb.toString();
        }
        return o;
    }

    private static String messageText(Message msg) {
        StringBuilder sb = new StringBuilder();
        if (msg.parts() != null) {
            for (Part<?> p : msg.parts()) {
                if (p instanceof TextPart tp) sb.append(tp.text());
            }
        }
        return sb.toString();
    }

    // ---- streaming (raw proto events) -------------------------------------------------

    static Observation runStreaming(SlimA2AClient client, String text) throws Exception {
        Observation o = new Observation();
        o.kind = "task";
        StringBuilder sb = new StringBuilder();
        int events = 0;
        ResponseStreamReader reader = client.sendStreamingMessage(Main.sendParams(text, null, null));
        ClientResponseStream<org.a2aproject.sdk.grpc.StreamResponse> stream =
                ClientResponseStream.create(reader, Observation::parse);
        org.a2aproject.sdk.grpc.StreamResponse ev;
        while ((ev = stream.recv()) != null) {
            events++;
            switch (ev.getPayloadCase()) {
                case TASK -> {
                    o.state = stateToken(ev.getTask().getStatus().getState().name());
                    appendProtoArtifacts(ev.getTask().getArtifactsList(), sb, o);
                }
                case STATUS_UPDATE -> o.state = stateToken(ev.getStatusUpdate().getStatus().getState().name());
                case ARTIFACT_UPDATE -> appendProtoParts(ev.getArtifactUpdate().getArtifact().getPartsList(), sb, o);
                case MESSAGE -> { o.kind = "message"; appendProtoParts(ev.getMessage().getPartsList(), sb, o); }
                default -> { }
            }
        }
        o.streamEvents = events;
        o.text = sb.toString();
        return o;
    }

    private static void appendProtoArtifacts(
            java.util.List<org.a2aproject.sdk.grpc.Artifact> artifacts, StringBuilder sb, Observation o) {
        for (var a : artifacts) appendProtoParts(a.getPartsList(), sb, o);
    }

    private static void appendProtoParts(
            java.util.List<org.a2aproject.sdk.grpc.Part> parts, StringBuilder sb, Observation o) {
        for (var p : parts) {
            if (p.getContentCase() == org.a2aproject.sdk.grpc.Part.ContentCase.TEXT) {
                sb.append(p.getText());
                o.artifactPresent = true;
            }
        }
    }

    private static org.a2aproject.sdk.grpc.StreamResponse parse(byte[] bytes) {
        try {
            return org.a2aproject.sdk.grpc.StreamResponse.parseFrom(bytes);
        } catch (Exception e) {
            throw new RuntimeException("failed to parse StreamResponse", e);
        }
    }

    /** Normalize any TaskState enum name to the canonical proto token (TASK_STATE_*). */
    private static String stateToken(String name) {
        if (name == null || name.isEmpty()) return "";
        if (name.startsWith("TASK_STATE_")) return name;
        StringBuilder sb = new StringBuilder("TASK_STATE_");
        for (int i = 0; i < name.length(); i++) {
            char c = name.charAt(i);
            if (i > 0 && Character.isUpperCase(c) && !Character.isUpperCase(name.charAt(i - 1))) sb.append('_');
            sb.append(Character.toUpperCase(c));
        }
        return sb.toString();
    }
}
