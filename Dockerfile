# veraPDF (PDF/A and PDF/UA validation) for integration tests, sourced from
# gotenberg/integration-tools. Declared as a global ARG so it can name a stage.
ARG INTEGRATION_TOOLS=gotenberg/integration-tools:latest

FROM ${INTEGRATION_TOOLS} AS tools

FROM debian:trixie-slim

ARG GO_VERSION=1.26.0
ARG GOLANGCI_LINT_VERSION=v2.10.1

COPY --from=tools /lib/verapdf /lib/verapdf

RUN apt-get update \
 && apt-get install -y --no-install-recommends \
  ca-certificates \
  curl \
  gcc \
  libc6-dev \
  libreoffice-core \
  libreoffice-writer \
  libreoffice-calc \
  libreoffice-impress \
  libreofficekit-dev \
  fontconfig \
  fonts-liberation \
  poppler-utils \
  default-jre-headless \
 && rm -rf /var/lib/apt/lists/* \
 && fc-cache -f \
 && ln -s /lib/verapdf/verapdf /usr/bin/verapdf

RUN arch="$(dpkg --print-architecture)" \
 && curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${arch}.tar.gz" \
  -o /tmp/go.tar.gz \
 && tar -xzf /tmp/go.tar.gz -C /usr/local \
 && rm /tmp/go.tar.gz
ENV PATH="/usr/local/go/bin:/root/go/bin:${PATH}"

RUN curl -fsSL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh \
 | sh -s -- -b /usr/local/bin "${GOLANGCI_LINT_VERSION}"

# Debian 13 (trixie) ships LibreOffice 25.2, which provides the trimMemory API
# (LibreOffice 7.6+). Enable it in the C bridge.
ENV CGO_CFLAGS="-DLOK_HAS_TRIM_MEMORY"

WORKDIR /src
COPY . .
