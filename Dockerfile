FROM debian:bookworm-slim

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
 && rm -rf /var/lib/apt/lists/* \
 && fc-cache -f

RUN arch="$(dpkg --print-architecture)" \
 && curl -fsSL "https://go.dev/dl/go1.26.0.linux-${arch}.tar.gz" \
  -o /tmp/go.tar.gz \
 && tar -xzf /tmp/go.tar.gz -C /usr/local \
 && rm /tmp/go.tar.gz
ENV PATH="/usr/local/go/bin:/root/go/bin:${PATH}"

RUN curl -fsSL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh \
 | sh -s -- -b /usr/local/bin

WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
a-certificates \
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
 && rm -rf /var/lib/apt/lists/* \
 && fc-cache -f
