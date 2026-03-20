#!/usr/bin/env python3
"""
Eval runner for testing tool call capabilities across models.
Connects to model endpoints and evaluates tool call generation.
"""

import json
import yaml
import requests
import time
import os
import fnmatch
from datetime import datetime
from typing import Dict, List, Any, Optional
from dataclasses import dataclass, asdict
from pathlib import Path

@dataclass
class EvalResult:
    task_id: str
    model_name: str
    prompt: str
    success: bool
    tool_calls_made: int
    expected_tools: List[str]
    actual_tools: List[str]
    error: Optional[str]
    turns: int
    timestamp: str

class ModelClient:
    def __init__(self, config: Dict[str, Any]):
        self.config = config
        self.endpoint = config.get("endpoint", "http://localhost:8080")
        self.model = config.get("model", config.get("name", ""))
        self.temperature = config.get("temperature", 0.7)
        self.max_tokens = config.get("max_tokens", 4096)
        
    def chat(self, messages: List[Dict[str, str]], tools: Optional[List[Dict]] = None) -> Dict:
        payload = {
            "model": self.model,
            "messages": messages,
            "temperature": self.temperature,
            "max_tokens": self.max_tokens,
        }
        if tools:
            payload["tools"] = tools
            
        try:
            resp = requests.post(f"{self.endpoint}/v1/chat/completions", json=payload, timeout=60)
            resp.raise_for_status()
            return resp.json()
        except requests.exceptions.RequestException as e:
            return {"error": str(e)}

class ToolExecutor:
    def __init__(self, tools_config: Dict):
        self.tools = tools_config.get("tools", [])
        
    def execute(self, tool_name: str, arguments: Dict) -> Dict:
        if tool_name == "Glob":
            return self._glob(arguments)
        elif tool_name == "Read":
            return self._read(arguments)
        elif tool_name == "Grep":
            return self._grep(arguments)
        return {"error": f"Unknown tool: {tool_name}"}
    
    def _glob(self, args: Dict) -> Dict:
        pattern = args.get("pattern", "*")
        path = args.get("path", ".")
        matches = []
        try:
            for root, dirs, files in os.walk(path):
                for f in files:
                    if fnmatch.fnmatch(f, pattern):
                        matches.append(os.path.join(root, f))
        except Exception as e:
            return {"error": str(e), "matches": []}
        return {"matches": matches, "count": len(matches)}
    
    def _read(self, args: Dict) -> Dict:
        filepath = args.get("filePath", "")
        offset = args.get("offset", 1)
        limit = args.get("limit", 2000)
        try:
            with open(filepath, 'r') as f:
                lines = f.readlines()
                start = max(0, offset - 1)
                end = min(len(lines), start + limit)
                content = ''.join(lines[start:end])
                return {"content": content, "lines": end - start}
        except Exception as e:
            return {"error": str(e), "content": ""}
    
    def _grep(self, args: Dict) -> Dict:
        pattern = args.get("pattern", "")
        path = args.get("path", ".")
        include = args.get("include", "*")
        matches = []
        try:
            for root, dirs, files in os.walk(path):
                for f in files:
                    if not fnmatch.fnmatch(f, include):
                        continue
                    filepath = os.path.join(root, f)
                    try:
                        with open(filepath, 'r') as file:
                            for i, line in enumerate(file, 1):
                                if pattern in line:
                                    matches.append({"file": filepath, "line": i, "content": line.strip()})
                    except:
                        pass
        except Exception as e:
            return {"error": str(e), "matches": []}
        return {"matches": matches, "count": len(matches)}

