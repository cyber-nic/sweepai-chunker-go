CYAN=\033[0;36m
NC=\033[0m # No Color

.PHONY: run

run:
	@ printf "$(CYAN)run$(NC)\n"
	go run ./cmd/main.go
