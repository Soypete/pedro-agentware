import type { Message } from "../llm/index.js";

export interface ToolResultEntry {
  call_id: string;
  tool_name: string;
  args: Record<string, unknown>;
  output: string;
  success: boolean;
}

export interface ContextManager {
  appendPrompt(jobId: string, msg: Message): void;
  appendResponse(jobId: string, msg: Message): void;
  appendToolCalls(jobId: string, calls: unknown[]): void;
  appendToolResults(jobId: string, results: ToolResultEntry[]): void;
  getHistory(jobId: string): Message[];
  purge(jobId: string): void;
}

export class InMemoryContextManager implements ContextManager {
  private history: Map<string, Message[]> = new Map();

  appendPrompt(jobId: string, msg: Message): void {
    this.ensureHistory(jobId).push(msg);
  }

  appendResponse(jobId: string, msg: Message): void {
    this.ensureHistory(jobId).push(msg);
  }

  appendToolCalls(_jobId: string, _calls: unknown[]): void {
    // Not stored in this implementation
  }

  appendToolResults(_jobId: string, _results: ToolResultEntry[]): void {
    // Not stored in this implementation
  }

  getHistory(jobId: string): Message[] {
    return this.history.get(jobId) || [];
  }

  purge(jobId: string): void {
    this.history.delete(jobId);
  }

  private ensureHistory(jobId: string): Message[] {
    if (!this.history.has(jobId)) {
      this.history.set(jobId, []);
    }
    return this.history.get(jobId)!;
  }
}