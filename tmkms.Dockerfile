# Build TMKMS from official iqlusioninc/tmkms repository
FROM rust:1.85-bookworm AS builder

# Install dependencies for softsign
RUN apt-get update && apt-get install -y \
    libusb-1.0-0-dev \
    pkg-config \
    && rm -rf /var/lib/apt/lists/*

# Install tmkms with softsign feature
RUN cargo install tmkms --features=softsign --locked

# Runtime image
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y \
    libusb-1.0-0 \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /usr/local/cargo/bin/tmkms /usr/local/bin/tmkms

WORKDIR /tmkms

# Create directories for config, secrets, and state
RUN mkdir -p /tmkms/secrets /tmkms/state

ENTRYPOINT ["tmkms"]
CMD ["start", "-c", "/tmkms/tmkms.toml"]
