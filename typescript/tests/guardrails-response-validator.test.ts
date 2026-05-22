import { ResponseValidator } from "../src/middleware/guardrails/response_validator";
import { NudgeKind } from "../src/middleware/guardrails/nudge";

describe("ResponseValidator", () => {
  describe("new ResponseValidator", () => {
    it("should create a new validator", () => {
      const rv = new ResponseValidator(["tool1", "tool2"], true);
      expect(rv).toBeDefined();
    });
  });

  describe("validateToolCalls", () => {
    it("should validate valid tool calls", () => {
      const rv = new ResponseValidator(["tool1", "tool2"], false);
      const calls = [{ tool: "tool1", args: { key: "value" } }];
      const result = rv.validateToolCalls(calls);

      expect(result.needsRetry).toBe(false);
      expect(result.nudge).toBeNull();
      expect(result.toolCalls.length).toBe(1);
    });

    it("should reject unknown tools", () => {
      const rv = new ResponseValidator(["tool1", "tool2"], false);
      const calls = [{ tool: "unknown_tool", args: {} }];
      const result = rv.validateToolCalls(calls);

      expect(result.needsRetry).toBe(true);
      expect(result.nudge).not.toBeNull();
      expect(result.nudge!.kind).toBe(NudgeKind.UNKNOWN_TOOL);
    });
  });

  describe("validateTextResponse", () => {
    it("should rescue tool calls from text", () => {
      const rv = new ResponseValidator(["tool1", "tool2"], true);
      const response = '{"tool": "tool1", "args": {"key": "value"}}';
      const result = rv.validateTextResponse(response);

      expect(result.needsRetry).toBe(false);
      expect(result.toolCalls.length).toBe(1);
    });

    it("should return nudge for text without rescue", () => {
      const rv = new ResponseValidator(["tool1", "tool2"], false);
      const response = "This is just text.";
      const result = rv.validateTextResponse(response);

      expect(result.needsRetry).toBe(true);
      expect(result.nudge).not.toBeNull();
      expect(result.nudge!.kind).toBe(NudgeKind.RETRY);
    });

    it("should return nudge for empty response", () => {
      const rv = new ResponseValidator(["tool1", "tool2"], true);
      const result = rv.validateTextResponse("");

      expect(result.needsRetry).toBe(true);
    });
  });

  describe("rescueToolCall", () => {
    it("should strip think tags", () => {
      const rv = new ResponseValidator(["tool1", "tool2"], true);
      const response = "[THINK] thinking [/THINK]\n{\"tool\": \"tool1\", \"args\": {}}";

      // Access private method through any cast
      const calls = (rv as unknown as { rescueToolCall: (r: string) => { tool: string; args: Record<string, unknown> }[] }).rescueToolCall(response);

      expect(calls.length).toBe(1);
    });

    it("should strip python tag", () => {
      const rv = new ResponseValidator(["tool1", "tool2"], true);
      const response = "<|python_tag|>{\"tool\": \"tool1\", \"args\": {}}";

      const calls = (rv as unknown as { rescueToolCall: (r: string) => { tool: string; args: Record<string, unknown> }[] }).rescueToolCall(response);

      expect(calls.length).toBe(1);
    });
  });

  describe("extractJsonToolCalls", () => {
    it("should extract from code fence", () => {
      const rv = new ResponseValidator(["tool1", "tool2"], true);
      const response = '```json\n{"tool": "tool1", "args": {"key": "value"}}\n```';

      const calls = (rv as unknown as { extractJsonToolCalls: (r: string) => { tool: string; args: Record<string, unknown> }[] }).extractJsonToolCalls(response);

      expect(calls.length).toBe(1);
    });
  });

  describe("extractRehearsalToolCalls", () => {
    it("should extract rehearsal format", () => {
      const rv = new ResponseValidator(["tool1", "tool2"], true);
      const response = 'tool1[ARGS]{"key": "value"}';

      const calls = (rv as unknown as { extractRehearsalToolCalls: (r: string) => { tool: string; args: Record<string, unknown> }[] }).extractRehearsalToolCalls(response);

      expect(calls.length).toBe(1);
    });
  });

  describe("tryParseToolCall", () => {
    it("should parse valid JSON", () => {
      const rv = new ResponseValidator(["tool1", "tool2"], true);
      const jsonStr = '{"tool": "tool1", "args": {"key": "value"}}';

      const call = (rv as unknown as { tryParseToolCall: (j: string) => { tool: string; args: Record<string, unknown> } | null }).tryParseToolCall(jsonStr);

      expect(call).not.toBeNull();
      expect(call!.tool).toBe("tool1");
    });

    it("should return null for invalid JSON", () => {
      const rv = new ResponseValidator(["tool1", "tool2"], true);
      const call = (rv as unknown as { tryParseToolCall: (j: string) => { tool: string; args: Record<string, unknown> } | null }).tryParseToolCall("not valid json");

      expect(call).toBeNull();
    });

    it("should return null for unknown tool", () => {
      const rv = new ResponseValidator(["tool1"], true);
      const jsonStr = '{"tool": "unknown", "args": {}}';

      const call = (rv as unknown as { tryParseToolCall: (j: string) => { tool: string; args: Record<string, unknown> } | null }).tryParseToolCall(jsonStr);

      expect(call).toBeNull();
    });
  });

  describe("extractQwenXmlToolCalls", () => {
    it("should return empty (not implemented)", () => {
      const rv = new ResponseValidator(["tool1", "tool2"], true);
      const response = "<function=tool1>content</function>";

      const calls = (rv as unknown as { extractQwenXmlToolCalls: (r: string) => { tool: string; args: Record<string, unknown> }[] }).extractQwenXmlToolCalls(response);

      expect(calls.length).toBe(0);
    });
  });
});