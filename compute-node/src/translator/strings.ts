import type { Statement, Translation, ResultRow } from "./types";
import { ErrNotInteger, ErrWrongType, errWrongArgs } from "./errors";
import { deleteIfExpired, parseIntStrict, rowString, wrongTypeFor } from "./helpers";

function hasWrongType(results: ResultRow[][], index: number): boolean {
  return wrongTypeFor(results, index, "string");
}

type SetOptions = {
  nx: boolean;
  xx: boolean;
  get: boolean;
  keepTTL: boolean;
  expiresAt: number | null;
};

function parseSetOptions(args: string[]): SetOptions {
  const options: SetOptions = {
    nx: false,
    xx: false,
    get: false,
    keepTTL: false,
    expiresAt: null,
  };
  let hasExpiry = false;

  for (let i = 0; i < args.length; i++) {
    const token = args[i].toUpperCase();
    switch (token) {
      case "NX":
        options.nx = true;
        break;
      case "XX":
        options.xx = true;
        break;
      case "GET":
        options.get = true;
        break;
      case "KEEPTTL":
        options.keepTTL = true;
        break;
      case "EX":
      case "PX":
      case "EXAT":
      case "PXAT": {
        if (hasExpiry) {
          throw errWrongArgs("set");
        }
        if (i + 1 >= args.length) {
          throw errWrongArgs("set");
        }
        const parsed = parseIntStrict(args[i + 1]);
        if (parsed === null) {
          throw ErrNotInteger;
        }
        const now = Math.floor(Date.now() / 1000);
        let expiresAt: number;
        switch (token) {
          case "EX":
            expiresAt = now + parsed;
            break;
          case "PX":
            expiresAt = now + Math.trunc(parsed / 1000);
            break;
          case "EXAT":
            expiresAt = parsed;
            break;
          case "PXAT":
            expiresAt = Math.trunc(parsed / 1000);
            break;
          default:
            throw errWrongArgs("set");
        }
        options.expiresAt = expiresAt;
        hasExpiry = true;
        i++;
        break;
      }
      default:
        throw errWrongArgs("set");
    }
  }

  if (options.nx && options.xx) {
    throw errWrongArgs("set");
  }
  if (options.keepTTL && hasExpiry) {
    throw errWrongArgs("set");
  }

  return options;
}

export function translateGet(args: string[]): Translation {
  if (args.length !== 1) {
    throw errWrongArgs("get");
  }

  const key = args[0];
  const statements: Statement[] = [
    deleteIfExpired(key),
    { sql: "SELECT type FROM keys WHERE key = ?", params: [key] },
    { sql: "SELECT value FROM strings WHERE key = ?", params: [key] },
  ];

  return {
    statements,
    mapResult: (results) => {
      if (hasWrongType(results, 1)) {
        throw ErrWrongType;
      }
      if (results[2].length === 0) {
        return null;
      }
      return rowString(results[2][0], "value");
    },
  };
}

export function translateSet(args: string[]): Translation {
  if (args.length < 2) {
    throw errWrongArgs("set");
  }

  const key = args[0];
  const value = args[1];
  const options = parseSetOptions(args.slice(2));

  let expiresAt: number | null = null;
  if (!options.keepTTL) {
    expiresAt = options.expiresAt ?? null;
  }

  const statements: Statement[] = [deleteIfExpired(key)];
  let getIndex = -1;
  if (options.get) {
    getIndex = statements.length;
    statements.push({ sql: "SELECT value FROM strings WHERE key = ?", params: [key] });
  }

  let okIndex = -1;
  if (options.nx) {
    okIndex = statements.length;
    statements.push(
      {
        sql: "INSERT OR IGNORE INTO keys (key, type, expires_at) VALUES (?, 'string', ?) RETURNING key",
        params: [key, expiresAt],
      },
      {
        sql: "INSERT INTO strings (key, value) SELECT ?, ? WHERE changes() > 0",
        params: [key, value],
      }
    );
  } else if (options.xx) {
    okIndex = statements.length;
    if (options.keepTTL) {
      statements.push({
        sql: "UPDATE keys SET type = 'string' WHERE key = ? RETURNING key",
        params: [key],
      });
    } else {
      statements.push({
        sql: "UPDATE keys SET type = 'string', expires_at = ? WHERE key = ? RETURNING key",
        params: [expiresAt, key],
      });
    }
    statements.push({
      sql: "INSERT OR REPLACE INTO strings (key, value) SELECT ?, ? WHERE changes() > 0",
      params: [key, value],
    });
  } else {
    if (options.keepTTL) {
      statements.push(
        {
          sql: "UPDATE keys SET type = 'string' WHERE key = ? RETURNING key",
          params: [key],
        },
        {
          sql: "INSERT INTO keys (key, type) SELECT ?, 'string' WHERE changes() = 0",
          params: [key],
        },
        {
          sql: "INSERT OR REPLACE INTO strings (key, value) VALUES (?, ?)",
          params: [key, value],
        }
      );
    } else {
      statements.push(
        { sql: "DELETE FROM keys WHERE key = ?", params: [key] },
        {
          sql: "INSERT INTO keys (key, type, expires_at) VALUES (?, 'string', ?)",
          params: [key, expiresAt],
        },
        {
          sql: "INSERT INTO strings (key, value) VALUES (?, ?)",
          params: [key, value],
        }
      );
    }
  }

  return {
    statements,
    mapResult: (results) => {
      let oldValue: unknown = null;
      if (getIndex >= 0) {
        if (results[getIndex].length === 0) {
          oldValue = null;
        } else {
          oldValue = rowString(results[getIndex][0], "value");
        }
      }

      if (okIndex >= 0 && results[okIndex].length === 0) {
        return null;
      }
      if (options.get) {
        return oldValue;
      }
      return "OK";
    },
  };
}

