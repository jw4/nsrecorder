FROM alpine:latest

RUN apk --no-cache add bind-tools ca-certificates openssl && update-ca-certificates

ADD nsrecorder /nsrecorder

ENTRYPOINT ["/nsrecorder"]
