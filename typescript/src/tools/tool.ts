export interface Tool {
  readonly name: string;
  readonly description: string;
  execute(args: Record<string, unknown>): Result;
}

export interface ToolExample {
  input: Record<string, unknown>;
  output: string;
  explanation?: string;
}

export interface ExtendedTool extends Tool {
  inputSchema(): Record<string, unknown>;
  examples(): ToolExample[];
}

export class BaseTool implements Tool {
  readonly name: string;
  readonly description: string;

  constructor(name: string, description: string) {
    this.name = name;
    this.description = description;
  }

  execute(_args: Record<string, unknown>): Result {
    throw new Error("Subclass must implement execute");
  }
}

export class Result {
  success: boolean;
  data: unknown = null;
  error: string | null = null;
  timestamp: Date;

  constructor(success: boolean, data?: unknown, error?: string) {
    this.success = success;
    this.data = data ?? null;
    this.error = error ?? null;
    this.timestamp = new Date();
  }

  toJSON(): object {
    return {
      success: this.success,
      data: this.data,
      error: this.error,
      timestamp: this.timestamp.toISOString(),
    };
  }
}