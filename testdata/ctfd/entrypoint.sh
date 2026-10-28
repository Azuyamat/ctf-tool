#!/bin/bash
set -euo pipefail

cd /opt/CTFd
PYTHONPATH=/opt/CTFd python /opt/seed/seed.py
exec /opt/CTFd/docker-entrypoint.sh
