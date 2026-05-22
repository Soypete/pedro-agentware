import { describe, it, expect } from "@jest/globals";
import { Message, Role } from "../src/llm/request.js";
import { MessageType } from "../src/middleware/types.js";
import { TieredCompact, findEligibleEnd } from "../src/llmcontext/strategies.js";

const simpleTokenCounter = (messages: Message[]): number => {
  return Math.floor(
    messages.reduce((sum, m) => sum + m.content.length, 0) / 4
  );
};

describe("TieredCompact", () => {
  describe("name", () => {
    it("returns TieredCompact", () => {
      const compact = new TieredCompact();
      expect(compact.name()).toBe("TieredCompact");
    });
  });

  describe("empty messages", () => {
    it("returns empty array for empty input", () => {
      const compact = new TieredCompact();
      const result = compact.compact([], 100, simpleTokenCounter);
      expect(result).toEqual([]);
    });
  });

  describe("phase1 - drops nudges", () => {
    it("drops step nudge in phase 1", () => {
      const compact = new TieredCompact(0);

      const messages: Message[] = [
        {
          role: Role.SYSTEM,
          content: "system",
          meta: { type: MessageType.SYSTEM_PROMPT },
        },
        {
          role: Role.USER,
          content: "user",
          meta: { type: MessageType.USER_INPUT },
        },
        {
          role: Role.ASSISTANT,
          content: "",
          meta: { type: MessageType.STEP_NUDGE, step_index: 0 },
        },
        {
          role: Role.ASSISTANT,
          content: "text",
          meta: { type: MessageType.TEXT_RESPONSE, step_index: 0 },
        },
      ];

      const result = compact.compact(messages, 2, simpleTokenCounter);

      const nudgeFound = result.some((m) => m.meta?.type === MessageType.STEP_NUDGE);
      expect(nudgeFound).toBe(false);
    });
  });

  describe("phase1 - truncates tool results", () => {
    it("truncates tool result to truncateChars", () => {
      const compact = new TieredCompact(0, 200);

      const longContent = "x".repeat(500);

      const messages: Message[] = [
        {
          role: Role.SYSTEM,
          content: "system",
          meta: { type: MessageType.SYSTEM_PROMPT, step_index: 0 },
        },
        {
          role: Role.USER,
          content: "user",
          meta: { type: MessageType.USER_INPUT, step_index: 0 },
        },
        {
          role: Role.TOOL,
          content: longContent,
          meta: { type: MessageType.TOOL_RESULT, step_index: 0 },
        },
        {
          role: Role.TOOL,
          content: longContent,
          meta: { type: MessageType.TOOL_RESULT, step_index: 1 },
        },
      ];

      const result = compact.compact(messages, 150, simpleTokenCounter);

      expect(result.length).toBe(4);
      for (const m of result) {
        if (m.meta?.type === MessageType.TOOL_RESULT) {
          expect(m.content.length).toBeLessThanOrEqual(200);
        }
      }
    });
  });

  describe("phase2 - drops tool results", () => {
    it("drops tool result entirely in phase 2", () => {
      const compact = new TieredCompact(0);

      const messages: Message[] = [
        {
          role: Role.SYSTEM,
          content: "system",
          meta: { type: MessageType.SYSTEM_PROMPT },
        },
        {
          role: Role.USER,
          content: "user",
          meta: { type: MessageType.USER_INPUT },
        },
        {
          role: Role.TOOL,
          content: "tool result",
          meta: { type: MessageType.TOOL_RESULT, step_index: 0 },
        },
        {
          role: Role.ASSISTANT,
          content: "text",
          meta: { type: MessageType.TEXT_RESPONSE, step_index: 0 },
        },
      ];

      const result = compact.compact(messages, 5, simpleTokenCounter);

      const toolResultFound = result.some(
        (m) => m.meta?.type === MessageType.TOOL_RESULT
      );
      expect(toolResultFound).toBe(false);
    });
  });

  describe("phase3 - drops reasoning and text response", () => {
    it("drops reasoning and text response in phase 3", () => {
      const compact = new TieredCompact(0);

      const messages: Message[] = [
        {
          role: Role.SYSTEM,
          content: "system",
          meta: { type: MessageType.SYSTEM_PROMPT },
        },
        {
          role: Role.USER,
          content: "user",
          meta: { type: MessageType.USER_INPUT },
        },
        {
          role: Role.ASSISTANT,
          content: "reasoning text",
          meta: { type: MessageType.REASONING, step_index: 0 },
        },
        {
          role: Role.ASSISTANT,
          content: "text response",
          meta: { type: MessageType.TEXT_RESPONSE, step_index: 0 },
        },
      ];

      const result = compact.compact(messages, 5, simpleTokenCounter);

      const reasoningFound = result.some(
        (m) => m.meta?.type === MessageType.REASONING
      );
      const textResponseFound = result.some(
        (m) => m.meta?.type === MessageType.TEXT_RESPONSE
      );
      expect(reasoningFound).toBe(false);
      expect(textResponseFound).toBe(false);
    });
  });

  describe("protected messages", () => {
    it("messages 0 and 1 always protected", () => {
      const compact = new TieredCompact(0);

      const messages: Message[] = [
        {
          role: Role.SYSTEM,
          content: "system",
          meta: { type: MessageType.SYSTEM_PROMPT },
        },
        {
          role: Role.USER,
          content: "user",
          meta: { type: MessageType.USER_INPUT },
        },
        {
          role: Role.ASSISTANT,
          content: "reasoning",
          meta: { type: MessageType.REASONING, step_index: 0 },
        },
      ];

      const result = compact.compact(messages, 1, simpleTokenCounter);

      expect(result.length).toBeGreaterThanOrEqual(2);
      expect(result[0].meta?.type).toBe(MessageType.SYSTEM_PROMPT);
      expect(result[1].meta?.type).toBe(MessageType.USER_INPUT);
    });
  });

  describe("keepRecent", () => {
    it("preserves last keepRecent steps", () => {
      const compact = new TieredCompact(2);

      const messages: Message[] = [
        {
          role: Role.SYSTEM,
          content: "system",
          meta: { type: MessageType.SYSTEM_PROMPT, step_index: 0 },
        },
        {
          role: Role.USER,
          content: "user",
          meta: { type: MessageType.USER_INPUT, step_index: 0 },
        },
        {
          role: Role.TOOL,
          content: "tool0",
          meta: { type: MessageType.TOOL_RESULT, step_index: 0 },
        },
        {
          role: Role.TOOL,
          content: "tool1",
          meta: { type: MessageType.TOOL_RESULT, step_index: 1 },
        },
        {
          role: Role.TOOL,
          content: "tool2",
          meta: { type: MessageType.TOOL_RESULT, step_index: 2 },
        },
        {
          role: Role.TOOL,
          content: "tool3",
          meta: { type: MessageType.TOOL_RESULT, step_index: 3 },
        },
        {
          role: Role.TOOL,
          content: "tool4",
          meta: { type: MessageType.TOOL_RESULT, step_index: 4 },
        },
      ];

      const result = compact.compact(messages, 1, simpleTokenCounter);

      const step3Preserved = result.some(
        (m) => m.meta?.step_index === 3
      );
      const step4Preserved = result.some(
        (m) => m.meta?.step_index === 4
      );
      expect(step3Preserved).toBe(true);
      expect(step4Preserved).toBe(true);
    });
  });

  describe("no compaction needed", () => {
    it("returns unchanged when under target", () => {
      const compact = new TieredCompact();

      const messages: Message[] = [
        {
          role: Role.SYSTEM,
          content: "system prompt",
          meta: { type: MessageType.SYSTEM_PROMPT },
        },
        {
          role: Role.USER,
          content: "user input",
          meta: { type: MessageType.USER_INPUT },
        },
      ];

      const result = compact.compact(messages, 100, simpleTokenCounter);

      expect(result.length).toBe(2);
      expect(result[0].content).toBe("system prompt");
      expect(result[1].content).toBe("user input");
    });
  });
});

