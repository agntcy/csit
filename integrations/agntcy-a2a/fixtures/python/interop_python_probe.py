# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

"""A2A Python v1.0 interoperability probe.

This probe is the Python-client analogue of the existing Rust and .NET probes.
It exercises one shared behavior slice at a time so the Go Ginkgo layer can
reuse the same behavior matrix across SDKs.
"""

from __future__ import annotations

import argparse
import asyncio
import json
from collections.abc import AsyncIterator
from typing import Any
from urllib.parse import urlparse
from uuid import uuid4

import grpc
import httpx

from google.protobuf.json_format import MessageToDict, ParseDict

from a2a.client import Client, ClientConfig, create_client
from a2a.types import a2a_pb2
from a2a.utils.constants import (
    PROTOCOL_VERSION_1_0,
    TransportProtocol,
    VERSION_HEADER,
)


REQUEST_TEXT = 'ping'
PENDING_REQUEST_TEXT = 'pending'
MESSAGE_ONLY_REQUEST_TEXT = 'message-only'
TASK_FAILURE_REQUEST_TEXT = 'task-failure'
MULTI_TURN_START_REQUEST_TEXT = 'multi-turn start'
MULTI_TURN_CONTINUE_REQUEST_TEXT = 'multi-turn continue'
STREAMING_REQUEST_TEXT = 'streaming'
LONG_RUNNING_REQUEST_TEXT = 'long-running'
DATA_TYPES_REQUEST_TEXT = 'data-types'
REQUEST_DATA_KIND = 'structured'
REQUEST_DATA_SCOPE = 'interop'
REQUEST_METADATA_KEY = 'csit'
REQUEST_METADATA_VALUE = 'multipart'
EXTENDED_CARD_SCHEME_ID = 'bearer_token'


def build_text_part(text: str) -> a2a_pb2.Part:
    return a2a_pb2.Part(text=text)


def build_data_part(data: dict[str, object]) -> a2a_pb2.Part:
    part = a2a_pb2.Part()
    ParseDict({'data': data}, part)
    return part


def new_request(
    text: str,
    return_immediately: bool,
    task_id: str = '',
    context_id: str = '',
) -> a2a_pb2.SendMessageRequest:
    message = a2a_pb2.Message(
        role=a2a_pb2.Role.ROLE_USER,
        message_id=str(uuid4()),
    )
    if task_id:
        message.task_id = task_id
    if context_id:
        message.context_id = context_id
    message.parts.append(build_text_part(text))
    message.parts.append(build_data_part({'kind': REQUEST_DATA_KIND, 'scope': REQUEST_DATA_SCOPE}))
    message.metadata.update({REQUEST_METADATA_KEY: REQUEST_METADATA_VALUE})

    return a2a_pb2.SendMessageRequest(
        message=message,
        configuration=a2a_pb2.SendMessageConfiguration(
            return_immediately=return_immediately,
        ),
    )


def first_message_text(message: a2a_pb2.Message) -> str:
    for part in message.parts:
        if part.text:
            return part.text
    raise AssertionError('message did not include a text part')


def task_status_text(task: a2a_pb2.Task) -> str:
    if not task.HasField('status') or not task.status.HasField('message'):
        raise AssertionError('task status did not include a message')
    return first_message_text(task.status.message)


def value_to_dict(part: a2a_pb2.Part) -> dict[str, object]:
    return MessageToDict(part.data)


def metadata_to_dict(message: a2a_pb2.Message) -> dict[str, object]:
    return MessageToDict(message.metadata)


def assert_message_payload(message: a2a_pb2.Message, expected_text: str, kind: str) -> None:
    if first_message_text(message) != expected_text:
        raise AssertionError(f'{kind}: unexpected text payload')
    if len(message.parts) != 2:
        raise AssertionError(f'{kind}: expected 2 message parts, got {len(message.parts)}')
    data_part = value_to_dict(message.parts[1])
    if data_part.get('kind') != REQUEST_DATA_KIND or data_part.get('scope') != REQUEST_DATA_SCOPE:
        raise AssertionError(f'{kind}: unexpected structured data payload {data_part!r}')
    metadata = metadata_to_dict(message)
    if metadata.get(REQUEST_METADATA_KEY) != REQUEST_METADATA_VALUE:
        raise AssertionError(f'{kind}: unexpected metadata payload {metadata!r}')


def assert_task_history_payloads(task: a2a_pb2.Task, expected_texts: list[str], kind: str) -> None:
    if len(task.history) != len(expected_texts):
        raise AssertionError(f'{kind}: expected {len(expected_texts)} history entries, got {len(task.history)}')
    for index, expected_text in enumerate(expected_texts):
        assert_message_payload(task.history[index], expected_text, kind)


