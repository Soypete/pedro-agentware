"""Policy loader for YAML-based policy configuration."""

from typing import Any
import yaml

from middleware_py.types import Action, Condition, RateLimitConfig
from middleware_py.policy import Policy, Rule


def load_policy_from_yaml(data: str | bytes) -> Policy:
    """Load policy from YAML string or bytes."""
    if isinstance(data, str):
        data = data.encode("utf-8")

    config = yaml.safe_load(data) or {}

    rules = []
    for rule_config in config.get("rules", []) or []:
        rule = _parse_rule(rule_config)
        rules.append(rule)

    return Policy(
        rules=rules,
        default_deny=config.get("default_deny", False),
    )


def load_policy_from_file(path: str) -> Policy:
    """Load policy from a YAML file."""
    with open(path, "r") as f:
        return load_policy_from_yaml(f.read())


def _parse_rule(config: dict[str, Any]) -> Rule:
    """Parse a rule from config dict."""
    conditions = []
    for cond_config in config.get("conditions", []):
        conditions.append(
            Condition(
                field=cond_config["field"],
                operator=cond_config["operator"],
                value=cond_config.get("value"),
            )
        )

    max_rate = None
    if "max_rate" in config:
        max_rate = RateLimitConfig(
            count=config["max_rate"]["count"],
            window=config["max_rate"]["window"],
        )

    return Rule(
        name=config["name"],
        tools=config.get("tools", ["*"]),
        action=Action(config["action"]),
        conditions=conditions,
        max_rate=max_rate,
        max_turns=config.get("max_turns"),
        max_iterations=config.get("max_iterations"),
        redact_fields=config.get("redact_fields", []),
    )