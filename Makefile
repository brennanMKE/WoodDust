.PHONY: build web go test check dev dev-memory seed seed-check clean \
	provision-preprod provision-prod deploy-preprod deploy-prod status-preprod status-prod \
	logs-preprod logs-prod rollback-preprod rollback-prod \
	posters shortlinks-key-preprod shortlinks-key-prod

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

# Shop posters: stock QR codes and print files (docs/posters.md). Local only.
posters:
	./scripts/posters.sh

# ---- photon (docs/deployment.md). preprod = wooddust.sstools.co, prod = www.dustalmanac.com
provision-preprod: ; ./scripts/provision.sh preprod
provision-prod:    ; ./scripts/provision.sh prod
deploy-preprod:    ; ./scripts/deploy.sh preprod
deploy-prod:       ; ./scripts/deploy.sh prod
status-preprod:    ; ./scripts/status.sh preprod
status-prod:       ; ./scripts/status.sh prod
logs-preprod:      ; ./scripts/status.sh preprod --logs 100
logs-prod:         ; ./scripts/status.sh prod --logs 100
rollback-preprod:  ; ./scripts/rollback.sh preprod
rollback-prod:     ; ./scripts/rollback.sh prod
# Store the ShortLinks API token (prompts, input hidden; docs/deployment.md).
shortlinks-key-preprod: ; ./scripts/set-setting.sh preprod shortlinks_api_key
shortlinks-key-prod:    ; ./scripts/set-setting.sh prod shortlinks_api_key
