FROM debian:12-slim

LABEL description="UGREEN NAS Go dev environment"

# Use Aliyun Debian mirror for faster package downloads
RUN sed -i 's|deb.debian.org|mirrors.aliyun.com|g' /etc/apt/sources.list.d/debian.sources && \
    apt-get update && apt-get install -y --no-install-recommends \
    curl wget ca-certificates gnupg \
    git build-essential \
    vim nano tmux \
    jq ripgrep fd-find \
    netcat-openbsd iputils-ping dnsutils \
    procps htop \
    && rm -rf /var/lib/apt/lists/*

# Install Go (match host version if possible, or use latest stable)
ARG GO_VERSION=1.26.3
RUN curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" \
    | tar -C /usr/local -xz

ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOPATH="/go"
ENV PATH="${GOPATH}/bin:${PATH}"

RUN mkdir -p "$GOPATH/src" "$GOPATH/bin" && chmod -R 777 "$GOPATH"

# Install common Go tools
RUN go install golang.org/x/tools/gopls@latest 2>/dev/null; \
    go install github.com/go-delve/delve/cmd/dlv@latest 2>/dev/null; \
    go install honnef.co/go/tools/cmd/staticcheck@latest 2>/dev/null; \
    true

# Set up working dir
WORKDIR /workspace

CMD ["/bin/bash"]
