import { Message } from "../llm/request.js";
import { MessageType } from "../middleware/types.js";

export type TokenCounter = (messages: Message[]) => number;

export interface CompactionStrategy {
  compact(
    messages: Message[],
    targetTokens: number,
    counter: TokenCounter
  ): Message[];
  name(): string;
}

function isNudgeType(msgType: MessageType): boolean {
  return (
    msgType === MessageType.STEP_NUDGE ||
    msgType === MessageType.RETRY_NUDGE ||
    msgType === MessageType.PREREQUISITE_NUDGE
  );
}

export class TieredCompact implements CompactionStrategy {
  private keepRecent: number;
  private truncateChars: number;
  public lastPhase: number = 0;

  constructor(keepRecent = 2, truncateChars = 200) {
    this.keepRecent = keepRecent;
    this.truncateChars = truncateChars;
  }

  name(): string {
    return "TieredCompact";
  }

  compact(
    messages: Message[],
    targetTokens: number,
    counter: TokenCounter
  ): Message[] {
    this.lastPhase = 0;

    if (!messages || messages.length === 0) {
      return [];
    }

    const result = [...messages];

    const eligibleEnd = findEligibleEnd(result, this.keepRecent);

    let currentTokens = counter(result);

    if (currentTokens <= targetTokens) {
      return result;
    }

    const phase1Result = this.phase1Compact(result, eligibleEnd, counter);
    currentTokens = counter(phase1Result);
    this.lastPhase = 1;
    if (currentTokens <= targetTokens) {
      return phase1Result;
    }

    const phase2Result = this.phase2Compact(phase1Result, eligibleEnd, counter);
    currentTokens = counter(phase2Result);
    this.lastPhase = 2;
    if (currentTokens <= targetTokens) {
      return phase2Result;
    }

    const phase3Result = this.phase3Compact(phase2Result, eligibleEnd, counter);
    this.lastPhase = 3;
    return phase3Result;
  }

  private protectedSteps(
    messages: Message[]
  ): Map<number, boolean> {
    const protectedSteps = new Map<number, boolean>();

    if (this.keepRecent <= 0) {
      return protectedSteps;
    }

    const steps: number[] = [];
    const stepSet = new Set<number>();

    for (const m of messages) {
      if (m.meta?.step_index !== undefined && !stepSet.has(m.meta.step_index)) {
        steps.push(m.meta.step_index);
        stepSet.add(m.meta.step_index);
      }
    }

    if (steps.length === 0) {
      return protectedSteps;
    }

    if (this.keepRecent >= steps.length) {
      for (const s of steps) {
        protectedSteps.set(s, true);
      }
      return protectedSteps;
    }

    const startIdx = steps.length - this.keepRecent;
    for (let i = startIdx; i < steps.length; i++) {
      protectedSteps.set(steps[i], true);
    }

    return protectedSteps;
  }

  private isProtected(
    msg: Message,
    protectedSteps: Map<number, boolean>
  ): boolean {
    if (msg.meta?.step_index === undefined) {
      return false;
    }
    return protectedSteps.get(msg.meta.step_index) ?? false;
  }

  private phase1Compact(
    messages: Message[],
    _eligibleEnd: number,
    _counter: TokenCounter
  ): Message[] {
    const result: Message[] = [];
    const protectedSteps = this.protectedSteps(messages);

    for (let i = 0; i < messages.length; i++) {
      const m = messages[i];

      if (i === 0 || i === 1) {
        result.push(m);
        continue;
      }

      if (this.isProtected(m, protectedSteps)) {
        result.push(m);
        continue;
      }

      if (isNudgeType(m.meta?.type ?? MessageType.USER_INPUT)) {
        continue;
      }

      if (m.meta?.type === MessageType.TOOL_RESULT) {
        if (m.content.length > this.truncateChars) {
          result.push({
            ...m,
            content: m.content.slice(0, this.truncateChars),
          });
          continue;
        }
      }

      result.push(m);
    }

    return result;
  }

  private phase2Compact(
    messages: Message[],
    _eligibleEnd: number,
    _counter: TokenCounter
  ): Message[] {
    const result: Message[] = [];
    const protectedSteps = this.protectedSteps(messages);

    for (let i = 0; i < messages.length; i++) {
      const m = messages[i];

      if (i === 0 || i === 1) {
        result.push(m);
        continue;
      }

      if (this.isProtected(m, protectedSteps)) {
        result.push(m);
        continue;
      }

      if (m.meta?.type === MessageType.TOOL_RESULT) {
        continue;
      }

      result.push(m);
    }

    return result;
  }

  private phase3Compact(
    messages: Message[],
    _eligibleEnd: number,
    _counter: TokenCounter
  ): Message[] {
    const result: Message[] = [];
    const protectedSteps = this.protectedSteps(messages);

    for (let i = 0; i < messages.length; i++) {
      const m = messages[i];

      if (i === 0 || i === 1) {
        result.push(m);
        continue;
      }

      if (this.isProtected(m, protectedSteps)) {
        result.push(m);
        continue;
      }

      const msgType = m.meta?.type ?? MessageType.TEXT_RESPONSE;
      if (msgType === MessageType.REASONING || msgType === MessageType.TEXT_RESPONSE) {
        continue;
      }

      result.push(m);
    }

    return result;
  }
}

export function findEligibleEnd(
  messages: Message[],
  keepRecent: number
): number {
  if (keepRecent <= 0) {
    return messages.length - 1;
  }

  let maxStep = -1;
  for (const m of messages) {
    if (m.meta?.step_index !== undefined && m.meta.step_index > maxStep) {
      maxStep = m.meta.step_index;
    }
  }

  if (maxStep < 0) {
    const protectedCount = Math.min(keepRecent, messages.length);
    return messages.length - 1 - protectedCount;
  }

  const protectedSteps = new Map<number, boolean>();
  let currentStep = maxStep;
  for (let i = 0; i < keepRecent; i++) {
    protectedSteps.set(currentStep, true);
    currentStep--;
    if (currentStep < 0) {
      break;
    }
  }

  for (let i = messages.length - 1; i >= 0; i--) {
    const stepIdx = messages[i].meta?.step_index;
    if (stepIdx === undefined || !protectedSteps.get(stepIdx)) {
      return i;
    }
  }

  return -1;
}