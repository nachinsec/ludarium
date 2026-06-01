.PHONY: dev dev-api dev-web build build-web run tidy clean

# Run backend (Go) and frontend (Vite) together for local development.
dev:
	@echo "Run 'make dev-api' and 'make dev-web' in two terminals."

dev-api:
	ENV=development go run ./cmd/server

dev-web:
	pnpm --dir web dev

# Production build: compile the frontend, then embed it into the Go binary.
build: build-web
	go build -o bin/ludarium ./cmd/server

build-web:
	pnpm --dir web install
	pnpm --dir web build

run: build
	./bin/ludarium

tidy:
	go mod tidy

clean:
	rm -rf bin data internal/web/dist/assets
