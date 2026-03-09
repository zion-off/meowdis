import { describe, expect, it } from "vitest";
import { translate } from "../../translator";
import type { ResultRow } from "../../translator/types";

function expectIndexQuoted(sql: string): void {
  expect(sql).toContain('"index"');
  const stripped = sql.replaceAll('"index"', "");
  expect(stripped.includes("index")).toBe(false);
}

describe("lists translator", () => {
  it("LPUSH returns count", () => {
    const translation = translate(["LPUSH", "k", "a", "b"]);
    expect(translation.statements).toHaveLength(6);
    const results: ResultRow[][] = [[], [], [], [], [], [{ count: "2" }]];
    expect(translation.mapResult(results)).toBe(2);
  });

  it("RPUSH returns count", () => {
    const translation = translate(["RPUSH", "k", "a"]);
    expect(translation.statements).toHaveLength(5);
    const results: ResultRow[][] = [[], [], [], [], [{ count: "1" }]];
    expect(translation.mapResult(results)).toBe(1);
  });

  it("LPOP uses quoted index", () => {
    const translation = translate(["LPOP", "k"]);
    expectIndexQuoted(translation.statements[2].sql);
    const results: ResultRow[][] = [[], [], [{ value: "v" }], []];
    expect(translation.mapResult(results)).toBe("v");
  });

  it("RPOP uses quoted index", () => {
    const translation = translate(["RPOP", "k"]);
    expectIndexQuoted(translation.statements[2].sql);
    const results: ResultRow[][] = [[], [], [{ value: "v" }], []];
    expect(translation.mapResult(results)).toBe("v");
  });

  it("LRANGE handles positive and negative indices", () => {
    const translation = translate(["LRANGE", "k", "1", "2"]);
    const results: ResultRow[][] = [
      [],
      [],
      [{ value: "a" }, { value: "b" }, { value: "c" }, { value: "d" }],
    ];
    expect(translation.mapResult(results)).toEqual(["b", "c"]);

    const negative = translate(["LRANGE", "k", "-2", "-1"]);
    expect(negative.mapResult(results)).toEqual(["c", "d"]);
  });
});
