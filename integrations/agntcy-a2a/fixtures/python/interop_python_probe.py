# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

"""Thin A2A probe adapter.

Executes a single A2A operation and writes the raw JSON result to stdout.
On A2A error, writes {"code":…,"message":"…"} to stderr and exits 1.
No assertions or orchestration live here — all test logic is in the
Go/Ginkgo spec tree.
"""

from __future__ import annotations

import argparse
import asyncio
import json
import sys
from typing import Any
from urllib.parse import urlparse

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


# ── error helpers ────────────────────────────────────────────────────────────

def fail(code: int, message: str) -> None:
    print(json.dumps({'code': code, 'message': message}), file=sys.stderr)
    sys.exit(1)


def extract_a2a_error(exc: Exception) -> tuple[int, str]:
    """Return (code, message) from any A2A / gRPC / HTTP exception."""
    if isinstance(exc, grpc.aio.AioRpcError):
        code_value = exc.code().value
        num = code_value[0] if isinstance(code_value, tuple) else int(code_value)
        return num, exc.details() or str(exc)
    code = getattr(exc, 'code', None)
    if isinstance(code, int):
        return code, str(exc)
    status_code = getattr(exc, 'status_code', None)
    if isinstance(status_code, int):
        return status_code, str(exc)
    return -32000, str(exc)


# ── serialization helpers ────────────────────────────────────────────────────

def to_json_str(msg: Any) -> str:
    return json.dumps(MessageToDict(msg, preserving_proto_field_name=False,
                                    including_default_value_fields=False))


def event_envelope(event_type: str, msg: Any) -> str:
    data = MessageToDict(msg, preserving_proto_field_name=False,
                         including_default_value_fields=False)
    return json.dumps({'type': event_type, 'data': data})


def push_config_to_dict(cfg: a2a_pb2.TaskPushNotificationConfig) -> dict:
    """Serialize push config to the JSON format expected by the Go probe client."""
    obj: dict[str, Any] = {'taskId': cfg.task_id, 'url': cfg.url}
    if cfg.id:
        obj['id'] = cfg.id
    if cfg.token:
        obj['token'] = cfg.token
    if cfg.HasField('authentication'):
        auth: dict[str, str] = {'scheme': cfg.authentication.scheme}
        if cfg.authentication.credentials:
            auth['credentials'] = cfg.authentication.credentials
        obj['authentication'] = auth
    return obj


# ── client creation ──────────────────────────────────────────────────────────

def grpc_target(address: str) -> str:
    if '://' not in address:
        return address
    return urlparse(address).netloc or address


async def make_client(card_url: str, *, streaming: bool) -> Client:
    config = ClientConfig(
        streaming=streaming,
        grpc_channel_factory=lambda addr: grpc.aio.insecure_channel(grpc_target(addr)),
        supported_protocol_bindings=[
            TransportProtocol.GRPC,
            TransportProtocol.JSONRPC,
            TransportProtocol.HTTP_JSON,
        ],
        httpx_client=httpx.AsyncClient(timeout=5.0),
    )
    return await create_client(card_url, client_config=config)


# ── push-config transport dispatch ───────────────────────────────────────────

class JsonRpcCompatError(RuntimeError):
    def __init__(self, code: int, message: str) -> None:
        super().__init__(message)
        self.code = code


class RestCompatError(RuntimeError):
    def __init__(self, status_code: int, message: str) -> None:
        super().__init__(message)
        self.status_code = status_code


def resolve_card_url(base_url: str) -> str:
    if base_url.rstrip('/').endswith('/.well-known/agent-card.json'):
        return base_url
    return f"{base_url.rstrip('/')}/.well-known/agent-card.json"


async def load_agent_card(card_url: str) -> a2a_pb2.AgentCard:
    async with httpx.AsyncClient(timeout=5.0) as http:
        r = await http.get(resolve_card_url(card_url))
        r.raise_for_status()
    return ParseDict(r.json(), a2a_pb2.AgentCard())


def primary_interface(card: a2a_pb2.AgentCard) -> a2a_pb2.AgentInterface:
    if not card.supported_interfaces:
        raise RuntimeError('agent card did not expose any supported interfaces')
    for binding in (TransportProtocol.JSONRPC.value, TransportProtocol.HTTP_JSON.value):
        for iface in card.supported_interfaces:
            if iface.protocol_binding == binding:
                return iface
    return card.supported_interfaces[0]


async def _jsonrpc(url: str, method: str, params: dict,
                   allow_no_result: bool = False) -> Any:
    from uuid import uuid4
    body = {'jsonrpc': '2.0', 'id': str(uuid4()), 'method': method, 'params': params}
    async with httpx.AsyncClient(timeout=5.0) as http:
        r = await http.post(url, json=body, headers={VERSION_HEADER: PROTOCOL_VERSION_1_0})
        r.raise_for_status()
        resp = r.json()
    if 'error' in resp and isinstance(resp['error'], dict):
        err = resp['error']
        raise JsonRpcCompatError(err.get('code', 0), str(err.get('message', 'error')))
    if 'result' not in resp:
        if allow_no_result:
            return None
        raise RuntimeError(f'JSON-RPC {method}: missing result')
    return resp['result']