describe("findEligibleEnd", () => {
  it("returns adjusted index when no steps", () => {
    const messages: Message[] = [
      {
        role: Role.SYSTEM,
        content: "system",
        meta: { type: MessageType.SYSTEM_PROMPT },
      },
      {
        role: Role.USER,
        content: "user",
        meta: { type: MessageType.USER_INPUT },
      },
      {
        role: Role.ASSISTANT,
        content: "msg1",
        meta: { type: MessageType.TEXT_RESPONSE },
      },
      {
        role: Role.ASSISTANT,
        content: "msg2",
        meta: { type: MessageType.TEXT_RESPONSE },
      },
      {
        role: Role.ASSISTANT,
        content: "msg3",
        meta: { type: MessageType.TEXT_RESPONSE },
      },
    ];

    const result = findEligibleEnd(messages, 1);
    const expected = messages.length - 1 - 1;
    expect(result).toBe(expected);
  });

  it("finds boundary with steps", () => {
    const messages: Message[] = [
      {
        role: Role.SYSTEM,
        content: "system",
        meta: { type: MessageType.SYSTEM_PROMPT, step_index: 0 },
      },
      {
        role: Role.USER,
        content: "user",
        meta: { type: MessageType.USER_INPUT, step_index: 0 },
      },
      {
        role: Role.ASSISTANT,
        content: "step1",
        meta: { type: MessageType.REASONING, step_index: 1 },
      },
      {
        role: Role.ASSISTANT,
        content: "step2",
        meta: { type: MessageType.REASONING, step_index: 2 },
      },
      {
        role: Role.ASSISTANT,
        content: "step3",
        meta: { type: MessageType.REASONING, step_index: 3 },
      },
    ];

    const result = findEligibleEnd(messages, 2);
    expect(result).toBe(2);
  });
});