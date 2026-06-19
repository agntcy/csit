// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

using System.Text.Json;
using System.Text.Json.Nodes;
using System.Text.Json.Serialization;
using System.Text;
using System.Net;
using System.Net.Http.Headers;
using System.Net.ServerSentEvents;
using System.Runtime.CompilerServices;
using A2A;

namespace InteropProbe;

// Thin A2A probe adapter: executes a single A2A operation and writes the raw
// JSON result to stdout.  On A2A error, writes {"code":…,"message":"…"} to
// stderr and exits 1.  No assertions or orchestration live here — all test
// logic is in the Go/Ginkgo spec tree.

internal static class Program
{
    public static async Task<int> Main(string[] args)
    {
        ParsedArgs parsed;
        try
        {
            parsed = ParseArgs(args);
        }
        catch (Exception error)
        {
            Fail(-32600, error.Message);
            return 1;
        }

        try
        {
            var resolver = new A2ACardResolver(new Uri(parsed.CardUrl));
            var card = await resolver.GetAgentCardAsync().ConfigureAwait(false);
            var client = CreateClient(card);
            await RunSubcommandAsync(parsed, card, client).ConfigureAwait(false);
            return 0;
        }
        catch (A2AException error)
        {
            Fail((int)error.ErrorCode, error.Message);
            return 1;
        }
        catch (Exception error)
        {
            Fail(-32000, error.Message);
            return 1;
        }
    }

    private static void Fail(int code, string message)
    {
        Console.Error.WriteLine(JsonSerializer.Serialize(new { code, message }));
    }

    // ── argument parsing ────────────────────────────────────────────────────

    private record ParsedArgs(string CardUrl, string Subcommand, string[] Rest);

    private static ParsedArgs ParseArgs(string[] args)
    {
        string? cardUrl = null;
        var i = 0;
        while (i < args.Length)
        {
            if (args[i] == "--card-url")
            {
                i++;
                if (i >= args.Length) throw new ArgumentException("--card-url requires a value");
                cardUrl = args[i++];
            }
            else break;
        }
        if (cardUrl is null) throw new ArgumentException("missing --card-url");
        if (i >= args.Length) throw new ArgumentException("missing subcommand");
        var subcommand = args[i++];
        return new ParsedArgs(cardUrl, subcommand, args[i..]);
    }

    private static string RequireFlag(string[] args, string flag)
    {
        for (var i = 0; i < args.Length - 1; i++)
        {
            if (args[i] == flag) return args[i + 1];
        }
        throw new ArgumentException($"missing {flag}");
    }

    // ── subcommand dispatch ─────────────────────────────────────────────────

