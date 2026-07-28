// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0
//
// CSIT fixture: A2A echo agent + probe over SLIMRPC (lf.a2a.v1) for cross-language interop.
// One entrypoint, two modes mirroring fixtures/{go,python,dotnet,java}:
//   node dist/main.js server --slim-endpoint <url> --identity <ns/grp/name> --secret <s>
//   node dist/main.js probe  --slim-endpoint <url> --local <id> --remote <id> --secret <s> --scenario <sc> --text <t>
//
// Consumes the PUBLISHED @agntcy/slim-a2a from npm (like the sibling fixtures consume their
// published packages). It connects to the EXTERNAL SLIM node at --slim-endpoint (no in-process
// broker) so the interop runs over the shared dataplane. NOTE: @agntcy/slim-a2a pins
// slim-bindings 2.0-alpha, so this fixture speaks the slim 2.0 wire — run it against a slim
// 2.0 node (the launcher/CI point node rows at ghcr.io/agntcy/slim:2.0.0-alpha.3).

import { randomUUID } from 'node:crypto';
import { AgentCard, CancelTaskRequest, SendMessageRequest, type Task } from '@a2a-js/sdk';
import type { Client } from '@a2a-js/sdk/client';
import { DefaultRequestHandler, InMemoryTaskStore } from '@a2a-js/sdk/server';
import { Server } from '@agntcy/slim-bindings';
import {
  createSlimClient,
  registerSlimA2AHandler,
  setupSlimClient,
  SRPCHandler,
} from '@agntcy/slim-a2a';
import { CsitEchoExecutor } from './executor.js';
import * as S from './scenarios.js';
import { emit, observeResult, observeStream, observeTask, type Observation } from './observation.js';

const READY_MARKER = 'CSIT_SLIM_SERVER_READY';
const DEFAULT_ENDPOINT = 'http://127.0.0.1:46357';
const DEFAULT_SECRET = 'my_shared_secret_for_testing_purposes_only';

async function main(): Promise<void> {
  const args = process.argv.slice(2);
  const mode = firstPositional(args);
  const endpoint = opt(args, '--slim-endpoint', envOr('SLIM_SERVER', DEFAULT_ENDPOINT));
  const secret = opt(args, '--secret', envOr('SLIM_SHARED_SECRET', DEFAULT_SECRET));

  switch (mode) {
    case 'server':
      await runServer(endpoint, secret, opt(args, '--identity', 'agntcy/a2a_csit_slim/server_node'));
      break;
    case 'probe':
      process.exit(
        await runProbe(
          endpoint,
          secret,
          opt(args, '--local', 'agntcy/a2a_csit_slim/client_node'),
          opt(args, '--remote', 'agntcy/a2a_csit_slim/server_node'),
          opt(args, '--scenario', S.ECHO),
          opt(args, '--text', 'Hello there!'),
        ),
      );
      break;
    default:
      process.stderr.write('usage: node dist/main.js [server|probe] [flags]\n');
      process.exit(2);
  }
}

// ---- server -----------------------------------------------------------------------

async function runServer(endpoint: string, secret: string, identity: string): Promise<void> {
  const [org, ns, name] = splitIdentity(identity);

  const agentCard = AgentCard.fromJSON({
    name: 'CSIT Echo Agent',
    description: 'Echoes messages over A2A on SLIMRPC for CSIT interop.',
    version: '1.0.0',
    capabilities: { streaming: true },
    supportedInterfaces: [
      { url: `${org}/${ns}/${name}`, protocolBinding: 'slimrpc', protocolVersion: '1.0' },
    ],
    defaultInputModes: ['text/plain'],
    defaultOutputModes: ['text/plain'],
    skills: [{ id: 'echo', name: 'Echo', description: 'Echoes the input text back.', tags: ['echo'] }],
  });

  const requestHandler = new DefaultRequestHandler(
    agentCard,
    new InMemoryTaskStore(),
    new CsitEchoExecutor(),
  );

  // Connect to the EXTERNAL SLIM node (no in-process broker); setupSlimClient subscribes localName.
  const { app, name: localName, connId } = await setupSlimClient(org, ns, name, {
    slimUrl: endpoint,
    secret,
  });

  const server = Server.newWithConnection(app, localName, connId);
  registerSlimA2AHandler(server, new SRPCHandler(agentCard, requestHandler));

  // Ready marker on stdout (the harness waits for this line), matching the other fixtures.
  console.log(READY_MARKER);
  await server.serveAsync();
}

// ---- probe ------------------------------------------------------------------------

