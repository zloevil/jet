#!/usr/bin/env bash
#
# Wait until the CI/local integration-test infrastructure is ready to serve.
#
# `docker compose up --wait` only gates on services that declare a healthcheck
# (redis, kafka, clickhouse). This script additionally confirms readiness for
# minio and centrifugo from the host, and is safe to run on its own. It probes
# the published ports/endpoints that the integration tests actually use.
#
# Usage: bash ci/wait-for-infra.sh [timeout_seconds]   (default 180)

set -euo pipefail

TIMEOUT="${1:-180}"
deadline=$(( $(date +%s) + TIMEOUT ))

remaining() { echo $(( deadline - $(date +%s) )); }

wait_tcp() { # name host port
  local name="$1" host="$2" port="$3"
  echo "waiting for ${name} (tcp ${host}:${port})..."
  until (exec 3<>"/dev/tcp/${host}/${port}") 2>/dev/null; do
    if [ "$(remaining)" -le 0 ]; then
      echo "ERROR: timed out waiting for ${name} (tcp ${host}:${port})" >&2
      return 1
    fi
    sleep 2
  done
  exec 3>&- 2>/dev/null || true
  echo "OK: ${name} is up"
}

wait_http() { # name url
  local name="$1" url="$2"
  echo "waiting for ${name} (http ${url})..."
  until curl -fsS -o /dev/null "$url"; do
    if [ "$(remaining)" -le 0 ]; then
      echo "ERROR: timed out waiting for ${name} (http ${url})" >&2
      return 1
    fi
    sleep 2
  done
  echo "OK: ${name} is up"
}

wait_tcp  redis           127.0.0.1 6379
wait_tcp  kafka           127.0.0.1 9092
wait_tcp  clickhouse      127.0.0.1 19000
wait_http minio           "http://127.0.0.1:29000/minio/health/live"
wait_tcp  centrifugo-ws   127.0.0.1 18000
wait_tcp  centrifugo-grpc 127.0.0.1 20000

echo "all CI infrastructure is ready"
