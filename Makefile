help:
	@echo "HELP: make clean|build|dev-start"
clean:
	@echo "---cleaning---"
	rm -rfv gqlsch
build: clean
	@echo "---building---"
	go mod tidy; go build -o gqlsch main.go
install: build
	@echo "---installing---"
	@if [ -z "$$GOPATH" ]; then echo "cp gqlsch <path-to-your-exec-path>"; else go install; fi
dev-start:
	@echo "\n---running---\n"
	@if [ -f gqlsch ]; then ./gqlsch; else echo "invalid binary"; fi
