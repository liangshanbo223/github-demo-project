#!/bin/bash

# 设置错误即退出
set -e

echo "=== 1. 开始构建前端生产资源 ==="
cd frontend
npm i
npm run build
cd ..

# 创建前端嵌入目录并同步静态资源
mkdir -p web/html
rm -fr web/html/*
cp -R frontend/dist/* web/html/

echo "=== 2. 初始化发布目录 ==="
mkdir -p bin
TEMP_DIR="release_temp"
rm -rf "$TEMP_DIR"
mkdir -p "$TEMP_DIR/s-ui"

# 定义公共编译参数 (移除了在 macOS 交叉编译 Linux 时会引起冲突的 Naive 和 Musl 标签)
BUILD_TAGS="with_quic,with_grpc,with_utls,with_acme,with_gvisor,with_tailscale"
LDFLAGS="-w -s -extldflags \"-Wl,-no_warn_duplicate_libraries\""

build_and_pack() {
    local os=$1
    local arch=$2
    local output_name=$3

    echo ">>> 正在编译 ${os}/${arch} 二进制程序..."
    GOOS=$os GOARCH=$arch go build -ldflags "$LDFLAGS" -tags "$BUILD_TAGS" -o "$TEMP_DIR/s-ui/sui" main.go

    echo ">>> 正在组装发布包文件..."
    cp s-ui.sh "$TEMP_DIR/s-ui/"
    cp s-ui.service "$TEMP_DIR/s-ui/"
    
    # 保持底层 bin 目录结构 (如有必要，如为空则创建)
    mkdir -p "$TEMP_DIR/s-ui/bin"

    echo ">>> 正在打包为 $output_name..."
    cd "$TEMP_DIR"
    tar -zcvf "../bin/$output_name" s-ui
    cd ..

    # 清理二进制以防混淆
    rm -rf "$TEMP_DIR/s-ui/sui"
    rm -rf "$TEMP_DIR/s-ui/s-ui.sh"
    rm -rf "$TEMP_DIR/s-ui/s-ui.service"
}

# 1. 编译并打包 Linux amd64 (x86_64)
build_and_pack "linux" "amd64" "s-ui-linux-amd64.tar.gz"

# 2. 编译并打包 Linux arm64 (aarch64)
build_and_pack "linux" "arm64" "s-ui-linux-arm64.tar.gz"

echo "=== 3. 清理临时构建文件 ==="
rm -rf "$TEMP_DIR"

echo "=== 🎉 构建完成！生成的文件存放在 bin/ 目录中 ==="
ls -lh bin/
