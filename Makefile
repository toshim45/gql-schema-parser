help:
	@echo "HELP: make clean|build|dev-start|install"
clean:
	@echo "---cleaning---"
	rm -rfv gqlsch
build: clean
	@echo "---building---"
	go mod tidy; go build -o gqlsch main.go
	
dev-start:
	@echo "\n---running---\n"
	@if [ -f gqlsch ]; then ./gqlsch; else echo "invalid binary"; fi

install: build
	@echo "---installing---"
ifneq (${GOPATH},) 
	@go install -v
else
	@echo "GOPATH not defined, try BINPATH"
ifneq (${BINPATH},)
	cp -v gqlsch ${BINPATH}
else
	@echo "BINPATH not defined"
endif
endif
