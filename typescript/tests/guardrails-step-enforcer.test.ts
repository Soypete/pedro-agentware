import { StepEnforcer, StepNotAllowedError } from "../src/middleware/guardrails/step_enforcer";

describe("StepEnforcer", () => {
  describe("addStep", () => {
    it("should add a step with prerequisites", () => {
      const se = new StepEnforcer();
      se.addStep("deploy", ["build", "test"]);

      const [allowed, missing] = se.canExecute("session1", "deploy");
      expect(allowed).toBe(false);
      expect(missing).toHaveLength(2);
    });
  });

  describe("markStepComplete", () => {
    it("should mark a step as complete", () => {
      const se = new StepEnforcer();
      se.addStep("deploy", ["build", "test"]);

      se.markStepComplete("session1", "build");
      const [allowed, missing] = se.canExecute("session1", "deploy");

      expect(allowed).toBe(false);
      expect(missing).toContain("test");
    });
  });

  describe("validateExecution", () => {
    it("should not throw when prerequisites are met", () => {
      const se = new StepEnforcer();
      se.addStep("deploy", ["build"]);

      se.markStepComplete("session1", "build");
      expect(() => se.validateExecution("session1", "deploy")).not.toThrow();
    });

    it("should throw when prerequisites are not met", () => {
      const se = new StepEnforcer();
      se.addStep("deploy", ["build"]);

      expect(() => se.validateExecution("session1", "deploy")).toThrow(StepNotAllowedError);
    });
  });

  describe("resetSession", () => {
    it("should reset completed steps for a session", () => {
      const se = new StepEnforcer();
      se.addStep("deploy", ["build"]);

      se.markStepComplete("session1", "build");
      se.resetSession("session1");

      const [allowed] = se.canExecute("session1", "deploy");
      expect(allowed).toBe(false);
    });
  });

  describe("isTerminalAllowed", () => {
    it("should return true when terminal is allowed", () => {
      const se = new StepEnforcer();
      se.addStep("deploy", ["build"]);
      se.addStep("build", []);

      se.markStepComplete("session1", "build");
      const allowed = se.isTerminalAllowed("session1", "deploy");

      expect(allowed).toBe(true);
    });
  });

  describe("getAllowedTerminals", () => {
    it("should return allowed terminals", () => {
      const se = new StepEnforcer();
      se.addStep("deploy", ["build"]);
      se.addStep("test", []);

      const allowed = se.getAllowedTerminals("session1");

      expect(allowed).toContain("test");
      expect(allowed).not.toContain("deploy");
    });
  });

  describe("no prerequisites", () => {
    it("should allow tools with no prerequisites", () => {
      const se = new StepEnforcer();
      se.addStep("build", []);

      const [allowed] = se.canExecute("session1", "build");
      expect(allowed).toBe(true);
    });
  });

  describe("invalid session", () => {
    it("should not allow for nonexistent session", () => {
      const se = new StepEnforcer();
      se.addStep("deploy", ["build"]);

      const [allowed] = se.canExecute("nonexistent", "deploy");
      expect(allowed).toBe(false);
    });
  });
});