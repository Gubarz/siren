// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { cleanup, render, screen, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

const listSessionTasks = vi.fn();
const getTaskCallPayloads = vi.fn();

vi.mock('../../../api/sessionTasks.js', () => ({
  listSessionTasks: (...a) => listSessionTasks(...a),
  getTaskCallPayloads: (...a) => getTaskCallPayloads(...a),
}));

import SessionTasks from '../SessionTasks.svelte';
import {
  statusChip,
  shortMethod,
  parsePayload,
  normalizeKeys,
  inferKind,
  decodeTextPayload,
  formatDurationNs,
  humanizeKey,
  requestSummary,
} from '../sessionTaskRender.js';

afterEach(cleanup);
beforeEach(() => {
  listSessionTasks.mockReset();
  getTaskCallPayloads.mockReset();
});

const task = {
  id: 7,
  ts: 1757000000000,
  method: '/rpcpb.SliverRPC/Ls',
  status: 'OK',
  evidence: '',
  runId: '',
  stageId: '',
  chainRef: 'chain-1',
  requestBytes: 7,
  responseBytes: 42,
  error: '',
};

const FIXTURES = {
  processes: '{"Processes":[{"Pid":1,"Ppid":2,"Executable":"proc-a","Owner":"user-a"}]}',
  files: '{"Path":"/test/dir","Exists":true,"Files":[{"Name":"file-a.txt","IsDir":false,"Size":"42","ModTime":"1700000000","Mode":"-rw-r--r--"}]}',
  image: '{"Data":"cG5n"}',
  env: '{"Variables":[{"Key":"ENV_KEY","Value":"value-a"}]}',
  services: '{"Details":[{"Name":"svc-a","DisplayName":"svc-a display","Status":4,"StartupType":2}]}',
  netstat: '{"Entries":[{"LocalAddr":{"Ip":"127.0.0.1","Port":1},"RemoteAddr":{"Ip":"127.0.0.1","Port":2},"SkState":"LISTEN","Protocol":"tcp"}]}',
  text: '{"Stdout":"aGVsbG8=","Stderr":"","Response":{"Err":""}}',
  textErr: '{"Stdout":"aGVsbG8=","Response":{"Err":"boom"}}',
};

function payload(kind, json) {
  return { direction: 'response', ts: task.ts, preview: '', json, base64: '', kind };
}

function deferred() {
  let resolve;
  const promise = new Promise((r) => { resolve = r; });
  return { promise, resolve };
}