export function translateDel(args: string[]): Translation {
  if (args.length === 0) {
    throw errWrongArgs("del");
  }

  const statements = args.map((key) => ({
    sql: "DELETE FROM keys WHERE key = ? RETURNING key",
    params: [key],
  }));

  return {
    statements,
    mapResult: (results) => {
      let count = 0;
      for (const res of results) {
        count += res.length;
      }
      return count;
    },
  };
}

export function translateExists(args: string[]): Translation {
  if (args.length === 0) {
    throw errWrongArgs("exists");
  }

  const statements = args.map((key) => ({
    sql: "SELECT 1 FROM keys WHERE key = ? AND (expires_at IS NULL OR expires_at > unixepoch())",
    params: [key],
  }));

  return {
    statements,
    mapResult: (results) => {
      let count = 0;
      for (const res of results) {
        if (res.length > 0) {
          count++;
        }
      }
      return count;
    },
  };
}

function translateIncrByDelta(args: string[], delta: number, cmd: string): Translation {
  if (args.length !== 1) {
    throw errWrongArgs(cmd);
  }

  const key = args[0];
  const statements: Statement[] = [
    deleteIfExpired(key),
    { sql: "SELECT type FROM keys WHERE key = ?", params: [key] },
    { sql: "INSERT OR IGNORE INTO keys (key, type) VALUES (?, 'string')", params: [key] },
    { sql: "INSERT OR IGNORE INTO strings (key, value) VALUES (?, '0')", params: [key] },
    { sql: "SELECT value FROM strings WHERE key = ?", params: [key] },
    {
      sql: "UPDATE strings SET value = CAST(CAST(value AS INTEGER) + ? AS INTEGER) WHERE key = ? RETURNING value",
      params: [delta, key],
    },
  ];

  return {
    statements,
    mapResult: (results) => {
      if (hasWrongType(results, 1)) {
        throw ErrWrongType;
      }
      if (results[4].length === 0) {
        throw ErrNotInteger;
      }
      const before = rowString(results[4][0], "value");
      if (parseIntStrict(before) === null) {
        throw ErrNotInteger;
      }
      if (results[5].length === 0) {
        throw ErrNotInteger;
      }
      const after = rowString(results[5][0], "value");
      const value = parseIntStrict(after);
      if (value === null) {
        throw ErrNotInteger;
      }
      return value;
    },
  };
}

export function translateIncr(args: string[]): Translation {
  return translateIncrByDelta(args, 1, "incr");
}

export function translateIncrBy(args: string[]): Translation {
  if (args.length !== 2) {
    throw errWrongArgs("incrby");
  }
  const delta = parseIntStrict(args[1]);
  if (delta === null) {
    throw ErrNotInteger;
  }
  return translateIncrByDelta(args.slice(0, 1), delta, "incrby");
}

export function translateDecr(args: string[]): Translation {
  return translateIncrByDelta(args, -1, "decr");
}

export function translateDecrBy(args: string[]): Translation {
  if (args.length !== 2) {
    throw errWrongArgs("decrby");
  }
  const delta = parseIntStrict(args[1]);
  if (delta === null) {
    throw ErrNotInteger;
  }
  return translateIncrByDelta(args.slice(0, 1), -delta, "decrby");
}

export function translateMGet(args: string[]): Translation {
  if (args.length === 0) {
    throw errWrongArgs("mget");
  }

  const statements: Statement[] = [];
  for (const key of args) {
    statements.push(deleteIfExpired(key));
    statements.push({ sql: "SELECT value FROM strings WHERE key = ?", params: [key] });
  }

  return {
    statements,
    mapResult: (results) => {
      const out: unknown[] = [];
      for (let i = 0; i < args.length; i++) {
        const res = results[i * 2 + 1];
        if (res.length === 0) {
          out.push(null);
        } else {
          out.push(rowString(res[0], "value"));
        }
      }
      return out;
    },
  };
}

export function translateMSet(args: string[]): Translation {
  if (args.length === 0 || args.length % 2 !== 0) {
    throw errWrongArgs("mset");
  }

  const statements: Statement[] = [];
  for (let i = 0; i < args.length; i += 2) {
    const key = args[i];
    const value = args[i + 1];
    statements.push(
      { sql: "DELETE FROM keys WHERE key = ?", params: [key] },
      { sql: "INSERT INTO keys (key, type) VALUES (?, 'string')", params: [key] },
      { sql: "INSERT INTO strings (key, value) VALUES (?, ?)", params: [key, value] }
    );
  }

  return {
    statements,
    mapResult: () => "OK",
  };
}