    private static async Task RunSubcommandAsync(ParsedArgs parsed, AgentCard card, IA2AClient? client)
    {
        switch (parsed.Subcommand)
        {
            case "send-message":
            {
                var msgJson = RequireFlag(parsed.Rest, "--message-json");
                var reqNode = JsonNode.Parse(msgJson)
                    ?? throw new InvalidOperationException("failed to parse send-message request");
                var response = await SendMessageAsync(client, card, reqNode).ConfigureAwait(false);
                PrintMessageResponse(response);
                return;
            }

            case "send-streaming-message":
            {
                var msgJson = RequireFlag(parsed.Rest, "--message-json");
                var request = JsonSerializer.Deserialize<SendMessageRequest>(msgJson, A2AJsonUtilities.DefaultOptions)
                    ?? throw new InvalidOperationException("failed to parse send-streaming-message request");
                await foreach (var ev in SendStreamingMessageAsync(client, card, request, msgJson).ConfigureAwait(false))
                {
                    PrintStreamEvent(ev);
                }
                return;
            }

            case "get-task":
            {
                var taskId = RequireFlag(parsed.Rest, "--task-id");
                var task = await GetTaskAsync(client, card, new GetTaskRequest { Id = taskId }).ConfigureAwait(false);
                Console.WriteLine(JsonSerializer.Serialize(task, A2AJsonUtilities.DefaultOptions));
                return;
            }

            case "cancel-task":
            {
                var taskId = RequireFlag(parsed.Rest, "--task-id");
                var task = await CancelTaskAsync(client, card, new CancelTaskRequest { Id = taskId }).ConfigureAwait(false);
                Console.WriteLine(JsonSerializer.Serialize(task, A2AJsonUtilities.DefaultOptions));
                return;
            }

            case "list-tasks":
            {
                var contextId = RequireFlag(parsed.Rest, "--context-id");
                var tasks = await ListTasksAsync(client, card, new ListTasksRequest { ContextId = contextId }).ConfigureAwait(false);
                var output = new { tasks };
                Console.WriteLine(JsonSerializer.Serialize(output, A2AJsonUtilities.DefaultOptions));
                return;
            }

            case "create-push-config":
            {
                var configJson = RequireFlag(parsed.Rest, "--config-json");
                var flat = JsonSerializer.Deserialize<FlatPushConfig>(configJson)
                    ?? throw new InvalidOperationException("failed to parse push config");
                var created = await CreatePushConfigAsync(client, card, flat).ConfigureAwait(false);
                Console.WriteLine(JsonSerializer.Serialize(created));
                return;
            }

            case "get-push-config":
            {
                var taskId = RequireFlag(parsed.Rest, "--task-id");
                var configId = RequireFlag(parsed.Rest, "--config-id");
                var result = await GetPushConfigAsync(client, card,
                    new GetTaskPushNotificationConfigRequest { TaskId = taskId, Id = configId }).ConfigureAwait(false);
                Console.WriteLine(JsonSerializer.Serialize(result));
                return;
            }

            case "list-push-configs":
            {
                var taskId = RequireFlag(parsed.Rest, "--task-id");
                var configs = await ListPushConfigsAsync(client, card,
                    new ListTaskPushNotificationConfigRequest { TaskId = taskId }).ConfigureAwait(false);
                Console.WriteLine(JsonSerializer.Serialize(new { configs }));
                return;
            }

            case "delete-push-config":
            {
                var taskId = RequireFlag(parsed.Rest, "--task-id");
                var configId = RequireFlag(parsed.Rest, "--config-id");
                await DeletePushConfigAsync(client, card,
                    new DeleteTaskPushNotificationConfigRequest { TaskId = taskId, Id = configId }).ConfigureAwait(false);
                return;
            }

            case "get-extended-card":
            {
                var extCard = await GetExtendedAgentCardAsync(client, card).ConfigureAwait(false);
                Console.WriteLine(JsonSerializer.Serialize(extCard, A2AJsonUtilities.DefaultOptions));
                return;
            }

            default:
                throw new ArgumentException($"unknown subcommand: {parsed.Subcommand}");
        }
    }

    // ── response printing ────────────────────────────────────────────────────

    private static void PrintMessageResponse(SendMessageResponse response)
    {
        if (response.Task is not null)
        {
            var data = JsonSerializer.SerializeToNode(response.Task, A2AJsonUtilities.DefaultOptions);
            Console.WriteLine(JsonSerializer.Serialize(new { type = "task", data }));
        }
        else if (response.Message is not null)
        {
            var data = JsonSerializer.SerializeToNode(response.Message, A2AJsonUtilities.DefaultOptions);
            Console.WriteLine(JsonSerializer.Serialize(new { type = "message", data }));
        }
        else
        {
            throw new InvalidOperationException("send-message: empty response");
        }
    }

    private static void PrintStreamEvent(StreamResponse ev)
    {
        switch (ev.PayloadCase)
        {
            case StreamResponseCase.Task:
            {
                var data = JsonSerializer.SerializeToNode(ev.Task!, A2AJsonUtilities.DefaultOptions);
                Console.WriteLine(JsonSerializer.Serialize(new { type = "task", data }));
                break;
            }
            case StreamResponseCase.StatusUpdate:
            {
                var data = JsonSerializer.SerializeToNode(ev.StatusUpdate!, A2AJsonUtilities.DefaultOptions);
                Console.WriteLine(JsonSerializer.Serialize(new { type = "task-status-update", data }));
                break;
            }
            case StreamResponseCase.ArtifactUpdate:
            {
                var data = JsonSerializer.SerializeToNode(ev.ArtifactUpdate!, A2AJsonUtilities.DefaultOptions);
                Console.WriteLine(JsonSerializer.Serialize(new { type = "task-artifact-update", data }));
                break;
            }
            case StreamResponseCase.Message:
            {
                var data = JsonSerializer.SerializeToNode(ev.Message!, A2AJsonUtilities.DefaultOptions);
                Console.WriteLine(JsonSerializer.Serialize(new { type = "message", data }));
                break;
            }
        }
    }

    // ── push config flat DTO (matches Go SDK's a2a.PushConfig JSON fields) ──

    private sealed class FlatPushConfig
    {
        [JsonPropertyName("taskId")]
        public string TaskId { get; set; } = string.Empty;

        [JsonPropertyName("id")]
        public string? Id { get; set; }

        [JsonPropertyName("url")]
        public string Url { get; set; } = string.Empty;

