import type { Statement, Translation } from "./types";
import { ErrNotInteger, errWrongArgs } from "./errors";
import { deleteIfExpired, parseIntStrict, rowString } from "./helpers";

export function translateExpireWith(args: string[], cmd: string, absolute: boolean): Translation {
  if (args.length < 2 || args.length > 3) {
    throw errWrongArgs(cmd);
  }

  const key = args[0];
  const value = parseIntStrict(args[1]);
  if (value === null) {
    throw ErrNotInteger;
  }

  const expiresAt = absolute ? value : Math.floor(Date.now() / 1000) + value;
  const option = args.length === 3 ? args[2].toUpperCase() : "";

  let updateSQL = "UPDATE keys SET expires_at = ? WHERE key = ? RETURNING key";
  let params: unknown[] = [expiresAt, key];

  switch (option) {
    case "":
      break;
    case "NX":
      updateSQL =
        "UPDATE keys SET expires_at = ? WHERE key = ? AND expires_at IS NULL RETURNING key";
      break;
    case "XX":
      updateSQL =
        "UPDATE keys SET expires_at = ? WHERE key = ? AND expires_at IS NOT NULL RETURNING key";
      break;
    case "GT":
      updateSQL =
        "UPDATE keys SET expires_at = ? WHERE key = ? AND (expires_at IS NULL OR expires_at < ?) RETURNING key";
      params = [expiresAt, key, expiresAt];
      break;
    case "LT":
      updateSQL =
        "UPDATE keys SET expires_at = ? WHERE key = ? AND expires_at IS NOT NULL AND expires_at > ? RETURNING key";
      params = [expiresAt, key, expiresAt];
      break;
    default:
      throw errWrongArgs(cmd);
  }

  const statements: Statement[] = [
    deleteIfExpired(key),
    { sql: "SELECT expires_at FROM keys WHERE key = ?", params: [key] },
    { sql: updateSQL, params },
  ];

  return {
    statements,
    mapResult: (results) => {
      if (results[1].length === 0) {
        return 0;
      }
      if (results[2].length === 0) {
        return 0;
      }
      return 1;
    },
  };
}

export function translateExpire(args: string[]): Translation {
  return translateExpireWith(args, "expire", false);
}

export function translateExpireAt(args: string[]): Translation {
  return translateExpireWith(args, "expireat", true);
}

export function translateTTLWith(args: string[], cmd: string, millis: boolean): Translation {
  if (args.length !== 1) {
    throw errWrongArgs(cmd);
  }

  const key = args[0];
  const statements: Statement[] = [
    deleteIfExpired(key),
    { sql: "SELECT expires_at, unixepoch() as now FROM keys WHERE key = ?", params: [key] },
  ];

  return {
    statements,
    mapResult: (results) => {
      if (results[1].length === 0) {
        return -2;
      }
      const row = results[1][0];
      const expiresValue = row["expires_at"];
      if (expiresValue === null || expiresValue === undefined) {
        return -1;
      }
      const expiresAt = parseIntStrict(rowString(row, "expires_at"));
      const now = parseIntStrict(rowString(row, "now"));
      if (expiresAt === null || now === null) {
        throw ErrNotInteger;
      }
      const ttl = expiresAt - now;
      return millis ? ttl * 1000 : ttl;
    },
  };
}

export function translateTTL(args: string[]): Translation {
  return translateTTLWith(args, "ttl", false);
}

export function translatePTTL(args: string[]): Translation {
  return translateTTLWith(args, "pttl", true);
}

export function translatePersist(args: string[]): Translation {
  if (args.length !== 1) {
    throw errWrongArgs("persist");
  }

  const key = args[0];
  const statements: Statement[] = [
    deleteIfExpired(key),
    {
      sql: "UPDATE keys SET expires_at = NULL WHERE key = ? AND expires_at IS NOT NULL RETURNING key",
      params: [key],
    },
  ];

  return {
    statements,
    mapResult: (results) => (results[1].length === 0 ? 0 : 1),
  };
}
