"""Hermes Agent client - subprocess-based wrapper."""

import json
import subprocess
from dataclasses import dataclass
from pathlib import Path
from typing import Any


@dataclass
class HermesTool:
    """Represents a Hermes tool/skill."""

    name: str
    description: str


class Client:
    """Client for Hermes Agent CLI.

    Wraps the hermes CLI for tool execution and skill management.
    """

    def __init__(
        self,
        config_path: Path | None = None,
        hermes_path: str = "hermes",
    ):
        self.config_path = config_path or Path.home() / ".hermes"
        self.hermes_path = hermes_path
        self._base_args = [hermes_path]

    def list_tools(self) -> list[HermesTool]:
        """List available tools/skills from Hermes."""
        try:
            result = subprocess.run(
                self._base_args + ["tools", "list", "--json"],
                capture_output=True,
                text=True,
                timeout=30,
            )
            if result.returncode != 0:
                return []
            data = json.loads(result.stdout)
            return [
                HermesTool(
                    name=t.get("name", ""),
                    description=t.get("description", ""),
                )
                for t in data.get("tools", [])
            ]
        except (subprocess.TimeoutExpired, json.JSONDecodeError):
            return []

    def execute(
        self,
        tool_name: str,
        args: dict[str, Any],
        timeout: float = 60.0,
    ) -> tuple[str, str | None]:
        """Execute a tool via Hermes.

        Returns:
            Tuple of (output, error)
        """
        try:
            cmd = self._base_args + [
                "tool",
                "run",
                tool_name,
                "--args",
                json.dumps(args),
            ]
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=timeout,
            )
            if result.returncode != 0:
                return "", result.stderr
            return result.stdout, None
        except subprocess.TimeoutExpired:
            return "", "timeout"
        except Exception as e:
            return "", str(e)

    def get_skills(self) -> list[str]:
        """List available skills."""
        try:
            result = subprocess.run(
                self._base_args + ["skills", "list"],
                capture_output=True,
                text=True,
                timeout=30,
            )
            if result.returncode != 0:
                return []
            return [
                line.strip()
                for line in result.stdout.split("\n")
                if line.strip() and not line.startswith("#")
            ]
        except subprocess.TimeoutExpired:
            return []

    def get_config(self) -> dict[str, Any]:
        """Get Hermes configuration."""
        config_file = self.config_path / "config.json"
        if config_file.exists():
            with open(config_file) as f:
                return json.load(f)
        return {}