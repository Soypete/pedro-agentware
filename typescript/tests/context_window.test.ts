import { describe, it, expect } from "@jest/globals";
import { Message, Role } from "../src/llm/request.js";
import {
  ContextWindowManager,
  defaultCounter,
  defaultContextWarning,
} from "../src/llmcontext/context_window.js";

describe("ContextWindowManager_UpdateTokenCount", () => {
  it("records actual token count from backend", () => {
    const mgr = new ContextWindowManager(1000, defaultCounter);

    mgr.updateTokenCount(500);

    const [tokens, atThreshold] = mgr.check([
      { role: Role.USER, content: "test" },
    ]);
    expect(tokens).toBe(500);
    expect(atThreshold).toBe(false);
  });
});

describe("ContextWindowManager_Check_UsesActualCount", () => {
  it("uses counter when no actual count provided", () => {
    const counter = (_messages: Message[]): number => 800;
    const mgr = new ContextWindowManager(1000, counter);
    mgr.setCompactionRatio(0.75);

    const [tokens, atThreshold] = mgr.check([{ role: Role.USER, content: "test" }]);
    expect(tokens).toBe(800);
    expect(atThreshold).toBe(true);
  });

  it("uses actual count when provided", () => {
    const counter = (_messages: Message[]): number => 800;
    const mgr = new ContextWindowManager(1000, counter);
    mgr.setCompactionRatio(0.75);

    mgr.updateTokenCount(600);

    const [tokens, atThreshold] = mgr.check([{ role: Role.USER, content: "test" }]);
    expect(tokens).toBe(600);
    expect(atThreshold).toBe(false);
  });
});

describe("ContextWindowManager_ShouldCompact_UsesActualCount", () => {
  it("uses counter when no actual count provided", () => {
    const counter = (_messages: Message[]): number => 900;
    const mgr = new ContextWindowManager(1000, counter);
    mgr.setCompactionRatio(0.75);

    expect(mgr.shouldCompact([{ role: Role.USER, content: "test" }])).toBe(true);
  });

  it("uses actual count when provided", () => {
    const counter = (_messages: Message[]): number => 900;
    const mgr = new ContextWindowManager(1000, counter);
    mgr.setCompactionRatio(0.75);

    mgr.updateTokenCount(700);

    expect(mgr.shouldCompact([{ role: Role.USER, content: "test" }])).toBe(false);
  });
});

describe("ContextWindowManager_Compact_ResetsTokenCount", () => {
  it("resets lastKnownTokens after compaction", () => {
    const mgr = new ContextWindowManager(1000, defaultCounter);
    mgr.setCompactionRatio(0.75);

    mgr.updateTokenCount(1500);

    const compacted = mgr.compact([{ role: Role.USER, content: "short" }]);

    const [tokens] = mgr.check(compacted);
    expect(tokens).not.toBe(1500);
    expect(tokens).toBeLessThan(100);
  });
});

describe("ContextWindowManager_ThreadSafety", () => {
  it("handles concurrent updates and reads", async () => {
    const mgr = new ContextWindowManager(1000, defaultCounter);

    const updates: Promise<void>[] = [];
    for (let i = 0; i < 100; i++) {
      updates.push(
        (async () => {
          for (let j = 0; j < 10; j++) {
            mgr.updateTokenCount(i * 10 + j);
          }
        })()
      );
    }

    const reads: Promise<void>[] = [];
    for (let i = 0; i < 100; i++) {
      reads.push(
        (async () => {
          mgr.check([{ role: Role.USER, content: "test" }]);
        })()
      );
    }

    await Promise.all([...updates, ...reads]);
  });
});

describe("defaultCounter", () => {
  it("calculates token count using character-based estimation", () => {
    const messages: Message[] = [
      { role: Role.USER, content: "Hello world test content" },
      { role: Role.ASSISTANT, content: "Response with some text" },
    ];

    const count = defaultCounter(messages);

    const expected =
      Math.floor("Hello world test content".length / 4) +
      "user".length +
      4 +
      (Math.floor("Response with some text".length / 4) +
        "assistant".length +
        4);

    expect(count).toBe(expected);
  });
});

describe("ContextWindowManager_CheckThresholds", () => {
  it("fires once per threshold", () => {
    const counter = (_messages: Message[]): number => 700;
    const mgr = new ContextWindowManager(1000, counter, null, [0.5, 0.65]);

    const warning1 = mgr.checkThresholds([{ role: Role.USER, content: "test" }]);
    expect(warning1).not.toBeNull();
    expect(warning1).toContain("filling up");

    const warning2 = mgr.checkThresholds([{ role: Role.USER, content: "test" }]);
    expect(warning2).not.toBeNull();
    expect(warning2).toContain("filling up");
  });

  it("resets after compact", () => {
    const mgr = new ContextWindowManager(1000, defaultCounter);
    mgr.setCompactionRatio(0.75);

    mgr.checkThresholds([{ role: Role.USER, content: "test" }]);

    mgr.compact([{ role: Role.USER, content: "short" }]);

    mgr.updateTokenCount(850);

    const warning = mgr.checkThresholds([{ role: Role.USER, content: "test" }]);
    expect(warning).not.toBeNull();
  });

  it("highest threshold fires first", () => {
    const counter = (_messages: Message[]): number => 900;
    const mgr = new ContextWindowManager(
      1000,
      counter,
      null,
      [0.5, 0.8, 0.65]
    );

    const warning = mgr.checkThresholds([{ role: Role.USER, content: "test" }]);
    expect(warning).not.toBeNull();
    expect(warning).toContain("nearly full");
  });

  it("default thresholds", () => {
    const counter = (_messages: Message[]): number => 700;
    const mgr = new ContextWindowManager(1000, counter);

    const warning = mgr.checkThresholds([{ role: Role.USER, content: "test" }]);
    expect(warning).not.toBeNull();
    expect(warning).toContain("filling up");
  });

  it("custom callback", () => {
    const counter = (_messages: Message[]): number => 700;
    const customCb = (_tokens: number, _budget: number, _pct: number): string | null => "Custom warning!";
    const mgr = new ContextWindowManager(
      1000,
      counter,
      null,
      [0.5],
      customCb
    );

    const warning = mgr.checkThresholds([{ role: Role.USER, content: "test" }]);
    expect(warning).toBe("Custom warning!");
  });

  it("zero tokens returns null", () => {
    const mgr = new ContextWindowManager(1000, () => 0);

    const warning = mgr.checkThresholds([{ role: Role.USER, content: "test" }]);
    expect(warning).toBeNull();
  });
});

describe("ContextWindowManager_ThreadSafety_CheckThresholds", () => {
  it("concurrent check thresholds", async () => {
    const counter = (_messages: Message[]): number => 700;
    const mgr = new ContextWindowManager(1000, counter, null, [0.5, 0.65]);

    const calls: Promise<string | null>[] = [];
    for (let i = 0; i < 50; i++) {
      calls.push(Promise.resolve(mgr.checkThresholds([{ role: Role.USER, content: "test" }])));
    }

    const results = await Promise.all(calls);
    const nonNull = results.filter((r) => r !== null).length;
    expect(nonNull).toBeGreaterThan(0);
  });
});