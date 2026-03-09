import { describe, expect, it } from "vitest";
import { translate } from "../../translator";
import type { ResultRow } from "../../translator/types";

describe("expiry translator", () => {
  it("EXPIRE returns 1 when updated", () => {
    const translation = translate(["EXPIRE", "k", "10"]);
    expect(translation.statements).toHaveLength(3);
    const results: ResultRow[][] = [[], [{ expires_at: 1 }], [{ key: "k" }]];
    expect(translation.mapResult(results)).toBe(1);
  });

  it("PERSIST returns 0 when nothing updated", () => {
    const translation = translate(["PERSIST", "k"]);
    expect(translation.statements).toHaveLength(2);
    const results: ResultRow[][] = [[], []];
    expect(translation.mapResult(results)).toBe(0);
  });

  it("TTL and PTTL return expected values", () => {
    const ttl = translate(["TTL", "k"]);
    const pttl = translate(["PTTL", "k"]);
    const results: ResultRow[][] = [[], [{ expires_at: "110", now: "100" }]];
    expect(ttl.mapResult(results)).toBe(10);
    expect(pttl.mapResult(results)).toBe(10000);
  });

  it("TTL handles missing and persistent keys", () => {
    const ttl = translate(["TTL", "k"]);
    const missing: ResultRow[][] = [[], []];
    expect(ttl.mapResult(missing)).toBe(-2);

    const persistent: ResultRow[][] = [[], [{ expires_at: null, now: "100" }]];
    expect(ttl.mapResult(persistent)).toBe(-1);
  });
});
