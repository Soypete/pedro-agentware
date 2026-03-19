# Agent Middleware Evaluation Framework

This directory contains the evaluation framework for testing tool call capabilities across different models.

## Structure

- `configs/` - Evaluation task definitions and model configurations
- `tools/` - Tool definitions for testing (file search, read, etc.)
- `results/` - Output directory for eval results (gitignored)

## Usage

Run evals with different models to compare tool call capabilities.

## References

- PedroCLI's `pkg/evals` for the evaluation harness
- `middleware/format/` for model-specific tool call formatters