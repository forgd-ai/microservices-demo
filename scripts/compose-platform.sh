#!/bin/sh
# Emit BUILD_PLATFORM and TARGETARCH for docker-compose (.env or shell).
# Usage: eval "$(./scripts/compose-platform.sh)"   then: docker-compose build
case "$(uname -m)" in
  arm64|aarch64) arch=arm64 ;;
  x86_64|amd64) arch=amd64 ;;
  *)
    echo "compose-platform: unsupported uname -m: $(uname -m)" >&2
    exit 1
    ;;
esac
printf 'BUILD_PLATFORM=linux/%s\n' "$arch"
printf 'TARGETARCH=%s\n' "$arch"
