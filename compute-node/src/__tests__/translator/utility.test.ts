import { describe, expect, it } from "vitest";
import { translate } from "../../translator";
import type { ResultRow } from "../../translator/types";

describe("utility translator", () => {
  it("PING returns PONG", () => {
    const translation = translate(["PING"]);
    expect(translation.statements).toHaveLength(0);
    const results: ResultRow[][] = [];
    expect(translation.mapResult(results)).toBe("PONG");
  });
});
