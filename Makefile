APP_NAME = godns
APP_VERSION = 0.1.1
REGISTRY = dkr.lonord.name

DOCKER_BUILD = docker buildx build --platform=linux/amd64,linux/arm64,linux/arm/v7

.PHONY: all docker-build clean

all: docker-build

docker-build:
	$(DOCKER_BUILD) -t $(REGISTRY)/$(APP_NAME) -t $(REGISTRY)/$(APP_NAME):$(APP_VERSION) . --push

clean:
	echo "clean"