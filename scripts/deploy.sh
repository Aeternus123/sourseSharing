#!/bin/bash
# deploy.sh

# 启动以太坊私有链
echo "启动以太坊私有链..."
geth --datadir ./chaindata init genesis.json
geth --datadir ./chaindata --networkid 2023 --http --http.addr "0.0.0.0" --http.port 8545 --http.api "web3,eth,net,personal" --http.corsdomain "*" --mine --miner.threads 1 --miner.etherbase "0x7e5fde38f1233b8a6c4e8e5f6a7b8c9d0e1f2a3b" &

# 等待节点启动
sleep 5

# 编译和部署合约
echo "编译智能合约..."
solc --bin --abi contracts/DataShare.sol -o build/

echo "部署智能合约..."
node scripts/deploy.js

# 启动API服务
echo "启动API服务..."
go run main.go &

echo "部署完成！"
echo "API服务运行在: http://localhost:8080"
echo "以太坊节点运行在: http://localhost:8545"