describe('sessionTaskRender', () => {
  it('maps statuses with evidence dominance', () => {
    expect(statusChip({ status: 'OK', evidence: 'verified' })).toEqual({ label: 'verified', class: 'text-success-500' });
    expect(statusChip({ status: 'OK', evidence: '' })).toEqual({ label: 'ok', class: 'text-success-500' });
    expect(statusChip({ status: 'NotFound', evidence: '' })).toEqual({ label: 'NotFound', class: 'text-danger-500' });
    expect(statusChip({ status: 'OK', evidence: 'failed' })).toEqual({ label: 'failed', class: 'text-danger-500' });
    expect(statusChip({ status: 'attempted', evidence: '' })).toEqual({ label: 'attempted', class: 'text-fg-muted' });
  });

  it('derives short method and parses payloads', () => {
    expect(shortMethod('/rpcpb.SliverRPC/Ls')).toBe('Ls');
    expect(parsePayload('{"a":1}')).toEqual({ a: 1 });
    expect(parsePayload('not json')).toBe(null);
  });

  it('normalizes protojson keys recursively', () => {
    expect(normalizeKeys({ Processes: [{ Pid: 1, Extra: { DeepKey: 'x' } }], Data: 'abc' })).toEqual({
      processes: [{ pid: 1, extra: { deepkey: 'x' } }],
      data: 'abc',
    });
    expect(normalizeKeys([{ A: 1 }, 'b'])).toEqual([{ a: 1 }, 'b']);
    expect(normalizeKeys(null)).toBe(null);
    expect(normalizeKeys(7)).toBe(7);
  });

  it('infers kinds from legacy payload shapes', () => {
    expect(inferKind('/rpcpb.SliverRPC/Ps', { processes: [] })).toBe('processes');
    expect(inferKind('/rpcpb.SliverRPC/Ls', { files: [] })).toBe('files');
    expect(inferKind('/rpcpb.SliverRPC/GetEnv', { variables: [] })).toBe('env');
    expect(inferKind('/rpcpb.SliverRPC/Services', { details: [] })).toBe('services');
    expect(inferKind('/rpcpb.SliverRPC/Netstat', { entries: [] })).toBe('netstat');
    expect(inferKind('/rpcpb.SliverRPC/Screenshot', { data: 'cG5n' })).toBe('image');
    expect(inferKind('/rpcpb.SliverRPC/Execute', { stdout: 'aGVsbG8=' })).toBe('text');
    expect(inferKind('/rpcpb.SliverRPC/Execute', { response: { output: 'aGVsbG8=' } })).toBe('text');
    expect(inferKind('/rpcpb.SliverRPC/GetVersion', {})).toBe('json');
    expect(inferKind('/rpcpb.SliverRPC/GetVersion', null)).toBe('json');
  });

  it('decodes base64 text payloads', () => {
    expect(decodeTextPayload({ stdout: 'aGVsbG8=', response: { err: '' } })).toEqual({ text: 'hello', err: '' });
    expect(decodeTextPayload({ response: { output: 'aGVsbG8=' } })).toEqual({ text: 'hello', err: '' });
    expect(decodeTextPayload({ stdout: 'aGVsbG8=', response: { err: 'boom' } })).toEqual({ text: 'hello', err: 'boom' });
    expect(decodeTextPayload({ path: '/test/dir' })).toEqual({ text: '/test/dir', err: '' });
    expect(decodeTextPayload(null)).toEqual({ text: '', err: '' });
  });

  it('formats nanosecond durations compactly', () => {
    expect(formatDurationNs('299999999999')).toBe('5m');
    expect(formatDurationNs('5000000000')).toBe('5s');
    expect(formatDurationNs('90000000000')).toBe('1m 30s');
    expect(formatDurationNs('9000000000000')).toBe('2h 30m');
    expect(formatDurationNs('nope')).toBe('nope');
  });

  it('humanizes request keys', () => {
    expect(humanizeKey('sessionid')).toBe('Session');
    expect(humanizeKey('ppid')).toBe('PPID');
    expect(humanizeKey('pid')).toBe('PID');
    expect(humanizeKey('fullinfo')).toBe('Full info');
    expect(humanizeKey('displayname')).toBe('Display name');
    expect(humanizeKey('path')).toBe('Path');
  });

  it('summarizes request parameters with session first', () => {
    expect(requestSummary({ request: { timeout: '299999999999', sessionid: 'sess-9' } })).toEqual([
      { label: 'Session', value: 'sess-9' },
      { label: 'Timeout', value: '5m' },
    ]);
    expect(requestSummary({ request: { fullinfo: false, pid: 42, path: '/test/dir' } })).toEqual([
      { label: 'PID', value: '42' },
      { label: 'Path', value: '/test/dir' },
    ]);
    expect(requestSummary({ request: {} })).toEqual([]);
    expect(requestSummary(null)).toEqual([]);
    const [hosts] = requestSummary({ request: { hosts: Array.from({ length: 40 }, (_, i) => `host-${i}`) } });
    expect(hosts.label).toBe('Hosts');
    expect(hosts.value.endsWith('…')).toBe(true);
    expect(hosts.value.length).toBe(121);
  });
});

