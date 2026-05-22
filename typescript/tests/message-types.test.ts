import { Role, Message } from "../src/llm/request.js";
import { MessageType, MessageMeta } from "../src/middleware/types.js";

describe("MessageType", () => {
  it("should have all 11 message types defined", () => {
    const expectedTypes = [
      "system_prompt",
      "user_input",
      "tool_call",
      "tool_result",
      "reasoning",
      "text_response",
      "step_nudge",
      "prerequisite_nudge",
      "retry_nudge",
      "context_warning",
      "summary",
    ];
    const actualTypes = Object.values(MessageType);
    expect(actualTypes).toEqual(expectedTypes);
  });

  it("should be string enum for easy serialization", () => {
    expect(MessageType.USER_INPUT).toBe("user_input");
    expect(typeof MessageType.TOOL_RESULT).toBe("string");
  });
});

describe("MessageMeta", () => {
  it("should have correct default values", () => {
    const meta: MessageMeta = { type: MessageType.USER_INPUT };
    expect(meta.type).toBe(MessageType.USER_INPUT);
    expect(meta.step_index).toBeUndefined();
    expect(meta.original_type).toBeUndefined();
    expect(meta.token_estimate).toBeUndefined();
  });

  it("should accept custom values", () => {
    const meta: MessageMeta = {
      type: MessageType.TOOL_RESULT,
      step_index: 5,
      original_type: MessageType.TOOL_CALL,
      token_estimate: 150,
    };
    expect(meta.type).toBe(MessageType.TOOL_RESULT);
    expect(meta.step_index).toBe(5);
    expect(meta.original_type).toBe(MessageType.TOOL_CALL);
    expect(meta.token_estimate).toBe(150);
  });
});

describe("Message", () => {
  it("should have optional meta field", () => {
    const msg: Message = {
      role: Role.USER,
      content: "Hello",
    };
    expect(msg.meta).toBeUndefined();
  });

  it("should accept custom meta", () => {
    const msg: Message = {
      role: Role.SYSTEM,
      content: "You are helpful",
      meta: { type: MessageType.SYSTEM_PROMPT, step_index: 0 },
    };
    expect(msg.meta?.type).toBe(MessageType.SYSTEM_PROMPT);
    expect(msg.meta?.step_index).toBe(0);
  });

  it("should maintain backward compatibility", () => {
    const msg: Message = {
      role: Role.ASSISTANT,
      content: "Response",
      tool_call_id: "call_123",
    };
    expect(msg.role).toBe(Role.ASSISTANT);
    expect(msg.content).toBe("Response");
    expect(msg.tool_call_id).toBe("call_123");
    expect(msg.meta).toBeUndefined();
  });
});