// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

using System.Text.Json;
using System.Text.Json.Nodes;
using System.Text;
using System.Net;
using System.Net.Http.Headers;
using System.Net.ServerSentEvents;
using System.Runtime.CompilerServices;
using A2A;

namespace InteropProbe;

internal static class Program
{
    private const string RequestText = "ping";
    private const string PendingRequestText = "pending";
    private const string RequestDataKind = "structured";
    private const string RequestDataScope = "interop";
    private const string RequestMetadataKey = "csit";
    private const string RequestMetadataValue = "multipart";

    public static async Task<int> Main(string[] args)
    {
        ProbeOptions options;

        try
        {
            options = ProbeOptions.Parse(args);
        }
        catch (Exception error)
        {
            Console.Error.WriteLine(error);
            return 2;
        }

        try
        {
            await RunAsync(options);
            return 0;
        }
        catch (Exception error)
        {
            Console.Error.WriteLine(error);
            return 1;
        }
    }

    private static async Task RunAsync(ProbeOptions options)
    {
        var resolver = new A2ACardResolver(new Uri(options.CardUrl));
        var card = await resolver.GetAgentCardAsync().ConfigureAwait(false);
        var client = CreateClient(card);

        var expectedPingText = ExpectedResponseText(options.ServerPrefix, RequestText);
        var expectedPendingText = ExpectedResponseText(options.ServerPrefix, PendingRequestText);
        var expectedCancelText = ExpectedCancelText(options.ServerPrefix);

        var request = BuildRequest(RequestText, false);

        var completedTask = TaskFromResponse(
            await SendMessageAsync(client, card, request, false).ConfigureAwait(false),
            "unary");
        AssertState(completedTask.Status.State, TaskState.Completed, "unary");
        AssertText(TaskText(completedTask), expectedPingText, "unary");
        AssertTaskHistory(completedTask, RequestText, "unary");

        var fetchedTask = await GetTaskAsync(client, card, new GetTaskRequest
        {
            Id = completedTask.Id,
            HistoryLength = 1,
        }).ConfigureAwait(false);
        AssertState(fetchedTask.Status.State, TaskState.Completed, "get_task");
        AssertText(TaskText(fetchedTask), expectedPingText, "get_task");
        AssertTaskHistory(fetchedTask, RequestText, "get_task");

        var listedTasks = await ListTasksAsync(client, card, new ListTasksRequest
        {
            ContextId = completedTask.ContextId,
        }).ConfigureAwait(false);
        if (!listedTasks.Any(task => task.Id == completedTask.Id))
        {
            throw new InvalidOperationException($"list_tasks did not include expected task {completedTask.Id}");
        }

        var streamingText = await ReadStreamingTextAsync(SendStreamingMessageAsync(client, card, request)).ConfigureAwait(false);
        AssertText(streamingText, expectedPingText, "streaming");

        var pendingTask = TaskFromResponse(
            await SendMessageAsync(client, card, BuildRequest(PendingRequestText, true), true).ConfigureAwait(false),
            "pending unary");
        AssertState(pendingTask.Status.State, TaskState.Working, "pending unary");
        AssertText(TaskText(pendingTask), expectedPendingText, "pending unary");

        var canceledTask = await CancelTaskAsync(client, card, new CancelTaskRequest
        {
            Id = pendingTask.Id,
        }).ConfigureAwait(false);
        AssertState(canceledTask.Status.State, TaskState.Canceled, "cancel_task");
        AssertText(TaskText(canceledTask), expectedCancelText, "cancel_task");

        var fetchedCanceledTask = await GetTaskAsync(client, card, new GetTaskRequest
        {
            Id = pendingTask.Id,
        }).ConfigureAwait(false);
        AssertState(fetchedCanceledTask.Status.State, TaskState.Canceled, "get_task after cancel");
        AssertText(TaskText(fetchedCanceledTask), expectedCancelText, "get_task after cancel");

        if (options.RelaxedErrorChecks)
        {
            await ExpectFailureAsync(() => GetTaskAsync(client, card, new GetTaskRequest { Id = Guid.NewGuid().ToString("N") }), "get missing task").ConfigureAwait(false);
            await ExpectFailureAsync(() => CancelTaskAsync(client, card, new CancelTaskRequest { Id = completedTask.Id }), "cancel completed task").ConfigureAwait(false);
        }
        else
        {
            await ExpectA2AErrorAsync(() => GetTaskAsync(client, card, new GetTaskRequest { Id = Guid.NewGuid().ToString("N") }), A2AErrorCode.TaskNotFound, "get missing task").ConfigureAwait(false);
            await ExpectA2AErrorAsync(() => CancelTaskAsync(client, card, new CancelTaskRequest { Id = completedTask.Id }), A2AErrorCode.TaskNotCancelable, "cancel completed task").ConfigureAwait(false);
        }

        if (options.ExpectPushUnsupported)
        {
            var pushConfig = new PushNotificationConfig
            {
                Id = "interop-config",
                Url = "https://example.invalid/webhook",
            };

            if (options.RelaxedErrorChecks)
            {
                await ExpectFailureAsync(() => CreateTaskPushNotificationConfigAsync(client, card, new CreateTaskPushNotificationConfigRequest
                {
                    TaskId = completedTask.Id,
                    ConfigId = "interop-config",
                    Config = pushConfig,
                }), "create_push_config").ConfigureAwait(false);
                await ExpectFailureAsync(() => GetTaskPushNotificationConfigAsync(client, card, new GetTaskPushNotificationConfigRequest
                {
                    TaskId = completedTask.Id,
                    Id = "interop-config",
                }), "get_push_config").ConfigureAwait(false);
                await ExpectFailureAsync(() => ListTaskPushNotificationConfigAsync(client, card, new ListTaskPushNotificationConfigRequest
                {
                    TaskId = completedTask.Id,
                }), "list_push_configs").ConfigureAwait(false);
                await ExpectFailureAsync(() => DeleteTaskPushNotificationConfigAsync(client, card, new DeleteTaskPushNotificationConfigRequest
                {
                    TaskId = completedTask.Id,
                    Id = "interop-config",
                }), "delete_push_config").ConfigureAwait(false);
            }
            else
            {
                await ExpectA2AErrorAsync(() => CreateTaskPushNotificationConfigAsync(client, card, new CreateTaskPushNotificationConfigRequest
                {
                    TaskId = completedTask.Id,
                    ConfigId = "interop-config",
                    Config = pushConfig,
                }), options.ExpectedPushErrorCode, "create_push_config").ConfigureAwait(false);
                await ExpectA2AErrorAsync(() => GetTaskPushNotificationConfigAsync(client, card, new GetTaskPushNotificationConfigRequest
                {
                    TaskId = completedTask.Id,
                    Id = "interop-config",
                }), options.ExpectedPushErrorCode, "get_push_config").ConfigureAwait(false);
                await ExpectA2AErrorAsync(() => ListTaskPushNotificationConfigAsync(client, card, new ListTaskPushNotificationConfigRequest
                {
                    TaskId = completedTask.Id,
                }), options.ExpectedPushErrorCode, "list_push_configs").ConfigureAwait(false);
                await ExpectA2AErrorAsync(() => DeleteTaskPushNotificationConfigAsync(client, card, new DeleteTaskPushNotificationConfigRequest
                {
                    TaskId = completedTask.Id,
                    Id = "interop-config",
                }), options.ExpectedPushErrorCode, "delete_push_config").ConfigureAwait(false);
            }
        }
        else if (options.ExpectPushSupported)
        {
            var pushConfig = new PushNotificationConfig
            {
                Id = "interop-config",
                Url = "https://example.invalid/webhook",
                Token = "interop-token",
                Authentication = new AuthenticationInfo
                {
                    Scheme = "Bearer",
                    Credentials = "interop-credential",
                },
            };

            var createdConfig = await CreateTaskPushNotificationConfigAsync(client, card, new CreateTaskPushNotificationConfigRequest
            {
                TaskId = completedTask.Id,
                ConfigId = "interop-config",
                Config = pushConfig,
            }).ConfigureAwait(false);
            AssertPushConfig(createdConfig, completedTask.Id, pushConfig, "create_push_config");

            var fetchedConfig = await GetTaskPushNotificationConfigAsync(client, card, new GetTaskPushNotificationConfigRequest
            {
                TaskId = completedTask.Id,
                Id = "interop-config",
            }).ConfigureAwait(false);
            AssertPushConfig(fetchedConfig, completedTask.Id, pushConfig, "get_push_config");

            var listedConfigs = await ListTaskPushNotificationConfigAsync(client, card, new ListTaskPushNotificationConfigRequest
            {
                TaskId = completedTask.Id,
            }).ConfigureAwait(false);
            if (listedConfigs.Count != 1)
            {
                throw new InvalidOperationException($"unexpected list_push_configs result count: got {listedConfigs.Count}, want 1");
            }
            AssertPushConfig(listedConfigs[0], completedTask.Id, pushConfig, "list_push_configs");

            await DeleteTaskPushNotificationConfigAsync(client, card, new DeleteTaskPushNotificationConfigRequest
            {
                TaskId = completedTask.Id,
                Id = "interop-config",
            }).ConfigureAwait(false);

            var listedAfterDelete = await ListTaskPushNotificationConfigAsync(client, card, new ListTaskPushNotificationConfigRequest
            {
                TaskId = completedTask.Id,
            }).ConfigureAwait(false);
            if (listedAfterDelete.Count > 0)
            {
                throw new InvalidOperationException($"expected list_push_configs after delete to be empty, got {listedAfterDelete.Count}");
            }
        }

        var protocol = card.SupportedInterfaces.FirstOrDefault()?.ProtocolBinding ?? "unknown";
        Console.WriteLine($"validated {options.ServerPrefix} {protocol} lifecycle against {options.CardUrl}");
    }

