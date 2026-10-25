FROM golang:1.22-bookworm AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /cht-app-github

COPY . .

RUN GOOS=$TARGETOS GOARCH=$TARGETARCH make build

FROM 360732775913.dkr.ecr.ap-northeast-2.amazonaws.com/base-image/debian:bookworm-slim-curl AS runtime

ARG BUILD_VERSION
ARG BUILD_COMMIT
ARG BUILD_TIME
ENV BUILD_VERSION=${BUILD_VERSION}
ENV BUILD_COMMIT=${BUILD_COMMIT}
ENV BUILD_TIME=${BUILD_TIME}

WORKDIR /cht-app-github

RUN apt-get update && apt-get install -qy \
    ca-certificates libbrotli1 tzdata

COPY --from=builder /cht-app-github/target/app ./app
COPY --from=builder /cht-app-github/config/*.yml ./config/

EXPOSE 4000

CMD ["./app"]
