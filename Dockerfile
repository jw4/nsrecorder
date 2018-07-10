#
# Build on Debian Stretch
#

FROM golang:stretch as builder

COPY . /go/src/jw4.us/nsrecorder

WORKDIR /go/src/jw4.us/nsrecorder

RUN go get -v -u ./... && go build -o nsr ./cmd/nsr


#
# Create Image on Stretch Slim
#

FROM debian:stretch-slim

COPY --from=builder /go/src/jw4.us/nsrecorder/nsr /nsr

ENTRYPOINT ["/nsr"]
