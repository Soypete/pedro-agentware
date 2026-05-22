import { ErrorCategory, ErrorTracker } from "../src/middleware/guardrails/error_tracker";

describe("ErrorTracker", () => {
  describe("recordError", () => {
    it("should record an error", () => {
      const et = new ErrorTracker();
      et.recordError(
        "session1",
        "tool1",
        { key: "value" },
        new Error("test error"),
        ErrorCategory.UNKNOWN,
      );

      expect(et.getErrorCount("session1", "tool1")).toBe(1);
    });
  });

  describe("getErrorCount", () => {
    it("should return correct error count", () => {
      const et = new ErrorTracker();
      et.recordError("session1", "tool1", {}, new Error("error1"), ErrorCategory.TIMEOUT);
      et.recordError("session1", "tool1", {}, new Error("error2"), ErrorCategory.TIMEOUT);
      et.recordError("session1", "tool2", {}, new Error("error3"), ErrorCategory.NOT_FOUND);

      expect(et.getErrorCount("session1", "tool1")).toBe(2);
      expect(et.getErrorCount("session1", "tool2")).toBe(1);
    });
  });

  describe("getRecentErrors", () => {
    it("should return recent errors", () => {
      const et = new ErrorTracker();
      et.recordError("session1", "tool1", {}, new Error("error1"), ErrorCategory.UNKNOWN);

      const errors = et.getRecentErrors("session1");
      expect(errors).toHaveLength(1);
    });
  });

  describe("getErrorsByCategory", () => {
    it("should filter by category", () => {
      const et = new ErrorTracker();
      et.recordError("session1", "tool1", {}, new Error("timeout"), ErrorCategory.TIMEOUT);
      et.recordError("session1", "tool1", {}, new Error("not found"), ErrorCategory.NOT_FOUND);

      const timeoutErrors = et.getErrorsByCategory("session1", ErrorCategory.TIMEOUT);
      expect(timeoutErrors).toHaveLength(1);
    });
  });

  describe("isErrorRateExceeded", () => {
    it("should detect rate exceeded", () => {
      const et = new ErrorTracker(3);

      for (let i = 0; i < 3; i++) {
        et.recordError("session1", "tool1", {}, new Error("error"), ErrorCategory.UNKNOWN);
      }

      expect(et.isErrorRateExceeded("session1", "tool1")).toBe(true);
    });
  });

  describe("resetSession", () => {
    it("should clear session errors", () => {
      const et = new ErrorTracker();
      et.recordError("session1", "tool1", {}, new Error("error"), ErrorCategory.UNKNOWN);

      et.resetSession("session1");
      expect(et.getErrorCount("session1", "tool1")).toBe(0);
    });
  });

  describe("shouldBlockTool", () => {
    it("should block tool when rate exceeded", () => {
      const et = new ErrorTracker(2);

      et.recordError("session1", "tool1", {}, new Error("error1"), ErrorCategory.UNKNOWN);
      et.recordError("session1", "tool1", {}, new Error("error2"), ErrorCategory.UNKNOWN);

      expect(et.shouldBlockTool("session1", "tool1")).toBe(true);
      expect(et.shouldBlockTool("session1", "tool2")).toBe(false);
    });
  });

  describe("setThresholds", () => {
    it("should update thresholds", () => {
      const et = new ErrorTracker();
      et.setThresholds(10, 10);

      expect(et["maxErrorsPerTool"]).toBe(10);
      expect(et["windowDurationMs"]).toBe(10 * 60 * 1000);
    });
  });
});