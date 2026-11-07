#!/usr/bin/env bash
# End-to-end suite for i3-next.
#
# Runs inside the KVM guest (via "tests/vm/vm.sh test") but works on any Linux
# machine that has sway, grim, jq and foot. Sway is started with the wlroots
# *headless* backend, so no display or GPU is needed; OUTPUTS virtual monitors
# are created to exercise multi-monitor setups.
set -u

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

export ARTIFACT_DIR="${ARTIFACT_DIR:-/tmp/i3-next-artifacts}"
mkdir -p "$ARTIFACT_DIR"
# shellcheck source=tests/e2e/lib.sh
. "$ROOT/tests/e2e/lib.sh"

BIN="$ROOT/build"
OUTPUTS="${OUTPUTS:-2}"

export XDG_RUNTIME_DIR="${XDG_RUNTIME_DIR:-/run/user/$(id -u)}"
if [ ! -d "$XDG_RUNTIME_DIR" ]; then
  XDG_RUNTIME_DIR="$(mktemp -d)"; chmod 700 "$XDG_RUNTIME_DIR"; export XDG_RUNTIME_DIR
fi

SWAY_CONFIG="$ROOT/configs/sway/config"
if [ ! -s "$SWAY_CONFIG" ]; then SWAY_CONFIG=/etc/sway/config; fi
SWAY_PID=""

# ---- sway lifecycle ---------------------------------------------------------
sway_start() {
  unset WAYLAND_DISPLAY SWAYSOCK I3SOCK DISPLAY
  export WLR_BACKENDS=headless WLR_LIBINPUT_NO_DEVICES=1 WLR_RENDERER=pixman
  export WLR_HEADLESS_OUTPUTS="$OUTPUTS"

  sway -c "$SWAY_CONFIG" >"$ARTIFACT_DIR/sway.log" 2>&1 &
  SWAY_PID=$!
  export SWAYSOCK="$XDG_RUNTIME_DIR/sway-ipc.$(id -u).$SWAY_PID.sock"

  retry 50 0.2 swaymsg -t get_version >/dev/null 2>&1 || return 1

  # Ask sway which Wayland socket it picked: children inherit WAYLAND_DISPLAY.
  local f="$ARTIFACT_DIR/.wayland-display"
  rm -f "$f"
  swaymsg exec "printenv WAYLAND_DISPLAY > $f" >/dev/null
  retry 50 0.2 test -s "$f" || return 1
  WAYLAND_DISPLAY="$(<"$f")"; export WAYLAND_DISPLAY
}

sway_stop() {
  if [ -n "$SWAY_PID" ] && kill -0 "$SWAY_PID" 2>/dev/null; then
    swaymsg exit >/dev/null 2>&1 || kill "$SWAY_PID" 2>/dev/null
    wait "$SWAY_PID" 2>/dev/null
  fi
}
trap sway_stop EXIT

# ---- helpers ----------------------------------------------------------------
output_count()      { swaymsg -t get_outputs -r | jq length; }
outputs_count_is()  { [ "$(output_count)" -eq "$1" ]; }
foot_windows()      { swaymsg -t get_tree -r | jq '[recurse(.nodes[]?, .floating_nodes[]?) | select(.app_id == "foot")] | length'; }
foot_is_mapped()    { [ "$(foot_windows)" -ge 1 ]; }

# ---- tests: build & CLI -----------------------------------------------------
t_binaries_exist() {
  local b
  for b in i3-nextd i3-nextctl i3-next-session i3-next-doctor; do
    test -x "$BIN/$b" || { echo "missing $BIN/$b"; return 1; }
  done
}

t_version_flags() {
  "$BIN/i3-nextd" --version        | grep -q '^i3_next 0\.1'
  "$BIN/i3-nextctl" --version      | grep -q '^i3-nextctl 0\.1'
  "$BIN/i3-next-session" --version | grep -q '^i3-next-session 0\.1'
  "$BIN/i3-next-doctor" --version  | grep -q '^i3-next-doctor 0\.1'
}

t_ctl_status() { "$BIN/i3-nextctl" status | grep -q 'i3_next: not connected'; }

