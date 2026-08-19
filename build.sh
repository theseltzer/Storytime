#!/usr/bin/env bash
# Build the game to WebAssembly and copy in the matching Go JS shim.
set -euo pipefail
cd "$(dirname "$0")"

GOOS=js GOARCH=wasm go build -o web/game.wasm ./cmd/game
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" web/wasm_exec.js

echo "built web/game.wasm ($(du -h web/game.wasm | cut -f1))"