        [JsonPropertyName("token")]
        public string? Token { get; set; }

        [JsonPropertyName("authentication")]
        public FlatAuthInfo? Authentication { get; set; }
    }

    private sealed class FlatAuthInfo
    {
        [JsonPropertyName("scheme")]
        public string Scheme { get; set; } = string.Empty;

        [JsonPropertyName("credentials")]
        public string? Credentials { get; set; }
    }

    private static FlatPushConfig ToFlatPushConfig(string taskId, PushNotificationConfig config) =>
        new()
        {
            TaskId = taskId,
            Id = config.Id,
            Url = config.Url,
            Token = config.Token,
            Authentication = config.Authentication is not null
                ? new FlatAuthInfo { Scheme = config.Authentication.Scheme, Credentials = config.Authentication.Credentials }
                : null,
        };

    private static PushNotificationConfig FromFlatPushConfig(FlatPushConfig flat) =>
        new()
        {
            Id = flat.Id,
            Url = flat.Url,
            Token = flat.Token,
            Authentication = flat.Authentication is not null
                ? new AuthenticationInfo { Scheme = flat.Authentication.Scheme, Credentials = flat.Authentication.Credentials }
                : null,
        };

    // ── push config operations ──────────────────────────────────────────────

    private static async Task<FlatPushConfig> CreatePushConfigAsync(
        IA2AClient? client,
        AgentCard card,
        FlatPushConfig flat)
    {
        var config = FromFlatPushConfig(flat);
        if (!UsesJsonRpcCompat(card))
        {
            var result = await PostRestAsync<PushNotificationConfig, TaskPushNotificationConfig>(
                card,
                $"/tasks/{Uri.EscapeDataString(flat.TaskId)}/pushNotificationConfigs",
                config).ConfigureAwait(false);
            return ToFlatPushConfig(flat.TaskId, result.PushNotificationConfig ?? config);
        }

        var rpcResult = await SendJsonRpcAsync<CompatibleTaskPushNotificationConfig>(
            card, "CreateTaskPushNotificationConfig",
            new CreateTaskPushNotificationConfigRequest { TaskId = flat.TaskId, ConfigId = flat.Id ?? string.Empty, Config = config }
        ).ConfigureAwait(false);
        return ToFlatPushConfig(rpcResult.TaskId, rpcResult.GetConfig());
    }

    private static async Task<FlatPushConfig> GetPushConfigAsync(
        IA2AClient? client,
        AgentCard card,
        GetTaskPushNotificationConfigRequest request)
    {
        if (!UsesJsonRpcCompat(card))
        {
            var result = await GetRestAsync<TaskPushNotificationConfig>(
                card,
                $"/tasks/{Uri.EscapeDataString(request.TaskId)}/pushNotificationConfigs/{Uri.EscapeDataString(request.Id)}"
            ).ConfigureAwait(false);
            return ToFlatPushConfig(request.TaskId, result.PushNotificationConfig ?? new PushNotificationConfig { Id = request.Id, Url = string.Empty });
        }

        var rpcResult = await SendJsonRpcAsync<CompatibleTaskPushNotificationConfig>(
            card, "GetTaskPushNotificationConfig", request).ConfigureAwait(false);
        return ToFlatPushConfig(rpcResult.TaskId, rpcResult.GetConfig());
    }

    private static async Task<List<FlatPushConfig>> ListPushConfigsAsync(
        IA2AClient? client,
        AgentCard card,
        ListTaskPushNotificationConfigRequest request)
    {
        if (!UsesJsonRpcCompat(card))
        {
            var restResponse = await GetRestAsync<ListTaskPushNotificationConfigResponse>(
                card,
                BuildListPushConfigsPath(request)).ConfigureAwait(false);
            return (restResponse.Configs ?? [])
                .Select(c => ToFlatPushConfig(request.TaskId, c.PushNotificationConfig ?? new PushNotificationConfig { Url = string.Empty }))
                .ToList();
        }

        var rpcResponse = await SendJsonRpcAsync<JsonElement>(
            card, "ListTaskPushNotificationConfigs", request).ConfigureAwait(false);

        var compatList = rpcResponse.ValueKind switch
        {
            JsonValueKind.Array => JsonSerializer.Deserialize<List<CompatibleTaskPushNotificationConfig>>(
                rpcResponse.GetRawText(), A2AJsonUtilities.DefaultOptions) ?? [],
            JsonValueKind.Object => JsonSerializer.Deserialize<CompatibleListTaskPushNotificationConfigResponse>(
                rpcResponse.GetRawText(), A2AJsonUtilities.DefaultOptions)?.Configs ?? [],
            _ => throw new InvalidOperationException("unexpected list push config JSON-RPC result shape"),
        };
        return compatList.Select(c => ToFlatPushConfig(c.TaskId, c.GetConfig())).ToList();
    }