t_ctl_unknown_command() {
  local rc=0
  "$BIN/i3-nextctl" definitely-not-a-command || rc=$?
  [ "$rc" -eq 1 ]
}

t_doctor_sees_sway() { "$BIN/i3-next-doctor" | grep -q '\[OK\] sway installed'; }
t_doctor_json() {
  "$BIN/i3-next-doctor" --json | python3 -c '
import sys, json
d = json.load(sys.stdin)
assert isinstance(d, list) and len(d) > 0
ids = {c["id"] for c in d}
assert "sway" in ids
for c in d:
    assert {"id", "status", "detail"} <= set(c.keys())
    assert c["status"] in ("ok", "warn", "fail")
'
}
t_doctor_exit_ok() { "$BIN/i3-next-doctor" >/dev/null 2>&1; }

# ---- tests: Go quality gates ------------------------------------------------
t_gofmt()  { local out; out="$(gofmt -l .)"; [ -z "$out" ] || { echo "needs gofmt:"; echo "$out"; return 1; }; }
t_go_vet() { go vet ./...; }
t_go_test() { go test -race -count=1 ./...; }

# ---- tests: daemon ----------------------------------------------------------
t_daemon_lifecycle() {
  local log="$ARTIFACT_DIR/i3-nextd.log" pid wd rc=0
  "$BIN/i3-nextd" --debug >"$log" 2>&1 &
  pid=$!
  retry 50 0.1 grep -q 'starting i3_next desktop core' "$log" ||
    { kill -9 "$pid" 2>/dev/null; echo "daemon did not start"; return 1; }
  kill -TERM "$pid"
  ( sleep 5; kill -KILL "$pid" 2>/dev/null ) & wd=$!      # watchdog against hangs
  wait "$pid" || rc=$?
  kill "$wd" 2>/dev/null || true
  [ "$rc" -eq 0 ] || { echo "exit code $rc (137 = had to be killed)"; return 1; }
  grep -q 'stopped successfully' "$log"
}

# ---- tests: sway / simulated hardware ---------------------------------------
t_sway_config_valid() { sway --validate -c "$SWAY_CONFIG"; }

t_sway_outputs() {
  local n; n="$(output_count)"; echo "outputs=$n expected=$OUTPUTS"
  [ "$n" -eq "$OUTPUTS" ]
}

t_sway_monitor_hotplug() {
  local before new
  before="$(swaymsg -t get_outputs -r | jq -r '.[].name' | sort)"
  swaymsg create_output >/dev/null
  retry 30 0.1 outputs_count_is $((OUTPUTS + 1))
  new="$(comm -13 <(echo "$before") <(swaymsg -t get_outputs -r | jq -r '.[].name' | sort))"
  [ -n "$new" ]
  swaymsg output "$new" unplug >/dev/null
  retry 30 0.1 outputs_count_is "$OUTPUTS"
}

t_sway_workspaces() {
  swaymsg workspace number 2 >/dev/null
  swaymsg -t get_workspaces -r | jq -e '.[] | select(.focused) | .num == 2' >/dev/null
  swaymsg workspace number 1 >/dev/null
}

t_sway_spawn_window() {
  swaymsg exec foot >/dev/null
  retry 50 0.2 foot_is_mapped
}

t_screenshot() {
  local f="$ARTIFACT_DIR/screen.png"
  grim "$f"
  [ "$(file -b --mime-type "$f")" = image/png ]
  [ "$(stat -c %s "$f")" -gt 1000 ]
}

t_sway_close_window() {
  swaymsg '[app_id="foot"] kill' >/dev/null
  retry 50 0.2 bash -c "[ \"\$(swaymsg -t get_tree -r | jq '[recurse(.nodes[]?, .floating_nodes[]?) | select(.app_id == \"foot\")] | length')\" -eq 0 ]"
}