def assert_task_push_config(
    config: a2a_pb2.TaskPushNotificationConfig,
    task_id: str,
    expected: a2a_pb2.TaskPushNotificationConfig,
    kind: str,
) -> None:
    if config.task_id != task_id:
        raise AssertionError(f'{kind}: unexpected task_id {config.task_id!r}')
    if config.id != expected.id or config.url != expected.url or config.token != expected.token:
        raise AssertionError(f'{kind}: unexpected push config payload')
    if config.authentication.scheme != expected.authentication.scheme:
        raise AssertionError(f'{kind}: unexpected auth scheme {config.authentication.scheme!r}')
    if config.authentication.credentials != expected.authentication.credentials:
        raise AssertionError(f'{kind}: unexpected auth credentials')


def assert_data_types_task(task: a2a_pb2.Task) -> None:
    if len(task.artifacts) != 1:
        raise AssertionError(f'data-types: expected 1 artifact, got {len(task.artifacts)}')
    artifact = task.artifacts[0]
    if len(artifact.parts) != 3:
        raise AssertionError(f'data-types: expected 3 artifact parts, got {len(artifact.parts)}')
    if artifact.parts[0].text != 'structured summary':
        raise AssertionError('data-types: unexpected artifact summary text')
    data_part = value_to_dict(artifact.parts[1])
    if data_part != {'kind': 'report', 'items': 2.0}:
        raise AssertionError(f'data-types: unexpected structured artifact payload {data_part!r}')
    if artifact.parts[2].url != 'https://example.invalid/diagram.svg':
        raise AssertionError('data-types: unexpected artifact URL')
    if artifact.parts[2].media_type != 'image/svg+xml':
        raise AssertionError('data-types: unexpected artifact media type')


def assert_extended_card_metadata(card: a2a_pb2.AgentCard) -> None:
    if not card.capabilities.extended_agent_card:
        raise AssertionError('extended-card: capability flag was not set')
    if '(extended)' not in card.description:
        raise AssertionError('extended-card: description did not include extended marker')
    if card.security_schemes:
        if EXTENDED_CARD_SCHEME_ID not in card.security_schemes:
            raise AssertionError('extended-card: bearer_token security scheme missing')
        scheme = card.security_schemes[EXTENDED_CARD_SCHEME_ID]
        if scheme.http_auth_security_scheme.scheme != 'Bearer':
            raise AssertionError('extended-card: unexpected HTTP auth scheme')
    skill_ids = {skill.id for skill in card.skills}
    expected_skills = {
        'message-only',
        'task-lifecycle',
        'task-failure',
        'task-cancel',
        'multi-turn',
        'streaming',
        'long-running',
        'data-types',
    }
    if skill_ids != expected_skills:
        raise AssertionError(f'extended-card: unexpected skills {sorted(skill_ids)!r}')


class JsonRpcCompatError(RuntimeError):
    def __init__(self, code: int, message: str, data: Any = None) -> None:
        super().__init__(message)
        self.code = code
        self.data = data


class RestCompatError(RuntimeError):
    def __init__(self, status_code: int, message: str, data: Any = None) -> None:
        super().__init__(message)
        self.status_code = status_code
        self.data = data


def resolve_card_url(base_url: str) -> str:
    if base_url.rstrip('/').endswith('/.well-known/agent-card.json'):
        return base_url
    return f"{base_url.rstrip('/')}/.well-known/agent-card.json"


def grpc_target(address: str) -> str:
    if '://' not in address:
        return address

    parsed = urlparse(address)
    return parsed.netloc or address


async def load_agent_card(card_url: str) -> a2a_pb2.AgentCard:
    async with httpx.AsyncClient(timeout=5.0) as client:
        response = await client.get(resolve_card_url(card_url))
        response.raise_for_status()
        payload = response.json()
    return ParseDict(payload, a2a_pb2.AgentCard())


def primary_interface(card: a2a_pb2.AgentCard) -> a2a_pb2.AgentInterface:
    if not card.supported_interfaces:
        raise AssertionError('agent card did not expose any supported interfaces')

    for protocol_binding in (TransportProtocol.JSONRPC.value, TransportProtocol.HTTP_JSON.value):
        for interface in card.supported_interfaces:
            if interface.protocol_binding == protocol_binding:
                return interface

    return card.supported_interfaces[0]


def build_create_push_config_params(
    config: a2a_pb2.TaskPushNotificationConfig,
) -> dict[str, Any]:
    return MessageToDict(config, preserving_proto_field_name=False)


