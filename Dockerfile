ARG GOLANG_VERSION
FROM 290268990387.dkr.ecr.us-west-2.amazonaws.com/ecr-public/docker/library/golang:${GOLANG_VERSION:-1.15.6} as build
ENV CGO_ENABLED=0
WORKDIR /go/src/github.com/roverdotcom/snagsby
COPY . /go/src/github.com/roverdotcom/snagsby
RUN make build
RUN /go/src/github.com/roverdotcom/snagsby/snagsby -v


# Image with more tools installed and no entrypoint
FROM 290268990387.dkr.ecr.us-west-2.amazonaws.com/ecr-public/docker/library/alpine:3 as dev
WORKDIR /app/
RUN apk add --no-cache \
    ca-certificates \
    bash \
    python3
COPY --from=build /go/src/github.com/roverdotcom/snagsby/snagsby /app/snagsby


FROM 290268990387.dkr.ecr.us-west-2.amazonaws.com/ecr-public/docker/library/alpine:3
WORKDIR /app/
RUN apk add --no-cache \
    ca-certificates
COPY --from=build /go/src/github.com/roverdotcom/snagsby/snagsby /app/snagsby
ENTRYPOINT [ "/app/snagsby" ]