    private static async Task DeletePushConfigAsync(
        IA2AClient? client,
        AgentCard card,
        DeleteTaskPushNotificationConfigRequest request)
    {
        if (!UsesJsonRpcCompat(card))
        {
            await SendRestAsync(
                HttpMethod.Delete,
                card,
                $"/tasks/{Uri.EscapeDataString(request.TaskId)}/pushNotificationConfigs/{Uri.EscapeDataString(request.Id)}"
            ).ConfigureAwait(false);
            return;
        }

        await SendJsonRpcWithoutResultAsync(card, "DeleteTaskPushNotificationConfig", request).ConfigureAwait(false);
    }

    // ── A2A operations ──────────────────────────────────────────────────────

    private static async Task<SendMessageResponse> SendMessageAsync(
        IA2AClient? client,
        AgentCard card,
        JsonNode request)
    {
        if (UsesJsonRpcCompat(card))
        {
            return await SendJsonRpcAsync<SendMessageResponse>(card, "SendMessage", request).ConfigureAwait(false);
        }

        RemoveNullProperties(request);
        return await PostRestAsync<JsonNode, SendMessageResponse>(card, "/message:send", request).ConfigureAwait(false);
    }

    private static IAsyncEnumerable<StreamResponse> SendStreamingMessageAsync(
        IA2AClient? client,
        AgentCard card,
        SendMessageRequest request,
        string rawJson)
    {
        if (UsesJsonRpcCompat(card))
        {
            return client!.SendStreamingMessageAsync(request);
        }

        var reqNode = JsonNode.Parse(rawJson) ?? throw new InvalidOperationException("failed to re-parse streaming request");
        RemoveNullProperties(reqNode);
        return PostRestStreamingAsync(card, "/message:stream", reqNode);
    }

    private static async Task<AgentTask> GetTaskAsync(IA2AClient? client, AgentCard card, GetTaskRequest request)
    {
        if (UsesJsonRpcCompat(card))
        {
            return await client!.GetTaskAsync(request).ConfigureAwait(false);
        }

        return await GetRestAsync<AgentTask>(card, BuildGetTaskPath(request)).ConfigureAwait(false);
    }

    private static async Task<AgentTask> CancelTaskAsync(IA2AClient? client, AgentCard card, CancelTaskRequest request)
    {
        if (UsesJsonRpcCompat(card))
        {
            return await client!.CancelTaskAsync(request).ConfigureAwait(false);
        }

        var path = $"/tasks/{Uri.EscapeDataString(request.Id)}:cancel";
        return request.Metadata is null
            ? await PostRestEmptyAsync<AgentTask>(card, path).ConfigureAwait(false)
            : await PostRestAsync<object, AgentTask>(card, path, new { metadata = request.Metadata }).ConfigureAwait(false);
    }

    private static async Task<List<AgentTask>> ListTasksAsync(IA2AClient? client, AgentCard card, ListTasksRequest request)
    {
        if (!UsesJsonRpcCompat(card))
        {
            var restResponse = await GetRestAsync<ListTasksResponse>(card, BuildListTasksPath(request)).ConfigureAwait(false);
            return restResponse.Tasks;
        }

        var rpcResponse = await SendJsonRpcAsync<CompatibleListTasksResponse>(card, "ListTasks", request).ConfigureAwait(false);
        return rpcResponse.Tasks;
    }

    private static async Task<AgentCard> GetExtendedAgentCardAsync(IA2AClient? client, AgentCard card)
    {
        if (UsesJsonRpcCompat(card))
        {
            return await client!.GetExtendedAgentCardAsync(new GetExtendedAgentCardRequest()).ConfigureAwait(false);
        }

        return await GetRestAsync<AgentCard>(card, "/extendedAgentCard").ConfigureAwait(false);
    }

    // ── client creation ─────────────────────────────────────────────────────

    private static IA2AClient? CreateClient(AgentCard card)
    {
        var agentInterface = GetPrimaryInterface(card);

        if (string.Equals(agentInterface.ProtocolBinding, "JSONRPC", StringComparison.OrdinalIgnoreCase))
        {
            return new A2AClient(new Uri(agentInterface.Url));
        }

        if (string.Equals(agentInterface.ProtocolBinding, "HTTP+JSON", StringComparison.OrdinalIgnoreCase))
        {
            return null;
        }

        throw new InvalidOperationException($"unsupported protocol binding: {agentInterface.ProtocolBinding}");
    }

