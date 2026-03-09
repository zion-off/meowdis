import type { Statement, Translation } from "./types";
import { ErrNotInteger, errWrongArgs } from "./errors";
import { parseIntStrict, rowString } from "./helpers";

function escapeRegExpChar(value: string): string {
  return value.replace(/[\\^$.*+?()|{}[\]]/g, "\\$&");
}

function escapeClassChar(value: string): string {
  return value.replace(/[\\\]\\-^]/g, "\\$&");
}

function pathMatch(pattern: string, value: string): boolean {
  let regex = "^";
  for (let i = 0; i < pattern.length; i++) {
    const ch = pattern[i];
    if (ch === "*") {
      regex += "[^/]*";
      continue;
    }
    if (ch === "?") {
      regex += "[^/]";
      continue;
    }
    if (ch === "[") {
      let j = i + 1;
      if (j >= pattern.length) {
        throw new Error("bad pattern");
      }
      let negate = false;
      if (pattern[j] === "!" || pattern[j] === "^") {
        negate = true;
        j++;
      }
      let classBody = "";
      let closed = false;
      for (; j < pattern.length; j++) {
        const c = pattern[j];
        if (c === "]" && classBody.length > 0) {
          closed = true;
          break;
        }
        if (c === "\\") {
          if (j + 1 >= pattern.length) {
            throw new Error("bad pattern");
          }
          classBody += escapeClassChar(pattern[j + 1]);
          j++;
          continue;
        }
        classBody += escapeClassChar(c);
      }
      if (!closed) {
        throw new Error("bad pattern");
      }
      regex += `[${negate ? "^" : ""}${classBody}]`;
      i = j;
      continue;
    }
    if (ch === "\\") {
      if (i + 1 >= pattern.length) {
        throw new Error("bad pattern");
      }
      regex += escapeRegExpChar(pattern[i + 1]);
      i++;
      continue;
    }
    regex += escapeRegExpChar(ch);
  }
  regex += "$";
  return new RegExp(regex).test(value);
}

export function translatePing(args: string[]): Translation {
  if (args.length > 1) {
    throw errWrongArgs("ping");
  }

  const message = args.length === 1 ? args[0] : "PONG";
  return {
    statements: [],
    mapResult: () => message,
  };
}

export function translateDBSize(args: string[]): Translation {
  if (args.length !== 0) {
    throw errWrongArgs("dbsize");
  }

  const statements: Statement[] = [
    {
      sql: "SELECT COUNT(*) as count FROM keys WHERE (expires_at IS NULL OR expires_at > unixepoch())",
      params: [],
    },
  ];

  return {
    statements,
    mapResult: (results) => {
      if (results[0].length === 0) {
        return 0;
      }
      const count = parseIntStrict(rowString(results[0][0], "count"));
      if (count === null) {
        throw ErrNotInteger;
      }
      return count;
    },
  };
}

export function translateFlushDB(args: string[]): Translation {
  if (args.length !== 0) {
    throw errWrongArgs("flushdb");
  }

  const statements: Statement[] = [{ sql: "DELETE FROM keys", params: [] }];
  return {
    statements,
    mapResult: () => "OK",
  };
}

export function translateKeys(args: string[]): Translation {
  if (args.length !== 1) {
    throw errWrongArgs("keys");
  }
  const pattern = args[0];
  if (pattern === "") {
    throw errWrongArgs("keys");
  }

  const statements: Statement[] = [
    {
      sql: "SELECT key FROM keys WHERE (expires_at IS NULL OR expires_at > unixepoch()) ORDER BY key",
      params: [],
    },
  ];

  return {
    statements,
    mapResult: (results) => {
      const out: unknown[] = [];
      for (const row of results[0]) {
        const key = rowString(row, "key");
        if (pathMatch(pattern, key)) {
          out.push(key);
        }
      }
      return out;
    },
  };
}
