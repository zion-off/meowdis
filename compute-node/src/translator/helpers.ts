import type { ResultRow, Statement } from "./types";

export function deleteIfExpired(key: string): Statement {
  return {
    sql: "DELETE FROM keys WHERE key = ? AND expires_at IS NOT NULL AND expires_at <= unixepoch()",
    params: [key],
  };
}

export function rowString(row: ResultRow, field: string): string {
  if (!(field in row)) {
    return "";
  }
  const value = row[field];
  if (value === null || value === undefined) {
    return "";
  }
  if (typeof value === "string") {
    return value;
  }
  return String(value);
}

export function parseIntStrict(value: string): number | null {
  if (!/^-?\d+$/.test(value)) {
    return null;
  }
  const parsed = Number(value);
  if (!Number.isSafeInteger(parsed)) {
    return null;
  }
  return parsed;
}

export function wrongTypeFor(results: ResultRow[][], index: number, expected: string): boolean {
  if (index < 0 || index >= results.length) {
    return false;
  }
  if (results[index].length === 0) {
    return false;
  }
  const row = results[index][0];
  const value = row["type"];
  if (value === null || value === undefined) {
    return false;
  }
  return String(value).toLowerCase() !== expected;
}
