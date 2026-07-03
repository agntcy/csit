// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0
//
// CSIT fixture: A2A echo agent + probe over SLIMRPC (lf.a2a.v1) for cross-language interop.
// One jar, two modes mirroring fixtures/go, fixtures/python, fixtures/dotnet:
//   csit-slim-a2a-java server --slim-endpoint <url> --identity <ns/grp/name> --secret <s>
//   csit-slim-a2a-java probe  --slim-endpoint <url> --local <id> --remote <id> --secret <s> --scenario <sc> --text <t>

package io.agntcy.csit.slim;

import java.time.Duration;
import java.util.List;
import java.util.UUID;
import java.util.concurrent.Executors;

import io.agntcy.slim.a2a.SlimA2AClient;
import io.agntcy.slim.a2a.SlimA2AHandler;
import io.agntcy.slim.a2a.SlimHelper;
import io.agntcy.slim.bindings.Channel;
import io.agntcy.slim.bindings.Server;
import org.a2aproject.sdk.grpc.A2AServiceSlimrpc;
import org.a2aproject.sdk.server.events.InMemoryQueueManager;
import org.a2aproject.sdk.server.events.MainEventBus;
import org.a2aproject.sdk.server.events.MainEventBusProcessor;
import org.a2aproject.sdk.server.requesthandlers.DefaultRequestHandler;
import org.a2aproject.sdk.server.tasks.InMemoryPushNotificationConfigStore;
import org.a2aproject.sdk.server.tasks.InMemoryTaskStore;
import org.a2aproject.sdk.server.tasks.PushNotificationSender;
import org.a2aproject.sdk.spec.AgentCapabilities;
import org.a2aproject.sdk.spec.AgentCard;
import org.a2aproject.sdk.spec.AgentSkill;
import org.a2aproject.sdk.spec.EventKind;
import org.a2aproject.sdk.spec.Message;
import org.a2aproject.sdk.spec.MessageSendParams;
import org.a2aproject.sdk.spec.Task;
import org.a2aproject.sdk.spec.TextPart;

public final class Main {

    static final String READY_MARKER = "CSIT_SLIM_SERVER_READY";
    static final String DEFAULT_ENDPOINT = "http://127.0.0.1:46357";
    static final String DEFAULT_SECRET = "my_shared_secret_for_testing_purposes_only";

    public static void main(String[] args) throws Exception {
        String mode = firstPositional(args);
        String endpoint = opt(args, "--slim-endpoint", envOr("SLIM_SERVER", DEFAULT_ENDPOINT));
        String secret = opt(args, "--secret", envOr("SLIM_SHARED_SECRET", DEFAULT_SECRET));

        switch (mode) {
            case "server" -> runServer(endpoint, secret,
                    opt(args, "--identity", "agntcy/a2a_csit_slim/server_java"));
            case "probe" -> System.exit(runProbe(endpoint, secret,
                    opt(args, "--local", "agntcy/a2a_csit_slim/client_java"),
                    opt(args, "--remote", "agntcy/a2a_csit_slim/server_java"),
                    opt(args, "--scenario", Scenarios.ECHO),
                    opt(args, "--text", "Hello there!")));
            default -> {
                System.err.println("usage: csit-slim-a2a-java [server|probe] [flags]");
                System.exit(2);
            }
        }
    }

    // ---- server -----------------------------------------------------------------------