async function runProbe(
  endpoint: string,
  secret: string,
  local: string,
  remote: string,
  scenario: string,
  text: string,
): Promise<number> {
  const req = S.request(scenario, text);
  const [lorg, lns, lname] = splitIdentity(local);

  const { app, connId } = await setupSlimClient(lorg, lns, lname, {
    slimUrl: endpoint,
    secret,
  });

  // A minimal card describing how to reach the remote agent over slimrpc.
  const card = AgentCard.fromJSON({
    name: 'CSIT remote',
    description: 'Remote CSIT echo agent.',
    version: '1.0.0',
    capabilities: { streaming: true },
    supportedInterfaces: [
      { url: trimSlashes(remote), protocolBinding: 'slimrpc', protocolVersion: '1.0' },
    ],
    defaultInputModes: ['text/plain'],
    defaultOutputModes: ['text/plain'],
    skills: [],
  });

  const client = await createSlimClient(app, connId, card);

  let obs: Observation;
  switch (req.mode) {
    case S.MODE_STREAMING:
      obs = await runStreaming(client, req.want);
      break;
    case S.MODE_CANCEL:
      obs = await runCancel(client, req.want);
      break;
    case S.MODE_MULTI_TURN:
      obs = await runMultiTurn(client);
      break;
    default:
      obs = await runUnary(client, req.want);
      break;
  }

  emit(obs);
  if (req.enforceEcho && !obs.text.includes(req.want)) {
    process.stderr.write(`response ${obs.text} does not contain sent text ${req.want}\n`);
    return 1;
  }
  return 0;
}

async function runUnary(client: Client, want: string): Promise<Observation> {
  return observeResult(await client.sendMessage(sendRequest(want)));
}

async function runStreaming(client: Client, want: string): Promise<Observation> {
  return observeStream(client.sendMessageStream(sendRequest(want)));
}

// Read the stream just far enough to learn the server-assigned task id, then cancel it and
// observe the terminal (canceled) task, mirroring the Go/Java probes.
async function runCancel(client: Client, want: string): Promise<Observation> {
  let taskId = '';
  for await (const response of client.sendMessageStream(sendRequest(want))) {
    const payload = response.payload;
    if (payload?.$case === 'task') {
      taskId = payload.value.id;
    } else if (payload?.$case === 'statusUpdate') {
      taskId = payload.value.taskId;
    } else if (payload?.$case === 'artifactUpdate') {
      taskId = payload.value.taskId;
    }
    if (taskId) {
      break;
    }
  }
  if (!taskId) {
    throw new Error('task-cancel scenario: no task id observed');
  }
  const canceled = await client.cancelTask(CancelTaskRequest.fromJSON({ id: taskId }));
  return observeTask(canceled);
}

// Drive a two-turn conversation on a single task: turn 1 reaches input-required (capturing the
// server-assigned task/context IDs); turn 2 references those IDs to continue the same task, which
// the server completes with the multi-turn artifact. Only the final observation is emitted.
async function runMultiTurn(client: Client): Promise<Observation> {
  const res1 = await client.sendMessage(sendRequest(S.SENTINEL_MULTI_TURN));
  if (!('id' in res1) || typeof res1.id !== 'string') {
    throw new Error('multi-turn start: expected a task');
  }
  const task1 = res1 as Task;
  const res2 = await client.sendMessage(
    sendRequest(S.SENTINEL_MULTI_TURN_CONTINUE, task1.id, task1.contextId),
  );
  return observeResult(res2);
}

function sendRequest(text: string, taskId?: string, contextId?: string): SendMessageRequest {
  const message: Record<string, unknown> = {
    messageId: randomUUID(),
    role: 'ROLE_USER',
    parts: [{ text }],
  };
  if (taskId) {
    message.taskId = taskId;
  }
  if (contextId) {
    message.contextId = contextId;
  }
  return SendMessageRequest.fromJSON({ message });
}

// ---- helpers ----------------------------------------------------------------------

function trimSlashes(s: string): string {
  return s.replace(/^\/+|\/+$/g, '');
}

function splitIdentity(s: string): [string, string, string] {
  const parts = trimSlashes(s).split('/');
  if (parts.length !== 3) {
    throw new Error(`identity must be ns/group/name, got ${s}`);
  }
  return [parts[0], parts[1], parts[2]];
}

function firstPositional(args: string[]): string {
  for (const a of args) {
    if (!a.startsWith('-')) {
      return a.toLowerCase();
    }
  }
  return 'server';
}

function opt(args: string[], name: string, def: string): string {
  for (let i = 0; i < args.length - 1; i++) {
    if (args[i] === name) {
      return args[i + 1];
    }
  }
  return def;
}

function envOr(key: string, def: string): string {
  const v = process.env[key];
  return v ? v : def;
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
