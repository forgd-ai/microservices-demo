#!/usr/bin/env bash
set -Eeuo pipefail

# Builds the 10 Online Boutique service images locally and tags them with
# the tag referenced in docker-compose.yml so `docker compose up -d` finds
# them. Run once during workshop setup; rerun when source changes.
#
# Override the tag with TAG=foo ./scripts/build-images.sh

readonly TAG="${TAG:-5096a85}"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly REPO_ROOT

# Detect host arch for the cartservice (.NET) build. The cartservice
# Dockerfile takes TARGETARCH as a build arg with default amd64; on Apple
# Silicon we must pass arm64 explicitly per docs/run-options.md.
case "$(uname -m)" in
  arm64|aarch64) HOST_ARCH="arm64" ;;
  x86_64|amd64)  HOST_ARCH="amd64" ;;
  *) echo "Unsupported host arch: $(uname -m)" >&2; exit 1 ;;
esac
readonly HOST_ARCH

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "Required command not found: $1" >&2
    exit 1
  }
}
require_cmd docker

# service_name : context_subdir
readonly SERVICES=(
  "adservice:src/adservice"
  "checkoutservice:src/checkoutservice"
  "currencyservice:src/currencyservice"
  "emailservice:src/emailservice"
  "frontend:src/frontend"
  "paymentservice:src/paymentservice"
  "productcatalogservice:src/productcatalogservice"
  "recommendationservice:src/recommendationservice"
  "shippingservice:src/shippingservice"
)

build() {
  local name="$1" context="$2"
  shift 2
  echo "==> Building ${name}:${TAG}"
  docker build -t "${name}:${TAG}" "$@" "${REPO_ROOT}/${context}"
}

for entry in "${SERVICES[@]}"; do
  build "${entry%%:*}" "${entry#*:}"
done

# cartservice is .NET and needs --platform + TARGETARCH on Apple Silicon
build cartservice src/cartservice/src \
  --platform "linux/${HOST_ARCH}" \
  --build-arg "TARGETARCH=${HOST_ARCH}"

echo
echo "All 10 service images built and tagged :${TAG}"
echo "Now run: docker compose up -d"
