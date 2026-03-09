import type { Statement, Translation } from "./types";
import { ErrNotInteger, ErrWrongType, errWrongArgs } from "./errors";
import { deleteIfExpired, parseIntStrict, rowString, wrongTypeFor } from "./helpers";

export function translateLPush(args: string[]): Translation {
  if (args.length < 2) {
    throw errWrongArgs("lpush");
  }

  const key = args[0];
  const values = args.slice(1);
  const statements: Statement[] = [
    deleteIfExpired(key),
    { sql: "SELECT type FROM keys WHERE key = ?", params: [key] },
    { sql: "INSERT OR IGNORE INTO keys (key, type) VALUES (?, 'list')", params: [key] },
  ];

  for (const value of values) {
    statements.push({
      sql: 'INSERT INTO lists (key, "index", value) SELECT ?, COALESCE(MIN("index"), 1.0) - 1.0, ? FROM lists WHERE key = ?',
      params: [key, value, key],
    });
  }

  const countIndex = statements.length;
  statements.push({ sql: "SELECT COUNT(*) as count FROM lists WHERE key = ?", params: [key] });

  return {
    statements,
    mapResult: (results) => {
      if (wrongTypeFor(results, 1, "list")) {
        throw ErrWrongType;
      }
      if (results[countIndex].length === 0) {
        return 0;
      }
      const count = parseIntStrict(rowString(results[countIndex][0], "count"));
      if (count === null) {
        throw ErrNotInteger;
      }
      return count;
    },
  };
}

export function translateRPush(args: string[]): Translation {
  if (args.length < 2) {
    throw errWrongArgs("rpush");
  }

  const key = args[0];
  const values = args.slice(1);
  const statements: Statement[] = [
    deleteIfExpired(key),
    { sql: "SELECT type FROM keys WHERE key = ?", params: [key] },
    { sql: "INSERT OR IGNORE INTO keys (key, type) VALUES (?, 'list')", params: [key] },
  ];

  for (const value of values) {
    statements.push({
      sql: 'INSERT INTO lists (key, "index", value) SELECT ?, COALESCE(MAX("index"), 0.0) + 1.0, ? FROM lists WHERE key = ?',
      params: [key, value, key],
    });
  }

  const countIndex = statements.length;
  statements.push({ sql: "SELECT COUNT(*) as count FROM lists WHERE key = ?", params: [key] });

  return {
    statements,
    mapResult: (results) => {
      if (wrongTypeFor(results, 1, "list")) {
        throw ErrWrongType;
      }
      if (results[countIndex].length === 0) {
        return 0;
      }
      const count = parseIntStrict(rowString(results[countIndex][0], "count"));
      if (count === null) {
        throw ErrNotInteger;
      }
      return count;
    },
  };
}

function translatePop(args: string[], cmd: string, left: boolean): Translation {
  if (args.length < 1 || args.length > 2) {
    throw errWrongArgs(cmd);
  }

  const key = args[0];
  let count = 0;
  let hasCount = false;
  if (args.length === 2) {
    const parsed = parseIntStrict(args[1]);
    if (parsed === null || parsed < 0) {
      throw ErrNotInteger;
    }
    hasCount = true;
    count = parsed;
  }

  const statements: Statement[] = [
    deleteIfExpired(key),
    { sql: "SELECT type FROM keys WHERE key = ?", params: [key] },
  ];

  let popSQL = "";
  let popParams: unknown[] = [];
  if (hasCount) {
    const order = left ? "ASC" : "DESC";
    popSQL = `DELETE FROM lists WHERE key = ? AND "index" IN (SELECT "index" FROM lists WHERE key = ? ORDER BY "index" ${order} LIMIT ?) RETURNING "index", value`;
    popParams = [key, key, count];
  } else if (left) {
    popSQL =
      'DELETE FROM lists WHERE key = ? AND "index" = (SELECT MIN("index") FROM lists WHERE key = ?) RETURNING "index", value';
    popParams = [key, key];
  } else {
    popSQL =
      'DELETE FROM lists WHERE key = ? AND "index" = (SELECT MAX("index") FROM lists WHERE key = ?) RETURNING "index", value';
    popParams = [key, key];
  }

  const popIndex = statements.length;
  statements.push({ sql: popSQL, params: popParams });
  statements.push({
    sql: "DELETE FROM keys WHERE key = ? AND NOT EXISTS (SELECT 1 FROM lists WHERE key = ?)",
    params: [key, key],
  });

  return {
    statements,
    mapResult: (results) => {
      if (wrongTypeFor(results, 1, "list")) {
        throw ErrWrongType;
      }
      const rows = results[popIndex];
      if (!hasCount) {
        if (rows.length === 0) {
          return null;
        }
        return rowString(rows[0], "value");
      }
      if (count === 0) {
        return [];
      }
      const values = rows.map((row) => {
        const indexValue = Number(rowString(row, "index"));
        if (!Number.isFinite(indexValue)) {
          throw ErrNotInteger;
        }
        return { index: indexValue, value: rowString(row, "value") };
      });

      values.sort((a, b) => (left ? a.index - b.index : b.index - a.index));
      return values.map((item) => item.value);
    },
  };
}

export function translateLPop(args: string[]): Translation {
  return translatePop(args, "lpop", true);
}

export function translateRPop(args: string[]): Translation {
  return translatePop(args, "rpop", false);
}

export function translateLRange(args: string[]): Translation {
  if (args.length !== 3) {
    throw errWrongArgs("lrange");
  }

  const key = args[0];
  const start = parseIntStrict(args[1]);
  const stop = parseIntStrict(args[2]);
  if (start === null || stop === null) {
    throw ErrNotInteger;
  }

  const statements: Statement[] = [
    deleteIfExpired(key),
    { sql: "SELECT type FROM keys WHERE key = ?", params: [key] },
    { sql: 'SELECT value FROM lists WHERE key = ? ORDER BY "index" ASC', params: [key] },
  ];

  return {
    statements,
    mapResult: (results) => {
      if (wrongTypeFor(results, 1, "list")) {
        throw ErrWrongType;
      }
      if (results[2].length === 0) {
        return [];
      }
      const values = results[2].map((row) => rowString(row, "value"));
      const length = values.length;
      let startIndex = start;
      let stopIndex = stop;
      if (startIndex < 0) {
        startIndex = length + startIndex;
      }
      if (stopIndex < 0) {
        stopIndex = length + stopIndex;
      }
      if (startIndex < 0) {
        startIndex = 0;
      }
      if (stopIndex > length - 1) {
        stopIndex = length - 1;
      }
      if (startIndex > stopIndex || startIndex >= length || stopIndex < 0) {
        return [];
      }
      return values.slice(startIndex, stopIndex + 1);
    },
  };
}

export function translateLLen(args: string[]): Translation {
  if (args.length !== 1) {
    throw errWrongArgs("llen");
  }

  const key = args[0];
  const statements: Statement[] = [
    deleteIfExpired(key),
    { sql: "SELECT type FROM keys WHERE key = ?", params: [key] },
    { sql: "SELECT COUNT(*) as count FROM lists WHERE key = ?", params: [key] },
  ];

  return {
    statements,
    mapResult: (results) => {
      if (wrongTypeFor(results, 1, "list")) {
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
