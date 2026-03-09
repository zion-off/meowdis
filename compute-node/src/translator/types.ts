export interface Statement {
  sql: string;
  params: unknown[];
}

export type ResultRow = Record<string, unknown>;

export interface Translation {
  statements: Statement[];
  mapResult(results: ResultRow[][]): unknown;
}
