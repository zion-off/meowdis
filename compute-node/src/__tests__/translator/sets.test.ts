import { describe, expect, it } from "vitest";
import { translate } from "../../translator";
import type { ResultRow } from "../../translator/types";

describe("sets translator", () => {
  it("SADD returns count", () => {
    const translation = translate(["SADD", "k", "a", "b"]);
    expect(translation.statements).toHaveLength(5);
    const results: ResultRow[][] = [[], [], [], [{ member: "a" }], [{ member: "b" }]];
    expect(translation.mapResult(results)).toBe(2);
  });

  it("SMEMBERS returns members", () => {
    const translation = translate(["SMEMBERS", "k"]);
    const results: ResultRow[][] = [[], [], [{ member: "a" }, { member: "b" }]];
    expect(translation.mapResult(results)).toEqual(["a", "b"]);
  });

  it("SREM returns count", () => {
    const translation = translate(["SREM", "k", "a"]);
    expect(translation.statements).toHaveLength(4);
    const results: ResultRow[][] = [[], [], [{ member: "a" }], []];
    expect(translation.mapResult(results)).toBe(1);
  });
});