async def _rest(method: str, url: str, json_body: dict | None = None) -> Any:
    async with httpx.AsyncClient(timeout=5.0) as http:
        r = await http.request(method, url, json=json_body,
                               headers={VERSION_HEADER: PROTOCOL_VERSION_1_0})
    if r.status_code >= 400:
        raise RestCompatError(r.status_code, r.text)
    if r.status_code == 204 or not r.content:
        return None
    return r.json()


async def push_create(
    card: a2a_pb2.AgentCard,
    client: Client,
    config: a2a_pb2.TaskPushNotificationConfig,
) -> a2a_pb2.TaskPushNotificationConfig:
    iface = primary_interface(card)
    if iface.protocol_binding == TransportProtocol.GRPC.value:
        return await client.create_task_push_notification_config(config)
    payload = MessageToDict(config, preserving_proto_field_name=False)
    if iface.protocol_binding == TransportProtocol.JSONRPC.value:
        result = await _jsonrpc(iface.url, 'CreateTaskPushNotificationConfig', payload)
        return ParseDict(result, a2a_pb2.TaskPushNotificationConfig())
    payload.pop('taskId', None)
    result = await _rest('POST',
                         f"{iface.url.rstrip('/')}/tasks/{config.task_id}/pushNotificationConfigs",
                         payload)
    return ParseDict(result, a2a_pb2.TaskPushNotificationConfig())


async def push_get(
    card: a2a_pb2.AgentCard,
    client: Client,
    req: a2a_pb2.GetTaskPushNotificationConfigRequest,
) -> a2a_pb2.TaskPushNotificationConfig:
    iface = primary_interface(card)
    if iface.protocol_binding == TransportProtocol.GRPC.value:
        return await client.get_task_push_notification_config(req)
    payload = MessageToDict(req, preserving_proto_field_name=False)
    if iface.protocol_binding == TransportProtocol.JSONRPC.value:
        result = await _jsonrpc(iface.url, 'GetTaskPushNotificationConfig', payload)
        return ParseDict(result, a2a_pb2.TaskPushNotificationConfig())
    result = await _rest(
        'GET',
        f"{iface.url.rstrip('/')}/tasks/{req.task_id}/pushNotificationConfigs/{req.id}",
    )
    return ParseDict(result, a2a_pb2.TaskPushNotificationConfig())


async def push_list(
    card: a2a_pb2.AgentCard,
    client: Client,
    req: a2a_pb2.ListTaskPushNotificationConfigsRequest,
) -> list[a2a_pb2.TaskPushNotificationConfig]:
    iface = primary_interface(card)
    if iface.protocol_binding == TransportProtocol.GRPC.value:
        resp = await client.list_task_push_notification_configs(req)
        return list(resp.configs)
    payload = MessageToDict(req, preserving_proto_field_name=False)
    if iface.protocol_binding == TransportProtocol.JSONRPC.value:
        result = await _jsonrpc(iface.url, 'ListTaskPushNotificationConfigs', payload)
        entries = result.get('configs', []) if isinstance(result, dict) else []
        return [ParseDict(e, a2a_pb2.TaskPushNotificationConfig()) for e in entries]
    result = await _rest(
        'GET',
        f"{iface.url.rstrip('/')}/tasks/{req.task_id}/pushNotificationConfigs",
    )
    entries = result.get('configs', []) if isinstance(result, dict) else []
    return [ParseDict(e, a2a_pb2.TaskPushNotificationConfig()) for e in entries]


async def push_delete(
    card: a2a_pb2.AgentCard,
    client: Client,
    req: a2a_pb2.DeleteTaskPushNotificationConfigRequest,
) -> None:
    iface = primary_interface(card)
    if iface.protocol_binding == TransportProtocol.GRPC.value:
        await client.delete_task_push_notification_config(req)
        return
    payload = MessageToDict(req, preserving_proto_field_name=False)
    if iface.protocol_binding == TransportProtocol.JSONRPC.value:
        await _jsonrpc(iface.url, 'DeleteTaskPushNotificationConfig', payload,
                       allow_no_result=True)
        return
    await _rest(
        'DELETE',
        f"{iface.url.rstrip('/')}/tasks/{req.task_id}/pushNotificationConfigs/{req.id}",
    )


# ── subcommand handlers ──────────────────────────────────────────────────────

