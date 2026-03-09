import { describe, expect, it } from "vitest";
import type {
  DurableObjectId,
  DurableObjectJurisdiction,
  DurableObjectNamespace,
  DurableObjectStub,
} from "@cloudflare/workers-types";
import worker from "../index";
import type { ResultRow, Statement } from "../translator/types";

const authToken = "test-token";

type StorageStub = {
  init(): Promise<void>;
  execBatch(statements: Statement[]): Promise<ResultRow[][]>;
  execPipeline(batches: { statements: Statement[] }[]): Promise<ResultRow[][][]>;
};

type Env = {
  AUTH_TOKEN: string;
  STORAGE: DurableObjectNamespace;
};

function makeNamespace(stub: StorageStub): DurableObjectNamespace {
  const id = {} as unknown as DurableObjectId;
  const stubValue = stub as unknown as DurableObjectStub;
  const namespace: DurableObjectNamespace = {
    newUniqueId: () => id,
    idFromName: () => id,
    idFromString: () => id,
    get: () => stubValue,
    getByName: () => stubValue,
    jurisdiction: (jurisdiction: DurableObjectJurisdiction) => {
      void jurisdiction;
      return namespace;
    },
  };

  return namespace;
}

function makeEnv(stub: StorageStub): Env {
  return {
    AUTH_TOKEN: authToken,
    STORAGE: makeNamespace(stub),
  };
}

describe("handler", () => {
  const fetchHandler = worker.fetch as (request: Request, env: Env) => Promise<Response>;

  it("rejects missing auth", async () => {
    const env = makeEnv({
      init: async () => {},
      execBatch: async () => [],
      execPipeline: async () => [],
    });
    const response = await fetchHandler(new Request("http://localhost", { method: "POST" }), env);
    expect(response.status).toBe(401);
  });

  it("rejects wrong auth", async () => {
    const env = makeEnv({
      init: async () => {},
      execBatch: async () => [],
      execPipeline: async () => [],
    });
    const request = new Request("http://localhost", {
      method: "POST",
      headers: { Authorization: "Bearer wrong" },
    });
    const response = await fetchHandler(request, env);
    expect(response.status).toBe(401);
  });

  it("rejects non-POST", async () => {
    const env = makeEnv({
      init: async () => {},
      execBatch: async () => [],
      execPipeline: async () => [],
    });
    const request = new Request("http://localhost", {
      method: "GET",
      headers: { Authorization: `Bearer ${authToken}` },
    });
    const response = await fetchHandler(request, env);
    expect(response.status).toBe(405);
  });

  it("handles PING", async () => {
    const stub: StorageStub = {
      init: async () => {},
      execBatch: async () => [],
      execPipeline: async () => [],
    };
    const env = makeEnv(stub);
    const request = new Request("http://localhost", {
      method: "POST",
      headers: { Authorization: `Bearer ${authToken}` },
      body: JSON.stringify(["PING"]),
    });
    const response = await fetchHandler(request, env);
    const body = (await response.json()) as { result: string };
    expect(body).toEqual({ result: "PONG" });
  });

  it("accepts numeric arguments", async () => {
    const results: ResultRow[][] = [[], [], [], []];
    const stub: StorageStub = {
      init: async () => {},
      execBatch: async () => results,
      execPipeline: async () => [],
    };
    const env = makeEnv(stub);
    const request = new Request("http://localhost", {
      method: "POST",
      headers: { Authorization: `Bearer ${authToken}` },
      body: JSON.stringify(["SET", "k", 0]),
    });
    const response = await fetchHandler(request, env);
    const body = (await response.json()) as { result: string };
    expect(body).toEqual({ result: "OK" });
  });

  it("handles pipelines", async () => {
    const pipelineResults: ResultRow[][][] = [
      [[], [], [], []],
      [[], [], [{ value: "v" }]],
    ];
    const stub: StorageStub = {
      init: async () => {},
      execBatch: async () => [],
      execPipeline: async () => pipelineResults,
    };
    const env = makeEnv(stub);
    const request = new Request("http://localhost", {
      method: "POST",
      headers: { Authorization: `Bearer ${authToken}` },
      body: JSON.stringify([
        ["SET", "k", "v"],
        ["GET", "k"],
      ]),
    });
    const response = await fetchHandler(request, env);
    const body = (await response.json()) as { result: unknown[] };
    expect(body).toEqual({ result: ["OK", "v"] });
  });

  it("encodes base64 responses", async () => {
    const stub: StorageStub = {
      init: async () => {},
      execBatch: async () => [],
      execPipeline: async () => [],
    };
    const env = makeEnv(stub);
    const request = new Request("http://localhost", {
      method: "POST",
      headers: {
        Authorization: `Bearer ${authToken}`,
        "Upstash-Encoding": "base64",
      },
      body: JSON.stringify(["PING"]),
    });
    const response = await fetchHandler(request, env);
    const body = (await response.json()) as { result: string };
    expect(body).toEqual({ result: "UE9ORw==" });
  });
});
