AIR := $(HOME)/go/bin/air
FRONTEND_DIR := frontend
BACKEND_DIR  := backend

.PHONY: dev backend frontend install-air

# Start both backend (hot-reload) and frontend in parallel.
# A single Ctrl+C sends SIGINT to the process group, stopping both.
dev:
	@trap 'kill 0' INT; \
	(cd $(BACKEND_DIR) && $(AIR)) & \
	(sleep 3 && cd $(FRONTEND_DIR) && rm -rf node_modules/.vite && npm run dev) & \
	wait

# Run backend only (hot-reload via air)
backend:
	cd $(BACKEND_DIR) && $(AIR)

# Run frontend only (clears Vite cache first)
frontend:
	cd $(FRONTEND_DIR) && rm -rf node_modules/.vite && npm run dev

# Install air if missing
install-air:
	go install github.com/air-verse/air@latest
	@echo "air installed at $(HOME)/go/bin/air"
	@echo "Add $(HOME)/go/bin to your PATH to use 'air' directly:"
	@echo "  echo 'export PATH=\$$PATH:\$$HOME/go/bin' >> ~/.zshrc && source ~/.zshrc"