describe('SessionTasks', () => {
  it('renders rows and expands payloads', async () => {
    listSessionTasks.mockResolvedValue([task]);
    getTaskCallPayloads.mockResolvedValue([
      {
        direction: 'request',
        ts: task.ts,
        preview: '{}',
        json: '{"Request":{"Timeout":"299999999999","SessionID":"sess-9"}}',
        base64: 'cmVx',
      },
      payload('files', FIXTURES.files),
    ]);

    render(SessionTasks, { sessionID: 'sess-1', active: true });
    expect(await screen.findByText('Ls')).toBeTruthy();

    await fireEvent.click(screen.getByRole('button', { name: /view/i }));
    expect(await screen.findByText('Request')).toBeTruthy();
    expect(await screen.findByText('Response')).toBeTruthy();
    expect(await screen.findByText('Session')).toBeTruthy();
    expect(await screen.findByText('sess-9')).toBeTruthy();
    expect(await screen.findByText('Timeout')).toBeTruthy();
    expect(await screen.findByText('5m')).toBeTruthy();
    expect(await screen.findByText('file-a.txt')).toBeTruthy();
    expect(document.body.textContent).not.toContain('{');
    expect(document.body.textContent).not.toContain('"request"');
    expect(getTaskCallPayloads).toHaveBeenCalledWith('chain-1');
  });

  it('renders an image payload', async () => {
    listSessionTasks.mockResolvedValue([{ ...task, method: '/rpcpb.SliverRPC/Screenshot', chainRef: 'chain-2' }]);
    getTaskCallPayloads.mockResolvedValue([payload('image', FIXTURES.image)]);
    render(SessionTasks, { sessionID: 'sess-1', active: true });
    await fireEvent.click(await screen.findByRole('button', { name: /view/i }));
    const img = await screen.findByAltText('Screenshot');
    expect(img.getAttribute('src')).toBe('data:image/png;base64,cG5n');
  });

  it('shows a per-chain payload error instead of staying on loading', async () => {
    listSessionTasks.mockResolvedValue([task]);
    getTaskCallPayloads.mockRejectedValue(new Error('payload failed'));

    render(SessionTasks, { sessionID: 'sess-1', active: true });
    await fireEvent.click(await screen.findByRole('button', { name: /view/i }));

    expect(await screen.findByText('payload failed')).toBeTruthy();
    expect(screen.queryByText('Loading…')).toBeNull();
  });

  it('clears the expanded payloads when the session changes', async () => {
    listSessionTasks.mockResolvedValue([task]);
    getTaskCallPayloads.mockResolvedValue([
      { direction: 'request', ts: task.ts, preview: '', json: '{"Path":"/test/dir"}', base64: 'cmVx' },
      payload('files', FIXTURES.files),
    ]);

    const { rerender } = render(SessionTasks, { sessionID: 'sess-1', active: true });
    await fireEvent.click(await screen.findByRole('button', { name: /view/i }));
    expect(await screen.findByText('Request')).toBeTruthy();
    expect(await screen.findByText('Response')).toBeTruthy();

    listSessionTasks.mockResolvedValue([]);
    await rerender({ sessionID: 'sess-2', active: true });
    expect(await screen.findByText('No recorded calls for this session yet.')).toBeTruthy();

    expect(screen.queryByText('Request')).toBeNull();
    expect(screen.queryByText('Response')).toBeNull();
  });

  it('does not load tasks when inactive', async () => {
    render(SessionTasks, { sessionID: 'sess-1', active: false });
    await tick();
    expect(listSessionTasks).not.toHaveBeenCalled();
    expect(screen.getByText('No recorded calls for this session yet.')).toBeTruthy();
  });

  it('keeps the newer session rows when responses resolve out of order', async () => {
    const older = deferred();
    const newer = deferred();
    listSessionTasks.mockReturnValueOnce(older.promise).mockReturnValueOnce(newer.promise);

    const { rerender } = render(SessionTasks, { sessionID: 'sess-1', active: true });
    await rerender({ sessionID: 'sess-2', active: true });

    newer.resolve([{ ...task, id: 2, method: '/rpcpb.SliverRPC/Ps', chainRef: 'chain-2' }]);
    expect(await screen.findByText('Ps')).toBeTruthy();

    older.resolve([{ ...task, id: 1, method: '/rpcpb.SliverRPC/Ls', chainRef: 'chain-1' }]);
    await new Promise((resolve) => setTimeout(resolve, 0));
    expect(screen.queryByText('Ls')).toBeNull();
    expect(screen.getByText('Ps')).toBeTruthy();
  });

  it('renders a processes payload as a typed table', async () => {
    listSessionTasks.mockResolvedValue([{ ...task, method: '/rpcpb.SliverRPC/Ps', chainRef: 'chain-ps' }]);
    getTaskCallPayloads.mockResolvedValue([
      { direction: 'response', ts: task.ts, preview: '', json: FIXTURES.processes, base64: '' },
    ]);
    render(SessionTasks, { sessionID: 'sess-1', active: true });
    await fireEvent.click(await screen.findByRole('button', { name: /view/i }));
    expect(await screen.findByText('proc-a')).toBeTruthy();
    expect(await screen.findByText('user-a')).toBeTruthy();
  });

  it('renders a files payload as a typed table', async () => {
    listSessionTasks.mockResolvedValue([task]);
    getTaskCallPayloads.mockResolvedValue([payload('files', FIXTURES.files)]);
    render(SessionTasks, { sessionID: 'sess-1', active: true });
    await fireEvent.click(await screen.findByRole('button', { name: /view/i }));
    expect(await screen.findByText('file-a.txt')).toBeTruthy();
    expect(await screen.findByText('42 B')).toBeTruthy();
  });

  it('renders an env payload as a typed table', async () => {
    listSessionTasks.mockResolvedValue([{ ...task, method: '/rpcpb.SliverRPC/Env', chainRef: 'chain-env' }]);
    getTaskCallPayloads.mockResolvedValue([payload('env', FIXTURES.env)]);
    render(SessionTasks, { sessionID: 'sess-1', active: true });
    await fireEvent.click(await screen.findByRole('button', { name: /view/i }));
    expect(await screen.findByText('ENV_KEY')).toBeTruthy();
    expect(await screen.findByText('value-a')).toBeTruthy();
  });

  it('renders a services payload as a typed table', async () => {
    listSessionTasks.mockResolvedValue([{ ...task, method: '/rpcpb.SliverRPC/Services', chainRef: 'chain-svc' }]);
    getTaskCallPayloads.mockResolvedValue([payload('services', FIXTURES.services)]);
    render(SessionTasks, { sessionID: 'sess-1', active: true });
    await fireEvent.click(await screen.findByRole('button', { name: /view/i }));
    expect(await screen.findByText('svc-a')).toBeTruthy();
    expect(await screen.findByText('svc-a display')).toBeTruthy();
  });

  it('renders a netstat payload as a typed table', async () => {
    listSessionTasks.mockResolvedValue([{ ...task, method: '/rpcpb.SliverRPC/Netstat', chainRef: 'chain-net' }]);
    getTaskCallPayloads.mockResolvedValue([payload('netstat', FIXTURES.netstat)]);
    render(SessionTasks, { sessionID: 'sess-1', active: true });
    await fireEvent.click(await screen.findByRole('button', { name: /view/i }));
    expect(await screen.findByText('127.0.0.1:1')).toBeTruthy();
    expect(await screen.findByText('LISTEN')).toBeTruthy();
  });

  it('renders a text payload', async () => {
    listSessionTasks.mockResolvedValue([{ ...task, method: '/rpcpb.SliverRPC/Execute', chainRef: 'chain-exec' }]);
    getTaskCallPayloads.mockResolvedValue([payload('text', FIXTURES.text)]);
    render(SessionTasks, { sessionID: 'sess-1', active: true });
    await fireEvent.click(await screen.findByRole('button', { name: /view/i }));
    expect(await screen.findByText('hello')).toBeTruthy();
  });

  it('renders a text payload error', async () => {
    listSessionTasks.mockResolvedValue([{ ...task, method: '/rpcpb.SliverRPC/Execute', chainRef: 'chain-exec' }]);
    getTaskCallPayloads.mockResolvedValue([payload('text', FIXTURES.textErr)]);
    render(SessionTasks, { sessionID: 'sess-1', active: true });
    await fireEvent.click(await screen.findByRole('button', { name: /view/i }));
    expect(await screen.findByText('hello')).toBeTruthy();
    expect(await screen.findByText('boom')).toBeTruthy();
  });

  it('discloses the 100-row cap', async () => {
    listSessionTasks.mockResolvedValue(
      Array.from({ length: 100 }, (_, i) => ({ ...task, id: i + 1, chainRef: `chain-${i}` })),
    );
    render(SessionTasks, { sessionID: 'sess-1', active: true });
    expect(await screen.findByText('Showing the latest 100 calls.')).toBeTruthy();
  });

  it('renders duplicate-direction payloads without crashing', async () => {
    listSessionTasks.mockResolvedValue([{ ...task, method: '/rpcpb.SliverRPC/GetVersion' }]);
    getTaskCallPayloads.mockResolvedValue([
      { direction: 'response', ts: task.ts, preview: '', json: '{"A":1}', base64: '' },
      { direction: 'response', ts: task.ts, preview: '', json: '{"B":2}', base64: '' },
    ]);
    render(SessionTasks, { sessionID: 'sess-1', active: true });
    await fireEvent.click(await screen.findByRole('button', { name: /view/i }));
    expect(await screen.findAllByText('Response')).toHaveLength(2);
    expect(screen.getByText(/"a": 1/)).toBeTruthy();
    expect(screen.getByText(/"b": 2/)).toBeTruthy();
  });

  it('shows a no-payload message when a chain has no payload rows', async () => {
    listSessionTasks.mockResolvedValue([task]);
    getTaskCallPayloads.mockResolvedValue([]);
    render(SessionTasks, { sessionID: 'sess-1', active: true });
    await fireEvent.click(await screen.findByRole('button', { name: /view/i }));
    expect(await screen.findByText('(no payload)')).toBeTruthy();
  });

  it('marks truncated base64 fallbacks', async () => {
    listSessionTasks.mockResolvedValue([task]);
    getTaskCallPayloads.mockResolvedValue([
      { direction: 'response', ts: task.ts, preview: '', json: '', base64: 'A'.repeat(3000) },
    ]);
    render(SessionTasks, { sessionID: 'sess-1', active: true });
    await fireEvent.click(await screen.findByRole('button', { name: /view/i }));
    expect(await screen.findByText(/truncated \(3000 base64 chars total\)/)).toBeTruthy();
  });
});
