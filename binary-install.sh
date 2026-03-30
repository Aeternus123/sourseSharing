#!/bin/bash

set -e  # 遇到错误立即退出

echo "开始安装区块链开发环境（修复版）..."
echo "当前目录: $(pwd)"

# 创建安装目录
INSTALL_DIR="$HOME/blockchain-tools"
mkdir -p $INSTALL_DIR
cd $INSTALL_DIR

# 1. 安装Geth
echo "=== 安装 Geth ==="
if ! command -v geth &> /dev/null; then
    echo "下载Geth..."
    
    # 尝试多个下载源
    if ! wget -q --timeout=30 https://github.com/ethereum/go-ethereum/releases/download/v1.13.15/geth-linux-amd64-1.13.15-6dc8946e.tar.gz; then
        echo "第一个源失败，尝试备用源..."
        wget -q --timeout=30 https://gethstore.blob.core.windows.net/builds/geth-linux-amd64-1.13.15-6dc8946e.tar.gz || {
            echo "下载失败，请检查网络连接"
            exit 1
        }
    fi
    
    echo "解压Geth..."
    tar -xzf geth-linux-amd64-1.13.15-6dc8946e.tar.gz
    sudo cp geth-linux-amd64-1.13.15-6dc8946e/geth /usr/local/bin/
    sudo chmod +x /usr/local/bin/geth
    rm -rf geth-linux-amd64-1.13.15-6dc8946e*
    echo "✅ Geth 安装完成"
else
    echo "✅ Geth 已安装"
fi

# 3. 安装Node.js（可选，用于solc）
echo "=== 安装 Node.js ==="
if ! command -v node &> /dev/null; then
    echo "下载Node.js..."
    wget -q https://nodejs.org/dist/v18.19.0/node-v18.19.0-linux-x64.tar.xz
    
    echo "安装Node.js..."
    sudo tar -xJf node-v18.19.0-linux-x64.tar.xz -C /usr/local --strip-components=1
    rm node-v18.19.0-linux-x64.tar.xz
    echo "✅ Node.js 安装完成"
else
    echo "✅ Node.js 已安装"
fi

# 4. 配置环境变量
echo "=== 配置环境变量 ==="
cat >> ~/.bashrc << 'EOF'

# Blockchain Development Environment
export PATH=$PATH:/usr/local/go/bin
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin
export PATH=$PATH:/usr/local/bin
EOF

# 立即生效
export PATH=$PATH:/usr/local/go/bin
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin

# 5. 安装Solidity编译器（可选）
echo "=== 安装 Solidity 编译器 ==="
if command -v npm &> /dev/null; then
    sudo npm install -g solc
    echo "✅ Solidity 编译器安装完成"
else
    echo "⚠️  npm不可用，跳过Solidity安装"
fi

# 6. 安装abigen
echo "=== 安装 abigen ==="
if command -v go &> /dev/null; then
    go install github.com/ethereum/go-ethereum/cmd/abigen@latest
    echo "✅ abigen 安装完成"
else
    echo "❌ Go不可用，无法安装abigen"
fi

# 7. 最终验证
echo ""
echo "=== 最终验证 ==="
command -v geth >/dev/null && echo "✅ Geth: $(geth version | head -1)" || echo "❌ Geth 未安装"
command -v go >/dev/null && echo "✅ Go: $(go version)" || echo "❌ Go 未安装"
command -v node >/dev/null && echo "✅ Node.js: $(node --version)" || echo "❌ Node.js 未安装"
command -v solc >/dev/null && echo "✅ Solidity: $(solc --version | head -1)" || echo "❌ Solidity 未安装"
[ -f "$GOPATH/bin/abigen" ] && echo "✅ abigen 已安装" || echo "❌ abigen 未安装"

echo ""
echo "安装完成！如果环境变量未生效，请运行: source ~/.bashrc"