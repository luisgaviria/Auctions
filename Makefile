AIR := $(HOME)/go/bin/air
FRONTEND_DIR := frontend
BACKEND_DIR  := backend

.PHONY: dev backend frontend test test-backend test-frontend install-air build-backend 

dev:
	make -j2 backend frontend

# Run backend only (hot-reload via air)
backend:
	cd $(BACKEND_DIR) && $(AIR)

# Run frontend only (clears Vite cache first)
frontend:
	cd $(FRONTEND_DIR) && rm -rf node_modules/.vite && npm run dev

test: test-backend test-frontend

test-backend:
	cd $(BACKEND_DIR) && go test ./...

test-frontend:
	cd $(FRONTEND_DIR) && npm test

# Install air if missing
install-air:
	go install github.com/air-verse/air@latest
	@echo "air installed at $(HOME)/go/bin/air"
	@echo "Add $(HOME)/go/bin to your PATH to use 'air' directly:"
	@echo "  echo 'export PATH=\$$PATH:\$$HOME/go/bin' >> ~/.zshrc && source ~/.zshrc"

build-backend:
	cd $(BACKEND_DIR) && go build -o output/backend

build-frontend:
	cd $(FRONTEND_DIR) && npm run build

build: build-backend build-frontend