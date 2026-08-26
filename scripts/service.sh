#!/bin/bash
# nuteo-web service manager — convenience wrapper around systemctl.
#
# Usage:
#   scripts/service.sh status          # show status
#   scripts/service.sh start           # start (if not running)
#   scripts/service.sh stop            # stop
#   scripts/service.sh restart         # restart
#   scripts/service.sh logs            # follow logs (Ctrl-C to exit)
#   scripts/service.sh logs -n 50      # last 50 lines
#   scripts/service.sh build           # rebuild binary and reload service
#   scripts/service.sh disable-boot    # disable auto-start on boot
#   scripts/service.sh enable-boot     # enable auto-start on boot
set -e

PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$PROJECT_DIR"

SERVICE_NAME="nuteo-web"

case "${1:-status}" in
    status)
        sudo systemctl status "$SERVICE_NAME" --no-pager
        ;;

    start)
        sudo systemctl start "$SERVICE_NAME"
        sudo systemctl status "$SERVICE_NAME" --no-pager
        ;;

    stop)
        sudo systemctl stop "$SERVICE_NAME"
        ;;

    restart)
        sudo systemctl restart "$SERVICE_NAME"
        sleep 1
        sudo systemctl status "$SERVICE_NAME" --no-pager
        ;;

    logs)
        # Pass-through any additional args (e.g. -n 50)
        shift
        sudo journalctl -u "$SERVICE_NAME" --no-pager "$@"
        ;;

    follow)
        sudo journalctl -u "$SERVICE_NAME" -f
        ;;

    build)
        echo "→ Building binary..."
        make build
        echo "→ Restarting service..."
        sudo systemctl restart "$SERVICE_NAME"
        sleep 1
        sudo systemctl status "$SERVICE_NAME" --no-pager
        ;;

    enable-boot)
        sudo systemctl enable "$SERVICE_NAME"
        echo "→ Auto-start on boot: enabled"
        sudo systemctl is-enabled "$SERVICE_NAME"
        ;;

    disable-boot)
        sudo systemctl disable "$SERVICE_NAME"
        echo "→ Auto-start on boot: disabled"
        sudo systemctl is-enabled "$SERVICE_NAME" || echo "(disabled)"
        ;;

    health)
        if curl -fsS http://localhost:8080/healthz >/dev/null; then
            echo "✓ Health check passed"
            exit 0
        else
            echo "✗ Health check failed"
            exit 1
        fi
        ;;

    *)
        cat <<EOF
nuteo-web service manager

Usage: $0 <command>

Commands:
    status         Show service status
    start          Start service (no-op if already running)
    stop           Stop service
    restart        Restart service
    logs [args]    Show logs (e.g. -n 50 for last 50 lines)
    follow         Follow logs in real-time (Ctrl-C to exit)
    build          Rebuild binary and restart service
    enable-boot    Enable auto-start on boot
    disable-boot   Disable auto-start on boot
    health         Run a single health check

Examples:
    $0 status
    $0 build           # edit code, then run this
    $0 logs -n 100
    $0 follow
EOF
        exit 1
        ;;
esac
