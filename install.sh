#!/bin/bash
#
# NixVis 安装脚本 Debian/Ubuntu
# 此脚本会下载并安装 NixVis 作为系统服务
#

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # 无颜色

# 配置
APP_NAME="nixvis"
APP_USER="nixvis"
APP_GROUP="nixvis"
INSTALL_DIR="/opt/nixvis"
DATA_DIR="/var/lib/nixvis"
CONFIG_DIR="/etc/nixvis"
SERVICE_FILE="/etc/systemd/system/nixvis.service"
DOWNLOAD_URL="https://github.com/woniu336/nixvis/releases/download/v2.2.3/nixvis-linux-amd64"
BINARY_NAME="nixvis-linux-amd64"

# 打印彩色消息
print_msg() {
    local msg=$1
    local color=$2
    echo -e "${color}${msg}${NC}"
}

# 检查是否为 root 用户
check_root() {
    if [ "$EUID" -ne 0 ]; then
        print_msg "错误：请使用 root 权限运行（使用 sudo）" "$RED"
        exit 1
    fi
}

# 检测系统架构
detect_arch() {
    ARCH=$(uname -m)
    case $ARCH in
        x86_64)
            print_msg "检测到架构：x86_64 (amd64)" "$GREEN"
            ;;
        aarch64)
            print_msg "检测到架构：aarch64 (arm64)" "$YELLOW"
            print_msg "注意：当前下载仅支持 amd64" "$YELLOW"
            ;;
        *)
            print_msg "警告：未经测试的架构：$ARCH" "$YELLOW"
            ;;
    esac
}

# 停止已运行的服务
stop_service() {
    if systemctl is-active --quiet nixvis 2>/dev/null; then
        print_msg "正在停止现有的 NixVis 服务..." "$YELLOW"
        systemctl stop nixvis
    fi
}

# 从 GitHub 下载二进制文件
download_binary() {
    print_msg "正在从 GitHub 下载 NixVis..." "$BLUE"

    # 创建临时下载目录
    TEMP_DIR=$(mktemp -d)
    cd "$TEMP_DIR"

    # 下载并显示进度
    if command -v wget &> /dev/null; then
        wget --show-progress -O "$BINARY_NAME" "$DOWNLOAD_URL"
    elif command -v curl &> /dev/null; then
        curl -L --progress-bar -o "$BINARY_NAME" "$DOWNLOAD_URL"
    else
        print_msg "错误：系统既没有 wget 也没有 curl" "$RED"
        rm -rf "$TEMP_DIR"
        exit 1
    fi

    # 验证下载
    if [ ! -f "$BINARY_NAME" ]; then
        print_msg "错误：下载失败" "$RED"
        rm -rf "$TEMP_DIR"
        exit 1
    fi

    # 获取文件大小
    SIZE=$(du -h "$BINARY_NAME" | cut -f1)
    print_msg "已下载：$BINARY_NAME ($SIZE)" "$GREEN"

    # 移动到安装目录
    mkdir -p "$INSTALL_DIR"
    mv "$BINARY_NAME" "$INSTALL_DIR/nixvis"
    chmod +x "$INSTALL_DIR/nixvis"

    # 清理临时目录
    cd - > /dev/null
    rm -rf "$TEMP_DIR"
}

# 创建用户和组
create_user() {
    if ! id "$APP_USER" &>/dev/null; then
        print_msg "正在创建用户和组：$APP_USER" "$YELLOW"
        useradd --system --user-group --home-dir "$DATA_DIR" --shell /bin/false $APP_USER
    else
        print_msg "用户 $APP_USER 已存在" "$GREEN"
    fi
}

# 创建目录
create_directories() {
    print_msg "正在创建目录..." "$YELLOW"
    mkdir -p "$INSTALL_DIR"
    mkdir -p "$DATA_DIR"
    mkdir -p "$CONFIG_DIR"
    mkdir -p "$DATA_DIR/logs"
    mkdir -p "/var/log/nixvis"
}

