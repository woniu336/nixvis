#!/bin/bash
#
# NixVis Uninstallation Script for Debian/Ubuntu
#

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

APP_NAME="nixvis"
APP_USER="nixvis"
INSTALL_DIR="/opt/nixvis"
DATA_DIR="/var/lib/nixvis"
CONFIG_DIR="/etc/nixvis"

print_msg() {
    local msg=$1
    local color=$2
    echo -e "${color}${msg}${NC}"
}

check_root() {
    if [ "$EUID" -ne 0 ]; then
        print_msg "Error: Please run as root (use sudo)" "$RED"
        exit 1
    fi
}

confirm() {
    read -p "This will remove NixVis from your system. Are you sure? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_msg "Uninstallation cancelled." "$YELLOW"
        exit 0
    fi
}

stop_service() {
    if systemctl is-active --quiet nixvis; then
        print_msg "Stopping NixVis service..." "$YELLOW"
        systemctl stop nixvis
    fi
}

disable_service() {
    if systemctl is-enabled --quiet nixvis; then
        print_msg "Disabling NixVis service..." "$YELLOW"
        systemctl disable nixvis
    fi
}

remove_service() {
    if [ -f "/etc/systemd/system/nixvis.service" ]; then
        print_msg "Removing systemd service..." "$YELLOW"
        rm -f /etc/systemd/system/nixvis.service
        systemctl daemon-reload
    fi
}

remove_files() {
    print_msg "Removing installed files..." "$YELLOW"

    # Ask about data removal
    read -p "Remove data directory ($DATA_DIR)? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -rf "$DATA_DIR"
    fi

    # Ask about config removal
    read -p "Remove configuration directory ($CONFIG_DIR)? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -rf "$CONFIG_DIR"
    fi

    # Ask about user removal
    read -p "Remove user $APP_USER? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        userdel $APP_USER 2>/dev/null || true
        groupdel $APP_USER 2>/dev/null || true
    fi

    # Remove binary
    rm -rf "$INSTALL_DIR"
}

main() {
    echo ""
    print_msg "=== NixVis Uninstallation Script ===" "$GREEN"
    echo ""

    check_root
    confirm
    stop_service
    disable_service
    remove_service
    remove_files

    print_msg "=== Uninstallation Complete ===" "$GREEN"
}

main
