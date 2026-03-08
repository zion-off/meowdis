import { DurableObject } from "cloudflare:workers";

type Statement = { sql: string; params: unknown[] };

type ResultRow = Record<string, unknown>;

interface Env {
  STORAGE: DurableObjectNamespace;
  STORAGE_SECRET: string;
}

export class StorageDurableObject extends DurableObject {
  private sql: SqlStorage;

  constructor(ctx: DurableObjectState, env: Env) {
    super(ctx, env);
    this.sql = ctx.storage.sql;
  }

  init(): void {
    const statements: Statement[] = [
      {
        sql: "CREATE TABLE IF NOT EXISTS keys (key TEXT PRIMARY KEY, type TEXT NOT NULL CHECK(type IN ('string', 'hash', 'list', 'set')), expires_at INTEGER)",
        params: [],
      },
      {
        sql: "CREATE TABLE IF NOT EXISTS strings (key TEXT PRIMARY KEY REFERENCES keys(key) ON DELETE CASCADE, value TEXT NOT NULL)",
        params: [],
      },
      {
        sql: "CREATE TABLE IF NOT EXISTS hashes (key TEXT NOT NULL REFERENCES keys(key) ON DELETE CASCADE, field TEXT NOT NULL, value TEXT NOT NULL, PRIMARY KEY (key, field))",
        params: [],
      },
      {
        sql: "CREATE TABLE IF NOT EXISTS lists (key TEXT NOT NULL REFERENCES keys(key) ON DELETE CASCADE, index REAL NOT NULL, value TEXT NOT NULL, PRIMARY KEY (key, index))",
        params: [],
      },
      {
        sql: "CREATE TABLE IF NOT EXISTS sets (key TEXT NOT NULL REFERENCES keys(key) ON DELETE CASCADE, member TEXT NOT NULL, PRIMARY KEY (key, member))",
        params: [],
      },
    ];

    this.ctx.storage.transactionSync(() => {
      for (const statement of statements) {
        this.sql.exec(statement.sql, ...statement.params).toArray();
      }
    });
  }

  execBatch(statements: Statement[]): ResultRow[][] {
    const results: ResultRow[][] = [];
    this.ctx.storage.transactionSync(() => {
      for (const { sql, params } of statements) {
        results.push(this.sql.exec(sql, ...params).toArray());
      }
    });
    return results;
  }

  execPipeline(batches: Statement[][]): ResultRow[][][] {
    const pipelineResults: ResultRow[][][] = [];
    for (const statements of batches) {
      const batch = this.execBatch(statements);
      pipelineResults.push(batch);
    }
    return pipelineResults;
  }

  private unauthorized(): Response {
    return new Response(JSON.stringify({ error: "UNAUTHORIZED" }), {
      status: 401,
      headers: { "Content-Type": "application/json" },
    });
  }

  private jsonResponse(body: unknown, status = 200): Response {
    return new Response(JSON.stringify(body), {
      status,
      headers: { "Content-Type": "application/json" },
    });
  }

  async fetch(request: Request): Promise<Response> {
    const authHeader = request.headers.get("Authorization");
    if (!authHeader || !authHeader.startsWith("Bearer ")) {
      return this.unauthorized();
    }
    const token = authHeader.slice("Bearer ".length);
    if (token !== (this.env as Env).STORAGE_SECRET) {
      return this.unauthorized();
    }

    if (request.method !== "POST") {
      return this.jsonResponse({ error: "Method not allowed" }, 405);
    }

    let payload: unknown;
    try {
      payload = await request.json();
    } catch (err) {
      return this.jsonResponse({ error: "ERR invalid JSON" }, 400);
    }

    if (payload && typeof payload === "object" && "init" in payload) {
      const initValue = (payload as { init?: boolean }).init;
      if (initValue) {
        try {
          this.init();
          return this.jsonResponse({ results: [] });
        } catch (err) {
          return this.jsonResponse({ error: "ERR " + String(err) }, 500);
        }
      }
    }

    if (payload && typeof payload === "object" && "statements" in payload) {
      const statements = (payload as { statements?: Statement[] }).statements;
      if (!Array.isArray(statements)) {
        return this.jsonResponse({ error: "ERR invalid statements" }, 400);
      }
      try {
        const results = this.execBatch(statements);
        return this.jsonResponse({ results });
      } catch (err) {
        return this.jsonResponse({ error: "ERR " + String(err) }, 500);
      }
    }

    if (payload && typeof payload === "object" && "pipeline" in payload) {
      const pipeline = (payload as { pipeline?: { statements: Statement[] }[] }).pipeline;
      if (!Array.isArray(pipeline)) {
        return this.jsonResponse({ error: "ERR invalid pipeline" }, 400);
      }

      const results = pipeline.map((item) => {
        try {
          if (!item || !Array.isArray(item.statements)) {
            return { error: "ERR invalid statements" };
          }
          return this.execBatch(item.statements);
        } catch (err) {
          return { error: "ERR " + String(err) };
        }
      });

      return this.jsonResponse({ results });
    }

    return this.jsonResponse({ error: "ERR invalid request" }, 400);
  }
}

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    const id = env.STORAGE.idFromName("global");
    const stub = env.STORAGE.get(id);
    return stub.fetch(request);
  },
};