    private static SendMessageRequest BuildRequest(string text, bool returnImmediately)
    {
        return new SendMessageRequest
        {
            Message = new Message
            {
                Role = Role.User,
                MessageId = Guid.NewGuid().ToString("N"),
                Parts =
                [
                    Part.FromText(text),
                    Part.FromData(JsonSerializer.SerializeToElement(new { kind = RequestDataKind, scope = RequestDataScope })),
                ],
                Metadata = new Dictionary<string, JsonElement>
                {
                    [RequestMetadataKey] = JsonSerializer.SerializeToElement(RequestMetadataValue),
                },
            },
            Configuration = returnImmediately
                ? new SendMessageConfiguration { Blocking = false }
                : null,
        };
    }

    private static async Task<SendMessageResponse> SendMessageAsync(
        IA2AClient? client,
        AgentCard card,
        SendMessageRequest request,
        bool returnImmediately)
    {
        if (!UsesJsonRpcCompat(card))
        {
            return await PostRestAsync<JsonNode, SendMessageResponse>(card, "/message:send", BuildRestSendMessagePayload(request)).ConfigureAwait(false);
        }

        if (!returnImmediately)
        {
            return await client!.SendMessageAsync(request).ConfigureAwait(false);
        }

        var payload = new
        {
            message = request.Message,
            configuration = new
            {
                returnImmediately = true,
            },
        };

        return await SendJsonRpcAsync<SendMessageResponse>(card, "SendMessage", payload).ConfigureAwait(false);
    }

