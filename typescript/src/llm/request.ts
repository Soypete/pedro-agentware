import type { ToolCall } from "./response.js";
import type { MessageMeta } from "../middleware/types.js";

export enum Role {
  SYSTEM = "system",
  USER = "user",
  ASSISTANT = "assistant",
  TOOL = "tool",
}

export interface Message {
  role: Role;
  content: string;
  tool_call_id?: string;
  tool_calls?: ToolCall[];
  meta?: MessageMeta;
}

export interface ToolDefinition {
  name: string;
  description: string;
  input_schema: Record<string, unknown>;
}

export interface Request {
  messages: Message[];
  tools: ToolDefinition[];
  temperature?: number;
  max_tokens?: number;
  stop?: string[];
}