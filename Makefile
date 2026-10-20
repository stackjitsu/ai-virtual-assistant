.PHONY: format lint static unit contract fuzz-smoke fuzz-full setup-check run

format lint static unit contract fuzz-smoke fuzz-full:
	python3 scripts/check.py $@

setup-check:
	python3 scripts/check.py setup

# Runtime is supplied by CHAT-01; this intentionally fails until it exists.
run:
	go run ./cmd/assistant