    private static void runServer(String endpoint, String secret, String identity) throws Exception {
        String[] id = splitIdentity(identity); // org, ns, instance

        SlimHelper.initializeDefaults();

        AgentCard card = AgentCard.builder()
                .name("CSIT Echo Agent")
                .description("Echoes messages over A2A on SLIMRPC for CSIT interop.")
                .version("1.0.0")
                .defaultInputModes(List.of("text"))
                .defaultOutputModes(List.of("text"))
                .supportedInterfaces(List.of())
                .capabilities(AgentCapabilities.builder().extendedAgentCard(true).build())
                .skills(List.of(AgentSkill.builder()
                        .id("echo").name("Echo").description("Echoes the input text back")
                        .tags(List.of("echo")).build()))
                .build();

        var taskStore = new InMemoryTaskStore();
        var eventBus = new MainEventBus();
        var queueManager = new InMemoryQueueManager(taskStore, eventBus);
        var pushConfigStore = new InMemoryPushNotificationConfigStore();
        PushNotificationSender noOpPush = (event, task) -> {};
        var eventProcessor = new MainEventBusProcessor(eventBus, taskStore, noOpPush, queueManager);
        var processorThread = new Thread(eventProcessor, "event-bus-processor");
        processorThread.setDaemon(true);
        processorThread.start();

        var executor = Executors.newCachedThreadPool();
        var requestHandler = DefaultRequestHandler.create(
                new CsitEchoExecutor(), taskStore, queueManager, pushConfigStore,
                eventProcessor, executor, executor);
        var handler = new SlimA2AHandler(requestHandler, card);

        Server rpcServer = SlimHelper.createServer(id[2], id[0], id[1], secret, endpoint);
        A2AServiceSlimrpc.registerA2AServiceServer(rpcServer, handler);

        // Ready marker on stdout (the harness waits for this line), matching the other fixtures.
        System.out.println(READY_MARKER);
        System.out.flush();
        rpcServer.serve();
    }

    // ---- probe ------------------------------------------------------------------------

    private static int runProbe(String endpoint, String secret, String local, String remote,
                                String scenario, String text) throws Exception {
        Scenarios.Request req = Scenarios.request(scenario, text);
        String[] localId = splitIdentity(local);   // org, ns, instance
        String[] remoteId = splitIdentity(remote);

        SlimHelper.initializeDefaults();
        // Local client instance lives in the server's org/ns (shared across CSIT identities).
        Channel channel = SlimHelper.createChannel(
                localId[2], remoteId[0], remoteId[1], remoteId[2], secret, endpoint);
        try {
            var client = new SlimA2AClient(channel);
            Observation obs = switch (req.mode()) {
                case Scenarios.MODE_STREAMING -> Observation.runStreaming(client, req.want());
                case Scenarios.MODE_CANCEL -> Observation.runCancel(client, req.want());
                case Scenarios.MODE_MULTI_TURN -> Observation.runMultiTurn(client);
                default -> Observation.runUnary(client, req.want());
            };
            obs.emit();
            if (req.enforceEcho() && !obs.text().contains(req.want())) {
                System.err.println("response " + obs.text() + " does not contain sent text " + req.want());
                return 1;
            }
            return 0;
        } finally {
            channel.close(Duration.ofSeconds(5));
        }
    }

    static Message userMessage(String text, String taskId, String contextId) {
        Message.Builder b = Message.builder()
                .role(Message.Role.ROLE_USER)
                .messageId(UUID.randomUUID().toString())
                .parts(List.of(new TextPart(text, null)));
        if (contextId != null) b.contextId(contextId);
        if (taskId != null) b.taskId(taskId);
        return b.build();
    }

    static MessageSendParams sendParams(String text, String taskId, String contextId) {
        return MessageSendParams.builder().message(userMessage(text, taskId, contextId)).build();
    }

    // ---- helpers ----------------------------------------------------------------------

    private static String[] splitIdentity(String s) {
        String[] p = s.replaceAll("^/+|/+$", "").split("/");
        if (p.length != 3) {
            throw new IllegalArgumentException("identity must be ns/group/name, got " + s);
        }
        return p;
    }

    private static String firstPositional(String[] a) {
        for (String x : a) {
            if (!x.startsWith("-")) return x.toLowerCase();
        }
        return "server";
    }

    private static String opt(String[] a, String name, String def) {
        for (int i = 0; i < a.length - 1; i++) {
            if (a[i].equals(name)) return a[i + 1];
        }
        return def;
    }

    private static String envOr(String key, String def) {
        String v = System.getenv(key);
        return (v == null || v.isEmpty()) ? def : v;
    }
}
