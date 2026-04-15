import type { Message } from "./request.js";
import type { Response } from "./response.js";

export interface Backend {
  complete(messages: Message[]): Response;
  supportsNativeToolCalling(): boolean;
  modelName(): string;
  contextWindowSize(): number;
}