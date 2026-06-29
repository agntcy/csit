// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0
//
// CSIT fixture: A2A echo agent + probe over SLIMRPC (lf.a2a.v1) for cross-language interop.
// One binary, two modes mirroring fixtures/go (cmd/server, cmd/probe):
//   csit-slim-a2a server --slim-endpoint <url> --identity <ns/grp/name> --secret <s>
//   csit-slim-a2a probe  --slim-endpoint <url> --local <id> --remote <id> --secret <s> --scenario <sc> --text <t>

using A2A;
using Agntcy.Slim;
using Agntcy.Slim.SlimRpc;
using Csit;
using Microsoft.Extensions.Logging.Abstractions;
using SlimA2A;

const string ReadyMarker = "CSIT_SLIM_SERVER_READY";
const string DefaultEndpoint = "http://127.0.0.1:46357";
const string DefaultSecret = "my_shared_secret_for_testing_purposes_only";

string mode = FirstPositional(args);
string endpoint = Opt(args, "--slim-endpoint") ?? Env("SLIM_SERVER") ?? DefaultEndpoint;
string secret = Opt(args, "--secret") ?? Env("SLIM_SHARED_SECRET") ?? DefaultSecret;

switch (mode)
{
    case "server":
        await RunServer(
            endpoint, secret,
            Opt(args, "--identity") ?? "agntcy/a2a_csit_slim/server_dotnet").ConfigureAwait(false);
        return 0;
    case "probe":
        return await RunProbe(
            endpoint, secret,
            local: Opt(args, "--local") ?? "agntcy/a2a_csit_slim/client_dotnet",
            remote: Opt(args, "--remote") ?? "agntcy/a2a_csit_slim/server_dotnet",
            scenario: Opt(args, "--scenario") ?? Scenarios.Echo,
            text: Opt(args, "--text") ?? "Hello there!").ConfigureAwait(false);
    default:
        Console.Error.WriteLine("usage: csit-slim-a2a [server|probe] [flags]");
        return 2;
}

// ---- server -------------------------------------------------------------------------

async Task RunServer(string ep, string sec, string identity)
{
    var card = MinimalCard(identity);
    var store = new InMemoryTaskStore();
    var notifier = new ChannelEventNotifier();
    var a2a = new A2AServer(new CsitEchoHandler(), store, notifier, NullLogger<A2AServer>.Instance);
    var slimHandler = new SlimA2AHandler(a2a, _ => Task.FromResult(card));

    var (app, connId) = await SlimHelper.ConnectAndSubscribeAsync(identity, sec, ep).ConfigureAwait(false);
    using (app)
    {
        using var localName = SlimName.Parse(identity);
        using var slimServer = SlimRpcServerFactory.CreateServer(app, localName, connId);
        SlimA2AServerRegistration.RegisterA2AService(slimServer, slimHandler);

        // Ready marker on stdout (the harness waits for this line), matching the Go fixture.
        Console.WriteLine(ReadyMarker);
        Console.Out.Flush();

        Console.CancelKeyPress += (_, e) =>
        {
            e.Cancel = true;
            _ = slimServer.ShutdownAsync();
        };
        try
        {
            await slimServer.ServeAsync().ConfigureAwait(false);
        }
        catch (OperationCanceledException)
        {
            // graceful shutdown
        }
    }
}

// ---- probe --------------------------------------------------------------------------

async Task<int> RunProbe(string ep, string sec, string local, string remote, string scenario, string text)
{
    (string want, bool enforceEcho, string pmode) = Scenarios.Request(scenario, text);

    var (app, connId) = await SlimHelper.ConnectAndSubscribeAsync(local, sec, ep).ConfigureAwait(false);
    using (app)
    {
        using var remoteName = SlimName.Parse(remote);
        using var channel = SlimRpcChannelFactory.CreateChannel(app, remoteName, connId);
        var client = new SlimA2AClient(channel);

        Observation obs = pmode switch
        {
            Scenarios.ModeStreaming => await RunStreaming(client, want).ConfigureAwait(false),
            Scenarios.ModeCancel => await RunCancel(client, want).ConfigureAwait(false),
            Scenarios.ModeMultiTurn => await RunMultiTurn(client).ConfigureAwait(false),
            _ => await RunUnary(client, want).ConfigureAwait(false),
        };

        EmitObservation(obs);

        if (enforceEcho && !obs.Text.Contains(want, StringComparison.Ordinal))
        {
            Console.Error.WriteLine($"response {obs.Text} does not contain sent text {want}");
            return 1;
        }
    }
    return 0;
}

static SendMessageRequest BuildRequest(string text, string? taskId = null, string? contextId = null)
{
    var msg = new Message
    {
        Role = Role.User,
        MessageId = Guid.NewGuid().ToString("N"),
        Parts = [Part.FromText(text)],
    };
    if (taskId is not null) msg.TaskId = taskId;
    if (contextId is not null) msg.ContextId = contextId;
    return new SendMessageRequest { Message = msg };
}