def build_rest_create_push_config_payload(
    config: a2a_pb2.TaskPushNotificationConfig,
) -> dict[str, Any]:
    flat_payload = MessageToDict(config, preserving_proto_field_name=False)
    flat_payload.pop('taskId', None)
    return flat_payload


def parse_push_config_entry(payload: dict[str, Any]) -> a2a_pb2.TaskPushNotificationConfig:
    return ParseDict(
        payload,
        a2a_pb2.TaskPushNotificationConfig(),
    )


def parse_push_config_list_result(result: Any) -> list[a2a_pb2.TaskPushNotificationConfig]:
    if not isinstance(result, dict):
        raise AssertionError(f'list push configs: unexpected result {result!r}')
    entries = result.get('configs', [])
    if not isinstance(entries, list):
        raise AssertionError(f'list push configs: unexpected configs field {result!r}')

    configs: list[a2a_pb2.TaskPushNotificationConfig] = []
    for entry in entries:
        if not isinstance(entry, dict):
            raise AssertionError(f'list push configs: unexpected JSON-RPC entry {entry!r}')
        configs.append(parse_push_config_entry(entry))
    return configs


async def send_jsonrpc_request(
    rpc_url: str,
    method: str,
    params: dict[str, Any],
    *,
    allow_missing_result: bool = False,
) -> Any:
    payload = {
        'jsonrpc': '2.0',
        'id': str(uuid4()),
        'method': method,
        'params': params,
    }
    async with httpx.AsyncClient(timeout=5.0) as client:
        response = await client.post(
            rpc_url,
            json=payload,
            headers={VERSION_HEADER: PROTOCOL_VERSION_1_0},
        )
        response.raise_for_status()
        response_payload = response.json()

    error = response_payload.get('error')
    if isinstance(error, dict):
        code = error.get('code')
        if not isinstance(code, int):
            code = 0
        raise JsonRpcCompatError(
            code,
            str(error.get('message', 'JSON-RPC request failed')),
            error.get('data'),
        )

    if 'result' not in response_payload:
        if allow_missing_result:
            return None
        raise AssertionError(f'JSON-RPC response for {method} did not include a result')

    return response_payload['result']


async def send_rest_request(
    method: str,
    url: str,
    *,
    json_body: dict[str, Any] | None = None,
) -> Any:
    async with httpx.AsyncClient(timeout=5.0) as client:
        response = await client.request(
            method,
            url,
            json=json_body,
            headers={VERSION_HEADER: PROTOCOL_VERSION_1_0},
        )

    if response.status_code >= 400:
        try:
            payload = response.json()
        except json.JSONDecodeError:
            payload = None
        raise RestCompatError(response.status_code, response.text, payload)

    if response.status_code == 204 or not response.content:
        return None
    return response.json()


async def create_task_push_config_jsonrpc(
    rpc_url: str,
    config: a2a_pb2.TaskPushNotificationConfig,
) -> a2a_pb2.TaskPushNotificationConfig:
    result = await send_jsonrpc_request(
        rpc_url,
        'CreateTaskPushNotificationConfig',
        build_create_push_config_params(config),
    )
    if not isinstance(result, dict):
        raise AssertionError(f'create push config: unexpected JSON-RPC result {result!r}')
    return parse_push_config_entry(result)


async def get_task_push_config_jsonrpc(
    rpc_url: str,
    request: a2a_pb2.GetTaskPushNotificationConfigRequest,
) -> a2a_pb2.TaskPushNotificationConfig:
    result = await send_jsonrpc_request(
        rpc_url,
        'GetTaskPushNotificationConfig',
        MessageToDict(request, preserving_proto_field_name=False),
    )
    if not isinstance(result, dict):
        raise AssertionError(f'get push config: unexpected JSON-RPC result {result!r}')
    return parse_push_config_entry(result)


async def list_task_push_configs_jsonrpc(
    rpc_url: str,
    request: a2a_pb2.ListTaskPushNotificationConfigsRequest,
) -> list[a2a_pb2.TaskPushNotificationConfig]:
    result = await send_jsonrpc_request(
        rpc_url,
        'ListTaskPushNotificationConfigs',
        MessageToDict(request, preserving_proto_field_name=False),
    )
    return parse_push_config_list_result(result)


async def delete_task_push_config_jsonrpc(
    rpc_url: str,
    request: a2a_pb2.DeleteTaskPushNotificationConfigRequest,
) -> None:
    await send_jsonrpc_request(
        rpc_url,
        'DeleteTaskPushNotificationConfig',
        MessageToDict(request, preserving_proto_field_name=False),
        allow_missing_result=True,
    )


