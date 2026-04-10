export interface ToolCall {
  id: string;
  name: string;
  arguments: Record<string, unknown>;
}

export interface TokenUsage {
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
}

export interface Response {
  content: string;
  tool_calls: ToolCall[];
  finish_reason: string;
  usage_tokens: TokenUsage;
}