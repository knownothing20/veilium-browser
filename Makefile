.PHONY: fmt vet test build check frontend-install frontend-test frontend-build desktop-dev desktop-build

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

vet:
	go vet ./cmd/... ./internal/...

test:
	go test ./cmd/... ./internal/...

build:
	go build ./cmd/... ./internal/...

frontend-install:
	cd frontend && npm install --no-audit --no-fund

frontend-test:
	cd frontend && npm run typecheck && npm test

frontend-build:
	cd frontend && npm run build

desktop-dev:
	wails dev

desktop-build:
	wails build

check:
	test -z "$$(gofmt -l .)"
	go vet ./cmd/... ./internal/...
	go test ./cmd/... ./internal/...
	go build ./cmd/... ./internal/...
	cd frontend && npm run typecheck
	cd frontend && npm test
	cd frontend && npm run build