# 创建默认配置
create_config() {
    if [ ! -f "$CONFIG_DIR/config.json" ]; then
        print_msg "正在创建默认配置..." "$YELLOW"
        cat > "$CONFIG_DIR/config.json" << 'EOF'
{
  "system": {
    "logDestination": "file",
    "taskInterval": "5m"
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
        chown $APP_USER:$APP_GROUP "$CONFIG_DIR/config.json"
    else
        print_msg "配置文件已存在，跳过..." "$YELLOW"
    fi
}

# 安装 systemd 服务
install_service() {
    print_msg "正在安装 systemd 服务..." "$YELLOW"
    cat > "$SERVICE_FILE" << EOF
[Unit]
Description=NixVis Nginx 日志分析工具
After=network.target

[Service]
Type=simple
User=$APP_USER
Group=$APP_GROUP
WorkingDirectory=$DATA_DIR
ExecStart=$INSTALL_DIR/nixvis
Restart=on-failure
RestartSec=5s

# 安全设置
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=$DATA_DIR $CONFIG_DIR /var/log/nixvis

# 环境变量
Environment="HOME=$DATA_DIR"
Environment="NIXVIS_SYSTEM_MODE=1"

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
}

# 设置权限
set_permissions() {
    print_msg "正在设置权限..." "$YELLOW"
    chown -R $APP_USER:$APP_GROUP "$INSTALL_DIR"
    chown -R $APP_USER:$APP_GROUP "$DATA_DIR"
    chown -R $APP_USER:$APP_GROUP "$CONFIG_DIR"
    chown -R $APP_USER:$APP_GROUP "/var/log/nixvis"
    chmod 750 "$DATA_DIR"
    chmod 750 "$CONFIG_DIR"
    chmod 755 "$INSTALL_DIR"
}

# 启用并启动服务
start_service() {
    print_msg "正在启用并启动 NixVis 服务..." "$YELLOW"
    systemctl enable nixvis
    systemctl start nixvis

    # 等待服务启动
    sleep 3

    if systemctl is-active --quiet nixvis; then
        print_msg "NixVis 服务启动成功！" "$GREEN"
    else
        print_msg "警告：服务可能未正常启动。请检查：journalctl -u nixvis" "$YELLOW"
    fi
}

# 打印状态信息
print_status() {
    echo ""
    print_msg "=== 安装完成 ===" "$GREEN"
    echo ""
    echo "服务状态：      systemctl status nixvis"
    echo "查看日志：      journalctl -u nixvis -f"
    echo "配置文件：      $CONFIG_DIR/config.json"
    echo "数据目录：      $DATA_DIR"
    echo "Web 界面：      http://localhost:9523"
    echo ""
    echo "常用命令："
    echo "  启动：   sudo systemctl start nixvis"
    echo "  停止：   sudo systemctl stop nixvis"
    echo "  重启：   sudo systemctl restart nixvis"
    echo "  状态：   sudo systemctl status nixvis"
    echo ""
    print_msg "下一步操作：" "$BLUE"
    echo "  1. 编辑配置：sudo nano $CONFIG_DIR/config.json"
    echo "  2. 添加您的网站和日志路径"
    echo "  3. 重启服务：sudo systemctl restart nixvis"
    echo "  4. 访问：    http://$(hostname -I | awk '{print $1}'):9523"
    echo ""
}

# 主安装流程
main() {
    echo ""
    print_msg "=== NixVis 安装脚本 ===" "$GREEN"
    echo ""
    print_msg "版本：v2.2.3" "$BLUE"
    print_msg "下载地址：$DOWNLOAD_URL" "$BLUE"
    echo ""

    check_root
    detect_arch
    stop_service
    download_binary
    create_user
    create_directories
    create_config
    install_service
    set_permissions
    start_service
    print_status
}

# 运行主函数
main
