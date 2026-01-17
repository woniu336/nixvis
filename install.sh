#!/bin/bash
#
# NixVis Installation Script for Debian/Ubuntu
# This script installs NixVis as a system service
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
APP_NAME="nixvis"
APP_USER="nixvis"
APP_GROUP="nixvis"
INSTALL_DIR="/opt/nixvis"
DATA_DIR="/var/lib/nixvis"
CONFIG_DIR="/etc/nixvis"
SERVICE_FILE="/etc/systemd/system/nixvis.service"
BINARY_NAME="nixvis-linux-amd64"

# Print colored message
print_msg() {
    local msg=$1
    local color=$2
    echo -e "${color}${msg}${NC}"
}

# Check if running as root
check_root() {
    if [ "$EUID" -ne 0 ]; then
        print_msg "Error: Please run as root (use sudo)" "$RED"
        exit 1
    fi
}

# Check if binary exists
check_binary() {
    if [ ! -f "$BINARY_NAME" ]; then
        print_msg "Error: Binary file '$BINARY_NAME' not found in current directory" "$RED"
        print_msg "Please build the binary first or download it" "$YELLOW"
        exit 1
    fi
}

# Detect system architecture
detect_arch() {
    ARCH=$(uname -m)
    case $ARCH in
        x86_64)
            print_msg "Detected architecture: x86_64 (amd64)" "$GREEN"
            ;;
        aarch64)
            print_msg "Detected architecture: aarch64 (arm64)" "$GREEN"
            ;;
        *)
            print_msg "Warning: Untested architecture: $ARCH" "$YELLOW"
            ;;
    esac
}

# Stop existing service if running
stop_service() {
    if systemctl is-active --quiet nixvis; then
        print_msg "Stopping existing NixVis service..." "$YELLOW"
        systemctl stop nixvis
    fi
}

# Create user and group
create_user() {
    if ! id "$APP_USER" &>/dev/null; then
        print_msg "Creating user and group: $APP_USER" "$YELLOW"
        useradd --system --user-group --home-dir "$DATA_DIR" --shell /bin/false $APP_USER
    else
        print_msg "User $APP_USER already exists" "$GREEN"
    fi
}

# Create directories
create_directories() {
    print_msg "Creating directories..." "$YELLOW"
    mkdir -p "$INSTALL_DIR"
    mkdir -p "$DATA_DIR"
    mkdir -p "$CONFIG_DIR"
    mkdir -p "$DATA_DIR/logs"
    mkdir -p "/var/log/nixvis"
}

# Install binary
install_binary() {
    print_msg "Installing binary to $INSTALL_DIR..." "$YELLOW"
    cp "$BINARY_NAME" "$INSTALL_DIR/nixvis"
    chmod +x "$INSTALL_DIR/nixvis"
    chown $APP_USER:$APP_GROUP "$INSTALL_DIR/nixvis"
}

# Create default config
create_config() {
    if [ ! -f "$CONFIG_DIR/config.json" ]; then
        print_msg "Creating default configuration..." "$YELLOW"
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
        print_msg "Configuration file already exists, skipping..." "$YELLOW"
    fi
}

# Install systemd service
install_service() {
    print_msg "Installing systemd service..." "$YELLOW"
    cat > "$SERVICE_FILE" << EOF
[Unit]
Description=NixVis Nginx Log Analyzer
After=network.target

[Service]
Type=simple
User=$APP_USER
Group=$APP_GROUP
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/nixvis
Restart=on-failure
RestartSec=5s

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=$DATA_DIR $CONFIG_DIR /var/log/nixvis

# Environment
Environment="HOME=$DATA_DIR"

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
}

# Set permissions
set_permissions() {
    print_msg "Setting permissions..." "$YELLOW"
    chown -R $APP_USER:$APP_GROUP "$INSTALL_DIR"
    chown -R $APP_USER:$APP_GROUP "$DATA_DIR"
    chown -R $APP_USER:$APP_GROUP "$CONFIG_DIR"
    chown -R $APP_USER:$APP_GROUP "/var/log/nixvis"
    chmod 750 "$DATA_DIR"
    chmod 750 "$CONFIG_DIR"
    chmod 755 "$INSTALL_DIR"
}

# Enable and start service
start_service() {
    print_msg "Enabling and starting NixVis service..." "$YELLOW"
    systemctl enable nixvis
    systemctl start nixvis

    # Wait a moment for service to start
    sleep 2

    if systemctl is-active --quiet nixvis; then
        print_msg "NixVis service started successfully!" "$GREEN"
    else
        print_msg "Warning: Service may not have started properly. Check with: journalctl -u nixvis" "$YELLOW"
    fi
}

# Print status info
print_status() {
    echo ""
    print_msg "=== Installation Complete ===" "$GREEN"
    echo ""
    echo "Service status: systemctl status nixvis"
    echo "View logs: journalctl -u nixvis -f"
    echo "Config file: $CONFIG_DIR/config.json"
    echo "Data directory: $DATA_DIR"
    echo "Web interface: http://localhost:9523"
    echo ""
    echo "Useful commands:"
    echo "  Start:   sudo systemctl start nixvis"
    echo "  Stop:    sudo systemctl stop nixvis"
    echo "  Restart: sudo systemctl restart nixvis"
    echo "  Status:  sudo systemctl status nixvis"
    echo ""
}

# Main installation flow
main() {
    echo ""
    print_msg "=== NixVis Installation Script ===" "$GREEN"
    echo ""

    check_root
    check_binary
    detect_arch
    stop_service
    create_user
    create_directories
    install_binary
    create_config
    install_service
    set_permissions
    start_service
    print_status
}

# Run main function
main