async def create_task_push_config_rest(
    base_url: str,
    config: a2a_pb2.TaskPushNotificationConfig,
) -> a2a_pb2.TaskPushNotificationConfig:
    result = await send_rest_request(
        'POST',
        f"{base_url.rstrip('/')}/tasks/{config.task_id}/pushNotificationConfigs",
        json_body=build_rest_create_push_config_payload(config),
    )
    if not isinstance(result, dict):
        raise AssertionError(f'create push config: unexpected REST result {result!r}')
    return parse_push_config_entry(result)


async def get_task_push_config_rest(
    base_url: str,
    request: a2a_pb2.GetTaskPushNotificationConfigRequest,
) -> a2a_pb2.TaskPushNotificationConfig:
    result = await send_rest_request(
        'GET',
        f"{base_url.rstrip('/')}/tasks/{request.task_id}/pushNotificationConfigs/{request.id}",
    )
    if not isinstance(result, dict):
        raise AssertionError(f'get push config: unexpected REST result {result!r}')
    return parse_push_config_entry(result)


async def list_task_push_configs_rest(
    base_url: str,
    request: a2a_pb2.ListTaskPushNotificationConfigsRequest,
) -> list[a2a_pb2.TaskPushNotificationConfig]:
    result = await send_rest_request(
        'GET',
        f"{base_url.rstrip('/')}/tasks/{request.task_id}/pushNotificationConfigs",
    )
    return parse_push_config_list_result(result)


async def delete_task_push_config_rest(
    base_url: str,
    request: a2a_pb2.DeleteTaskPushNotificationConfigRequest,
) -> None:
    await send_rest_request(
        'DELETE',
        f"{base_url.rstrip('/')}/tasks/{request.task_id}/pushNotificationConfigs/{request.id}",
    )


async def create_probe_client(card_url: str, streaming: bool) -> Client:
    config = ClientConfig(
        streaming=streaming,
        grpc_channel_factory=lambda address: grpc.aio.insecure_channel(grpc_target(address)),
        supported_protocol_bindings=[
            TransportProtocol.GRPC,
            TransportProtocol.JSONRPC,
            TransportProtocol.HTTP_JSON,
        ],
        httpx_client=httpx.AsyncClient(timeout=5.0),
    )
    return await create_client(card_url, client_config=config)


async def create_task_push_config_grpc(
    client: Client,
    config: a2a_pb2.TaskPushNotificationConfig,
) -> a2a_pb2.TaskPushNotificationConfig:
    return await client.create_task_push_notification_config(config)


async def get_task_push_config_grpc(
    client: Client,
    request: a2a_pb2.GetTaskPushNotificationConfigRequest,
) -> a2a_pb2.TaskPushNotificationConfig:
    return await client.get_task_push_notification_config(request)


async def list_task_push_configs_grpc(
    client: Client,
    request: a2a_pb2.ListTaskPushNotificationConfigsRequest,
) -> list[a2a_pb2.TaskPushNotificationConfig]:
    response = await client.list_task_push_notification_configs(request)
    return list(response.configs)


async def delete_task_push_config_grpc(
    client: Client,
    request: a2a_pb2.DeleteTaskPushNotificationConfigRequest,
) -> None:
    await client.delete_task_push_notification_config(request)


async def collect_responses(
    client: Client,
    request: a2a_pb2.SendMessageRequest,
) -> list[a2a_pb2.StreamResponse]:
    responses: list[a2a_pb2.StreamResponse] = []
    async for response in client.send_message(request):
        responses.append(response)
    return responses


def expect_task_response(response: a2a_pb2.StreamResponse, kind: str) -> a2a_pb2.Task:
    if not response.HasField('task'):
        raise AssertionError(f'{kind}: expected task response')
    return response.task


def expect_message_response(response: a2a_pb2.StreamResponse, kind: str) -> a2a_pb2.Message:
    if not response.HasField('message'):
        raise AssertionError(f'{kind}: expected message response')
    return response.message


async def expect_error(
    awaitable: asyncio.Future | asyncio.Task | object,
    expected_substring: str | tuple[str, ...],
) -> None:
    expected_substrings = (
        (expected_substring,)
        if isinstance(expected_substring, str)
        else expected_substring
    )
    try:
        await awaitable  # type: ignore[arg-type]
    except Exception as exc:  # noqa: BLE001
        message = str(exc).lower()
        if not any(candidate.lower() in message for candidate in expected_substrings):
            raise AssertionError(
                f'expected error containing one of {expected_substrings!r}, got {exc!r}'
            ) from exc
        return
    raise AssertionError(f'expected error containing one of {expected_substrings!r}')


