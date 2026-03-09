import { DurableObject } from "cloudflare:workers";

type Statement = { sql: string; params: unknown[] };

type ResultRow = Record<string, unknown>;

interface Env {}

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
        sql: 'CREATE TABLE IF NOT EXISTS lists (key TEXT NOT NULL REFERENCES keys(key) ON DELETE CASCADE, "index" REAL NOT NULL, value TEXT NOT NULL, PRIMARY KEY (key, "index"))',
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
}