# ---- tests: daemon services --------------------------------------------------
t_daemon_service_disabled() {
  local root log pid rc=0
  root="$(mktemp -d)"
  log="$ARTIFACT_DIR/i3-nextd-disabled.log"
  mkdir -p "$root/config/i3-next" "$root/run"
  chmod 700 "$root/run"
  cat >"$root/config/i3-next/services.yaml" <<'YAML'
version: 1
services:
  shell: { enabled: false, required: false }
YAML

  XDG_CONFIG_HOME="$root/config" XDG_RUNTIME_DIR="$root/run" \
    "$BIN/i3-nextd" --debug >"$log" 2>&1 &
  pid=$!
  retry 100 0.1 grep -q 'starting i3_next desktop core' "$log" ||
    { kill -9 "$pid" 2>/dev/null || true; echo "daemon did not start"; tail -n 40 "$log"; return 1; }
  sleep 0.5
  if pgrep -x quickshell >/dev/null 2>&1; then
    echo "quickshell started despite shell.enabled=false"
    kill -TERM "$pid" 2>/dev/null || true
    return 1
  fi
  kill -TERM "$pid" 2>/dev/null || true
  ( sleep 5; kill -KILL "$pid" 2>/dev/null || true ) & wd=$!
  wait "$pid" || rc=$?
  kill "$wd" 2>/dev/null || true
  [ "$rc" -eq 0 ] || { echo "exit code $rc"; return 1; }
}

t_daemon_workspace_event() {
  local log="$ARTIFACT_DIR/i3-nextd-workspace.log" pid rc=0
  "$BIN/i3-nextd" --debug >"$log" 2>&1 &
  pid=$!
  retry 100 0.1 grep -q 'starting i3_next desktop core' "$log" ||
    { kill -9 "$pid" 2>/dev/null; echo "daemon did not start"; return 1; }
  retry 100 0.1 grep -qi 'sway.connected' "$log" || true
  swaymsg workspace number 2 >/dev/null
  retry 100 0.1 grep -qi 'workspace.changed' "$log" ||
    { echo "no workspace.changed in daemon log"; kill -TERM "$pid" 2>/dev/null || true; return 1; }
  kill -TERM "$pid" 2>/dev/null || true
  ( sleep 5; kill -KILL "$pid" 2>/dev/null ) & wd=$!
  wait "$pid" || rc=$?
  kill "$wd" 2>/dev/null || true
  [ "$rc" -eq 0 ] || { echo "exit code $rc"; return 1; }
  grep -q 'stopped successfully' "$log"
}

t_daemon_shell_restart() {
  local log="$ARTIFACT_DIR/i3-nextd-shell.log" pid rc=0 pid1 pid2 i
  command -v quickshell >/dev/null 2>&1 || { echo "quickshell not installed, skipping"; return 0; }
  "$BIN/i3-nextd" --debug >"$log" 2>&1 &
  pid=$!
  retry 100 0.1 grep -q 'starting i3_next desktop core' "$log" ||
    { kill -9 "$pid" 2>/dev/null; echo "daemon did not start"; return 1; }
  retry 100 0.2 pgrep -x quickshell >/dev/null ||
    { echo "shell did not start"; kill -TERM "$pid" 2>/dev/null || true; return 1; }
  pid1="$(pgrep -n -x quickshell)"
  # kill -9 the shell: it should restart with backoff
  kill -9 "$pid1" 2>/dev/null || true
  retry 40 0.25 bash -c "pid2=\$(pgrep -n -x quickshell); [ -n \"\$pid2\" ] && [ \"\$pid2\" != \"$pid1\" ]" ||
    { echo "shell did not restart after kill -9"; kill -TERM "$pid" 2>/dev/null || true; return 1; }
  # exceed the restart limit to reach degraded
  for i in 1 2 3 4; do
    pid1="$(pgrep -n -x quickshell)" || true
    [ -n "$pid1" ] && kill -9 "$pid1" 2>/dev/null || true
    sleep 0.6
  done
  retry 60 0.5 grep -q 'state=failed' "$log" ||
    { echo "shell did not reach degraded/failed after restart limit"; kill -TERM "$pid" 2>/dev/null || true; return 1; }
  kill -TERM "$pid" 2>/dev/null || true
  ( sleep 5; kill -KILL "$pid" 2>/dev/null || true ) & wd=$!
  wait "$pid" || rc=$?
  kill "$wd" 2>/dev/null || true
  [ "$rc" -eq 0 ] || { echo "exit code $rc"; return 1; }
  grep -q 'stopped successfully' "$log"
}