    private static JsonNode BuildRestSendMessagePayload(SendMessageRequest request)
    {
        var payload = JsonSerializer.SerializeToNode(request, A2AJsonUtilities.DefaultOptions) as JsonObject
            ?? throw new InvalidOperationException("failed to serialize REST send message request");

        RemoveNullProperties(payload);

        if (payload["configuration"] is JsonObject configuration
            && configuration.TryGetPropertyValue("blocking", out var blockingNode)
            && blockingNode is not null)
        {
            configuration["returnImmediately"] = !blockingNode.GetValue<bool>();
            configuration.Remove("blocking");
        }

        return payload;
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

    private static async Task<CompatibleTaskPushNotificationConfig> CreateTaskPushNotificationConfigAsync(
        IA2AClient? client,
        AgentCard card,
        CreateTaskPushNotificationConfigRequest request)
    {
        if (!UsesJsonRpcCompat(card))
        {
            return ToCompatibleTaskPushNotificationConfig(
                await PostRestAsync<PushNotificationConfig, TaskPushNotificationConfig>(
                    card,
                    $"/tasks/{Uri.EscapeDataString(request.TaskId)}/pushNotificationConfigs",
                    request.Config).ConfigureAwait(false));
        }

        return await SendJsonRpcAsync<CompatibleTaskPushNotificationConfig>(card, "CreateTaskPushNotificationConfig", request).ConfigureAwait(false);
    }

    private static async Task<CompatibleTaskPushNotificationConfig> GetTaskPushNotificationConfigAsync(
        IA2AClient? client,
        AgentCard card,
        GetTaskPushNotificationConfigRequest request)
    {
        if (!UsesJsonRpcCompat(card))
        {
            return ToCompatibleTaskPushNotificationConfig(
                await GetRestAsync<TaskPushNotificationConfig>(
                    card,
                    $"/tasks/{Uri.EscapeDataString(request.TaskId)}/pushNotificationConfigs/{Uri.EscapeDataString(request.Id)}").ConfigureAwait(false));
        }

        return await SendJsonRpcAsync<CompatibleTaskPushNotificationConfig>(card, "GetTaskPushNotificationConfig", request).ConfigureAwait(false);
    }

    private static async Task<List<CompatibleTaskPushNotificationConfig>> ListTaskPushNotificationConfigAsync(
        IA2AClient? client,
        AgentCard card,
        ListTaskPushNotificationConfigRequest request)
    {
        if (!UsesJsonRpcCompat(card))
        {
            var restResponse = await GetRestAsync<ListTaskPushNotificationConfigResponse>(
                card,
                BuildListPushConfigsPath(request)).ConfigureAwait(false);
            return (restResponse.Configs ?? []).Select(ToCompatibleTaskPushNotificationConfig).ToList();
        }

        var rpcResponse = await SendJsonRpcAsync<JsonElement>(
            card,
            "ListTaskPushNotificationConfigs",
            request).ConfigureAwait(false);

        return rpcResponse.ValueKind switch
        {
            JsonValueKind.Array => JsonSerializer.Deserialize<List<CompatibleTaskPushNotificationConfig>>(
                rpcResponse.GetRawText(),
                A2AJsonUtilities.DefaultOptions) ?? [],
            JsonValueKind.Object => JsonSerializer.Deserialize<CompatibleListTaskPushNotificationConfigResponse>(
                rpcResponse.GetRawText(),
                A2AJsonUtilities.DefaultOptions)?.Configs ?? [],
            _ => throw new InvalidOperationException("unexpected list push config JSON-RPC result shape"),
        };
    }

    private static async Task DeleteTaskPushNotificationConfigAsync(
        IA2AClient? client,
        AgentCard card,
        DeleteTaskPushNotificationConfigRequest request)
    {
        if (!UsesJsonRpcCompat(card))
        {
            await SendRestAsync(
                HttpMethod.Delete,
                card,
                $"/tasks/{Uri.EscapeDataString(request.TaskId)}/pushNotificationConfigs/{Uri.EscapeDataString(request.Id)}").ConfigureAwait(false);
            return;
        }

        await SendJsonRpcWithoutResultAsync(card, "DeleteTaskPushNotificationConfig", request).ConfigureAwait(false);
    }

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

        throw new InvalidOperationException($"agent card did not advertise a supported interface: {agentInterface.ProtocolBinding}");
    }

