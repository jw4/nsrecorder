NAME=nsrecorder
IMAGE=docker.jw4.us/$(NAME)

ifeq ($(BUILD_VERSION),)
	BUILD_VERSION=$(shell git describe --dirty --first-parent --always --tags)
endif

all: image

clean:
	-rm ./nsr
	go clean ./...

image:
	docker build --build-arg BUILD_VERSION=$(BUILD_VERSION) -t $(IMAGE):latest -t $(IMAGE):$(BUILD_VERSION) .

push: clean image
	docker push $(IMAGE):$(BUILD_VERSION)
	docker push $(IMAGE):latest