# ---- main -------------------------------------------------------------------
echo "i3-next e2e  (outputs=$OUTPUTS, sway config=$SWAY_CONFIG)"
echo "artifacts: $ARTIFACT_DIR"
echo

if [ ! -x "$BIN/i3-nextd" ]; then
  echo "building..."; make build >"$ARTIFACT_DIR/build.log" 2>&1 || echo "build failed, see build.log"
fi

echo "build & CLI"
run_test binaries_exist        t_binaries_exist
run_test version_flags         t_version_flags
run_test ctl_status            t_ctl_status
run_test ctl_unknown_command   t_ctl_unknown_command
run_test doctor_sees_sway      t_doctor_sees_sway
run_test doctor_json            t_doctor_json
run_test doctor_exit_ok         t_doctor_exit_ok

echo "go quality gates"
run_test gofmt                 t_gofmt
run_test go_vet                t_go_vet
run_test go_test_race          t_go_test

echo "config (CFG-07/08)"
run_test cfg07_invalid_degrades_no_restart t_cfg07_invalid_config_degrades_no_restart_loop
run_test cfg07_invalid_uses_last_valid     t_cfg07_invalid_uses_last_valid_cache

echo "daemon"
run_test daemon_lifecycle      t_daemon_lifecycle
run_test service_disabled      t_daemon_service_disabled

echo "sway (headless, $OUTPUTS virtual outputs)"
if sway_start; then
  run_test sway_config_valid     t_sway_config_valid
  run_test sway_outputs          t_sway_outputs
  run_test sway_monitor_hotplug  t_sway_monitor_hotplug
  run_test sway_workspaces       t_sway_workspaces
  run_test sway_spawn_window     t_sway_spawn_window
  run_test screenshot            t_screenshot
  swaymsg -t get_tree -r    >"$ARTIFACT_DIR/tree.json"    2>/dev/null
  swaymsg -t get_outputs -r >"$ARTIFACT_DIR/outputs.json" 2>/dev/null
  run_test sway_close_window     t_sway_close_window
  run_test daemon_workspace_event t_daemon_workspace_event
  if command -v quickshell >/dev/null 2>&1; then
    run_test daemon_shell_restart t_daemon_shell_restart
  else
    skip_test daemon_shell_restart "quickshell not installed"
  fi
else
  echo "  sway failed to start, see sway.log:"; tail -n 10 "$ARTIFACT_DIR/sway.log" | sed 's/^/        | /'
  FAIL=$((FAIL + 1))
  for n in sway_config_valid sway_outputs sway_monitor_hotplug sway_workspaces sway_spawn_window screenshot sway_close_window; do
    skip_test "$n" "sway not running"
  done
fi

summary

