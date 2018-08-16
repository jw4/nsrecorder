NAME=nsrecorder
IMAGE=docker.jw4.us/$(NAME)

ifeq ($(BUILD_VERSION),)
	BUILD_VERSION=$(shell git describe --dirty --first-parent --always --tags)
endif

.PHONY: all
all: image

.PHONY: clean
clean:
	-rm ./nsr ./nsr.db*
	go clean ./...

.PHONY: local
local:
	go build -tags netgo -ldflags="-s -w -X jw4.us/nsrecorder.Version=${BUILD_VERSION}" -o nsr ./cmd/nsr/


.PHONY: image
image:
	docker build --build-arg BUILD_VERSION=$(BUILD_VERSION) -t $(IMAGE):latest -t $(IMAGE):$(BUILD_VERSION) .

.PHONY: push
push: clean image
	docker push $(IMAGE):$(BUILD_VERSION)
	docker push $(IMAGE):latest