    // ── REST helpers ────────────────────────────────────────────────────────

    private static async Task<TResult> GetRestAsync<TResult>(AgentCard card, string path)
    {
        using var httpClient = new HttpClient();
        using var request = CreateRestRequest(HttpMethod.Get, card, path);
        using var response = await httpClient.SendAsync(request).ConfigureAwait(false);
        await EnsureRestSuccessOrThrowA2AExceptionAsync(response).ConfigureAwait(false);
        return await ReadRestResponseAsync<TResult>(response.Content, path).ConfigureAwait(false);
    }

    private static async Task<TResult> PostRestAsync<TBody, TResult>(AgentCard card, string path, TBody body)
    {
        using var httpClient = new HttpClient();
        using var request = CreateRestRequest(HttpMethod.Post, card, path);
        request.Content = new StringContent(JsonSerializer.Serialize(body, A2AJsonUtilities.DefaultOptions), Encoding.UTF8, "application/json");
        using var response = await httpClient.SendAsync(request).ConfigureAwait(false);
        await EnsureRestSuccessOrThrowA2AExceptionAsync(response).ConfigureAwait(false);
        return await ReadRestResponseAsync<TResult>(response.Content, path).ConfigureAwait(false);
    }

    private static async Task<TResult> PostRestEmptyAsync<TResult>(AgentCard card, string path)
    {
        using var httpClient = new HttpClient();
        using var request = CreateRestRequest(HttpMethod.Post, card, path);
        using var response = await httpClient.SendAsync(request).ConfigureAwait(false);
        await EnsureRestSuccessOrThrowA2AExceptionAsync(response).ConfigureAwait(false);
        return await ReadRestResponseAsync<TResult>(response.Content, path).ConfigureAwait(false);
    }

    private static async Task SendRestAsync(HttpMethod method, AgentCard card, string path)
    {
        using var httpClient = new HttpClient();
        using var request = CreateRestRequest(method, card, path);
        using var response = await httpClient.SendAsync(request).ConfigureAwait(false);
        await EnsureRestSuccessOrThrowA2AExceptionAsync(response).ConfigureAwait(false);
    }

    private static async IAsyncEnumerable<StreamResponse> PostRestStreamingAsync(
        AgentCard card,
        string path,
        JsonNode body,
        [EnumeratorCancellation] CancellationToken cancellationToken = default)
    {
        using var httpClient = new HttpClient();
        using var request = CreateRestRequest(HttpMethod.Post, card, path);
        request.Content = new StringContent(JsonSerializer.Serialize(body, A2AJsonUtilities.DefaultOptions), Encoding.UTF8, "application/json");
        request.Headers.Accept.Add(new MediaTypeWithQualityHeaderValue("text/event-stream"));

        using var response = await httpClient.SendAsync(request, HttpCompletionOption.ResponseHeadersRead, cancellationToken).ConfigureAwait(false);
        await EnsureRestSuccessOrThrowA2AExceptionAsync(response).ConfigureAwait(false);

        await using var stream = await response.Content.ReadAsStreamAsync(cancellationToken).ConfigureAwait(false);
        await foreach (var sseItem in SseParser.Create(stream).EnumerateAsync(cancellationToken).ConfigureAwait(false))
        {
            yield return ReadRestPayload<StreamResponse>(sseItem.Data, path);
        }
    }

    private static async Task<TResult> ReadRestResponseAsync<TResult>(HttpContent content, string path)
    {
        await using var stream = await content.ReadAsStreamAsync().ConfigureAwait(false);
        var payload = await JsonNode.ParseAsync(stream).ConfigureAwait(false)
            ?? throw new InvalidOperationException($"received empty REST response for {path}");
        return ReadRestPayload<TResult>(payload, path);
    }

    private static TResult ReadRestPayload<TResult>(string json, string path)
    {
        var payload = JsonNode.Parse(json)
            ?? throw new InvalidOperationException($"received empty REST response for {path}");
        return ReadRestPayload<TResult>(payload, path);
    }

    private static TResult ReadRestPayload<TResult>(JsonNode payload, string path)
    {
        RemoveNullProperties(payload);
        payload = NormalizeRestPayload<TResult>(payload);
        var result = payload.Deserialize<TResult>(A2AJsonUtilities.DefaultOptions);
        return result ?? throw new InvalidOperationException($"failed to deserialize REST response for {path}: {payload.ToJsonString()}");
    }

