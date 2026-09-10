import { describe, expect, it, vi, beforeEach } from 'vitest';

const ListSessionTasks = vi.fn();
const GetTaskCallPayloads = vi.fn();

vi.mock('../../../../bindings/siren/cmd/gui/app.js', () => ({
  ListSessionTasks: (...args) => ListSessionTasks(...args),
  GetTaskCallPayloads: (...args) => GetTaskCallPayloads(...args),
}));

import { listSessionTasks, getTaskCallPayloads } from '../sessionTasks.js';

describe('sessionTasks api', () => {
  beforeEach(() => {
    ListSessionTasks.mockReset();
    GetTaskCallPayloads.mockReset();
  });

  it('lists session tasks with the default limit', async () => {
    ListSessionTasks.mockResolvedValue([{ method: '/rpcpb.SliverRPC/Ls' }]);
    const rows = await listSessionTasks('sess-1');
    expect(ListSessionTasks).toHaveBeenCalledWith('sess-1', 100);
    expect(rows).toHaveLength(1);
  });

  it('returns an empty array when the binding returns null', async () => {
    ListSessionTasks.mockResolvedValue(null);
    expect(await listSessionTasks('sess-1', 10)).toEqual([]);
  });

  it('fetches payloads by chain ref', async () => {
    GetTaskCallPayloads.mockResolvedValue([{ direction: 'request' }]);
    const rows = await getTaskCallPayloads('chain-1');
    expect(GetTaskCallPayloads).toHaveBeenCalledWith('chain-1');
    expect(rows).toHaveLength(1);
  });
});
