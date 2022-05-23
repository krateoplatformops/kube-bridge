# Build environment
# -----------------
FROM golang:1.18.0-bullseye as builder
LABEL stage=builder

ARG DEBIAN_FRONTEND=noninteractive

SHELL ["/bin/bash", "-o", "pipefail", "-c"]
# hadolint ignore=DL3008
RUN apt-get update && apt-get install -y ca-certificates openssl git tzdata && apt-get install -y --no-install-recommends && \
  update-ca-certificates && \
  rm -rf /var/lib/apt/lists/*

WORKDIR /src

COPY . .

# Build
RUN make print.vars && make deps && make build

# Deployment environment
# ----------------------
FROM scratch

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=builder /src/bin/service /service

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

ENTRYPOINT ["/service"]