def parse_tool_calls(response: str, tool_format: str = "direct") -> List[Dict]:
    """Parse tool calls from model response."""
    calls = []
    
    # Try JSON format (OpenAI-style)
    try:
        data = json.loads(response)
        if "tools" in data:
            for tool in data.get("tool_calls", []):
                calls.append({
                    "name": tool.get("function", {}).get("name", ""),
                    "arguments": json.loads(tool.get("function", {}).get("arguments", "{}"))
                })
        return calls
    except:
        pass
    
    # Try custom format: <|python_tag|>{"name": "...", "parameters": {...}}
    import re
    # Match from <|python_tag|> to end of JSON object (handles nested braces)
    python_tag_pattern = r'<\|python_tag\|>(\{.*\})'
    matches = re.findall(python_tag_pattern, response, re.DOTALL)
    for match in matches:
        try:
            data = json.loads(match)
            name = data.get("name", "")
            params = data.get("parameters", {})
            calls.append({"name": name, "arguments": params})
        except:
            pass
    
    # Try XML format (Mistral-style)
    xml_pattern = r'<tool>(.*?)</tool>\s*<input>(.*?)</input>'
    matches = re.findall(xml_pattern, response, re.DOTALL)
    for name, args in matches:
        calls.append({"name": name.strip(), "arguments": {}})
    
    return calls

def format_tools_for_prompt(tools: List[Dict]) -> str:
    """Format tools description for the prompt."""
    result = "You have access to the following tools:\n\n"
    for tool in tools:
        result += f"## {tool['name']}\n"
        result += f"{tool.get('description', '')}\n"
        result += "Parameters:\n"
        for param, spec in tool.get('parameters', {}).get('properties', {}).items():
            result += f"  - {param}: {spec.get('description', '')}\n"
        result += "\n"
    return result

def convert_tools_to_openai_format(tools: List[Dict]) -> List[Dict]:
    """Convert file_tools.yaml format to OpenAI function format."""
    openai_tools = []
    for tool in tools:
        openai_tools.append({
            "type": "function",
            "function": {
                "name": tool["name"],
                "description": tool.get("description", ""),
                "parameters": {
                    "type": "object",
                    "properties": tool.get("parameters", {}).get("properties", {}),
                    "required": tool.get("parameters", {}).get("required", [])
                }
            }
        })
    return openai_tools

def run_eval(client: ModelClient, task: Dict, tools: List[Dict], executor: ToolExecutor, max_turns: int = 10) -> EvalResult:
    """Run a single eval task."""
    prompt = task["input"]["prompt"]
    expected_tools = task["graders"][0]["config"]["expected_tools"]
    
    tools_desc = format_tools_for_prompt(tools)
    system_msg = f"You are a helpful coding assistant. {tools_desc}"
    
    messages = [
        {"role": "system", "content": system_msg},
        {"role": "user", "content": prompt}
    ]
    
    tool_calls_made = 0
    actual_tools = []
    error = None
    turns = 0
    
    openai_tools = convert_tools_to_openai_format(tools)
    
    for turn in range(max_turns):
        turns = turn + 1
        resp = client.chat(messages, openai_tools)
        
        if "error" in resp:
            error = resp["error"]
            break
            
        message = resp.get("choices", [{}])[0].get("message", {})
        content = message.get("content", "")
        tool_calls = message.get("tool_calls", [])
        
        if not tool_calls:
            # Try to parse from content
            tool_calls = parse_tool_calls(content)
        
        if not tool_calls:
            # No tool calls made, check if we got a final answer
            if content:
                messages.append({"role": "assistant", "content": content})
                break
            error = "No tool calls or response generated"
            break
        
        for tc in tool_calls:
            tool_calls_made += 1
            tool_name = tc.get("name", tc.get("function", {}).get("name", ""))
            args = tc.get("arguments", tc.get("function", {}).get("arguments", {}))
            
            if isinstance(args, str):
                try:
                    args = json.loads(args)
                except:
                    args = {}
            
            actual_tools.append(tool_name)
            
            # Execute tool
            result = executor.execute(tool_name, args)
            
            # Add tool result to messages
            messages.append({
                "role": "assistant",
                "tool_calls": [{"id": f"call_{tool_calls_made}", "type": "function", "function": {"name": tool_name, "arguments": json.dumps(args)}}]
            })
            messages.append({
                "role": "tool",
                "tool_call_id": f"call_{tool_calls_made}",
                "content": json.dumps(result)
            })
    
    # Determine success
    success = tool_calls_made > 0 and any(et in actual_tools for et in expected_tools)
    
    return EvalResult(
        task_id=task["id"],
        model_name=client.model or client.config.get("name", "unknown"),
        prompt=prompt,
        success=success,
        tool_calls_made=tool_calls_made,
        expected_tools=expected_tools,
        actual_tools=actual_tools,
        error=error,
        turns=turns,
        timestamp=datetime.now().isoformat()
    )

