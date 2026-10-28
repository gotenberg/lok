# veraPDF (PDF/A and PDF/UA validation) for integration tests, sourced from
# gotenberg/integration-tools. Declared as a global ARG so it can name a stage.
ARG INTEGRATION_TOOLS=gotenberg/integration-tools:latest

FROM ${INTEGRATION_TOOLS} AS tools

FROM debian:trixie-slim

ARG GO_VERSION=1.27.3
ARG GOLANGCI_LINT_VERSION=v2.15.0

COPY --from=tools /lib/verapdf /lib/verapdf

# LibreOffice packages, hyphenation dictionaries and the trixie-backports
# source match gotenberg's build/Dockerfile, so lok converts with the same
# LibreOffice as Gotenberg. python3-uno is left out: only unoconverter uses it.
RUN echo "deb http://deb.debian.org/debian trixie-backports main" >> /etc/apt/sources.list \
 && apt-get update \
 && apt-get install -y --no-install-recommends \
  ca-certificates \
  curl \
  gcc \
  libc6-dev \
  fontconfig \
  fonts-liberation \
  poppler-utils \
  default-jre-headless \
  hyphen-af hyphen-as hyphen-be hyphen-bg hyphen-bn hyphen-ca hyphen-cs hyphen-da hyphen-de hyphen-el \
  hyphen-en-gb hyphen-en-us hyphen-eo hyphen-es hyphen-fr hyphen-gl hyphen-gu hyphen-hi hyphen-hr hyphen-hu \
  hyphen-id hyphen-is hyphen-it hyphen-kn hyphen-lt hyphen-lv hyphen-ml hyphen-mn hyphen-mr hyphen-nl \
  hyphen-no hyphen-or hyphen-pa hyphen-pl hyphen-pt-br hyphen-pt-pt hyphen-ro hyphen-ru hyphen-sk hyphen-sl \
  hyphen-sr hyphen-sv hyphen-ta hyphen-te hyphen-th hyphen-uk hyphen-zu \
 && apt-get install -y --no-install-recommends -t trixie-backports \
  libreoffice-writer \
  libreoffice-calc \
  libreoffice-impress \
  libreoffice-draw \
  libreoffice-math \
  libreofficekit-dev \
 && rm -rf /var/lib/apt/lists/* \
 && fc-cache -f \
 && ln -s /lib/verapdf/verapdf /usr/bin/verapdf

RUN arch="$(dpkg --print-architecture)" \
 && curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${arch}.tar.gz" \
  -o /tmp/go.tar.gz \
 && tar -xzf /tmp/go.tar.gz -C /usr/local \
 && rm /tmp/go.tar.gz
ENV PATH="/usr/local/go/bin:/root/go/bin:${PATH}"

RUN GOBIN=/usr/local/bin go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@${GOLANGCI_LINT_VERSION}

# trixie-backports ships a LibreOffice newer than 7.6, which provides the
# trimMemory API. Enable it in the C bridge.
ENV CGO_CFLAGS="-DLOK_HAS_TRIM_MEMORY"

WORKDIR /src
COPY . .