async def wait_for_task_state(client: Client, task_id: str, expected_state: int) -> a2a_pb2.Task:
    deadline = asyncio.get_running_loop().time() + 2.0
    while asyncio.get_running_loop().time() < deadline:
        task = await client.get_task(a2a_pb2.GetTaskRequest(id=task_id))
        if task.status.state == expected_state:
            return task
        await asyncio.sleep(0.05)
    raise AssertionError(f'timed out waiting for task {task_id} to reach state {expected_state}')


async def assert_task_streaming(card_url: str, server_prefix: str) -> None:
    unary_client = await create_probe_client(card_url, streaming=False)
    try:
        unary_responses = await collect_responses(unary_client, new_request(REQUEST_TEXT, False))
        if len(unary_responses) != 1:
            raise AssertionError(f'unary: expected exactly one response, got {len(unary_responses)}')
        unary_task = expect_task_response(unary_responses[0], 'unary')
        if task_status_text(unary_task) != f'{server_prefix} server received: {REQUEST_TEXT}':
            raise AssertionError('unary: unexpected response text')
    finally:
        await unary_client.close()

    streaming_client = await create_probe_client(card_url, streaming=True)
    try:
        streaming_responses = await collect_responses(streaming_client, new_request(REQUEST_TEXT, False))
        texts: list[str] = []
        for response in streaming_responses:
            if response.HasField('task'):
                texts.append(task_status_text(response.task))
            elif response.HasField('message'):
                texts.append(first_message_text(response.message))
            elif response.HasField('status_update') and response.status_update.status.HasField('message'):
                texts.append(first_message_text(response.status_update.status.message))
        if f'{server_prefix} server received: {REQUEST_TEXT}' not in texts:
            raise AssertionError(f'streaming: expected response text in {texts!r}')
    finally:
        await streaming_client.close()


async def assert_task_lifecycle(card_url: str, server_prefix: str) -> None:
    client = await create_probe_client(card_url, streaming=False)
    try:
        completed_task = expect_task_response(
            (await collect_responses(client, new_request(REQUEST_TEXT, False)))[0],
            'completed task',
        )
        if completed_task.status.state != a2a_pb2.TaskState.TASK_STATE_COMPLETED:
            raise AssertionError('completed task: expected completed state')
        if task_status_text(completed_task) != f'{server_prefix} server received: {REQUEST_TEXT}':
            raise AssertionError('completed task: unexpected status text')
        assert_task_history_payloads(completed_task, [REQUEST_TEXT], 'completed task history')

        fetched_task = await client.get_task(a2a_pb2.GetTaskRequest(id=completed_task.id))
        if fetched_task.status.state != a2a_pb2.TaskState.TASK_STATE_COMPLETED:
            raise AssertionError('fetched task: expected completed state')
        if task_status_text(fetched_task) != f'{server_prefix} server received: {REQUEST_TEXT}':
            raise AssertionError('fetched task: unexpected status text')
        assert_task_history_payloads(fetched_task, [REQUEST_TEXT], 'fetched task history')

        listed_tasks = await client.list_tasks(
            a2a_pb2.ListTasksRequest(context_id=completed_task.context_id)
        )
        if not any(task.id == completed_task.id for task in listed_tasks.tasks):
            raise AssertionError('list tasks: completed task id was not returned')

        pending_task = expect_task_response(
            (await collect_responses(client, new_request(PENDING_REQUEST_TEXT, True)))[0],
            'pending task',
        )
        if pending_task.status.state != a2a_pb2.TaskState.TASK_STATE_WORKING:
            raise AssertionError('pending task: expected working state')
        if task_status_text(pending_task) != f'{server_prefix} server received: {PENDING_REQUEST_TEXT}':
            raise AssertionError('pending task: unexpected status text')

        canceled_task = await client.cancel_task(a2a_pb2.CancelTaskRequest(id=pending_task.id))
        if canceled_task.status.state != a2a_pb2.TaskState.TASK_STATE_CANCELED:
            raise AssertionError('canceled task: expected canceled state')
        if task_status_text(canceled_task) != f'{server_prefix} server canceled task':
            raise AssertionError('canceled task: unexpected status text')

        fetched_canceled_task = await client.get_task(a2a_pb2.GetTaskRequest(id=pending_task.id))
        if fetched_canceled_task.status.state != a2a_pb2.TaskState.TASK_STATE_CANCELED:
            raise AssertionError('fetched canceled task: expected canceled state')
        if task_status_text(fetched_canceled_task) != f'{server_prefix} server canceled task':
            raise AssertionError('fetched canceled task: unexpected status text')

        await expect_error(
            client.get_task(a2a_pb2.GetTaskRequest(id=str(uuid4()))),
            'task',
        )
        await expect_error(
            client.cancel_task(a2a_pb2.CancelTaskRequest(id=completed_task.id)),
            ('cancel', 'terminal state', 'not cancelable'),
        )
    finally:
        await client.close()


