import type { Statement, Translation } from "./types";
import { ErrWrongType, errWrongArgs } from "./errors";
import { deleteIfExpired, rowString, wrongTypeFor } from "./helpers";

export function translateHGet(args: string[]): Translation {
  if (args.length !== 2) {
    throw errWrongArgs("hget");
  }

  const key = args[0];
  const field = args[1];
  const statements: Statement[] = [
    deleteIfExpired(key),
    { sql: "SELECT type FROM keys WHERE key = ?", params: [key] },
    { sql: "SELECT value FROM hashes WHERE key = ? AND field = ?", params: [key, field] },
  ];

  return {
    statements,
    mapResult: (results) => {
      if (wrongTypeFor(results, 1, "hash")) {
        throw ErrWrongType;
      }
      if (results[2].length === 0) {
        return null;
      }
      return rowString(results[2][0], "value");
    },
  };
}

export function translateHSet(args: string[]): Translation {
  if (args.length < 3 || (args.length - 1) % 2 !== 0) {
    throw errWrongArgs("hset");
  }

  const key = args[0];
  const statements: Statement[] = [
    deleteIfExpired(key),
    { sql: "SELECT type FROM keys WHERE key = ?", params: [key] },
    { sql: "INSERT OR IGNORE INTO keys (key, type) VALUES (?, 'hash')", params: [key] },
  ];

  const insertIndexes: number[] = [];
  for (let i = 1; i < args.length; i += 2) {
    const field = args[i];
    const value = args[i + 1];
    insertIndexes.push(statements.length);
    statements.push(
      {
        sql: "INSERT OR IGNORE INTO hashes (key, field, value) VALUES (?, ?, ?) RETURNING field",
        params: [key, field, value],
      },
      {
        sql: "UPDATE hashes SET value = ? WHERE key = ? AND field = ? AND NOT EXISTS (SELECT 1 WHERE changes() > 0)",
        params: [value, key, field],
      }
    );
  }

  return {
    statements,
    mapResult: (results) => {
      if (wrongTypeFor(results, 1, "hash")) {
        throw ErrWrongType;
      }
      let count = 0;
      for (const index of insertIndexes) {
        count += results[index].length;
      }
      return count;
    },
  };
}

export function translateHDel(args: string[]): Translation {
  if (args.length < 2) {
    throw errWrongArgs("hdel");
  }

  const key = args[0];
  const statements: Statement[] = [
    deleteIfExpired(key),
    { sql: "SELECT type FROM keys WHERE key = ?", params: [key] },
  ];

  const deleteIndexes: number[] = [];
  for (let i = 1; i < args.length; i++) {
    const field = args[i];
    deleteIndexes.push(statements.length);
    statements.push({
      sql: "DELETE FROM hashes WHERE key = ? AND field = ? RETURNING field",
      params: [key, field],
    });
  }

  return {
    statements,
    mapResult: (results) => {
      if (wrongTypeFor(results, 1, "hash")) {
        throw ErrWrongType;
      }
      let count = 0;
      for (const index of deleteIndexes) {
        count += results[index].length;
      }
      return count;
    },
  };
}

export function translateHGetAll(args: string[]): Translation {
  if (args.length !== 1) {
    throw errWrongArgs("hgetall");
  }

  const key = args[0];
  const statements: Statement[] = [
    deleteIfExpired(key),
    { sql: "SELECT type FROM keys WHERE key = ?", params: [key] },
    { sql: "SELECT field, value FROM hashes WHERE key = ? ORDER BY field", params: [key] },
  ];

  return {
    statements,
    mapResult: (results) => {
      if (wrongTypeFor(results, 1, "hash")) {
        throw ErrWrongType;
      }
      const out: unknown[] = [];
      for (const row of results[2]) {
        out.push(rowString(row, "field"), rowString(row, "value"));
      }
      return out;
    },
  };
}

export function translateHExists(args: string[]): Translation {
  if (args.length !== 2) {
    throw errWrongArgs("hexists");
  }

  const key = args[0];
  const field = args[1];
  const statements: Statement[] = [
    deleteIfExpired(key),
    { sql: "SELECT type FROM keys WHERE key = ?", params: [key] },
    { sql: "SELECT 1 FROM hashes WHERE key = ? AND field = ?", params: [key, field] },
  ];

  return {
    statements,
    mapResult: (results) => {
      if (wrongTypeFor(results, 1, "hash")) {
        throw ErrWrongType;
      }
      return results[2].length > 0 ? 1 : 0;
    },
  };
}

export function translateHKeys(args: string[]): Translation {
  if (args.length !== 1) {
    throw errWrongArgs("hkeys");
  }

  const key = args[0];
  const statements: Statement[] = [
    deleteIfExpired(key),
    { sql: "SELECT type FROM keys WHERE key = ?", params: [key] },
    { sql: "SELECT field FROM hashes WHERE key = ? ORDER BY field", params: [key] },
  ];

  return {
    statements,
    mapResult: (results) => {
      if (wrongTypeFor(results, 1, "hash")) {
        throw ErrWrongType;
      }
      return results[2].map((row) => rowString(row, "field"));
    },
  };
}

export function translateHVals(args: string[]): Translation {
  if (args.length !== 1) {
    throw errWrongArgs("hvals");
  }

  const key = args[0];
  const statements: Statement[] = [
    deleteIfExpired(key),
    { sql: "SELECT type FROM keys WHERE key = ?", params: [key] },
    { sql: "SELECT value FROM hashes WHERE key = ? ORDER BY field", params: [key] },
  ];

  return {
    statements,
    mapResult: (results) => {
      if (wrongTypeFor(results, 1, "hash")) {
        throw ErrWrongType;
      }
      return results[2].map((row) => rowString(row, "value"));
    },
  };
}
