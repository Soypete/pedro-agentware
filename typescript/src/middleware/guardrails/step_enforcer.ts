export class StepNotAllowedError extends Error {
  tool: string;
  missingSteps: string[];

  constructor(tool: string, missingSteps: string[]) {
    super(`step not allowed: missing ${missingSteps.join(", ")}`);
    this.name = "StepNotAllowedError";
    this.tool = tool;
    this.missingSteps = missingSteps;
  }
}

export class StepEnforcer {
  private stepDefinitions: Map<string, string[]>;
  private completedSteps: Map<string, Map<string, boolean>>;
  private allowedTerminals: Map<string, Map<string, boolean>>;

  constructor() {
    this.stepDefinitions = new Map();
    this.completedSteps = new Map();
    this.allowedTerminals = new Map();
  }

  addStep(tool: string, prerequisites?: string[]): void {
    this.stepDefinitions.set(tool, prerequisites || []);
  }

  addTerminal(tool: string, allowed?: Map<string, boolean>): void {
    this.allowedTerminals.set(tool, allowed || new Map());
  }

  markStepComplete(sessionId: string, step: string): void {
    if (!this.completedSteps.has(sessionId)) {
      this.completedSteps.set(sessionId, new Map());
    }
    this.completedSteps.get(sessionId)!.set(step, true);
  }

  resetSession(sessionId: string): void {
    this.completedSteps.delete(sessionId);
  }

  canExecute(sessionId: string, tool: string): [boolean, string[]] {
    const prereqs = this.stepDefinitions.get(tool);
    if (prereqs === undefined) {
      return [true, []];
    }

    const completed = this.completedSteps.get(sessionId) || new Map();
    const missing = prereqs.filter((p) => completed.get(p) !== true);

    return [missing.length === 0, missing];
  }

  validateExecution(sessionId: string, tool: string): void {
    const [allowed, missing] = this.canExecute(sessionId, tool);
    if (allowed) {
      return;
    }
    throw new StepNotAllowedError(tool, missing);
  }

  isTerminalAllowed(sessionId: string, terminalTool: string): boolean {
    const [allowed] = this.canExecute(sessionId, terminalTool);
    return allowed;
  }

  getAllowedTerminals(sessionId: string): string[] {
    const result: string[] = [];
    for (const tool of this.stepDefinitions.keys()) {
      const [allowed] = this.canExecute(sessionId, tool);
      if (allowed) {
        result.push(tool);
      }
    }
    return result;
  }
}