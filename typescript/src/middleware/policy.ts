import type { CallerContext, Decision } from "./types.js";
import { Action } from "./types.js";

export enum Operator {
  EQ = "eq",
  NOT_EQ = "not_eq",
  CONTAINS = "contains",
  NOT_CONTAINS = "not_contains",
  MATCHES = "matches",
  NOT_MATCHES = "not_matches",
  EXISTS = "exists",
  NOT_EXISTS = "not_exists",
}

export interface PolicyEvaluator {
  evaluate(toolName: string, args: Record<string, unknown>, caller: CallerContext): Decision;
}

export interface Condition {
  field: string;
  operator: Operator;
  value?: string;
}

export interface Rule {
  name: string;
  tools?: string[];
  action: Action;
  conditions?: Condition[];
  max_rate?: { count: number; window_ms: number };
  redact_fields?: string[];
}

export interface Policy {
  rules: Rule[];
  default_deny: boolean;
}

export class SimplePolicyEvaluator implements PolicyEvaluator {
  private policy: Policy;

  constructor(policy: Policy) {
    this.policy = policy;
  }

  evaluate(toolName: string, args: Record<string, unknown>, caller: CallerContext): Decision {
    for (const rule of this.policy.rules) {
      if (!this.ruleMatchesTool(rule, toolName)) continue;
      if (!this.evaluateConditions(rule.conditions || [], args, caller)) continue;
      return {
        action: rule.action,
        rule: rule.name,
        reason: `matched rule ${rule.name}`,
        timestamp: new Date(),
      };
    }

    if (this.policy.default_deny) {
      return {
        action: Action.DENY,
        rule: "default",
        reason: "no matching rules and default deny is enabled",
        timestamp: new Date(),
      };
    }

    return {
      action: Action.ALLOW,
      rule: "default",
      reason: "no matching rules and default allow is enabled",
      timestamp: new Date(),
    };
  }

  private ruleMatchesTool(rule: Rule, toolName: string): boolean {
    if (!rule.tools || rule.tools.length === 0) return true;
    return rule.tools.includes("*") || rule.tools.includes(toolName);
  }

  private evaluateConditions(
    conditions: Condition[],
    args: Record<string, unknown>,
    caller: CallerContext
  ): boolean {
    if (conditions.length === 0) return true;
    return conditions.every((c) => this.evaluateCondition(c, args, caller));
  }

  private evaluateCondition(
    condition: Condition,
    args: Record<string, unknown>,
    caller: CallerContext
  ): boolean {
    const value = this.getValue(condition.field, args, caller);
    return this.compare(value, condition.operator, condition.value || "");
  }

  private getValue(
    field: string,
    args: Record<string, unknown>,
    caller: CallerContext
  ): string {
    if (field.startsWith("caller.")) {
      const key = field.slice(7);
      if (key === "role") return caller.role || "";
      if (key === "source") return caller.source || "";
      if (key === "trusted") return caller.trusted ? "true" : "false";
      if (key === "user_id") return caller.user_id || "";
      if (key === "session_id") return caller.session_id || "";
    } else if (field.startsWith("args.")) {
      const key = field.slice(5);
      const val = args[key];
      return val !== undefined ? String(val) : "";
    }
    return "";
  }

  private compare(value: string, operator: Operator, target: string): boolean {
    switch (operator) {
      case Operator.EQ:
        return value === target;
      case Operator.NOT_EQ:
        return value !== target;
      case Operator.CONTAINS:
        return value.includes(target);
      case Operator.NOT_CONTAINS:
        return !value.includes(target);
      case Operator.MATCHES:
        try {
          return new RegExp(target).test(value);
        } catch {
          return false;
        }
      case Operator.NOT_MATCHES:
        try {
          return !new RegExp(target).test(value);
        } catch {
          return true;
        }
      case Operator.EXISTS:
        return value !== "";
      case Operator.NOT_EXISTS:
        return value === "";
      default:
        return false;
    }
  }
}