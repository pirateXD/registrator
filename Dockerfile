FROM golang:1.10-alpine3.7 as builder
MAINTAINER YangJunhai <yangjunhai@xindong.com>
COPY . /go/src/github.com/pirateXD/registrator
RUN apk --no-cache add ca-certificates \
    && go build -ldflags "-X main.Version=$(grep "^version" /go/src/github.com/pirateXD/registrator/docker_run.config | awk -F'=' '{print $2}')" \
    -o /bin/registrator github.com/pirateXD/registrator && \
    rm -rf /go
FROM alpine:latest
COPY --from=builder /bin/registrator /bin/registrator
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
ENTRYPOINT ["/bin/registrator"]
CMD ["--help"]