.PHONY: build web go test check dev dev-memory seed seed-check clean

BIN := bin/wooddust

# Full production build: SPA, then the Go binary with web/dist embedded.
build: web go

web:
	cd web && npm ci && npm run build
	touch web/dist/.gitkeep

go:
	mkdir -p web/dist && touch web/dist/.gitkeep
	go build -trimpath -o $(BIN) ./cmd/wooddust

test:
	go test -p 1 ./...

# Canonical verification (vet, tests, schema, integrity).
check:
	./scripts/check.sh

dev:
	./scripts/dev.sh

dev-memory:
	./scripts/dev.sh --memory

seed: go
	$(BIN) seed

seed-check:
	go run ./cmd/wooddust seed -check

clean:
	rm -rf bin
