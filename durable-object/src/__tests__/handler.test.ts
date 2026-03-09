import { describe, expect, it, vi } from "vitest";
import type { DurableObjectNamespace, DurableObjectState } from "@cloudflare/workers-types";

type StorageEnv = {
  STORAGE_SECRET: string;
  STORAGE: DurableObjectNamespace;
};

vi.mock("cloudflare:workers", () => ({
  DurableObject: class {
    ctx: DurableObjectState;
    env: StorageEnv;
    constructor(ctx: DurableObjectState, env: StorageEnv) {
      this.ctx = ctx;
      this.env = env;
    }
  },
}));

import { StorageDurableObject } from "../index";

type ExecResult = { toArray: () => unknown[] };
type SqlExec = (sql: string, ...params: unknown[]) => ExecResult;

type StorageCtx = {
  storage: {
    sql: { exec: SqlExec };
    transactionSync: (fn: () => void) => void;
  };
};

function makeCtx(exec: SqlExec): StorageCtx {
  return {
    storage: {
      sql: { exec },
      transactionSync: (fn: () => void) => fn(),
    },
  };
}

describe("StorageDurableObject", () => {
  it("init creates tables", () => {
    const exec = vi.fn<SqlExec>(() => ({ toArray: () => [] }));
    const ctx = makeCtx(exec) as unknown as DurableObjectState;
    const env: StorageEnv = { STORAGE_SECRET: "secret", STORAGE: {} as DurableObjectNamespace };
    const storage = new StorageDurableObject(ctx, env);

    storage.init();

    expect(exec).toHaveBeenCalledTimes(5);
    const calls = exec.mock.calls.map((call) => call[0]);
    expect(calls[0]).toContain("CREATE TABLE IF NOT EXISTS keys");
    expect(calls[1]).toContain("CREATE TABLE IF NOT EXISTS strings");
    expect(calls[2]).toContain("CREATE TABLE IF NOT EXISTS hashes");
    expect(calls[3]).toContain("CREATE TABLE IF NOT EXISTS lists");
    expect(calls[4]).toContain("CREATE TABLE IF NOT EXISTS sets");
  });

  it("execBatch runs statements in order", () => {
    const exec = vi
      .fn<SqlExec>()
      .mockImplementationOnce(() => ({ toArray: () => [{ a: 1 }] }))
      .mockImplementationOnce(() => ({ toArray: () => [{ b: 2 }] }));
    const ctx = makeCtx(exec) as unknown as DurableObjectState;
    const env: StorageEnv = { STORAGE_SECRET: "secret", STORAGE: {} as DurableObjectNamespace };
    const storage = new StorageDurableObject(ctx, env);

    const results = storage.execBatch([
      { sql: "SELECT 1", params: [] },
      { sql: "SELECT 2", params: [] },
    ]);

    expect(exec).toHaveBeenCalledTimes(2);
    expect(results).toEqual([[{ a: 1 }], [{ b: 2 }]]);
  });

  it("execPipeline executes batches", () => {
    const exec = vi.fn<SqlExec>(() => ({ toArray: () => [] }));
    const ctx = makeCtx(exec) as unknown as DurableObjectState;
    const env: StorageEnv = { STORAGE_SECRET: "secret", STORAGE: {} as DurableObjectNamespace };
    const storage = new StorageDurableObject(ctx, env);
    const execBatchSpy = vi.spyOn(storage, "execBatch");

    const results = storage.execPipeline([
      [{ sql: "SELECT 1", params: [] }],
      [{ sql: "SELECT 2", params: [] }],
    ]);

    expect(execBatchSpy).toHaveBeenCalledTimes(2);
    expect(results).toEqual([[[]], [[]]]);
  });

  it("rejects missing or wrong auth", async () => {
    const exec = vi.fn<SqlExec>(() => ({ toArray: () => [] }));
    const ctx = makeCtx(exec) as unknown as DurableObjectState;
    const env: StorageEnv = { STORAGE_SECRET: "secret", STORAGE: {} as DurableObjectNamespace };
    const storage = new StorageDurableObject(ctx, env);

    const missing = await storage.fetch(new Request("http://localhost", { method: "POST" }));
    expect(missing.status).toBe(401);

    const wrong = await storage.fetch(
      new Request("http://localhost", {
        method: "POST",
        headers: { Authorization: "Bearer wrong" },
      })
    );
    expect(wrong.status).toBe(401);
  });

  it("handles statements payload", async () => {
    const exec = vi.fn<SqlExec>(() => ({ toArray: () => [{ ok: true }] }));
    const ctx = makeCtx(exec) as unknown as DurableObjectState;
    const env: StorageEnv = { STORAGE_SECRET: "secret", STORAGE: {} as DurableObjectNamespace };
    const storage = new StorageDurableObject(ctx, env);
    const execBatchSpy = vi.spyOn(storage, "execBatch");

    const response = await storage.fetch(
      new Request("http://localhost", {
        method: "POST",
        headers: { Authorization: "Bearer secret" },
        body: JSON.stringify({ statements: [{ sql: "SELECT 1", params: [] }] }),
      })
    );

    const body = await response.json();
    expect(execBatchSpy).toHaveBeenCalledTimes(1);
    expect(body).toEqual({ results: [[{ ok: true }]] });
  });

  it("handles pipeline payload", async () => {
    const exec = vi.fn<SqlExec>(() => ({ toArray: () => [] }));
    const ctx = makeCtx(exec) as unknown as DurableObjectState;
    const env: StorageEnv = { STORAGE_SECRET: "secret", STORAGE: {} as DurableObjectNamespace };
    const storage = new StorageDurableObject(ctx, env);
    const execBatchSpy = vi.spyOn(storage, "execBatch");

    const response = await storage.fetch(
      new Request("http://localhost", {
        method: "POST",
        headers: { Authorization: "Bearer secret" },
        body: JSON.stringify({
          pipeline: [
            { statements: [{ sql: "SELECT 1", params: [] }] },
            { statements: [{ sql: "SELECT 2", params: [] }] },
          ],
        }),
      })
    );

    const body = await response.json();
    expect(execBatchSpy).toHaveBeenCalledTimes(2);
    expect(body).toEqual({ results: [[[]], [[]]] });
  });
});