    private static JsonNode NormalizeRestPayload<TResult>(JsonNode payload)
    {
        if (typeof(TResult) == typeof(ListTaskPushNotificationConfigResponse) && payload is JsonArray configsArray)
        {
            foreach (var item in configsArray.OfType<JsonObject>())
            {
                NormalizeTaskPushNotificationConfig(item);
            }

            return new JsonObject
            {
                ["configs"] = configsArray,
                ["nextPageToken"] = string.Empty,
            };
        }

        if (payload is not JsonObject root)
        {
            return payload;
        }

        if (typeof(TResult) == typeof(ListTasksResponse))
        {
            EnsureStringProperty(root, "nextPageToken");
            EnsureIntProperty(root, "pageSize");
            EnsureIntProperty(root, "totalSize");
            return root;
        }

        if (typeof(TResult) == typeof(ListTaskPushNotificationConfigResponse))
        {
            EnsureStringProperty(root, "nextPageToken");
            if (root["configs"] is JsonArray configs)
            {
                foreach (var item in configs.OfType<JsonObject>())
                {
                    NormalizeTaskPushNotificationConfig(item);
                }
            }
            return root;
        }

        if (typeof(TResult) == typeof(TaskPushNotificationConfig))
        {
            NormalizeTaskPushNotificationConfig(root);
        }

        return root;
    }

    private static void NormalizeTaskPushNotificationConfig(JsonObject root)
    {
        if (root["config"] is not JsonObject config || root.ContainsKey("pushNotificationConfig"))
        {
            return;
        }

        if (!root.ContainsKey("id") && config["id"] is not null)
        {
            root["id"] = config["id"]!.DeepClone();
        }

        root["pushNotificationConfig"] = config.DeepClone();
        root.Remove("config");
    }

    private static void RemoveNullProperties(JsonNode? node)
    {
        if (node is JsonObject obj)
        {
            foreach (var property in obj.ToList())
            {
                if (property.Value is null)
                {
                    obj.Remove(property.Key);
                    continue;
                }

                RemoveNullProperties(property.Value);
            }

            return;
        }

        if (node is JsonArray array)
        {
            foreach (var item in array)
            {
                RemoveNullProperties(item);
            }
        }
    }

    private static void EnsureStringProperty(JsonObject obj, string propertyName)
    {
        if (!obj.ContainsKey(propertyName))
        {
            obj[propertyName] = string.Empty;
        }
    }

    private static void EnsureIntProperty(JsonObject obj, string propertyName)
    {
        if (!obj.ContainsKey(propertyName))
        {
            obj[propertyName] = 0;
        }
    }

    private static HttpRequestMessage CreateRestRequest(HttpMethod method, AgentCard card, string path)
    {
        var request = new HttpRequestMessage(method, GetPrimaryInterface(card).Url.TrimEnd('/') + path);
        request.Headers.TryAddWithoutValidation("A2A-Version", "1.0");
        return request;
    }

    private static async Task EnsureRestSuccessOrThrowA2AExceptionAsync(HttpResponseMessage response)
    {
        if (response.IsSuccessStatusCode)
        {
            return;
        }

        string? detail = null;
        A2AErrorCode? mappedCode = null;

        try
        {
            var contentType = response.Content.Headers.ContentType?.MediaType;
            if (string.Equals(contentType, "application/json", StringComparison.OrdinalIgnoreCase))
            {
                await using var stream = await response.Content.ReadAsStreamAsync().ConfigureAwait(false);
                var errorResponse = await JsonSerializer.DeserializeAsync<RestA2AErrorResponse>(stream, A2AJsonUtilities.DefaultOptions).ConfigureAwait(false);
                if (errorResponse?.Error is { } error)
                {
                    detail = error.Message;
                    var errorInfo = error.Details?.FirstOrDefault(value =>
                        string.Equals(value.Domain, "a2a-protocol.org", StringComparison.OrdinalIgnoreCase));
                    if (errorInfo?.Reason is not null && ReasonToErrorCode.TryGetValue(errorInfo.Reason, out var code))
                    {
                        mappedCode = code;
                    }
                }
            }
            else
            {
                detail = await response.Content.ReadAsStringAsync().ConfigureAwait(false);
            }
        }
        catch
        {
        }

        var errorCode = mappedCode ?? response.StatusCode switch
        {
            HttpStatusCode.NotFound => A2AErrorCode.TaskNotFound,
            HttpStatusCode.BadRequest => A2AErrorCode.InvalidRequest,
            HttpStatusCode.Conflict => A2AErrorCode.TaskNotCancelable,
            HttpStatusCode.UnsupportedMediaType => A2AErrorCode.ContentTypeNotSupported,
            HttpStatusCode.BadGateway => A2AErrorCode.InvalidAgentResponse,
            _ => A2AErrorCode.InternalError,
        };

        var message = !string.IsNullOrEmpty(detail)
            ? $"HTTP {(int)response.StatusCode}: {detail}"
            : $"HTTP {(int)response.StatusCode}: {response.ReasonPhrase}";

        throw new A2AException(message, errorCode);
    }