async def assert_push_config(
    card_url: str,
    server_prefix: str,
    expect_push_supported: bool,
    expected_push_error_code: int,
    relaxed_error_checks: bool,
) -> None:
    client = await create_probe_client(card_url, streaming=False)
    try:
        card = getattr(client, '_card', None)
        if not isinstance(card, a2a_pb2.AgentCard):
            card = await load_agent_card(card_url)
        agent_interface = primary_interface(card)
        uses_grpc = agent_interface.protocol_binding == TransportProtocol.GRPC.value
        uses_jsonrpc = agent_interface.protocol_binding == TransportProtocol.JSONRPC.value

        completed_task = expect_task_response(
            (await collect_responses(client, new_request(REQUEST_TEXT, False)))[0],
            'push-config task',
        )
        if completed_task.status.state != a2a_pb2.TaskState.TASK_STATE_COMPLETED:
            raise AssertionError('push-config task: expected completed state')
        if task_status_text(completed_task) != f'{server_prefix} server received: {REQUEST_TEXT}':
            raise AssertionError('push-config task: unexpected status text')
        assert_task_history_payloads(completed_task, [REQUEST_TEXT], 'push-config task history')

        push_config = a2a_pb2.TaskPushNotificationConfig(
            task_id=completed_task.id,
            id='interop-config',
            url='https://example.invalid/webhook',
            token='interop-token',
            authentication=a2a_pb2.AuthenticationInfo(
                scheme='Bearer',
                credentials='interop-credential',
            ),
        )

        if not expect_push_supported:
            try:
                if uses_grpc:
                    await create_task_push_config_grpc(client, push_config)
                elif uses_jsonrpc:
                    await create_task_push_config_jsonrpc(agent_interface.url, push_config)
                else:
                    await create_task_push_config_rest(
                        agent_interface.url,
                        push_config,
                    )
            except Exception as exc:  # noqa: BLE001
                if not relaxed_error_checks and expected_push_error_code != 0:
                    if isinstance(exc, JsonRpcCompatError):
                        error_text = f'{exc.code} {exc}'
                    elif isinstance(exc, RestCompatError):
                        error_text = f'{exc.status_code} {exc}'
                    else:
                        error_text = str(exc)
                    if str(expected_push_error_code) not in error_text:
                        normalized_error_text = error_text.upper()
                        accepts_rest_unsupported = (
                            isinstance(exc, RestCompatError)
                            and expected_push_error_code == -32003
                            and (
                                '501' in error_text
                                or 'UNIMPLEMENTED' in normalized_error_text
                                or 'PUSH_NOTIFICATION_NOT_SUPPORTED' in normalized_error_text
                            )
                        )
                        accepts_grpc_unsupported = (
                            uses_grpc
                            and expected_push_error_code == -32003
                            and 'UNIMPLEMENTED' in normalized_error_text
                        )
                        if accepts_rest_unsupported or accepts_grpc_unsupported:
                            return
                        raise AssertionError(
                            f'expected push error code {expected_push_error_code}, got {error_text!r}'
                        ) from exc
                return
            raise AssertionError('expected push configuration to be unsupported')

        if uses_grpc:
            created_config = await create_task_push_config_grpc(client, push_config)
        elif uses_jsonrpc:
            created_config = await create_task_push_config_jsonrpc(agent_interface.url, push_config)
        else:
            created_config = await create_task_push_config_rest(
                agent_interface.url,
                push_config,
            )
        assert_task_push_config(created_config, completed_task.id, push_config, 'created push config')

        get_request = a2a_pb2.GetTaskPushNotificationConfigRequest(
            task_id=completed_task.id,
            id=push_config.id,
        )
        if uses_grpc:
            fetched_config = await get_task_push_config_grpc(client, get_request)
        elif uses_jsonrpc:
            fetched_config = await get_task_push_config_jsonrpc(agent_interface.url, get_request)
        else:
            fetched_config = await get_task_push_config_rest(agent_interface.url, get_request)
        assert_task_push_config(fetched_config, completed_task.id, push_config, 'fetched push config')

        list_request = a2a_pb2.ListTaskPushNotificationConfigsRequest(task_id=completed_task.id)
        if uses_grpc:
            listed_configs = await list_task_push_configs_grpc(client, list_request)
        elif uses_jsonrpc:
            listed_configs = await list_task_push_configs_jsonrpc(agent_interface.url, list_request)
        else:
            listed_configs = await list_task_push_configs_rest(agent_interface.url, list_request)

        if len(listed_configs) != 1:
            raise AssertionError('list push configs: expected one config')
        assert_task_push_config(listed_configs[0], completed_task.id, push_config, 'listed push config')

        delete_request = a2a_pb2.DeleteTaskPushNotificationConfigRequest(
            task_id=completed_task.id,
            id=push_config.id,
        )
        if uses_grpc:
            await delete_task_push_config_grpc(client, delete_request)
            listed_configs = await list_task_push_configs_grpc(client, list_request)
        elif uses_jsonrpc:
            await delete_task_push_config_jsonrpc(agent_interface.url, delete_request)
            listed_configs = await list_task_push_configs_jsonrpc(agent_interface.url, list_request)
        else:
            await delete_task_push_config_rest(agent_interface.url, delete_request)
            listed_configs = await list_task_push_configs_rest(agent_interface.url, list_request)

        if listed_configs:
            raise AssertionError('list push configs after delete: expected no configs')
    finally:
        await client.close()


