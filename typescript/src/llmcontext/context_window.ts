import type { Message } from "../llm/request.js";
import type { CompactionStrategy, TokenCounter } from "./strategies.js";
import { TieredCompact } from "./strategies.js";

export interface CompactEvent {
  step_index: number;
  tokens_before: number;
  tokens_after: number;
  budget_tokens: number;
  messages_before: number;
  messages_after: number;
  phase_reached: number;
  strategy_name: string;
}

export type CompactCallback = (event: CompactEvent) => void;

export type ThresholdCallback = (
  tokens: number,
  budget: number,
  pct: number
) => string | null;

export function defaultContextWarning(
  tokens: number,
  budget: number,
  pct: number
): string | null {
  if (pct >= 0.8) {
    return "[Context is nearly full. Summarize critical findings and complete current task.]";
  }
  if (pct >= 0.65) {
    return "[Context is filling up. Be concise and front-load important information.]";
  }
  return null;
}

export class ContextWindowManager {
  private contextWindow: number;
  private compactionRatio: number;
  private counter: TokenCounter;
  private strategy: CompactionStrategy;
  private lastKnownTokens: number | null;
  private contextThresholds: number[];
  private onContextThreshold: ThresholdCallback;
  private firedThresholds: Set<number>;
  private onCompact: CompactCallback | null;

  constructor(
    contextWindow: number,
    counter: TokenCounter | null = null,
    strategy: CompactionStrategy | null = null,
    contextThresholds: number[] | null = null,
    onContextThreshold: ThresholdCallback | null = null,
    onCompact: CompactCallback | null = null
  ) {
    this.contextWindow = contextWindow;
    this.compactionRatio = 0.75;
    this.counter = counter ?? defaultCounter;
    this.strategy = strategy ?? new TieredCompact();
    this.lastKnownTokens = null;
    this.contextThresholds = contextThresholds
      ? [...contextThresholds].sort((a, b) => a - b)
      : [0.65, 0.8];
    this.onContextThreshold = onContextThreshold ?? defaultContextWarning;
    this.firedThresholds = new Set();
    this.onCompact = onCompact;
  }

  setCompactionRatio(ratio: number): void {
    this.compactionRatio = ratio;
  }

  updateTokenCount(totalTokens: number): void {
    this.lastKnownTokens = totalTokens;
  }

  check(messages: Message[]): [number, boolean] {
    const currentTokens = this.estimateTokens(messages);
    const threshold = Math.floor(this.contextWindow * this.compactionRatio);
    return [currentTokens, currentTokens >= threshold];
  }

  shouldCompact(messages: Message[]): boolean {
    const currentTokens = this.estimateTokens(messages);
    const threshold = Math.floor(this.contextWindow * this.compactionRatio);
    return currentTokens > threshold;
  }

  compact(messages: Message[]): Message[] {
    const tokensBefore = this.estimateTokens(messages);
    const messagesBefore = messages.length;
    const targetTokens = Math.floor(this.contextWindow * this.compactionRatio);
    const compacted = this.strategy.compact(messages, targetTokens, this.counter);
    const tokensAfter = this.counter(compacted);
    const messagesAfter = compacted.length;

    let phaseReached = 0;
    if (this.strategy instanceof TieredCompact) {
      phaseReached = (this.strategy as TieredCompact).lastPhase;
    }

    if (this.onCompact !== null) {
      const event: CompactEvent = {
        step_index: 0,
        tokens_before: tokensBefore,
        tokens_after: tokensAfter,
        budget_tokens: this.contextWindow,
        messages_before: messagesBefore,
        messages_after: messagesAfter,
        phase_reached: phaseReached,
        strategy_name: this.strategy.name(),
      };
      this.onCompact(event);
    }

    this.lastKnownTokens = null;
    this.firedThresholds.clear();
    return compacted;
  }

  checkThresholds(messages: Message[]): string | null {
    const currentTokens = this.estimateTokens(messages);
    const budget = this.contextWindow;
    if (currentTokens === 0 || budget === 0) {
      return null;
    }

    const pct = currentTokens / budget;

    for (let i = this.contextThresholds.length - 1; i >= 0; i--) {
      const threshold = this.contextThresholds[i];
      if (pct >= threshold) {
        if (!this.firedThresholds.has(threshold)) {
          this.firedThresholds.add(threshold);
          return this.onContextThreshold(currentTokens, budget, pct);
        }
      }
    }
    return null;
  }

  private estimateTokens(messages: Message[]): number {
    if (this.lastKnownTokens !== null) {
      return this.lastKnownTokens;
    }
    return this.counter(messages);
  }
}

export function defaultCounter(messages: Message[]): number {
  let total = 0;
  for (const m of messages) {
    let overhead = m.role.length + 4;
    if (m.tool_calls) {
      for (const tc of m.tool_calls) {
        overhead += tc.name.length + 1;
      }
    }
    total += Math.floor(m.content.length / 4) + overhead;
  }
  return total;
}