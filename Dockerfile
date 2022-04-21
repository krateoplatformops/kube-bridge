# Build environment
# -----------------
FROM golang:1.18.0-bullseye as builder
LABEL stage=builder

ARG upx_version=3.96
ARG DEBIAN_FRONTEND=noninteractive

ARG LD_FLAGS

SHELL ["/bin/bash", "-o", "pipefail", "-c"]
# hadolint ignore=DL3008
RUN apt-get update && apt-get install -y ca-certificates openssl git tzdata && apt-get install -y --no-install-recommends xz-utils && \
  update-ca-certificates && \
  curl -Ls https://github.com/upx/upx/releases/download/v${upx_version}/upx-${upx_version}-amd64_linux.tar.xz -o - | tar xvJf - -C /tmp && \
  cp /tmp/upx-${upx_version}-amd64_linux/upx /usr/local/bin/ && \
  chmod +x /usr/local/bin/upx && \
  apt-get remove -y xz-utils && \
  rm -rf /var/lib/apt/lists/*

WORKDIR /src

COPY . .

# Build
RUN make deps && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on \
    go build -tags netgo -a -v -ldflags "$LD_FLAGS" -o /bin/service ./main.go && \
    strip /bin/service && \
    /usr/local/bin/upx -9 /bin/service

# Deployment environment
# ----------------------
FROM scratch

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=builder /bin/service /bin/service

# Metadata params
ARG VERSION
ARG BUILD_DATE
ARG REPO_URL
ARG LAST_COMMIT
ARG PROJECT_NAME
ARG VENDOR

# Metadata
LABEL org.label-schema.build-date=$BUILD_DATE \
      org.label-schema.name=$PROJECT_NAME \
      org.label-schema.vcs-url=$REPO_URL \
      org.label-schema.vcs-ref=$LAST_COMMIT \
      org.label-schema.vendor=$VENDOR \
      org.label-schema.version=$VERSION \
      org.label-schema.docker.schema-version="1.0"

ARG KUBE_BRIDGE_PORT

EXPOSE ${KUBE_BRIDGE_PORT}

ENTRYPOINT ["/bin/service"]