async def assert_scenario_parity(card_url: str, server_prefix: str) -> None:
    unary_client = await create_probe_client(card_url, streaming=False)
    streaming_client = await create_probe_client(card_url, streaming=True)
    try:
        message_only = expect_message_response(
            (await collect_responses(unary_client, new_request(MESSAGE_ONLY_REQUEST_TEXT, False)))[0],
            'message-only',
        )
        if first_message_text(message_only) != f'{server_prefix} server message-only response':
            raise AssertionError('message-only: unexpected response text')

        failed_task = expect_task_response(
            (await collect_responses(unary_client, new_request(TASK_FAILURE_REQUEST_TEXT, False)))[0],
            'task-failure',
        )
        if failed_task.status.state != a2a_pb2.TaskState.TASK_STATE_FAILED:
            raise AssertionError('task-failure: expected failed state')
        if task_status_text(failed_task) != f'{server_prefix} server failed task':
            raise AssertionError('task-failure: unexpected status text')

        input_required_task = expect_task_response(
            (await collect_responses(unary_client, new_request(MULTI_TURN_START_REQUEST_TEXT, False)))[0],
            'multi-turn start',
        )
        if input_required_task.status.state != a2a_pb2.TaskState.TASK_STATE_INPUT_REQUIRED:
            raise AssertionError('multi-turn start: expected input-required state')
        if task_status_text(input_required_task) != f'{server_prefix} server needs more input':
            raise AssertionError('multi-turn start: unexpected status text')
        assert_task_history_payloads(input_required_task, [MULTI_TURN_START_REQUEST_TEXT], 'multi-turn start')

        continued_task = expect_task_response(
            (
                await collect_responses(
                    unary_client,
                    new_request(
                        MULTI_TURN_CONTINUE_REQUEST_TEXT,
                        False,
                        task_id=input_required_task.id,
                        context_id=input_required_task.context_id,
                    ),
                )
            )[0],
            'multi-turn continue',
        )
        if continued_task.status.state != a2a_pb2.TaskState.TASK_STATE_COMPLETED:
            raise AssertionError('multi-turn continue: expected completed state')
        if task_status_text(continued_task) != f'{server_prefix} server multi-turn completed':
            raise AssertionError('multi-turn continue: unexpected status text')
        assert_task_history_payloads(
            continued_task,
            [MULTI_TURN_START_REQUEST_TEXT, MULTI_TURN_CONTINUE_REQUEST_TEXT],
            'multi-turn continue',
        )

        streaming_responses = await collect_responses(
            streaming_client,
            new_request(STREAMING_REQUEST_TEXT, False),
        )
        saw_streaming_start = False
        saw_streaming_complete = False
        saw_append = False
        streaming_chunks: list[str] = []
        for response in streaming_responses:
            if response.HasField('task'):
                saw_streaming_start = True
                if response.task.status.state != a2a_pb2.TaskState.TASK_STATE_WORKING:
                    raise AssertionError('streaming: expected initial working task')
                if task_status_text(response.task) != f'{server_prefix} server streaming started':
                    raise AssertionError('streaming: unexpected start text')
            elif response.HasField('artifact_update'):
                artifact = response.artifact_update.artifact
                if not artifact.parts or not artifact.parts[0].text:
                    raise AssertionError('streaming: artifact update missing text part')
                streaming_chunks.append(artifact.parts[0].text)
                if response.artifact_update.append:
                    saw_append = True
            elif response.HasField('status_update'):
                saw_streaming_complete = True
                if response.status_update.status.state != a2a_pb2.TaskState.TASK_STATE_COMPLETED:
                    raise AssertionError('streaming: expected completed status update')
                if first_message_text(response.status_update.status.message) != f'{server_prefix} server streaming complete':
                    raise AssertionError('streaming: unexpected completion text')
            else:
                raise AssertionError('streaming: unexpected response variant')
        if not saw_streaming_start or not saw_streaming_complete or not saw_append:
            raise AssertionError('streaming: missing expected task/artifact/status events')
        if streaming_chunks != ['streaming chunk 1', 'streaming chunk 2']:
            raise AssertionError(f'streaming: unexpected chunks {streaming_chunks!r}')

        long_running_task = expect_task_response(
            (await collect_responses(unary_client, new_request(LONG_RUNNING_REQUEST_TEXT, True)))[0],
            'long-running',
        )
        if long_running_task.status.state == a2a_pb2.TaskState.TASK_STATE_WORKING:
            if task_status_text(long_running_task) != f'{server_prefix} server long-running started':
                raise AssertionError('long-running: unexpected start text')
            long_running_completed = await wait_for_task_state(
                unary_client,
                long_running_task.id,
                a2a_pb2.TaskState.TASK_STATE_COMPLETED,
            )
        elif long_running_task.status.state == a2a_pb2.TaskState.TASK_STATE_COMPLETED:
            long_running_completed = long_running_task
        else:
            raise AssertionError(
                f'long-running: unexpected initial state {long_running_task.status.state}'
            )
        if task_status_text(long_running_completed) != f'{server_prefix} server long-running complete':
            raise AssertionError('long-running: unexpected completion text')

        data_types_task = expect_task_response(
            (await collect_responses(unary_client, new_request(DATA_TYPES_REQUEST_TEXT, False)))[0],
            'data-types',
        )
        if data_types_task.status.state != a2a_pb2.TaskState.TASK_STATE_COMPLETED:
            raise AssertionError('data-types: expected completed state')
        if task_status_text(data_types_task) != f'{server_prefix} server data-types ready':
            raise AssertionError('data-types: unexpected status text')
        assert_data_types_task(data_types_task)

        extended_card = await unary_client.get_extended_agent_card(
            a2a_pb2.GetExtendedAgentCardRequest()
        )
        assert_extended_card_metadata(extended_card)
    finally:
        await unary_client.close()
        await streaming_client.close()


