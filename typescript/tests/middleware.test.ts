import {
  MiddlewareImpl,
  Action,
  CallerContext,
  Policy,
  Rule,
  InMemoryAuditor,
  SimplePolicyEvaluator,
} from "../src/middleware/index.js";

const mockExecutor = {
  execute: (
    _toolName: string,
    _args: Record<string, unknown>
  ): [unknown, boolean, string] => {
    return [{ result: "executed" }, true, ""];
  },
};

describe("Middleware", () => {
  it("should allow by default", () => {
    const mw = new MiddlewareImpl(mockExecutor);
    const [, success, error] = mw.execute("test_tool", { arg: "value" }, { trusted: true });
    expect(success).toBe(true);
    expect(error).toBe("");
  });

  it("should deny when policy denies", () => {
    const policy: Policy = {
      rules: [{ name: "deny_all", tools: ["test_tool"], action: Action.DENY }],
      default_deny: false,
    };
    const evaluator = new SimplePolicyEvaluator(policy);
    const mw = new MiddlewareImpl(mockExecutor).withPolicy(evaluator);
    const [, success, error] = mw.execute("test_tool", {}, { trusted: true });
    expect(success).toBe(false);
    expect(error).toContain("denied by policy");
  });

  it("should record audit", () => {
    const auditor = new InMemoryAuditor();
    const mw = new MiddlewareImpl(mockExecutor).withAuditor(auditor);
    mw.execute("test_tool", { arg: "value" }, { trusted: true });
    const records = auditor.query({ session_id: "" });
    expect(records.length).toBe(1);
  });
});

describe("Policy", () => {
  it("should allow by default", () => {
    const policy: Policy = { rules: [], default_deny: false };
    const evaluator = new SimplePolicyEvaluator(policy);
    const decision = evaluator.evaluate("test_tool", {}, { trusted: true });
    expect(decision.action).toBe(Action.ALLOW);
  });

  it("should deny when default_deny is true", () => {
    const policy: Policy = { rules: [], default_deny: true };
    const evaluator = new SimplePolicyEvaluator(policy);
    const decision = evaluator.evaluate("test_tool", {}, { trusted: true });
    expect(decision.action).toBe(Action.DENY);
  });
});

describe("Rule matching", () => {
  it("should match tool by name", () => {
    const policy: Policy = {
      rules: [{ name: "test", tools: ["foo", "bar"], action: Action.ALLOW }],
      default_deny: false,
    };
    const evaluator = new SimplePolicyEvaluator(policy);
    const decision = evaluator.evaluate("foo", {}, { trusted: true });
    expect(decision.action).toBe(Action.ALLOW);
  });
});