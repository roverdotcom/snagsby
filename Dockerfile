ARG GOLANG_VERSION
FROM golang:${GOLANG_VERSION:-1.15.6} as build
ENV CGO_ENABLED=0
WORKDIR /go/src/github.com/roverdotcom/snagsby
COPY . /go/src/github.com/roverdotcom/snagsby
RUN make build
RUN /go/src/github.com/roverdotcom/snagsby/snagsby -v


FROM alpine:3
WORKDIR /app/
RUN apk add --no-cache \
    ca-certificates \
    bash
COPY --from=build /go/src/github.com/roverdotcom/snagsby/snagsby /app/snagsby
ENTRYPOINT [ "/app/snagsby" ]
