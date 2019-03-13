#
#           Build on Debian Stretch
#

FROM        golang:stretch as builder

COPY        . /go/src/jw4.us/nsrecorder

WORKDIR     /go/src/jw4.us/nsrecorder

ARG         BUILD_VERSION=v0.0.0

ENV         BUILD_VERSION ${BUILD_VERSION}

RUN         go build -tags netgo -ldflags="-s -w -X jw4.us/nsrecorder.Version=${BUILD_VERSION}" -o nsr ./cmd/nsr/


#
#           Create Image on Stretch Slim
#

FROM        debian:stretch-slim

COPY        --from=builder /go/src/jw4.us/nsrecorder/nsr /nsr

ENV         TOPIC=dns \
            CHANNEL=recorder \
            LOOKUPD=nsq:4161 \
            DB_FILE=nsr.db \
            VERBOSE=false

ENTRYPOINT  ["/nsr"]