    private static readonly Dictionary<string, A2AErrorCode> ReasonToErrorCode = new(StringComparer.OrdinalIgnoreCase)
    {
        ["TASK_NOT_FOUND"] = A2AErrorCode.TaskNotFound,
        ["TASK_NOT_CANCELABLE"] = A2AErrorCode.TaskNotCancelable,
        ["PUSH_NOTIFICATION_NOT_SUPPORTED"] = A2AErrorCode.PushNotificationNotSupported,
        ["UNSUPPORTED_OPERATION"] = A2AErrorCode.UnsupportedOperation,
        ["CONTENT_TYPE_NOT_SUPPORTED"] = A2AErrorCode.ContentTypeNotSupported,
        ["INVALID_AGENT_RESPONSE"] = A2AErrorCode.InvalidAgentResponse,
        ["EXTENDED_AGENT_CARD_NOT_CONFIGURED"] = A2AErrorCode.ExtendedAgentCardNotConfigured,
        ["EXTENSION_SUPPORT_REQUIRED"] = A2AErrorCode.ExtensionSupportRequired,
        ["VERSION_NOT_SUPPORTED"] = A2AErrorCode.VersionNotSupported,
    };

    private static bool UsesJsonRpcCompat(AgentCard card) =>
        string.Equals(GetPrimaryInterface(card).ProtocolBinding, "JSONRPC", StringComparison.OrdinalIgnoreCase);

    private static AgentInterface GetPrimaryInterface(AgentCard card) =>
        card.SupportedInterfaces.FirstOrDefault()
        ?? throw new InvalidOperationException("agent card did not advertise any interfaces");

    private static AgentInterface GetJsonRpcInterface(AgentCard card)
    {
        var jsonRpcInterface = card.SupportedInterfaces.FirstOrDefault(candidate =>
            string.Equals(candidate.ProtocolBinding, "JSONRPC", StringComparison.OrdinalIgnoreCase));

        return jsonRpcInterface
            ?? throw new InvalidOperationException("agent card did not advertise a JSON-RPC interface");
    }

    private static string BuildGetTaskPath(GetTaskRequest request) =>
        $"/tasks/{Uri.EscapeDataString(request.Id)}{BuildQueryString(("historyLength", request.HistoryLength?.ToString(System.Globalization.CultureInfo.InvariantCulture)))}";

    private static string BuildListTasksPath(ListTasksRequest request) =>
        "/tasks" + BuildQueryString(
            ("contextId", request.ContextId),
            ("status", request.Status?.ToString()),
            ("pageSize", request.PageSize?.ToString(System.Globalization.CultureInfo.InvariantCulture)),
            ("pageToken", request.PageToken),
            ("historyLength", request.HistoryLength?.ToString(System.Globalization.CultureInfo.InvariantCulture)));

    private static string BuildListPushConfigsPath(ListTaskPushNotificationConfigRequest request) =>
        $"/tasks/{Uri.EscapeDataString(request.TaskId)}/pushNotificationConfigs" + BuildQueryString(
            ("pageSize", request.PageSize?.ToString(System.Globalization.CultureInfo.InvariantCulture)),
            ("pageToken", request.PageToken));

    private static string BuildQueryString(params (string Key, string? Value)[] parameters)
    {
        var parts = new List<string>();
        foreach (var (key, value) in parameters)
        {
            if (!string.IsNullOrEmpty(value))
            {
                parts.Add($"{key}={Uri.EscapeDataString(value)}");
            }
        }
        return parts.Count > 0 ? "?" + string.Join("&", parts) : string.Empty;
    }

    // ── JSON-RPC helpers ─────────────────────────────────────────────────────