def load_config(config_path: str) -> Dict:
    with open(config_path) as f:
        return yaml.safe_load(f)

def save_results(results: List[EvalResult], output_dir: str):
    os.makedirs(output_dir, exist_ok=True)
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    
    # Save JSON
    json_path = os.path.join(output_dir, f"eval_results_{timestamp}.json")
    with open(json_path, 'w') as f:
        json.dump([asdict(r) for r in results], f, indent=2)
    
    # Save Markdown summary
    md_path = os.path.join(output_dir, f"eval_summary_{timestamp}.md")
    with open(md_path, 'w') as f:
        f.write(f"# Eval Results - {timestamp}\n\n")
        f.write(f"Total Tasks: {len(results)}\n")
        f.write(f"Passed: {sum(1 for r in results if r.success)}\n")
        f.write(f"Failed: {sum(1 for r in results if not r.success)}\n\n")
        f.write("## Results\n\n")
        f.write("| Task | Model | Success | Tool Calls | Turns | Expected | Actual |\n")
        f.write("|------|-------|---------|------------|-------|----------|--------|\n")
        for r in results:
            f.write(f"| {r.task_id} | {r.model_name} | {'✅' if r.success else '❌'} | {r.tool_calls_made} | {r.turns} | {', '.join(r.expected_tools)} | {', '.join(r.actual_tools)} |\n")
        
        f.write("\n## Details\n\n")
        for r in results:
            f.write(f"### {r.task_id}\n")
            f.write(f"**Prompt:** {r.prompt}\n\n")
            if r.error:
                f.write(f"**Error:** {r.error}\n\n")
            f.write("---\n\n")
    
    print(f"Results saved to {json_path}")
    print(f"Summary saved to {md_path}")
    return json_path, md_path

def main():
    import argparse
    parser = argparse.ArgumentParser(description="Run tool call evals")
    parser.add_argument("--model", default="gpt-oss-20b", help="Model name from config")
    parser.add_argument("--endpoint", default="http://100.121.229.114:8000", help="Model endpoint")
    parser.add_argument("--config-dir", default="configs", help="Config directory")
    parser.add_argument("--output-dir", default="../results", help="Output directory")
    parser.add_argument("--max-turns", type=int, default=10, help="Max turns per task")
    args = parser.parse_args()
    
    # Load configs
    models_config = load_config(os.path.join(args.config_dir, "models.yaml"))
    tasks_config = load_config(os.path.join(args.config_dir, "toolcall_suite.yaml"))
    tools_config = load_config(os.path.join(args.config_dir, "../tools/file_tools.yaml"))
    
    # Find model config
    model_cfg = None
    for m in models_config["models"]:
        if m["name"] == args.model:
            model_cfg = m
            break
    
    if not model_cfg:
        print(f"Model {args.model} not found, using command-line endpoint")
        model_cfg = {"name": args.model, "endpoint": args.endpoint, "model": args.model}
    
    # Override endpoint for gpt-oss
    if args.model == "gpt-oss-20b":
        model_cfg["endpoint"] = args.endpoint
    
    client = ModelClient(model_cfg)
    executor = ToolExecutor(tools_config)
    
    results = []
    for task in tasks_config["tasks"]:
        print(f"Running task: {task['id']}...")
        result = run_eval(client, task, tools_config["tools"], executor, args.max_turns)
        results.append(result)
        print(f"  -> {'✅ PASS' if result.success else '❌ FAIL'} ({result.tool_calls_made} calls, {result.turns} turns)")
        if result.error:
            print(f"     Error: {result.error}")
    
    save_results(results, args.output_dir)

if __name__ == "__main__":
    main()