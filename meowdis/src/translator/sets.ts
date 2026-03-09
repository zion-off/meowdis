import type { Statement, Translation } from "./types";
import { ErrNotInteger, ErrWrongType, errWrongArgs } from "./errors";
import { deleteIfExpired, parseIntStrict, rowString, wrongTypeFor } from "./helpers";

export function translateSAdd(args: string[]): Translation {
  if (args.length < 2) {
    throw errWrongArgs("sadd");
  }

  const key = args[0];
  const members = args.slice(1);
  const statements: Statement[] = [
    deleteIfExpired(key),
    { sql: "SELECT type FROM keys WHERE key = ?", params: [key] },
    { sql: "INSERT OR IGNORE INTO keys (key, type) VALUES (?, 'set')", params: [key] },
  ];

  const insertIndexes: number[] = [];
  for (const member of members) {
    insertIndexes.push(statements.length);
    statements.push({
      sql: "INSERT OR IGNORE INTO sets (key, member) VALUES (?, ?) RETURNING member",
      params: [key, member],
    });
  }

  return {
    statements,
    mapResult: (results) => {
      if (wrongTypeFor(results, 1, "set")) {
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

export function translateSRem(args: string[]): Translation {
  if (args.length < 2) {
    throw errWrongArgs("srem");
  }

  const key = args[0];
  const members = args.slice(1);
  const statements: Statement[] = [
    deleteIfExpired(key),
    { sql: "SELECT type FROM keys WHERE key = ?", params: [key] },
  ];

  const deleteIndexes: number[] = [];
  for (const member of members) {
    deleteIndexes.push(statements.length);
    statements.push({
      sql: "DELETE FROM sets WHERE key = ? AND member = ? RETURNING member",
      params: [key, member],
    });
  }

  statements.push({
    sql: "DELETE FROM keys WHERE key = ? AND NOT EXISTS (SELECT 1 FROM sets WHERE key = ?)",
    params: [key, key],
  });

  return {
    statements,
    mapResult: (results) => {
      if (wrongTypeFor(results, 1, "set")) {
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

export function translateSMembers(args: string[]): Translation {
  if (args.length !== 1) {
    throw errWrongArgs("smembers");
  }

  const key = args[0];
  const statements: Statement[] = [
    deleteIfExpired(key),
    { sql: "SELECT type FROM keys WHERE key = ?", params: [key] },
    { sql: "SELECT member FROM sets WHERE key = ? ORDER BY member", params: [key] },
  ];

  return {
    statements,
    mapResult: (results) => {
      if (wrongTypeFor(results, 1, "set")) {
        throw ErrWrongType;
      }
      return results[2].map((row) => rowString(row, "member"));
    },
  };
}

export function translateSIsMember(args: string[]): Translation {
  if (args.length !== 2) {
    throw errWrongArgs("sismember");
  }

  const key = args[0];
  const member = args[1];
  const statements: Statement[] = [
    deleteIfExpired(key),
    { sql: "SELECT type FROM keys WHERE key = ?", params: [key] },
    { sql: "SELECT 1 FROM sets WHERE key = ? AND member = ?", params: [key, member] },
  ];

  return {
    statements,
    mapResult: (results) => {
      if (wrongTypeFor(results, 1, "set")) {
        throw ErrWrongType;
      }
      return results[2].length > 0 ? 1 : 0;
    },
  };
}

export function translateSCard(args: string[]): Translation {
  if (args.length !== 1) {
    throw errWrongArgs("scard");
  }

  const key = args[0];
  const statements: Statement[] = [
    deleteIfExpired(key),
    { sql: "SELECT type FROM keys WHERE key = ?", params: [key] },
    { sql: "SELECT COUNT(*) as count FROM sets WHERE key = ?", params: [key] },
  ];

  return {
    statements,
    mapResult: (results) => {
      if (wrongTypeFor(results, 1, "set")) {
        throw ErrWrongType;
      }
      if (results[2].length === 0) {
        return 0;
      }
      const count = parseIntStrict(rowString(results[2][0], "count"));
      if (count === null) {
        throw ErrNotInteger;
      }
      return count;
    },
  };
}
