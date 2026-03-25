"""Tests for policy_loader module."""

import os
import tempfile
from middleware_py.policy_loader import load_policy_from_yaml, load_policy_from_file
from middleware_py.middleware_types import Action


def test_load_policy_from_string():
    """Test loading policy from YAML string."""
    yaml = """
rules:
  - name: allow-all
    tools:
      - "*"
    action: allow
default_deny: false
"""
    policy = load_policy_from_yaml(yaml)

    assert len(policy.rules) == 1
    assert policy.rules[0].name == "allow-all"
    assert policy.default_deny is False


def test_load_policy_from_bytes():
    """Test loading policy from YAML bytes."""
    yaml = b"""
rules:
  - name: deny-tool
    tools:
      - "dangerous"
    action: deny
default_deny: true
"""
    policy = load_policy_from_yaml(yaml)

    assert len(policy.rules) == 1
    assert policy.rules[0].action == Action.DENY
    assert policy.default_deny is True


def test_load_policy_from_string_multiple_rules():
    """Test loading policy with multiple rules."""
    yaml = """
rules:
  - name: allow-tool1
    tools:
      - "tool1"
    action: allow
  - name: deny-tool2
    tools:
      - "tool2"
    action: deny
default_deny: false
"""
    policy = load_policy_from_yaml(yaml)

    assert len(policy.rules) == 2
    assert policy.rules[0].action == Action.ALLOW
    assert policy.rules[1].action == Action.DENY


def test_load_policy_with_conditions():
    """Test loading policy with conditions."""
    yaml = """
rules:
  - name: allow-admin-only
    tools:
      - "admin_tool"
    action: allow
    conditions:
      - field: caller.role
        operator: eq
        value: admin
"""
    policy = load_policy_from_yaml(yaml)

    assert len(policy.rules) == 1
    rule = policy.rules[0]
    assert len(rule.conditions) == 1
    assert rule.conditions[0].field == "caller.role"
    assert rule.conditions[0].operator == "eq"
    assert rule.conditions[0].value == "admin"


def test_load_policy_with_rate_limit():
    """Test loading policy with rate limiting."""
    yaml = """
rules:
  - name: rate-limited-tool
    tools:
      - "expensive_tool"
    action: allow
    max_rate:
      count: 10
      window: 60
"""
    policy = load_policy_from_yaml(yaml)

    assert len(policy.rules) == 1
    rule = policy.rules[0]
    assert rule.max_rate is not None
    assert rule.max_rate.count == 10
    assert rule.max_rate.window == 60


def test_load_policy_with_max_turns():
    """Test loading policy with max turns."""
    yaml = """
rules:
  - name: limited-turns
    tools:
      - "*"
    action: allow
    max_turns: 5
"""
    policy = load_policy_from_yaml(yaml)

    assert len(policy.rules) == 1
    assert policy.rules[0].max_turns == 5


def test_load_policy_with_max_iterations():
    """Test loading policy with max iterations."""
    yaml = """
rules:
  - name: limited-iterations
    tools:
      - "*"
    action: allow
    max_iterations: 100
"""
    policy = load_policy_from_yaml(yaml)

    assert len(policy.rules) == 1
    assert policy.rules[0].max_iterations == 100


def test_load_policy_with_redact_fields():
    """Test loading policy with redact fields."""
    yaml = """
rules:
  - name: filter-sensitive
    tools:
      - "secret_tool"
    action: filter
    redact_fields:
      - password
      - api_key
"""
    policy = load_policy_from_yaml(yaml)

    assert len(policy.rules) == 1
    assert policy.rules[0].redact_fields == ["password", "api_key"]


def test_load_policy_default_tools():
    """Test that tools defaults to wildcard."""
    yaml = """
rules:
  - name: allow-all
    action: allow
"""
    policy = load_policy_from_yaml(yaml)

    assert policy.rules[0].tools == ["*"]


def test_load_policy_invalid_yaml():
    """Test loading invalid YAML raises error."""
    invalid_yaml = "invalid: yaml: [[["

    try:
        load_policy_from_yaml(invalid_yaml)
        assert False, "Expected exception"
    except Exception:
        pass


def test_load_policy_from_file():
    """Test loading policy from file."""
    with tempfile.NamedTemporaryFile(mode="w", suffix=".yaml", delete=False) as f:
        f.write("""
rules:
  - name: test-rule
    tools:
      - "foo"
      - "bar"
    action: allow
default_deny: true
""")
        f.flush()
        path = f.name

    try:
        policy = load_policy_from_file(path)

        assert len(policy.rules) == 1
        assert policy.rules[0].name == "test-rule"
        assert policy.default_deny is True
    finally:
        os.unlink(path)


def test_load_policy_file_not_found():
    """Test loading from nonexistent file raises error."""
    try:
        load_policy_from_file("/nonexistent/path/policy.yaml")
        assert False, "Expected exception"
    except FileNotFoundError:
        pass


def test_load_policy_empty_rules():
    """Test loading policy with no rules."""
    yaml = """
default_deny: false
"""
    policy = load_policy_from_yaml(yaml)

    assert len(policy.rules) == 0
    assert policy.default_deny is False


def test_load_policy_empty_string():
    """Test loading empty YAML."""
    yaml = ""
    policy = load_policy_from_yaml(yaml)

    assert len(policy.rules) == 0
    assert policy.default_deny is False