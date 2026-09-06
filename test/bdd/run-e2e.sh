#!/usr/bin/env bash
# Sobe os containers, roda os testes e2e (godog) e aguarda ENTER antes de encerrar.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
SRC_DIR="$ROOT_DIR/src"

cd "$ROOT_DIR"

cleanup() {
	echo "==> Encerrando containers (docker compose down)..."
	(cd "$SRC_DIR" && docker compose down)
}
trap cleanup EXIT

echo "==> Subindo containers (docker compose up --build -d)..."
(cd "$SRC_DIR" && docker compose up --build -d)

echo "==> Executando testes e2e (E2E=1 go test ./test/bdd/...)..."
set +e
E2E=1 go test ./test/bdd/... -v
test_exit_code=$?
set -e

echo
read -r -p "Pressione ENTER para encerrar os containers..."

exit $test_exit_code
