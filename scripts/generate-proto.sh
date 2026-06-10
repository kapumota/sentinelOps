#!/usr/bin/env bash
set -euo pipefail

if ! command -v protoc >/dev/null 2>&1; then
  echo "protoc no está instalado. Instala protobuf-compiler antes de generar código." >&2
  exit 1
fi

if ! command -v protoc-gen-go >/dev/null 2>&1; then
  echo "protoc-gen-go no está instalado. Ejecuta make proto-tools." >&2
  exit 1
fi

if ! command -v protoc-gen-go-grpc >/dev/null 2>&1; then
  echo "protoc-gen-go-grpc no está instalado. Ejecuta make proto-tools." >&2
  exit 1
fi

mkdir -p gen/go
protoc \
  --proto_path=proto \
  --go_out=gen/go --go_opt=paths=source_relative \
  --go-grpc_out=gen/go --go-grpc_opt=paths=source_relative \
  proto/validator/v1/validator.proto

echo "Código Go generado en gen/go"
