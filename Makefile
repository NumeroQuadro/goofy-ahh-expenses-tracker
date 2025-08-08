APP_NAME=goofy-ahh-expenses-tracker
DOCKER_IMAGE_NAME=goofy-ahh-expenses-tracker

all: lint build

docker-build:
	@echo "Building from Dockerfile..."
	@docker build -t $(DOCKER_IMAGE_NAME) .

docker-run: docker-build
	@if [ -f .env ]; then \
		DATA_DIR=/var/lib/goofy-expenses-data; \
		sudo mkdir -p $$DATA_DIR; \
		sudo chown $$USER:$$USER $$DATA_DIR; \
		docker run -d --rm --name $(APP_NAME) \
		--env-file .env \
		-p 8088:8088 \
		-v $$DATA_DIR:/app/data \
		$(DOCKER_IMAGE_NAME); \
	else \
		@echo "Error: .env file not found. Please create one from env.example."; \
		exit 1; \
	fi

# Optional: run with TLS handled inside the container (no Nginx in front)
docker-run-https: docker-build
	@if [ -f .env ]; then \
		docker run -d --rm --name $(APP_NAME) \
		--env-file .env \
		-e WEB_ADDRESS=0.0.0.0:443 \
		-p 9443:443 \
		-v /etc/letsencrypt/live/tralalero-tralala.ru/fullchain.pem:/app/certs/fullchain.pem:ro \
		-v /etc/letsencrypt/live/tralalero-tralala.ru/privkey.pem:/app/certs/privkey.pem:ro \
		$(DOCKER_IMAGE_NAME); \
	else \
		@echo "Error: .env file not found. Please create one from env.example."; \
		exit 1; \
	fi

docker-logs:
	@echo "Getting Docker container logs..."
	@docker logs -f $(APP_NAME)

docker-stop:
	@echo "Stopping Docker container..."
	@docker stop $(APP_NAME)

docker-rm:
	@echo "Deleting Docker container..."
	@docker rm $(APP_NAME)

deps:
	go mod tidy
	go mod download

clean:
	rm -f $(BINARY_NAME)

lint:
	@echo "Running linter..."
	@golangci-lint run

build:
	@echo "Building application..."
	@go build -o bin/$(APP_NAME) ./cmd/main.go