async def run_scenario(args: argparse.Namespace) -> None:
    if args.scenario == 'core':
        await assert_task_streaming(args.card_url, args.server_prefix)
        await assert_task_lifecycle(args.card_url, args.server_prefix)
        await assert_push_config(
            args.card_url,
            args.server_prefix,
            not args.expect_push_unsupported,
            args.expected_push_error_code,
            args.relaxed_error_checks,
        )
        return
    if args.scenario == 'task-streaming':
        await assert_task_streaming(args.card_url, args.server_prefix)
        return
    if args.scenario == 'task-lifecycle':
        await assert_task_lifecycle(args.card_url, args.server_prefix)
        return
    if args.scenario == 'push-config':
        await assert_push_config(
            args.card_url,
            args.server_prefix,
            not args.expect_push_unsupported,
            args.expected_push_error_code,
            args.relaxed_error_checks,
        )
        return
    if args.scenario == 'parity':
        await assert_scenario_parity(args.card_url, args.server_prefix)
        return
    raise AssertionError(f'unsupported scenario {args.scenario!r}')


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description='Run the Python A2A interop probe.')
    parser.add_argument('--card-url', required=True)
    parser.add_argument('--server-prefix', required=True)
    parser.add_argument(
        '--scenario',
        default='core',
        choices=('core', 'task-streaming', 'task-lifecycle', 'push-config', 'parity'),
    )
    parser.add_argument('--expect-push-supported', action='store_true')
    parser.add_argument('--expect-push-unsupported', action='store_true')
    parser.add_argument('--expected-push-error-code', type=int, default=0)
    parser.add_argument('--relaxed-error-checks', action='store_true')
    return parser.parse_args()


def main() -> None:
    args = parse_args()
    asyncio.run(run_scenario(args))


if __name__ == '__main__':
    main()