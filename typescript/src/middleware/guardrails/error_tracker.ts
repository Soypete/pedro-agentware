export enum ErrorCategory {
  TIMEOUT = "timeout",
  NOT_FOUND = "not_found",
  INVALID_ARGS = "invalid_args",
  PERMISSION = "permission",
  RATE_LIMIT = "rate_limit",
  UNKNOWN = "unknown",
}

export interface ToolError {
  timestamp: Date;
  tool: string;
  args: Record<string, unknown>;
  category: ErrorCategory;
  message: string;
  sessionId: string;
  retryCount: number;
}

export class ErrorTracker {
  private errors: Map<string, ToolError[]>;
  private maxErrorsPerTool: number;
  private windowDurationMs: number;

  constructor(maxErrorsPerTool: number = 5, windowDurationMinutes: number = 5) {
    this.errors = new Map();
    this.maxErrorsPerTool = maxErrorsPerTool;
    this.windowDurationMs = windowDurationMinutes * 60 * 1000;
  }

  setThresholds(maxErrors: number, windowMinutes: number): void {
    this.maxErrorsPerTool = maxErrors;
    this.windowDurationMs = windowMinutes * 60 * 1000;
  }

  recordError(
    sessionId: string,
    tool: string,
    args: Record<string, unknown>,
    err: Error,
    category: ErrorCategory,
  ): void {
    if (!this.errors.has(sessionId)) {
      this.errors.set(sessionId, []);
    }

    const retryCount = this.getRetryCount(sessionId, tool);

    const toolErr: ToolError = {
      timestamp: new Date(),
      tool,
      args,
      category,
      message: err.message,
      sessionId,
      retryCount,
    };

    this.errors.get(sessionId)!.push(toolErr);
    this.pruneOldErrors(sessionId);
  }

  private getRetryCount(sessionId: string, tool: string): number {
    return (this.errors.get(sessionId) || []).filter((e) => e.tool === tool).length;
  }

  private pruneOldErrors(sessionId: string): void {
    const errors = this.errors.get(sessionId);
    if (!errors || errors.length <= this.maxErrorsPerTool) {
      return;
    }

    const cutoff = new Date(Date.now() - this.windowDurationMs);
    this.errors.set(
      sessionId,
      errors.filter((e) => e.timestamp > cutoff),
    );
  }

  getErrorCount(sessionId: string, tool: string): number {
    return (this.errors.get(sessionId) || []).filter((e) => e.tool === tool).length;
  }

  getRecentErrors(sessionId: string): ToolError[] {
    this.pruneOldErrors(sessionId);
    return [...(this.errors.get(sessionId) || [])];
  }

  getErrorsByCategory(sessionId: string, category: ErrorCategory): ToolError[] {
    return (this.errors.get(sessionId) || []).filter((e) => e.category === category);
  }

  isErrorRateExceeded(sessionId: string, tool: string): boolean {
    return this.getErrorCount(sessionId, tool) >= this.maxErrorsPerTool;
  }

  resetSession(sessionId: string): void {
    this.errors.delete(sessionId);
  }

  shouldBlockTool(sessionId: string, tool: string): boolean {
    return this.isErrorRateExceeded(sessionId, tool);
  }
}