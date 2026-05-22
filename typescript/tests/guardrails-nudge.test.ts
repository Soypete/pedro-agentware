import {
  NudgeKind,
  retryNudge,
  unknownToolNudge,
  stepNudge,
  prerequisiteNudge,
} from "../src/middleware/guardrails/nudge";

describe("NudgeKind", () => {
  it("should have correct values", () => {
    expect(NudgeKind.RETRY).toBe("retry");
    expect(NudgeKind.UNKNOWN_TOOL).toBe("unknown_tool");
    expect(NudgeKind.STEP).toBe("step");
    expect(NudgeKind.PREREQUISITE).toBe("prerequisite");
  });
});

describe("retryNudge", () => {
  it("should create a retry nudge", () => {
    const tools = ["get_weather", "echo", "search"];
    const nudge = retryNudge("some text", tools);

    expect(nudge.role).toBe("user");
    expect(nudge.kind).toBe(NudgeKind.RETRY);
    expect(nudge.tier).toBe(0);
    expect(nudge.content).toContain("get_weather");
  });

  it("should handle empty tools list", () => {
    const nudge = retryNudge("text", []);
    expect(nudge.content).toContain("(no tools available)");
  });
});

describe("unknownToolNudge", () => {
  it("should create an unknown tool nudge", () => {
    const tools = ["echo", "search"];
    const nudge = unknownToolNudge("nonexistent", tools);

    expect(nudge.role).toBe("user");
    expect(nudge.kind).toBe(NudgeKind.UNKNOWN_TOOL);
    expect(nudge.content).toContain("nonexistent");
  });
});

describe("stepNudge", () => {
  it("should create tier 1 nudge", () => {
    const pending = ["validate", "prepare"];
    const nudge = stepNudge("submit", pending, 1);

    expect(nudge.kind).toBe(NudgeKind.STEP);
    expect(nudge.tier).toBe(1);
  });

  it("should create tier 2 nudge", () => {
    const pending = ["validate", "prepare"];
    const nudge = stepNudge("submit", pending, 2);

    expect(nudge.tier).toBe(2);
  });

  it("should create tier 3 nudge", () => {
    const pending = ["validate"];
    const nudge = stepNudge("submit", pending, 3);

    expect(nudge.tier).toBe(3);
  });

  it("should clamp tier below minimum", () => {
    const nudge = stepNudge("submit", ["validate"], 0);
    expect(nudge.tier).toBe(1);
  });

  it("should clamp tier above maximum", () => {
    const nudge = stepNudge("submit", ["validate"], 10);
    expect(nudge.tier).toBe(3);
  });
});

describe("prerequisiteNudge", () => {
  it("should create a prerequisite nudge", () => {
    const missing = ["authenticate", "validate"];
    const nudge = prerequisiteNudge("submit", missing);

    expect(nudge.kind).toBe(NudgeKind.PREREQUISITE);
    expect(nudge.tier).toBe(0);
    expect(nudge.content).toContain("authenticate");
  });
});