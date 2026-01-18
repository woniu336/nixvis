#!/bin/bash
#
# NixVis 安装脚本（最终完整版）
# 适用于 Debian / Ubuntu
#

set -e

# ================= 基础配置 =================
APP_NAME="nixvis"
APP_USER="nixvis"
APP_GROUP="nixvis"

INSTALL_DIR="/opt/nixvis"
DATA_DIR="/var/lib/nixvis"
CONFIG_DIR="/etc/nixvis"
LOG_DIR="/var/log/nixvis"
SERVICE_FILE="/etc/systemd/system/nixvis.service"

VERSION="v2.2.3"
BASE_URL="https://github.com/woniu336/nixvis/releases/download/${VERSION}"

# ================= 颜色 =================
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

msg() { echo -e "${2}${1}${NC}"; }

# ================= Root 检查 =================
check_root() {
    if [ "$EUID" -ne 0 ]; then
        msg "错误：请使用 root 或 sudo 运行脚本" "$RED"
        exit 1
    fi
}

# ================= 架构检测 =================
detect_arch() {
    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64)
            BIN_NAME="nixvis-linux-amd64"
            ;;
        aarch64)
            BIN_NAME="nixvis-linux-arm64"
            ;;
        *)
            msg "不支持的架构：$ARCH" "$RED"
            exit 1
            ;;
    esac
    DOWNLOAD_URL="${BASE_URL}/${BIN_NAME}"
}

# ================= 停止服务 =================
stop_service() {
    systemctl stop nixvis 2>/dev/null || true
}

# ================= 创建用户 =================
create_user() {
    if ! id "$APP_USER" &>/dev/null; then
        msg "创建系统用户：$APP_USER" "$YELLOW"
        useradd \
            --system \
            --user-group \
            --home-dir "$DATA_DIR" \
            --shell /usr/sbin/nologin \
            "$APP_USER"
    fi
}

# ================= 创建目录 =================
create_directories() {
    msg "创建目录结构" "$YELLOW"
    mkdir -p "$INSTALL_DIR" "$DATA_DIR" "$CONFIG_DIR" "$LOG_DIR"
}

# ================= 下载程序 =================
download_binary() {
    msg "下载 NixVis 程序" "$BLUE"

    TMP_DIR=$(mktemp -d)
    cd "$TMP_DIR"

    if command -v curl >/dev/null; then
        curl -L -o nixvis "$DOWNLOAD_URL"
    else
        wget -O nixvis "$DOWNLOAD_URL"
    fi

    chmod +x nixvis
    mv nixvis "$INSTALL_DIR/nixvis"

    cd /
    rm -rf "$TMP_DIR"
}

# ================= 创建默认配置 =================
create_config() {
    if [ -f "$CONFIG_DIR/config.json" ]; then
        msg "配置文件已存在，跳过生成" "$YELLOW"
        return
    fi

    msg "生成默认配置文件" "$YELLOW"

    cat > "$CONFIG_DIR/config.json" << 'EOF'
{
  "system": {
    "logDestination": "file",
    "taskInterval": "5m",
    "timezone": "Asia/Shanghai"
  },
  "server": {
    "Port": ":9523"
  },
  "websites": [],
  "pvFilter": {
    "statusCodeInclude": [200],
    "excludePatterns": [
      "favicon.ico$",
      "robots.txt$",
      "sitemap.xml$",
      "\\.(?:js|css|jpg|jpeg|png|gif|svg|webp|woff|woff2|ttf|eot|ico)$",
      "^/(?:api|ajax)/",
      "^/index\\.php/ajax/",
      "^/index\\.php/user/ajax_ulog$",
      "^/health$",
      "^/_(?:nuxt|next)/",
      "rss.xml$",
      "feed.xml$",
      "atom.xml$"
    ],
    "excludeIPs": [
      "127.0.0.1",
      "::1"
    ]
  }
}
EOF
}

# ================= 安装 systemd 服务 =================
install_service() {
    msg "安装 systemd 服务" "$YELLOW"

    cat > "$SERVICE_FILE" << EOF
[Unit]
Description=NixVis Nginx 日志分析工具
After=network.target

[Service]
Type=simple
User=${APP_USER}
Group=${APP_GROUP}
ExecStart=${INSTALL_DIR}/nixvis
WorkingDirectory=${DATA_DIR}
Restart=on-failure
RestartSec=5

# 安全与隔离
PrivateTmp=true
NoNewPrivileges=false
ProtectSystem=false
ProtectHome=false
ReadWritePaths=${DATA_DIR} ${CONFIG_DIR} ${LOG_DIR}

Environment=HOME=${DATA_DIR}
Environment=NIXVIS_SYSTEM_MODE=1

# IP 屏蔽功能需要 CAP_NET_ADMIN 权限
AmbientCapabilities=CAP_NET_ADMIN
CapabilityBoundingSet=CAP_NET_ADMIN

[Install]
WantedBy=multi-user.target
EOF
}

# ================= 权限修复（关键） =================
set_permissions() {
    msg "设置文件权限（关键步骤）" "$GREEN"

    # 程序目录
    chown -R root:root "$INSTALL_DIR"
    chmod 755 "$INSTALL_DIR" "$INSTALL_DIR/nixvis"

    # 配置目录：nixvis 用户需要读写配置文件（添加站点时）
    chown -R $APP_USER:$APP_GROUP "$CONFIG_DIR"
    chmod 750 "$CONFIG_DIR"
    chmod 640 "$CONFIG_DIR/config.json"

    # 数据与日志
    chown -R $APP_USER:$APP_GROUP "$DATA_DIR" "$LOG_DIR"
    chmod 750 "$DATA_DIR" "$LOG_DIR"

    # 将 nixvis 用户添加到 adm 组，以便读取 nginx 日志文件
    msg "将 nixvis 用户添加到 adm 组（读取 nginx 日志）" "$YELLOW"
    usermod -aG adm "$APP_USER"
}

# ================= 启动服务 =================
start_service() {
    msg "启动 NixVis 服务" "$BLUE"

    systemctl daemon-reload
    systemctl enable nixvis
    systemctl restart nixvis

    sleep 2

    if systemctl is-active --quiet nixvis; then
        msg "🎉 NixVis 启动成功" "$GREEN"
    else
        msg "⚠ 服务启动失败，请检查日志" "$RED"
        journalctl -u nixvis --no-pager -n 30
        exit 1
    fi
}

# ================= 主流程 =================
main() {
    echo
    msg "=== NixVis 安装脚本 ===" "$GREEN"

    check_root
    detect_arch
    stop_service
    create_user
    create_directories
    download_binary
    create_config
    install_service
    set_permissions
    start_service

    echo
    msg "安装完成 🎉" "$GREEN"
    echo "配置文件： $CONFIG_DIR/config.json"
    echo "日志查看： journalctl -u nixvis -f"
    echo "Web 地址： http://$(hostname -I | awk '{print $1}'):9523"
    echo
}

main
