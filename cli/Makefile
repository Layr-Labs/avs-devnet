.PHONY: install deps

deps:
	@echo "Installing dependencies..."
	go mod tidy

install:
	@echo "Installing package..."
	go install ./...
	@-asdf reshim 2> /dev/null
