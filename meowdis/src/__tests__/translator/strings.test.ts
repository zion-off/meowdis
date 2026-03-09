import { describe, expect, it } from "vitest";
import { translate } from "../../translator";
import { ErrNotInteger } from "../../translator/errors";
import type { ResultRow } from "../../translator/types";

describe("strings translator", () => {
  it("GET missing key returns null", () => {
    const translation = translate(["GET", "missing"]);
    expect(translation.statements).toHaveLength(3);
    const results: ResultRow[][] = [[], [], []];
    expect(translation.mapResult(results)).toBeNull();
  });

  it("SET basic returns OK", () => {
    const translation = translate(["SET", "k", "v"]);
    expect(translation.statements).toHaveLength(4);
    const results: ResultRow[][] = [[], [], [], []];
    expect(translation.mapResult(results)).toBe("OK");
  });

  it("SET NX when key exists returns null", () => {
    const translation = translate(["SET", "k", "v", "NX"]);
    expect(translation.statements).toHaveLength(3);
    const results: ResultRow[][] = [[], [], []];
    expect(translation.mapResult(results)).toBeNull();
  });

  it("SET XX when key missing returns null", () => {
    const translation = translate(["SET", "k", "v", "XX"]);
    expect(translation.statements).toHaveLength(3);
    const results: ResultRow[][] = [[], [], []];
    expect(translation.mapResult(results)).toBeNull();
  });

  it("SET EX 10 sets expires_at param", () => {
    const translation = translate(["SET", "k", "v", "EX", "10"]);
    const hasExpiryValue = translation.statements.some((stmt) =>
      stmt.params.some((param) => typeof param === "number" && Number.isFinite(param))
    );
    expect(hasExpiryValue).toBe(true);
  });

  it("INCR returns new value from RETURNING", () => {
    const translation = translate(["INCR", "k"]);
    expect(translation.statements).toHaveLength(6);
    const results: ResultRow[][] = [[], [], [], [], [{ value: "0" }], [{ value: 1 }]];
    expect(translation.mapResult(results)).toBe(1);
  });

  it("INCRBY uses delta", () => {
    const translation = translate(["INCRBY", "k", "5"]);
    expect(translation.statements).toHaveLength(6);
    const results: ResultRow[][] = [[], [], [], [], [{ value: "0" }], [{ value: 5 }]];
    expect(translation.mapResult(results)).toBe(5);
  });

  it("INCR throws on non-integer key", () => {
    const translation = translate(["INCR", "k"]);
    const results: ResultRow[][] = [[], [], [], [], [], []];
    expect(() => translation.mapResult(results)).toThrow(ErrNotInteger);
  });

  it("rejects wrong args and invalid options", () => {
    expect(() => translate(["GET"])).toThrow("ERR wrong number of arguments for 'get' command");
    expect(() => translate(["SET", "k", "v", "NX", "XX"])).toThrow(
      "ERR wrong number of arguments for 'set' command"
    );
  });

  it("rejects unknown command", () => {
    expect(() => translate(["NOPE"])).toThrow("ERR unknown command 'nope'");
  });
});