# ---- tests: config (CFG-07/08) ---------------------------------------------
t_cfg07_invalid_config_degrades_no_restart_loop() {
  local root log pid rc=0 i prev_size=0 stable=0
  root="$(mktemp -d)"
  log="$ARTIFACT_DIR/cfg07-i3-nextd.log"
  mkdir -p "$root/config/i3-next" "$root/state" "$root/data" "$root/cache" "$root/run"
  chmod 700 "$root/run"
  cat >"$root/config/i3-next/desktop.yaml" <<'YAML'
version: 1
not_a_valid_key: true
YAML

  XDG_CONFIG_HOME="$root/config" \
  XDG_STATE_HOME="$root/state" \
  XDG_DATA_HOME="$root/data" \
  XDG_CACHE_HOME="$root/cache" \
  XDG_RUNTIME_DIR="$root/run" \
  "$BIN/i3-nextd" --debug >"$log" 2>&1 &
  pid=$!

  retry 50 0.1 grep -q 'starting i3_next desktop core' "$log" ||
    { kill -9 "$pid" 2>/dev/null || true; echo "daemon did not start"; tail -n 40 "$log"; return 1; }

  # Must not restart-loop: log size should stabilize while process stays alive.
  for i in $(seq 1 20); do
    kill -0 "$pid" 2>/dev/null || { echo "daemon exited unexpectedly"; tail -n 60 "$log"; return 1; }
    size=$(wc -c <"$log" 2>/dev/null || echo 0)
    if [ "$size" -eq "$prev_size" ]; then stable=$((stable+1)); else stable=0; fi
    prev_size="$size"
    [ "$stable" -ge 6 ] && break
    sleep 0.15
  done
  kill -0 "$pid" 2>/dev/null || { echo "daemon not running after observation window"; return 1; }

  grep -Eqi 'degraded|configuration degraded|using fallback|last valid|default config' "$log" ||
    { echo "missing degraded/actionable config error in log"; tail -n 80 "$log"; kill -TERM "$pid" 2>/dev/null || true; return 1; }
  grep -Eqi 'desktop\.yaml|not_a_valid_key|field|line|actionable|fix' "$log" ||
    { echo "missing actionable config detail"; tail -n 80 "$log"; kill -TERM "$pid" 2>/dev/null || true; return 1; }

  kill -TERM "$pid" 2>/dev/null || true
  ( sleep 5; kill -KILL "$pid" 2>/dev/null || true ) & wd=$!
  wait "$pid" || rc=$?
  kill "$wd" 2>/dev/null || true
  [ "$rc" -eq 0 ] || { echo "exit code $rc"; return 1; }
}

t_cfg07_invalid_uses_last_valid_cache() {
  local root log pid rc=0
  root="$(mktemp -d)"
  mkdir -p "$root/config/i3-next" "$root/state" "$root/data" "$root/cache" "$root/run"
  chmod 700 "$root/run"
  cat >"$root/config/i3-next/desktop.yaml" <<'YAML'
version: 1
session:
  name: i3_next
  compositor: sway
paths:
  config_dir: ~/.config/i3-next
  data_dir: ~/.local/share/i3-next
  state_dir: ~/.local/state/i3-next
  cache_dir: ~/.cache/i3-next
logging:
  level: warn
  format: text
ui:
  shell: quickshell
  theme: default
YAML

  log="$ARTIFACT_DIR/cfg07-valid.log"
  XDG_CONFIG_HOME="$root/config" XDG_STATE_HOME="$root/state" XDG_DATA_HOME="$root/data" \
  XDG_CACHE_HOME="$root/cache" XDG_RUNTIME_DIR="$root/run" \
  "$BIN/i3-nextd" --debug >"$log" 2>&1 & pid=$!
  retry 50 0.1 grep -q 'starting i3_next desktop core' "$log" || { kill -9 "$pid" 2>/dev/null || true; return 1; }
  kill -TERM "$pid"; wait "$pid" || true

  test -s "$root/state/i3-next/config/last-valid/desktop.yaml" || { echo "missing last-valid cache"; return 1; }

  cat >"$root/config/i3-next/desktop.yaml" <<'YAML'
::broken
YAML
  log="$ARTIFACT_DIR/cfg07-broken.log"
  XDG_CONFIG_HOME="$root/config" XDG_STATE_HOME="$root/state" XDG_DATA_HOME="$root/data" \
  XDG_CACHE_HOME="$root/cache" XDG_RUNTIME_DIR="$root/run" \
  "$BIN/i3-nextd" --debug >"$log" 2>&1 & pid=$!
  retry 50 0.1 grep -q 'starting i3_next desktop core' "$log" || { kill -9 "$pid" 2>/dev/null || true; tail -n 40 "$log"; return 1; }
  grep -Eqi 'last.valid|last valid|using fallback' "$log" || { echo "expected last-valid fallback"; tail -n 60 "$log"; kill -TERM "$pid" 2>/dev/null || true; return 1; }
  kill -TERM "$pid" 2>/dev/null || true
  ( sleep 5; kill -KILL "$pid" 2>/dev/null || true ) & wd=$!
  wait "$pid" || rc=$?
  kill "$wd" 2>/dev/null || true
  [ "$rc" -eq 0 ]
}
