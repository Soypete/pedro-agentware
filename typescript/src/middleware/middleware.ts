import type { CallerContext } from "./types.js";
import type { PolicyEvaluator } from "./policy.js";
import type { Auditor } from "./audit.js";
import { Action } from "./types.js";

export interface ToolExecutor {
  execute(toolName: string, args: Record<string, unknown>): [unknown, boolean, string];
}

export interface Middleware {
  execute(
    toolName: string,
    args: Record<string, unknown>,
    caller: CallerContext
  ): [unknown, boolean, string];
  withPolicy(evaluator: PolicyEvaluator): MiddlewareImpl;
  withAuditor(auditor: Auditor): MiddlewareImpl;
}

export class MiddlewareImpl implements Middleware {
  private executor: ToolExecutor;
  private evaluator: PolicyEvaluator | null = null;
  private auditor: Auditor | null = null;

  constructor(executor: ToolExecutor) {
    this.executor = executor;
  }

  execute(
    toolName: string,
    args: Record<string, unknown>,
    caller: CallerContext
  ): [unknown, boolean, string] {
    const decision = this.evaluator
      ? this.evaluator.evaluate(toolName, args, caller)
      : { action: Action.ALLOW, rule: "default", reason: "no policy configured", timestamp: new Date() };

    if (this.auditor) {
      this.auditor.record({
        session_id: caller.session_id || "",
        tool_name: toolName,
        args,
        decision,
        timestamp: new Date(),
      });
    }

    if (decision.action === Action.DENY) {
      return [null, false, `denied by policy: ${decision.reason}`];
    }

    if (decision.action === Action.FILTER && decision.redacted_args) {
      args = { ...args, ...decision.redacted_args };
    }

    return this.executor.execute(toolName, args);
  }

  withPolicy(evaluator: PolicyEvaluator): MiddlewareImpl {
    this.evaluator = evaluator;
    return this;
  }

  withAuditor(auditor: Auditor): MiddlewareImpl {
    this.auditor = auditor;
    return this;
  }
}

export function newMiddleware(executor: ToolExecutor): MiddlewareImpl {
  return new MiddlewareImpl(executor);
}