    private static CompatibleTaskPushNotificationConfig ToCompatibleTaskPushNotificationConfig(TaskPushNotificationConfig config)
    {
        return new CompatibleTaskPushNotificationConfig
        {
            TaskId = config.TaskId,
            Config = config.PushNotificationConfig,
        };
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

    private static IAsyncEnumerable<StreamResponse> SendStreamingMessageAsync(IA2AClient? client, AgentCard card, SendMessageRequest request)
    {
        if (UsesJsonRpcCompat(card))
        {
            return client!.SendStreamingMessageAsync(request);
        }

        return PostRestStreamingAsync(card, "/message:stream", request);
    }

    private static string BuildGetTaskPath(GetTaskRequest request)
    {
        return $"/tasks/{Uri.EscapeDataString(request.Id)}{BuildQueryString(("historyLength", request.HistoryLength?.ToString(System.Globalization.CultureInfo.InvariantCulture)))}";
    }

    private static string BuildListTasksPath(ListTasksRequest request)
    {
        return "/tasks" + BuildQueryString(
            ("contextId", request.ContextId),
            ("status", request.Status?.ToString()),
            ("pageSize", request.PageSize?.ToString(System.Globalization.CultureInfo.InvariantCulture)),
            ("pageToken", request.PageToken),
            ("historyLength", request.HistoryLength?.ToString(System.Globalization.CultureInfo.InvariantCulture)));
    }

    private static string BuildListPushConfigsPath(ListTaskPushNotificationConfigRequest request)
    {
        return $"/tasks/{Uri.EscapeDataString(request.TaskId)}/pushNotificationConfigs" + BuildQueryString(
            ("pageSize", request.PageSize?.ToString(System.Globalization.CultureInfo.InvariantCulture)),
            ("pageToken", request.PageToken));
    }

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
        SendMessageRequest body,
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
            var result = ReadRestPayload<StreamResponse>(sseItem.Data, path);
            yield return result;
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

    private static bool UsesJsonRpcCompat(AgentCard card)
    {
        return string.Equals(GetPrimaryInterface(card).ProtocolBinding, "JSONRPC", StringComparison.OrdinalIgnoreCase);
    }

    private static AgentInterface GetPrimaryInterface(AgentCard card)
    {
        return card.SupportedInterfaces.FirstOrDefault()
            ?? throw new InvalidOperationException("agent card did not advertise any interfaces");
    }

    private static AgentInterface GetJsonRpcInterface(AgentCard card)
    {
        var jsonRpcInterface = card.SupportedInterfaces.FirstOrDefault(candidate =>
            string.Equals(candidate.ProtocolBinding, "JSONRPC", StringComparison.OrdinalIgnoreCase));

        if (jsonRpcInterface is null)
        {
            throw new InvalidOperationException("agent card did not advertise a JSON-RPC interface");
        }

        return jsonRpcInterface;
    }

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
        var rpcResponse = await JsonSerializer.DeserializeAsync<JsonRpcResponse>(
            stream,
            A2AJsonUtilities.DefaultOptions).ConfigureAwait(false)
            ?? throw new InvalidOperationException("failed to deserialize JSON-RPC response");

        if (rpcResponse.Error is not null)
        {
            throw new A2AException(rpcResponse.Error.Message, (A2AErrorCode)rpcResponse.Error.Code);
        }

        if (rpcResponse.Result is null)
        {
            throw new InvalidOperationException($"failed to deserialize JSON-RPC result for {method}: null result payload");
        }

        var rawResult = rpcResponse.Result.ToJsonString();

        var result = JsonSerializer.Deserialize<TResult>(
            rawResult,
            A2AJsonUtilities.DefaultOptions);

        return result ?? throw new InvalidOperationException($"failed to deserialize JSON-RPC result for {method}: {rawResult}");
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
        var rpcResponse = await JsonSerializer.DeserializeAsync<JsonRpcResponse>(
            stream,
            A2AJsonUtilities.DefaultOptions).ConfigureAwait(false)
            ?? throw new InvalidOperationException($"failed to deserialize JSON-RPC response for {method}");

        if (rpcResponse.Error is not null)
        {
            throw new A2AException(rpcResponse.Error.Message, (A2AErrorCode)rpcResponse.Error.Code);
        }
    }

