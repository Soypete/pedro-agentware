export enum NudgeKind {
  RETRY = "retry",
  UNKNOWN_TOOL = "unknown_tool",
  STEP = "step",
  PREREQUISITE = "prerequisite",
}

export interface Nudge {
  role: string;
  content: string;
  kind: NudgeKind;
  tier: number;
}

function joinToolNames(names: string[]): string {
  if (names.length === 0) {
    return "(no tools available)";
  }
  if (names.length === 1) {
    return names[0];
  }
  if (names.length === 2) {
    return `${names[0]} and ${names[1]}`;
  }
  return names.slice(0, -1).join(", ") + `, and ${names[names.length - 1]}`;
}

export function retryNudge(_rawResponse: string, toolNames: string[]): Nudge {
  const toolsList = joinToolNames(toolNames);
  return {
    role: "user",
    content: `Your previous response was not a valid tool call. You must respond with a tool call, not free text. Available tools: ${toolsList}. Please try again with a valid tool call.`,
    kind: NudgeKind.RETRY,
    tier: 0,
  };
}

export function unknownToolNudge(toolName: string, toolNames: string[]): Nudge {
  const toolsList = joinToolNames(toolNames);
  return {
    role: "user",
    content: `Tool '${toolName}' does not exist. Available tools: ${toolsList}. Call one of them.`,
    kind: NudgeKind.UNKNOWN_TOOL,
    tier: 0,
  };
}

export function stepNudge(
  terminalTool: string,
  pendingSteps: string[],
  tier: number = 1,
): Nudge {
  const clampedTier = Math.max(1, Math.min(3, tier));
  const steps = joinToolNames(pendingSteps);

  let content: string;
  switch (clampedTier) {
    case 1:
      content = `You cannot call ${terminalTool} yet. You must first complete these required steps: ${steps}. Call one of them now.`;
      break;
    case 2:
      content = `You must call one of these tools now: ${steps}. Pick one.`;
      break;
    default:
      content = `STOP. You MUST call one of: ${steps}. Do NOT call ${terminalTool}. Your next response MUST be a tool call to one of: ${steps}.`;
  }

  return {
    role: "user",
    content,
    kind: NudgeKind.STEP,
    tier: clampedTier,
  };
}

export function prerequisiteNudge(
  toolName: string,
  missingPrereqs: string[],
): Nudge {
  const prereqs = joinToolNames(missingPrereqs);
  return {
    role: "user",
    content: `You cannot call ${toolName} yet. You must first call: ${prereqs}. Call the prerequisite tool now.`,
    kind: NudgeKind.PREREQUISITE,
    tier: 0,
  };
}