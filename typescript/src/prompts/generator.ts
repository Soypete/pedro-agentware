import type { ToolRegistry } from "../tools/index.js";

export interface PromptGenerator {
  generateToolSection(registry: ToolRegistry): string;
  generateToolSchemas(registry: ToolRegistry): object[];
}

export class DefaultPromptGenerator implements PromptGenerator {
  generateToolSection(registry: ToolRegistry): string {
    const tools = registry.all();
    if (tools.length === 0) return "";

    const lines = ["## Available Tools\n"];
    for (const tool of tools) {
      lines.push(`- **${tool.name}**: ${tool.description}`);
    }
    return lines.join("\n");
  }

  generateToolSchemas(registry: ToolRegistry): object[] {
    return registry.all().map((tool) => ({
      type: "function",
      function: {
        name: tool.name,
        description: tool.description,
        parameters:
          "inputSchema" in tool
            ? (tool as unknown as { inputSchema(): Record<string, unknown> }).inputSchema()
            : {},
      },
    }));
  }
}