    private static string ExpectedResponseText(string serverPrefix, string requestText) =>
        $"{serverPrefix} server received: {requestText}";

    private static string ExpectedCancelText(string serverPrefix) =>
        $"{serverPrefix} server canceled task";

    private static AgentTask TaskFromResponse(SendMessageResponse response, string kind)
    {
        return response.Task ?? throw new InvalidOperationException($"unexpected {kind} response type: Message");
    }

    private static string TaskText(AgentTask task)
    {
        return task.Status.Message is null
            ? throw new InvalidOperationException("task response contained no message")
            : FirstText(task.Status.Message);
    }

    private static string FirstText(Message message)
    {
        var part = message.Parts.FirstOrDefault(value => value.Text is not null);
        return part?.Text ?? throw new InvalidOperationException("message contained no text parts");
    }

    private static void AssertText(string actual, string expected, string kind)
    {
        if (!string.Equals(actual, expected, StringComparison.Ordinal))
        {
            throw new InvalidOperationException($"unexpected {kind} response text: got '{actual}', want '{expected}'");
        }
    }

    private static void AssertState(TaskState actual, TaskState expected, string kind)
    {
        if (actual != expected)
        {
            throw new InvalidOperationException($"unexpected {kind} task state: got {actual}, want {expected}");
        }
    }

