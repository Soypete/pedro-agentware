export enum Action {
  ALLOW = "allow",
  DENY = "deny",
  FILTER = "filter",
}

export interface CallerContext {
  user_id?: string;
  session_id?: string;
  role?: string;
  source?: string;
  trusted: boolean;
  metadata?: Record<string, string>;
}

export interface Decision {
  action: Action;
  rule: string;
  reason: string;
  redacted_args?: Record<string, unknown>;
  timestamp: Date;
}

export function createAllowDecision(reason = "allowed"): Decision {
  return {
    action: Action.ALLOW,
    rule: "default",
    reason,
    timestamp: new Date(),
  };
}

export function createDenyDecision(reason: string): Decision {
  return {
    action: Action.DENY,
    rule: "default",
    reason,
    timestamp: new Date(),
  };
}