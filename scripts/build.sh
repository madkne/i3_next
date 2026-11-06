#!/usr/bin/env bash
set -euo pipefail

# Thin wrapper around the single build source (Makefile).
exec make build