    private static void AssertTaskHistory(AgentTask task, string expectedText, string kind)
    {
        if (task.History is null || task.History.Count != 1)
        {
            throw new InvalidOperationException($"{kind} task did not include a single history entry");
        }

        var message = task.History[0];
        AssertText(FirstText(message), expectedText, kind);

        if (message.Parts.Count != 2)
        {
            throw new InvalidOperationException($"{kind} task history had {message.Parts.Count} parts, want 2");
        }

        var dataPart = message.Parts[1].Data ?? throw new InvalidOperationException($"{kind} task history second part was not a structured data part");
        var kindValue = dataPart.GetProperty("kind").GetString();
        var scopeValue = dataPart.GetProperty("scope").GetString();
        if (!string.Equals(kindValue, RequestDataKind, StringComparison.Ordinal) || !string.Equals(scopeValue, RequestDataScope, StringComparison.Ordinal))
        {
            throw new InvalidOperationException($"{kind} task history data part mismatch: got kind={kindValue} scope={scopeValue}");
        }

        if (message.Metadata is null || !message.Metadata.TryGetValue(RequestMetadataKey, out var metadataValue))
        {
            throw new InvalidOperationException($"{kind} task history metadata was missing {RequestMetadataKey}");
        }

        if (!string.Equals(metadataValue.GetString(), RequestMetadataValue, StringComparison.Ordinal))
        {
            throw new InvalidOperationException($"{kind} task history metadata mismatch: got '{metadataValue.GetString()}', want '{RequestMetadataValue}'");
        }
    }

    private static async Task<string> ReadStreamingTextAsync(IAsyncEnumerable<StreamResponse> stream)
    {
        await foreach (var response in stream.ConfigureAwait(false))
        {
            var text = StreamResponseText(response);
            if (text is not null)
            {
                return text;
            }
        }

        throw new InvalidOperationException("stream completed without a terminal response event");
    }

    private static string? StreamResponseText(StreamResponse response)
    {
        return response.PayloadCase switch
        {
            StreamResponseCase.Message => FirstText(response.Message!),
            StreamResponseCase.Task => TaskText(response.Task!),
            StreamResponseCase.StatusUpdate when response.StatusUpdate?.Status.Message is not null => FirstText(response.StatusUpdate.Status.Message),
            _ => null,
        };
    }

    private static void AssertPushConfig(CompatibleTaskPushNotificationConfig actual, string taskId, PushNotificationConfig expected, string kind)
    {
        if (!string.Equals(actual.TaskId, taskId, StringComparison.Ordinal))
        {
            throw new InvalidOperationException($"unexpected {kind} task id: got '{actual.TaskId}', want '{taskId}'");
        }

        if (!string.Equals(actual.Config.Id, expected.Id, StringComparison.Ordinal)
            || !string.Equals(actual.Config.Url, expected.Url, StringComparison.Ordinal)
            || !string.Equals(actual.Config.Token, expected.Token, StringComparison.Ordinal)
            || !string.Equals(actual.Config.Authentication?.Scheme, expected.Authentication?.Scheme, StringComparison.Ordinal)
            || !string.Equals(actual.Config.Authentication?.Credentials, expected.Authentication?.Credentials, StringComparison.Ordinal))
        {
            throw new InvalidOperationException($"unexpected {kind} push config result");
        }
    }

