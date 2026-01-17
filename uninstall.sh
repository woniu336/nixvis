#!/bin/bash
#
# NixVis 全自动卸载脚本 Debian/Ubuntu（无交互）
#

set -e

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
    echo -e "${2}${1}${NC}"
}

check_root() {
    if [ "$EUID" -ne 0 ]; then
        print_msg "错误：请使用 root 权限运行（sudo）" "$RED"
        exit 1
    fi
}

stop_service() {
    if systemctl is-active --quiet nixvis; then
        print_msg "停止 NixVis 服务..." "$YELLOW"
        systemctl stop nixvis
    fi
}

disable_service() {
    if systemctl is-enabled --quiet nixvis; then
        print_msg "禁用 NixVis 服务..." "$YELLOW"
        systemctl disable nixvis
    fi
}

remove_service() {
    if [ -f "/etc/systemd/system/nixvis.service" ]; then
        print_msg "删除 systemd 服务..." "$YELLOW"
        rm -f /etc/systemd/system/nixvis.service
        systemctl daemon-reload
    fi
}

remove_files() {
    print_msg "删除程序目录..." "$YELLOW"
    rm -rf "$INSTALL_DIR"

    print_msg "删除数据目录..." "$YELLOW"
    rm -rf "$DATA_DIR"

    print_msg "删除配置目录..." "$YELLOW"
    rm -rf "$CONFIG_DIR"

    print_msg "删除用户和用户组..." "$YELLOW"
    userdel "$APP_USER" 2>/dev/null || true
    groupdel "$APP_USER" 2>/dev/null || true
}

main() {
    echo ""
    print_msg "=== NixVis 全自动卸载开始 ===" "$GREEN"
    echo ""

    check_root
    stop_service
    disable_service
    remove_service
    remove_files

    echo ""
    print_msg "=== 卸载完成（无交互）===" "$GREEN"
    echo ""
}

main
