IMAGE_NAME := gomad
TAG := $(shell \
       LASTTAG=$$(git describe --tags --abbrev=0); \
       COMMITS_SINCE=$$(git rev-list $$LASTTAG..HEAD --count); \
       if [ "$$COMMITS_SINCE" = "0" ]; then \
           echo $$LASTTAG; \
       else \
           echo "$$LASTTAG-dev.$$COMMITS_SINCE-$$(git rev-parse --short HEAD)"; \
       fi)
REGISTRY := registry.werewolves.fyi
ARCHS := amd64 arm64

# Define targets
.PHONY: all build test clean install lint docker-build docker-push help

all: lint test build

build:
	@echo "Building gomad..."
	@go build -o bin/ ./cmd/...

test:
	@echo "Running tests..."
	@go test -v ./...

clean:
	@echo "Cleaning up..."
	@rm -rf bin/
	@rm -f coverage.out

install:
	@echo "Installing gomad..."
	@go install ./cmd/...

docker-build:
	@echo "Building Docker images for platforms: $(ARCHS)"
	@$(foreach arch, $(ARCHS), \
		docker build --platform linux/$(arch) -t $(REGISTRY)/$(IMAGE_NAME):$(TAG)-$(arch) .;)

push-images:
	@echo "Pushing individual architecture images"
	@$(foreach arch, $(ARCHS), \
		docker push $(REGISTRY)/$(IMAGE_NAME):$(TAG)-$(arch);)

manifest:
	@echo "Creating manifest for images $(REGISTRY)/$(IMAGE_NAME):$(TAG)"
	@docker manifest create $(REGISTRY)/$(IMAGE_NAME):$(TAG) \
		$(foreach arch,$(ARCHS),$(REGISTRY)/$(IMAGE_NAME):$(TAG)-$(arch))

push:
	@echo "Pushing Docker image $(REGISTRY)/$(IMAGE_NAME):$(TAG)"
	@docker manifest push $(REGISTRY)/$(IMAGE_NAME):$(TAG)

docker: docker-build push-images manifest push
