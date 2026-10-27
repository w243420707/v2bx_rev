#!/usr/bin/env bash
# V2bX 一键安装脚本
# 用法:
#   bash install.sh            交互菜单
#   bash install.sh install    安装(非交互)
#   bash install.sh update     更新(非交互)
#   bash install.sh uninstall  卸载(非交互)
set -u

REPO="w243420707/v2bx_rev"
INSTALL_DIR="/etc/V2bX"
BIN_PATH="/usr/local/bin/V2bX"
SERVICE_NAME="V2bX"

#---------- 输出 ----------

info()  { printf '\033[32m[信息]\033[0m %s\n' "$*"; }
warn()  { printf '\033[33m[警告]\033[0m %s\n' "$*"; }
error() { printf '\033[31m[错误]\033[0m %s\n' "$*" >&2; }
die()   { error "$*"; exit 1; }

#---------- 环境检测 ----------

check_root() {
  [ "$(id -u)" = "0" ] || die "请用 root 运行（sudo -i 后重试）"
}

detect_pkg_mgr() {
  if [ -f /etc/debian_version ]; then echo "apt"
  elif [ -f /etc/redhat-release ]; then echo "yum"
  elif [ -f /etc/alpine-release ]; then echo "apk"
  elif command -v apt-get >/dev/null 2>&1; then echo "apt"
  elif command -v dnf >/dev/null 2>&1; then echo "dnf"
  elif command -v yum >/dev/null 2>&1; then echo "yum"
  else echo ""
  fi
}

ensure_deps() {
  command -v curl  >/dev/null 2>&1 && \
  command -v unzip >/dev/null 2>&1 && return 0

  local mgr; mgr="$(detect_pkg_mgr)"
  case "$mgr" in
    apt) apt-get update -qq && apt-get install -y -qq curl unzip ;;
    dnf) dnf install -y curl unzip ;;
    yum) yum install -y curl unzip ;;
    apk) apk add --no-cache curl unzip ;;
    *)   die "未找到 curl/unzip 且无法识别包管理器，请手动安装后重试" ;;
  esac
}

# 架构 -> Release 资产名，映射与 .github/build/friendly-filenames.json 保持一致
detect_asset() {
  case "$(uname -m)" in
    x86_64|amd64)   echo "linux-64" ;;
    aarch64|arm64)  echo "linux-arm64-v8a" ;;
    armv7l)         echo "linux-arm32-v7a" ;;
    i386|i686)      echo "linux-32" ;;
    *) die "暂不支持的架构 $(uname -m)，请到 Releases 页面手动下载" ;;
  esac
}

download() { # $1=URL $2=输出文件
  info "下载: $1"
  if command -v curl >/dev/null 2>&1; then
    curl -fL --connect-timeout 15 --retry 2 -o "$2" "$1" || die "下载失败，请检查网络或 Releases 页面"
  else
    wget -q --tries=2 -O "$2" "$1" || die "下载失败，请检查网络或 Releases 页面"
  fi
}

#---------- 安装 / 更新 / 卸载 ----------

fetch_release() { # $1=临时目录，解压后文件落在 $1
  local asset; asset="$(detect_asset)"
  local url="https://github.com/${REPO}/releases/latest/download/V2bX-${asset}.zip"
  local tmpzip="$1/v2bx.zip"
  download "$url" "$tmpzip"
  unzip -q -o "$tmpzip" -d "$1"
  [ -f "$1/V2bX" ] || die "压缩包内未找到 V2bX 二进制，下载内容可能不完整"
}

write_service() {
  cat > /etc/systemd/system/${SERVICE_NAME}.service <<'EOF'
[Unit]
Description=V2bX Service
After=network.target network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/V2bX server --config /etc/V2bX/config.json
Restart=on-failure
RestartSec=5s
LimitNOFILE=1048576

[Install]
WantedBy=multi-user.target
EOF
  systemctl daemon-reload
}

do_install() {
  check_root
  ensure_deps
  [ -f "$BIN_PATH" ] && die "检测到已安装（$BIN_PATH），请使用 update 更新"

  local tmp; tmp="$(mktemp -d)"
  trap 'rm -rf "$tmp"' EXIT
  fetch_release "$tmp"

  install -Dm755 "$tmp/V2bX" "$BIN_PATH"

  # 配置与规则数据：config.json 仅在不存在时创建，避免覆盖用户配置
  mkdir -p "$INSTALL_DIR"
  [ -f "$INSTALL_DIR/config.json" ] || cp "$tmp/config.json" "$INSTALL_DIR/config.json"
  for f in geoip.dat geosite.dat; do
    [ -f "$tmp/$f" ] && install -Dm644 "$tmp/$f" "$INSTALL_DIR/$f"
  done

  if command -v systemctl >/dev/null 2>&1; then
    write_service
    systemctl enable --now "$SERVICE_NAME" >/dev/null 2>&1
    info "systemd 服务已安装并启动"
  else
    warn "未检测到 systemd（容器环境？），请手动运行: $BIN_PATH server --config $INSTALL_DIR/config.json"
  fi

  echo
  info "安装完成: $($BIN_PATH version | head -1)"
  info "配置文件: $INSTALL_DIR/config.json （填入面板信息后执行 systemctl restart $SERVICE_NAME）"
  info "常用命令: systemctl status|restart|stop $SERVICE_NAME   /   日志: journalctl -u $SERVICE_NAME -f"
}

do_update() {
  check_root
  ensure_deps
  [ -f "$BIN_PATH" ] || die "尚未安装，请先执行 install"

  local tmp; tmp="$(mktemp -d)"
  trap 'rm -rf "$tmp"' EXIT
  fetch_release "$tmp"

  install -Dm755 "$tmp/V2bX" "$BIN_PATH"
  for f in geoip.dat geosite.dat; do
    [ -f "$tmp/$f" ] && install -Dm644 "$tmp/$f" "$INSTALL_DIR/$f"
  done

  if command -v systemctl >/dev/null 2>&1 && systemctl list-unit-files | grep -q "^${SERVICE_NAME}.service"; then
    systemctl restart "$SERVICE_NAME"
    info "已更新并重启服务"
  else
    info "二进制已更新（服务未运行，未执行重启）"
  fi
  info "当前版本: $($BIN_PATH version | head -1)"
}

do_uninstall() {
  check_root
  if command -v systemctl >/dev/null 2>&1 && systemctl list-unit-files | grep -q "^${SERVICE_NAME}.service"; then
    systemctl disable --now "$SERVICE_NAME" >/dev/null 2>&1
    rm -f /etc/systemd/system/${SERVICE_NAME}.service
    systemctl daemon-reload
    info "服务已停止并移除"
  fi
  rm -f "$BIN_PATH"
  warn "配置目录 $INSTALL_DIR 已保留（含 config.json），确认不再需要请手动删除: rm -rf $INSTALL_DIR"
  info "卸载完成"
}

show_menu() {
  echo "====== V2bX 管理菜单 ======"
  echo "  1. 安装 V2bX"
  echo "  2. 更新 V2bX"
  echo "  3. 卸载 V2bX"
  echo "  0. 退出"
  echo "=========================="
  read -rp "请输入编号: " choice
  case "$choice" in
    1) do_install ;;
    2) do_update ;;
    3) do_uninstall ;;
    0) exit 0 ;;
    *) die "无效选择" ;;
  esac
}

#---------- 入口 ----------

case "${1:-}" in
  install)   do_install ;;
  update)    do_update ;;
  uninstall) do_uninstall ;;
  "")        show_menu ;;
  *)         die "用法: bash install.sh [install|update|uninstall]" ;;
esac
