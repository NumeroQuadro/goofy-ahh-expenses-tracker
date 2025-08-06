APP_NAME=goofy-ahh-expenses-tracker
DOCKER_IMAGE_NAME=goofy-ahh-expenses-tracker

all: lint build

docker-build:
	@echo "Building from Dockerfile..."
	@docker build -t $(DOCKER_IMAGE_NAME) .

docker-run: docker-build
	@if [ -f .env ]; then \
		docker run -d --rm --name $(APP_NAME) \
		--env-file .env \
		-p 8080:8080 \
		-p 8443:443 \
		-v /etc/letsencrypt/live/your-domain.com/fullchain.pem:/app/certs/fullchain.pem:ro \
		-v /etc/letsencrypt/live/your-domain.com/privkey.pem:/app/certs/privkey.pem:ro \
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