    private static async Task ExpectFailureAsync(Func<Task> action, string kind)
    {
        try
        {
            await action().ConfigureAwait(false);
        }
        catch
        {
            return;
        }

        throw new InvalidOperationException($"expected {kind} to fail, but it succeeded");
    }

    private static async Task ExpectFailureAsync<T>(Func<Task<T>> action, string kind)
    {
        try
        {
            _ = await action().ConfigureAwait(false);
        }
        catch
        {
            return;
        }

        throw new InvalidOperationException($"expected {kind} to fail, but it succeeded");
    }

    private static async Task ExpectA2AErrorAsync(Func<Task> action, A2AErrorCode expectedCode, string kind)
    {
        try
        {
            await action().ConfigureAwait(false);
        }
        catch (A2AException error) when (error.ErrorCode == expectedCode)
        {
            return;
        }
        catch (A2AException error)
        {
            throw new InvalidOperationException($"unexpected {kind} error code: got {(int)error.ErrorCode}, want {(int)expectedCode} ({error.Message})");
        }

        throw new InvalidOperationException($"expected {kind} to fail with code {(int)expectedCode}, but it succeeded");
    }

    private static async Task ExpectA2AErrorAsync<T>(Func<Task<T>> action, A2AErrorCode expectedCode, string kind)
    {
        try
        {
            _ = await action().ConfigureAwait(false);
        }
        catch (A2AException error) when (error.ErrorCode == expectedCode)
        {
            return;
        }
        catch (A2AException error)
        {
            throw new InvalidOperationException($"unexpected {kind} error code: got {(int)error.ErrorCode}, want {(int)expectedCode} ({error.Message})");
        }

        throw new InvalidOperationException($"expected {kind} to fail with code {(int)expectedCode}, but it succeeded");
    }
}

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

    public PushNotificationConfig Config { get; set; } = new();
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

internal sealed class ProbeOptions
{
    public required string CardUrl { get; init; }

    public required string ServerPrefix { get; init; }

    public bool ExpectPushSupported { get; init; }

    public bool ExpectPushUnsupported { get; init; }

    public bool RelaxedErrorChecks { get; init; }

    public A2AErrorCode ExpectedPushErrorCode { get; init; } = A2AErrorCode.PushNotificationNotSupported;

    public static ProbeOptions Parse(string[] args)
    {
        string? cardUrl = null;
        string? serverPrefix = null;
        var expectPushSupported = false;
        var expectPushUnsupported = false;
        var relaxedErrorChecks = false;
        var expectedPushErrorCode = A2AErrorCode.PushNotificationNotSupported;

        for (var index = 0; index < args.Length; index++)
        {
            switch (args[index])
            {
                case "--card-url":
                    index++;
                    cardUrl = args[index];
                    break;
                case "--server-prefix":
                    index++;
                    serverPrefix = args[index];
                    break;
                case "--expect-push-supported":
                    expectPushSupported = true;
                    break;
                case "--expect-push-unsupported":
                    expectPushUnsupported = true;
                    break;
                case "--relaxed-error-checks":
                    relaxedErrorChecks = true;
                    break;
                case "--expected-push-error-code":
                    index++;
                    expectedPushErrorCode = (A2AErrorCode)int.Parse(args[index], System.Globalization.CultureInfo.InvariantCulture);
                    break;
                default:
                    throw new ArgumentException($"unknown argument: {args[index]}");
            }
        }

        if (expectPushSupported && expectPushUnsupported)
        {
            throw new ArgumentException("--expect-push-supported and --expect-push-unsupported are mutually exclusive");
        }

        return new ProbeOptions
        {
            CardUrl = cardUrl ?? throw new ArgumentException("missing --card-url"),
            ServerPrefix = serverPrefix ?? throw new ArgumentException("missing --server-prefix"),
            ExpectPushSupported = expectPushSupported,
            ExpectPushUnsupported = expectPushUnsupported,
            RelaxedErrorChecks = relaxedErrorChecks,
            ExpectedPushErrorCode = expectedPushErrorCode,
        };
    }
}