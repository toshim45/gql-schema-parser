help:
	@echo "HELP: make clean|build|dev"
clean:
	@echo "cleaning"
	rm -rfv binary
build: clean
	@echo "building"
	go mod tidy; go build -o binary main.go	
dev:
	@echo "\n --running-- \n"
	@if [ -f binary ]; then ./binary; else echo "invalid binary"; fi
