.PHONY: help evals evals-file-search evals-general evals-clean

help:
	@echo "Available targets:"
	@echo "  evals              - Run all evals (file search + general) sequentially against models"
	@echo "  evals-file-search  - Run only file search tool call evals"
	@echo "  evals-general      - Run only general tool calling evals"
	@echo "  evals-clean        - Clean eval output files"
	@echo ""
	@echo "Environment variables / args:"
	@echo "  EVAL_BASE_URL      - API base URL (default: http://pedrogpt:8080/v1)"
	@echo "  EVAL_MODELS        - Comma-separated model list (default: gpt-oss,nemotron,qwen)"
	@echo "  --models           - Override models via CLI"
	@echo "  --base-url         - Override base URL via CLI"

evals:
	python3 -m testing.evals.main --all --models nemotron-3-super-120b

evals-file-search:
	python3 -m testing.evals.main --file-search --models nemotron-3-super-120b

evals-general:
	python3 -m testing.evals.main --general --models nemotron-3-super-120b

evals-clean:
	rm -rf testing/evals/output/*.json
	@echo "Cleaned eval output files"