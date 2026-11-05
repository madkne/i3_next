#!/usr/bin/env bash
# Tiny test harness for the i3-next end-to-end suite (sourced by run.sh).

PASS=0; FAIL=0; SKIP=0

if [ -t 1 ]; then
  C_OK=$'\033[32m'; C_BAD=$'\033[31m'; C_DIM=$'\033[2m'; C_OFF=$'\033[0m'
else
  C_OK=; C_BAD=; C_DIM=; C_OFF=
fi

# retry <attempts> <delay-seconds> <cmd...>
retry() {
  local attempts="$1" delay="$2" i
  shift 2
  for ((i = 0; i < attempts; i++)); do
    if "$@"; then return 0; fi
    sleep "$delay"
  done
  return 1
}

# run_test <name> <function> [args...]
# The function runs in a subshell with "set -Eeuo pipefail"; its output goes to
# $ARTIFACT_DIR/test-<name>.log and is echoed only when the test fails.
run_test() {
  local name="$1"; shift
  local log="$ARTIFACT_DIR/test-${name//[^A-Za-z0-9_.-]/_}.log"
  if ( set -Eeuo pipefail; "$@" ) >"$log" 2>&1; then
    PASS=$((PASS + 1))
    printf '  %sok%s    %s\n' "$C_OK" "$C_OFF" "$name"
  else
    FAIL=$((FAIL + 1))
    printf '  %sFAIL%s  %s  %s(%s)%s\n' "$C_BAD" "$C_OFF" "$name" "$C_DIM" "$log" "$C_OFF"
    tail -n 15 "$log" | sed 's/^/        | /'
  fi
}

skip_test() { # skip_test <name> <reason>
  SKIP=$((SKIP + 1))
  printf '  %sskip%s  %s  %s(%s)%s\n' "$C_DIM" "$C_OFF" "$1" "$C_DIM" "$2" "$C_OFF"
}

summary() {
  local line
  line="$(printf '%d tests: %d passed, %d failed, %d skipped' \
    $((PASS + FAIL + SKIP)) "$PASS" "$FAIL" "$SKIP")"
  printf '\n%s\n' "$line"
  printf '%s\n' "$line" >"$ARTIFACT_DIR/summary.txt"
  [ "$FAIL" -eq 0 ]
}
