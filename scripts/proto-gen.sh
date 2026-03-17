#!/bin/bash
# Generate Go code from protobuf definitions
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PROTO_DIR="$REPO_ROOT/api/proto"
GEN_DIR="$REPO_ROOT/api/gen/go"

echo "▸ Generating Go protobuf code..."

# Ensure output directory exists
mkdir -p "$GEN_DIR"

# Find all .proto files and generate
find "$PROTO_DIR" -name "*.proto" | while read -r proto_file; do
    echo "  → $proto_file"
    protoc \
        --proto_path="$PROTO_DIR" \
        --go_out="$GEN_DIR" \
        --go_opt=paths=source_relative \
        --go-grpc_out="$GEN_DIR" \
        --go-grpc_opt=paths=source_relative \
        "$proto_file"
done

echo "✅ Protobuf code generation complete."