async Task<Observation> RunUnary(SlimA2AClient client, string text)
{
    var resp = await client.SendMessageAsync(BuildRequest(text)).ConfigureAwait(false);
    return Observation.FromResponse(resp);
}

async Task<Observation> RunStreaming(SlimA2AClient client, string text)
{
    var obs = new Observation { Kind = "task" };
    int events = 0;
    await foreach (var ev in client.SendStreamingMessageAsync(BuildRequest(text)).ConfigureAwait(false))
    {
        events++;
        obs.Absorb(ev);
    }
    obs.StreamEvents = events;
    return obs;
}

async Task<Observation> RunCancel(SlimA2AClient client, string text)
{
    // Read just enough of the stream to learn the server-assigned task ID, then stop:
    // the task stays working (non-terminal), so cancel it via CancelTask and observe the
    // terminal (canceled) task.
    using var cts = new CancellationTokenSource();
    string? taskId = null;
    int events = 0;
    try
    {
        await foreach (var ev in client.SendStreamingMessageAsync(BuildRequest(text), cts.Token).ConfigureAwait(false))
        {
            events++;
            taskId = ev.PayloadCase switch
            {
                StreamResponseCase.Task => ev.Task?.Id,
                StreamResponseCase.StatusUpdate => ev.StatusUpdate?.TaskId,
                StreamResponseCase.ArtifactUpdate => ev.ArtifactUpdate?.TaskId,
                _ => taskId,
            };
            if (!string.IsNullOrEmpty(taskId)) break;
        }
    }
    finally
    {
        cts.Cancel();
    }

    if (string.IsNullOrEmpty(taskId))
        throw new InvalidOperationException("task-cancel scenario: no task id observed");

    var canceled = await client.CancelTaskAsync(new CancelTaskRequest { Id = taskId }).ConfigureAwait(false);
    var obs = Observation.FromTask(canceled);
    obs.StreamEvents = events;
    return obs;
}

async Task<Observation> RunMultiTurn(SlimA2AClient client)
{
    var res1 = await client.SendMessageAsync(BuildRequest(Scenarios.SentinelMultiTurn)).ConfigureAwait(false);
    if (res1.PayloadCase != SendMessageResponseCase.Task || res1.Task is null)
        throw new InvalidOperationException($"multi-turn start: expected a task, got {res1.PayloadCase}");
    var task1 = res1.Task;
    if (task1.Status?.State != TaskState.InputRequired)
        throw new InvalidOperationException($"multi-turn start: expected input-required, got {task1.Status?.State}");

    var res2 = await client.SendMessageAsync(
        BuildRequest(Scenarios.SentinelMultiTurnContinue, task1.Id, task1.ContextId)).ConfigureAwait(false);
    return Observation.FromResponse(res2);
}

void EmitObservation(Observation o)
{
    Console.WriteLine($"CSIT_SLIM_RESULT_KIND={o.Kind}");
    Console.WriteLine($"CSIT_SLIM_TASK_STATE={o.State}");
    Console.WriteLine($"CSIT_SLIM_ARTIFACT_PRESENT={(o.ArtifactPresent ? "true" : "false")}");
    Console.WriteLine($"CSIT_SLIM_STREAM_EVENTS={o.StreamEvents}");
    Console.WriteLine($"CSIT_SLIM_ARTIFACT_TEXT={o.Text}");
    Console.WriteLine(o.Text);
    Console.Out.Flush();
}

static AgentCard MinimalCard(string name) => new()
{
    Name = "CSIT Echo Agent (SLIM)",
    Description = "Echoes messages over A2A on SLIMRPC for CSIT interop.",
    Version = "1.0.0",
    SupportedInterfaces =
    [
        new AgentInterface { Url = $"slim://{name}", ProtocolBinding = "SLIMRPC", ProtocolVersion = "1.0" },
    ],
    DefaultInputModes = ["text/plain"],
    DefaultOutputModes = ["text/plain"],
    Capabilities = new AgentCapabilities { Streaming = true, PushNotifications = false },
    Skills = [new AgentSkill { Id = "echo", Name = "Echo", Description = "Echoes back the user message.", Tags = ["echo", "test"] }],
};

// ---- arg helpers --------------------------------------------------------------------

static string FirstPositional(string[] a)
{
    foreach (var x in a)
        if (!x.StartsWith('-')) return x.ToLowerInvariant();
    return "server";
}

static string? Opt(string[] a, string name)
{
    for (int i = 0; i < a.Length - 1; i++)
        if (a[i] == name) return a[i + 1];
    return null;
}

static string? Env(string key)
{
    var v = Environment.GetEnvironmentVariable(key);
    return string.IsNullOrEmpty(v) ? null : v;
}