    private static async Task<TResult> SendJsonRpcAsync<TResult>(AgentCard card, string method, object payload)
    {
        using var httpClient = new HttpClient();
        var request = new JsonRpcRequest
        {
            Id = Guid.NewGuid().ToString("N"),
            Method = method,
            Params = JsonSerializer.SerializeToElement(payload, A2AJsonUtilities.DefaultOptions),
        };

        using var message = new HttpRequestMessage(HttpMethod.Post, GetJsonRpcInterface(card).Url)
        {
            Content = new StringContent(
                JsonSerializer.Serialize(request, A2AJsonUtilities.DefaultOptions),
                Encoding.UTF8,
                "application/json"),
        };
        message.Headers.TryAddWithoutValidation("A2A-Version", "1.0");

        using var response = await httpClient.SendAsync(message).ConfigureAwait(false);
        response.EnsureSuccessStatusCode();

        await using var stream = await response.Content.ReadAsStreamAsync().ConfigureAwait(false);
        var rpcResponse = await JsonSerializer.DeserializeAsync<JsonRpcResponse>(stream, A2AJsonUtilities.DefaultOptions).ConfigureAwait(false)
            ?? throw new InvalidOperationException("failed to deserialize JSON-RPC response");

        if (rpcResponse.Error is not null)
        {
            throw new A2AException(rpcResponse.Error.Message, (A2AErrorCode)rpcResponse.Error.Code);
        }

        if (rpcResponse.Result is null)
        {
            throw new InvalidOperationException($"JSON-RPC {method}: null result payload");
        }

        var rawResult = rpcResponse.Result.ToJsonString();
        return JsonSerializer.Deserialize<TResult>(rawResult, A2AJsonUtilities.DefaultOptions)
            ?? throw new InvalidOperationException($"failed to deserialize JSON-RPC result for {method}");
    }

    private static async Task SendJsonRpcWithoutResultAsync(AgentCard card, string method, object payload)
    {
        using var httpClient = new HttpClient();
        var request = new JsonRpcRequest
        {
            Id = Guid.NewGuid().ToString("N"),
            Method = method,
            Params = JsonSerializer.SerializeToElement(payload, A2AJsonUtilities.DefaultOptions),
        };

        using var message = new HttpRequestMessage(HttpMethod.Post, GetJsonRpcInterface(card).Url)
        {
            Content = new StringContent(
                JsonSerializer.Serialize(request, A2AJsonUtilities.DefaultOptions),
                Encoding.UTF8,
                "application/json"),
        };
        message.Headers.TryAddWithoutValidation("A2A-Version", "1.0");

        using var response = await httpClient.SendAsync(message).ConfigureAwait(false);
        response.EnsureSuccessStatusCode();

        await using var stream = await response.Content.ReadAsStreamAsync().ConfigureAwait(false);
        var rpcResponse = await JsonSerializer.DeserializeAsync<JsonRpcResponse>(stream, A2AJsonUtilities.DefaultOptions).ConfigureAwait(false)
            ?? throw new InvalidOperationException($"failed to deserialize JSON-RPC response for {method}");

        if (rpcResponse.Error is not null)
        {
            throw new A2AException(rpcResponse.Error.Message, (A2AErrorCode)rpcResponse.Error.Code);
        }
    }
}

// ── supporting types ──────────────────────────────────────────────────────────

internal sealed class CompatibleListTasksResponse
{
    public List<AgentTask> Tasks { get; set; } = [];
    public string? NextPageToken { get; set; }
    public int? PageSize { get; set; }
    public int? TotalSize { get; set; }
}

internal sealed class CompatibleTaskPushNotificationConfig
{
    public string TaskId { get; set; } = string.Empty;
    public PushNotificationConfig? Config { get; set; }

    // Flat fields for v1 format
    public string? Id { get; set; }
    public string? Url { get; set; }
    public string? Token { get; set; }
    public AuthenticationInfo? Authentication { get; set; }

    public PushNotificationConfig GetConfig() =>
        Config ?? new PushNotificationConfig
        {
            Id = Id,
            Url = Url ?? string.Empty,
            Token = Token,
            Authentication = Authentication,
        };
}

internal sealed class CompatibleListTaskPushNotificationConfigResponse
{
    public List<CompatibleTaskPushNotificationConfig> Configs { get; set; } = [];
}

internal sealed class RestA2AErrorResponse
{
    public RestA2AErrorStatus? Error { get; set; }
}

internal sealed class RestA2AErrorStatus
{
    public int Code { get; set; }
    public string? Status { get; set; }
    public string? Message { get; set; }
    public List<RestA2AErrorDetail>? Details { get; set; }
}

internal sealed class RestA2AErrorDetail
{
    public string? Type { get; set; }
    public string? Reason { get; set; }
    public string? Domain { get; set; }
}
