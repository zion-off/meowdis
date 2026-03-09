import { describe, expect, it } from "vitest";
import { translate } from "../../translator";
import type { ResultRow } from "../../translator/types";

describe("hashes translator", () => {
  it("HSET returns count", () => {
    const translation = translate(["HSET", "k", "f", "v"]);
    expect(translation.statements).toHaveLength(5);
    const results: ResultRow[][] = [[], [], [], [{ field: "f" }], []];
    expect(translation.mapResult(results)).toBe(1);
  });

  it("HGET missing returns null", () => {
    const translation = translate(["HGET", "k", "f"]);
    expect(translation.statements).toHaveLength(3);
    const results: ResultRow[][] = [[], [], []];
    expect(translation.mapResult(results)).toBeNull();
  });

  it("HGETALL returns flattened list", () => {
    const translation = translate(["HGETALL", "k"]);
    const results: ResultRow[][] = [
      [],
      [],
      [
        { field: "a", value: "1" },
        { field: "b", value: "2" },
      ],
    ];
    expect(translation.mapResult(results)).toEqual(["a", "1", "b", "2"]);
  });
});
