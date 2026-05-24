import type { Message } from "../llm/request.js";
import type { CompactionStrategy, TokenCounter } from "./strategies.js";
import { TieredCompact } from "./strategies.js";

export class ContextWindowManager {
  private contextWindow: number;
  private compactionRatio: number;
  private counter: TokenCounter;
  private strategy: CompactionStrategy;
  private lastKnownTokens: number | null;
  private lock: Int32Array;

  constructor(
    contextWindow: number,
    counter: TokenCounter | null = null,
    strategy: CompactionStrategy | null = null
  ) {
    this.contextWindow = contextWindow;
    this.compactionRatio = 0.75;
    this.counter = counter ?? defaultCounter;
    this.strategy = strategy ?? new TieredCompact();
    this.lastKnownTokens = null;
    this.lock = new Int32Array(1);
  }

  setCompactionRatio(ratio: number): void {
    Atomics.store(this.lock, 0, 1);
    try {
      this.compactionRatio = ratio;
    } finally {
      Atomics.store(this.lock, 0, 0);
    }
  }

  updateTokenCount(totalTokens: number): void {
    Atomics.store(this.lock, 0, 1);
    try {
      this.lastKnownTokens = totalTokens;
    } finally {
      Atomics.store(this.lock, 0, 0);
    }
  }

  check(messages: Message[]): [number, boolean] {
    Atomics.store(this.lock, 0, 1);
    try {
      const currentTokens = this.estimateTokens(messages);
      const threshold = Math.floor(this.contextWindow * this.compactionRatio);
      return [currentTokens, currentTokens >= threshold];
    } finally {
      Atomics.store(this.lock, 0, 0);
    }
  }

  shouldCompact(messages: Message[]): boolean {
    Atomics.store(this.lock, 0, 1);
    try {
      const currentTokens = this.estimateTokens(messages);
      const threshold = Math.floor(this.contextWindow * this.compactionRatio);
      return currentTokens > threshold;
    } finally {
      Atomics.store(this.lock, 0, 0);
    }
  }

  compact(messages: Message[]): Message[] {
    Atomics.store(this.lock, 0, 1);
    try {
      const targetTokens = Math.floor(this.contextWindow * this.compactionRatio);
      const compacted = this.strategy.compact(messages, targetTokens, this.counter);
      this.lastKnownTokens = null;
      return compacted;
    } finally {
      Atomics.store(this.lock, 0, 0);
    }
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