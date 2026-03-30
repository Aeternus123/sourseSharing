#!/bin/bash

echo "生成Solidity合约绑定..."

# 创建构建目录
mkdir -p build

# 编译Solidity合约
echo "编译DataShare.sol..."
npx solc --bin --abi -o build/ contracts/DataShare.sol

# 生成Go绑定
echo "生成Go绑定文件..."
abigen --bin=build/DataShare.bin --abi=build/DataShare.abi --pkg=core --type=DataShare --out=core/datashare.go

# 检查是否成功
if [ $? -eq 0 ]; then
    echo "✅ 绑定文件生成成功: core/datashare.go"
else
    echo "❌ 绑定文件生成失败"
    exit 1
fi