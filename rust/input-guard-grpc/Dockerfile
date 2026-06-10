FROM rust:1-slim-bookworm AS builder
WORKDIR /workspace

RUN apt-get update \
    && apt-get install -y --no-install-recommends protobuf-compiler pkg-config ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY proto ./proto
COPY rust/input-guard ./rust/input-guard
COPY rust/input-guard-grpc ./rust/input-guard-grpc

WORKDIR /workspace/rust/input-guard-grpc
RUN cargo build --release

FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /workspace/rust/input-guard-grpc/target/release/input-guard-grpc /usr/local/bin/input-guard-grpc

EXPOSE 50051
ENV INPUT_GUARD_GRPC_ADDR=0.0.0.0:50051
ENV RUST_LOG=info,input_guard_grpc=debug

ENTRYPOINT ["/usr/local/bin/input-guard-grpc"]
