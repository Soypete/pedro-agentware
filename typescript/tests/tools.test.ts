import { ToolRegistry, BaseTool, Result } from "../src/tools/index.js";

class AddTool extends BaseTool {
  constructor() {
    super("add", "Add two numbers");
  }

  execute(args: Record<string, unknown>): Result {
    const a = (args.a as number) || 0;
    const b = (args.b as number) || 0;
    return new Result(true, a + b);
  }
}

describe("ToolRegistry", () => {
  it("should register a tool", () => {
    const registry = new ToolRegistry();
    const tool = new AddTool();
    registry.register(tool);
    expect(registry.names()).toContain("add");
  });

  it("should get a tool by name", () => {
    const registry = new ToolRegistry();
    const tool = new AddTool();
    registry.register(tool);
    const found = registry.get("add");
    expect(found).toBeDefined();
    expect(found?.name).toBe("add");
  });

  it("should return all tools", () => {
    const registry = new ToolRegistry();
    registry.register(new AddTool());
    const all = registry.all();
    expect(all.length).toBe(1);
  });
});

describe("Result", () => {
  it("should serialize to JSON", () => {
    const result = new Result(true, 42);
    const json = result.toJSON();
    expect(json).toHaveProperty("success", true);
    expect(json).toHaveProperty("data", 42);
  });
});