async def run(args: argparse.Namespace) -> None:
    card_url: str = args.card_url
    subcommand: str = args.subcommand

    if subcommand == 'send-message':
        req = ParseDict(json.loads(args.message_json), a2a_pb2.SendMessageRequest())
        client = await make_client(card_url, streaming=False)
        try:
            responses = [r async for r in client.send_message(req)]
        finally:
            await client.close()
        if not responses:
            fail(-32000, 'send-message: no response received')
        resp = responses[0]
        if resp.HasField('task'):
            print(event_envelope('task', resp.task))
        elif resp.HasField('message'):
            print(event_envelope('message', resp.message))
        else:
            fail(-32000, 'send-message: unexpected response variant')
        return

    if subcommand == 'send-streaming-message':
        req = ParseDict(json.loads(args.message_json), a2a_pb2.SendMessageRequest())
        client = await make_client(card_url, streaming=True)
        try:
            responses = [r async for r in client.send_message(req)]
        finally:
            await client.close()
        for resp in responses:
            if resp.HasField('task'):
                print(event_envelope('task', resp.task))
            elif resp.HasField('message'):
                print(event_envelope('message', resp.message))
            elif resp.HasField('status_update'):
                print(event_envelope('task-status-update', resp.status_update))
            elif resp.HasField('artifact_update'):
                print(event_envelope('task-artifact-update', resp.artifact_update))
        return

    if subcommand == 'get-task':
        client = await make_client(card_url, streaming=False)
        try:
            task = await client.get_task(a2a_pb2.GetTaskRequest(id=args.task_id))
        finally:
            await client.close()
        print(to_json_str(task))
        return

    if subcommand == 'cancel-task':
        client = await make_client(card_url, streaming=False)
        try:
            task = await client.cancel_task(a2a_pb2.CancelTaskRequest(id=args.task_id))
        finally:
            await client.close()
        print(to_json_str(task))
        return

    if subcommand == 'list-tasks':
        client = await make_client(card_url, streaming=False)
        try:
            resp = await client.list_tasks(
                a2a_pb2.ListTasksRequest(context_id=args.context_id)
            )
        finally:
            await client.close()
        print(to_json_str(resp))
        return

    if subcommand == 'create-push-config':
        config = ParseDict(json.loads(args.config_json),
                           a2a_pb2.TaskPushNotificationConfig())
        card = await load_agent_card(card_url)
        client = await make_client(card_url, streaming=False)
        try:
            result = await push_create(card, client, config)
        finally:
            await client.close()
        print(json.dumps(push_config_to_dict(result)))
        return

    if subcommand == 'get-push-config':
        req = a2a_pb2.GetTaskPushNotificationConfigRequest(
            task_id=args.task_id, id=args.config_id
        )
        card = await load_agent_card(card_url)
        client = await make_client(card_url, streaming=False)
        try:
            result = await push_get(card, client, req)
        finally:
            await client.close()
        print(json.dumps(push_config_to_dict(result)))
        return

    if subcommand == 'list-push-configs':
        req = a2a_pb2.ListTaskPushNotificationConfigsRequest(task_id=args.task_id)
        card = await load_agent_card(card_url)
        client = await make_client(card_url, streaming=False)
        try:
            configs = await push_list(card, client, req)
        finally:
            await client.close()
        print(json.dumps({'configs': [push_config_to_dict(c) for c in configs]}))
        return

    if subcommand == 'delete-push-config':
        req = a2a_pb2.DeleteTaskPushNotificationConfigRequest(
            task_id=args.task_id, id=args.config_id
        )
        card = await load_agent_card(card_url)
        client = await make_client(card_url, streaming=False)
        try:
            await push_delete(card, client, req)
        finally:
            await client.close()
        return

    if subcommand == 'get-extended-card':
        client = await make_client(card_url, streaming=False)
        try:
            card = await client.get_extended_agent_card(
                a2a_pb2.GetExtendedAgentCardRequest()
            )
        finally:
            await client.close()
        print(to_json_str(card))
        return

    fail(-32000, f'unknown subcommand: {subcommand}')


def make_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description='A2A Python probe adapter.')
    parser.add_argument('--card-url', required=True)
    sub = parser.add_subparsers(dest='subcommand', required=True)

    m = sub.add_parser('send-message')
    m.add_argument('--message-json', required=True)

    sm = sub.add_parser('send-streaming-message')
    sm.add_argument('--message-json', required=True)

    gt = sub.add_parser('get-task')
    gt.add_argument('--task-id', required=True)

    ct = sub.add_parser('cancel-task')
    ct.add_argument('--task-id', required=True)

    lt = sub.add_parser('list-tasks')
    lt.add_argument('--context-id', required=True)

    cpc = sub.add_parser('create-push-config')
    cpc.add_argument('--config-json', required=True)

    gpc = sub.add_parser('get-push-config')
    gpc.add_argument('--task-id', required=True)
    gpc.add_argument('--config-id', required=True)

    lpc = sub.add_parser('list-push-configs')
    lpc.add_argument('--task-id', required=True)

    dpc = sub.add_parser('delete-push-config')
    dpc.add_argument('--task-id', required=True)
    dpc.add_argument('--config-id', required=True)

    sub.add_parser('get-extended-card')

    return parser


def main() -> None:
    args = make_parser().parse_args()

    async def _run() -> None:
        try:
            await run(args)
        except SystemExit:
            raise
        except Exception as exc:  # noqa: BLE001
            code, message = extract_a2a_error(exc)
            fail(code, message)

    asyncio.run(_run())


if __name__ == '__main__':
    main()
