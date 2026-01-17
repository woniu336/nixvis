#!/bin/bash
#
# NixVis 卸载脚本 Debian/Ubuntu
#

set -e

# 颜色定义
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
        print_msg "错误：请使用 root 权限运行（使用 sudo）" "$RED"
        exit 1
    fi
}

confirm() {
    read -p "将从系统中卸载 NixVis。确定吗？(y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_msg "已取消卸载。" "$YELLOW"
        exit 0
    fi
}

stop_service() {
    if systemctl is-active --quiet nixvis; then
        print_msg "正在停止 NixVis 服务..." "$YELLOW"
        systemctl stop nixvis
    fi
}

disable_service() {
    if systemctl is-enabled --quiet nixvis; then
        print_msg "正在禁用 NixVis 服务..." "$YELLOW"
        systemctl disable nixvis
    fi
}

remove_service() {
    if [ -f "/etc/systemd/system/nixvis.service" ]; then
        print_msg "正在删除 systemd 服务..." "$YELLOW"
        rm -f /etc/systemd/system/nixvis.service
        systemctl daemon-reload
    fi
}

remove_files() {
    print_msg "正在删除已安装的文件..." "$YELLOW"

    # 询问是否删除数据目录
    read -p "删除数据目录 ($DATA_DIR)？(y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -rf "$DATA_DIR"
        print_msg "数据目录已删除" "$GREEN"
    else
        print_msg "数据目录已保留" "$YELLOW"
    fi

    # 询问是否删除配置目录
    read -p "删除配置目录 ($CONFIG_DIR)？(y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -rf "$CONFIG_DIR"
        print_msg "配置目录已删除" "$GREEN"
    else
        print_msg "配置目录已保留" "$YELLOW"
    fi

    # 询问是否删除用户
    read -p "删除用户 $APP_USER？(y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        userdel $APP_USER 2>/dev/null || true
        groupdel $APP_USER 2>/dev/null || true
        print_msg "用户已删除" "$GREEN"
    else
        print_msg "用户已保留" "$YELLOW"
    fi

    # 删除二进制文件目录
    rm -rf "$INSTALL_DIR"
    print_msg "程序目录已删除" "$GREEN"
}

main() {
    echo ""
    print_msg "=== NixVis 卸载脚本 ===" "$GREEN"
    echo ""

    check_root
    confirm
    stop_service
    disable_service
    remove_service
    remove_files

    echo ""
    print_msg "=== 卸载完成 ===" "$GREEN"
    echo ""
}

main
