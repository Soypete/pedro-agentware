# Python Adapters Implementation Plan

## Related Issue
GitHub Issue: [#29](https://github.com/Soypete/pedro-agentware/issues/29) - Research: Integration with Kitaru, Pydantic Agents, and Hermes Agent

## Goal
Create Python adapter packages for Kitaru, Hermes, and Pydantic agent backends that integrate with pedro-agentware middleware, following Python-native patterns.

## Architecture
```
Python Agent (Kitaru/PydanticAI/Hermes) <--Python Adapters--> pedro-agentware Middleware
                                                                  |
                                                                  v
                                                           Tool Execution
```

## Directory Structure
```
python/adapters/
├── __init__.py                 # Public API exports
├── base.py                     # Base classes/interfaces
├── kitaru/
│   ├── __init__.py
│   ├── adapter.py              # Main adapter
│   ├── client.py               # HTTP client
│   └── pyproject.toml
├── hermes/
│   ├── __init__.py
│   ├── adapter.py
│   ├── client.py
│   └── pyproject.toml
└── pydantic/
    ├── __init__.py
    ├── adapter.py
    ├── client.py
    └── pyproject.toml
```

## Implementation Steps

### Phase 1: Base Interface & Types
1. Create `python/adapters/base.py`:
   - `AgentBackend` Protocol (matches Python Executor pattern)
   - `AgentTool` dataclass (name, description, input_schema)
   - `AgentResult` dataclass (success, output, error, metadata)

2. Update `python/adapters/__init__.py` to export public API

### Phase 2: Kitaru Adapter
1. Create `python/adapters/kitaru/client.py`:
   - HTTP client using `httpx` for REST API on port 8080
   - Flow execution, polling, checkpoint/restore, artifacts

2. Create `python/adapters/kitaru/adapter.py`:
   - `KitaruAdapter` implementing `AgentBackend` Protocol
   - Map tool names to flow names
   - Configurable timeouts, retry logic

3. Create `python/adapters/kitaru/pyproject.toml`:
   - Dependencies: httpx, pydantic

### Phase 3: Hermes Adapter
1. Research Hermes Agent Python API:
   - Check if `hermes-agent` pip package available
   - Or use subprocess CLI wrapper

2. Create `python/adapters/hermes/client.py`:
   - HTTP or subprocess client for Hermes

3. Create `python/adapters/hermes/adapter.py`:
   - `HermesAdapter` implementing `AgentBackend` Protocol

4. Create `python/adapters/hermes/pyproject.toml`

### Phase 4: Pydantic Adapter
1. Create `python/adapters/pydantic/client.py`:
   - Wrap `pydantic-ai` Agent

2. Create `python/adapters/pydantic/adapter.py`:
   - `PydanticAdapter` implementing `AgentBackend` Protocol
   - Extract tools from @agent.tool decorators
   - Handle streaming responses

3. Create `python/adapters/pydantic/pyproject.toml`

### Phase 5: Testing Setup

#### Unit Tests (Mock-based)
Location: `python/tests/adapters/`
- Use `unittest.mock` for external dependencies
- Test success and failure paths
- Table-driven tests for edge cases

#### Integration Tests (Qwen3.6)
```python
# python/tests/adapters/conftest.py
@pytest.fixture
def qwen_backend():
    return Backend(
        base_url="http://pedrogpt:8080/v1",
        model="qwen3.6",
    )
```

#### Docker Setup
- `docker-compose.yml` for test infrastructure
- Services: pedrogpt (qwen3.6), websearch-agent

### Phase 6: PR Creation
1. Branch: `feature/python-adapters`
2. Link to issue #29

## Dependencies
```toml
[project.optional-dependencies]
adapters = [
    "httpx>=0.25",
    "pydantic-ai>=0.0.20",
]
dev = [
    "pytest>=7.0",
    "pytest-cov>=4.0",
    "pytest-asyncio>=0.21",
    "ruff>=0.1",
    "mypy>=1.0",
    "requests-mock>=1.11",
]
```

## Commands
```bash
# Unit tests
cd python && pytest tests/adapters/ -v

# With coverage
cd python && pytest tests/adapters/ --cov=adapters --cov-report=html

# Linting
cd python && ruff check adapters/

# Type checking
cd python && mypy adapters/
```