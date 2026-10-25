FROM golang:1.22-bookworm AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /cht-app-github

COPY . .

RUN GOOS=$TARGETOS GOARCH=$TARGETARCH make build

FROM public.ecr.aws/i8a4b9p4/circleci-base/debian:bookworm-slim-curl AS runtime

ARG BUILD_VERSION
ARG BUILD_COMMIT
ARG BUILD_TIME
ENV BUILD_VERSION=${BUILD_VERSION}
ENV BUILD_COMMIT=${BUILD_COMMIT}
ENV BUILD_TIME=${BUILD_TIME}

WORKDIR /cht-app-github

# Upgrade packages with published Debian security fixes, including the CA bundle.
RUN apt-get update && apt-get install -qy --only-upgrade \
    ca-certificates \
    curl \
    dpkg \
    gpgv \
    krb5-locales \
    libbrotli1 \
    libc-bin \
    libc6 \
    libcap2 \
    libcurl4 \
    libgcrypt20 \
    libgnutls30 \
    libgssapi-krb5-2 \
    libk5crypto3 \
    libkrb5-3 \
    libkrb5support0 \
    liblzma5 \
    libnghttp2-14 \
    libpam-modules \
    libpam-modules-bin \
    libpam-runtime \
    libpam0g \
    libpcre2-8-0 \
    libssh2-1 \
    libssl3 \
    libsystemd0 \
    libudev1 \
    openssl \
    perl-base \
    sed \
    tzdata

COPY --from=builder /cht-app-github/target/app ./app
COPY --from=builder /cht-app-github/config/*.yml ./config/

EXPOSE 4000

CMD ["./app"]
