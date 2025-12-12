# Build sekaid
FROM ubuntu:20.04 AS sekai-builder

# Avoid prompts from apt
ARG DEBIAN_FRONTEND=noninteractive

# Update packages and Install essentials in one layer
RUN apt-get update -y && \
    apt-get install -y git build-essential jq wget

# Install Golang
RUN wget https://go.dev/dl/go1.22.0.linux-amd64.tar.gz -O /tmp/go.tar.gz && \
    tar -C /usr/local -xzf /tmp/go.tar.gz && \
    rm /tmp/go.tar.gz

# Set PATH for Golang
ENV PATH="$PATH:/usr/local/go/bin"

# Cloning sekai repo and install
RUN git clone -c http.postBuffer=1048576000 --depth 1 https://github.com/kiracore/sekai.git /bsekai && \
    cd /bsekai && \
    make build-static

# Build scaller (CLI tool)
FROM golang:1.22 AS scaller-builder

WORKDIR /app

COPY ./src/sCaller /app

RUN go mod tidy && CGO_ENABLED=0 go build -a -tags netgo -installsuffix cgo -o /scaller ./cmd/main.go

# Final image
FROM alpine:latest

# Copy artifacts
COPY --from=sekai-builder /sekaid /sekaid
COPY --from=scaller-builder /scaller /scaller

# Entrypoint: wait for docker exec commands
ENTRYPOINT ["/scaller", "wait"]

# Expose Tendermint ports (no HTTP port - CLI only)
EXPOSE 26657
EXPOSE 26656
EXPOSE 9090
EXPOSE 1317
