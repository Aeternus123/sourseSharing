#!/bin/bash

# 数据共享区块链系统 - 服务器后台启动脚本

# 设置环境变量
export ETH_RPC_URL=http://localhost:8545
export ADMIN_PRIVATE_KEY=ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80
export CONTRACT_ADDRESS=0x5FbDB2315678afecb367f032d93F642f64180aa3
export STORAGE_PATH=./storage
export PORT=8080
export GIN_MODE=release

# 🔥 新增 Gas 配置
export GAS_PRICE=1000000000    # 1 Gwei
export GAS_LIMIT=1000000       # 100万 Gas
export CHAIN_ID=31337          # Hardhat 网络

# 配置文件
CONFIG_FILE="server.conf"
LOG_FILE="server.log"
PID_FILE="server.pid"

# 加载配置文件（如果存在）
load_config() {
    if [ -f "$CONFIG_FILE" ]; then
        source "$CONFIG_FILE"
        echo "📁 已加载配置文件: $CONFIG_FILE"
    fi
}

# 检查依赖
check_dependencies() {
    local deps=("go" "npx")
    for dep in "${deps[@]}"; do
        if ! command -v "$dep" &> /dev/null; then
            echo "❌ 缺少依赖: $dep"
            return 1
        fi
    done
    echo "✅ 所有依赖检查通过"
}

# 检查区块链节点
check_blockchain() {
    echo "🔗 检查区块链节点连接..."
    if curl -s $ETH_RPC_URL > /dev/null; then
        echo "✅ 区块链节点连接正常"
        
        # 检查合约是否可访问
        contract_addr=$(echo $CONTRACT_ADDRESS | cut -c3-) # 移除 0x 前缀
        if curl -s -X POST --data '{"jsonrpc":"2.0","method":"eth_getCode","params":["0x'$contract_addr'", "latest"],"id":1}' $ETH_RPC_URL | grep -q "0x"; then
            echo "✅ 合约可访问"
        else
            echo "⚠️  合约可能未部署或地址不正确"
        fi
    else
        echo "❌ 无法连接到区块链节点: $ETH_RPC_URL"
        echo "💡 请先运行: npx hardhat node"
        return 1
    fi
}

# 检查合约
check_contract() {
    echo "📝 检查合约状态..."
    if [ -z "$CONTRACT_ADDRESS" ]; then
        echo "❌ 未设置合约地址"
        return 1
    fi
    
    # 简单的合约地址格式检查
    if [[ ! $CONTRACT_ADDRESS =~ ^0x[0-9a-fA-F]{40}$ ]]; then
        echo "❌ 合约地址格式不正确: $CONTRACT_ADDRESS"
        return 1
    fi
    echo "✅ 合约地址格式正确: $CONTRACT_ADDRESS"
}

start_server() {
    echo "🚀 启动数据共享区块链服务器..."
    
    # 加载配置
    load_config
    
    # 检查依赖
    if ! check_dependencies; then
        return 1
    fi
    
    # 检查区块链节点
    if ! check_blockchain; then
        return 1
    fi
    
    # 检查合约
    if ! check_contract; then
        return 1
    fi
    
    # 检查是否已在运行
    if [ -f "$PID_FILE" ]; then
        PID=$(cat "$PID_FILE")
        if ps -p $PID > /dev/null 2>&1; then
            echo "❌ 服务器已在运行 (PID: $PID)"
            echo "💡 如需重启，请运行: $0 restart"
            return 1
        else
            # 清理陈旧的PID文件
            rm -f "$PID_FILE"
        fi
    fi
    
    # 创建必要的目录
    mkdir -p $STORAGE_PATH
    mkdir -p logs
    
    # 编译
    echo "🔨 编译服务器..."
    if ! go build -o soursesharing-server; then
        echo "❌ 编译失败"
        return 1
    fi
    
    # 显示配置信息
    echo "📋 服务器配置:"
    echo "   - 区块链节点: $ETH_RPC_URL"
    echo "   - 合约地址: $CONTRACT_ADDRESS"
    echo "   - Gas 价格: $GAS_PRICE wei"
    echo "   - Gas 限制: $GAS_LIMIT"
    echo "   - 存储路径: $STORAGE_PATH"
    echo "   - 服务端口: $PORT"
    
    # 启动服务器（后台运行）
    echo "🔄 启动服务器进程..."
    nohup ./soursesharing-server >> "$LOG_FILE" 2>&1 &
    SERVER_PID=$!
    
    # 等待进程启动
    sleep 2
    
    # 检查进程是否仍在运行
    if ! ps -p $SERVER_PID > /dev/null 2>&1; then
        echo "❌ 服务器启动失败，请检查日志: $LOG_FILE"
        tail -10 "$LOG_FILE"
        return 1
    fi
    
    # 保存 PID
    echo $SERVER_PID > "$PID_FILE"
    
    echo "✅ 服务器已启动 (PID: $SERVER_PID)"
    echo "📝 日志文件: $LOG_FILE"
    echo "🌐 访问地址: http://localhost:$PORT"
    echo "🔗 健康检查: http://localhost:$PORT/health"
    
    # 等待服务就绪
    echo "⏳ 等待服务就绪..."
    HEALTH_CHECK_URL="http://localhost:${PORT:-8080}/health"
    TIMEOUT=120  # 增加超时时间到120秒
    INTERVAL=10  # 每10秒检查一次
    elapsed=0
    
    while [ $elapsed -lt $TIMEOUT ]; do
        echo "⏱️  服务启动中... ($elapsed/$TIMEOUT秒)"
        
        # 检查进程是否仍在运行
        if ! ps -p $SERVER_PID > /dev/null; then
            echo "❌ 服务器进程已终止"
            echo "📋 建议查看日志: tail -50 $LOG_FILE"
            return 1
        fi
        
        # 显示最新日志
        echo "📝 最新日志:"
        tail -10 $LOG_FILE
        
        # 检查端口是否被占用
        echo "🔍 检查端口占用..."
        if command -v lsof > /dev/null; then
            PORT_INFO=$(lsof -ti:${PORT:-8080} || echo "端口未被占用")
            echo "   端口 ${PORT:-8080} 状态: $PORT_INFO"
        else
            echo "   警告: lsof 命令不可用，无法检查端口占用"
        fi
        
        # 尝试访问健康检查端点，获取详细响应
        echo "🔄 尝试健康检查..."
        HTTP_RESPONSE=$(curl -s -w "\n%{http_code}" $HEALTH_CHECK_URL 2>/dev/null)
        HTTP_STATUS=$(echo "$HTTP_RESPONSE" | tail -n 1)
        HTTP_BODY=$(echo "$HTTP_RESPONSE" | head -n -1)
        
        echo "   健康检查状态码: $HTTP_STATUS"
        echo "   健康检查响应: $HTTP_BODY"
        
        if [ "$HTTP_STATUS" = "200" ]; then
            echo "✅ 服务已就绪!"
            echo "📋 服务信息:"
            echo "  - 服务地址: http://localhost:${PORT:-8080}"
            echo "  - 健康检查: $HEALTH_CHECK_URL"
            echo "  - 日志文件: $LOG_FILE"
            return 0
        fi
        
        sleep $INTERVAL
        elapsed=$((elapsed + INTERVAL))
    done
    
    # 超时处理
    echo "⚠️  服务器启动超时（$TIMEOUT秒），但进程仍在运行"
    echo "📋 详细诊断信息："
    
    # 检查区块链节点连接
    echo "1. 区块链节点连接检查:"
    if curl -s http://localhost:8545 > /dev/null; then
        echo "   ✅ 区块链节点可访问"
    else
        echo "   ❌ 区块链节点不可访问，请确保Hardhat节点正在运行"
    fi
    
    # 检查服务器进程
    echo "2. 服务器进程状态:"
    if ps -p $SERVER_PID > /dev/null; then
        echo "   ✅ 服务器进程正在运行 (PID: $SERVER_PID)"
    else
        echo "   ❌ 服务器进程未运行"
    fi
    
    # 检查端口监听
    echo "3. 端口监听状态:"
    if command -v netstat > /dev/null; then
        PORT_LISTEN=$(netstat -tulpn 2>/dev/null | grep ":${PORT:-8080} " || echo "未监听")
        echo "   端口监听情况: $PORT_LISTEN"
    elif command -v ss > /dev/null; then
        PORT_LISTEN=$(ss -tulpn 2>/dev/null | grep ":${PORT:-8080} " || echo "未监听")
        echo "   端口监听情况: $PORT_LISTEN"
    else
        echo "   警告: netstat 和 ss 命令都不可用，无法检查端口监听"
    fi
    
    # 建议操作
    echo "📋 建议操作："
    echo "1. 查看详细日志: tail -50 $LOG_FILE"
    echo "2. 手动测试健康检查: curl -v $HEALTH_CHECK_URL"
    echo "3. 检查服务器是否正确绑定端口: grep -i '绑定' $LOG_FILE"
    echo "4. 尝试重启Hardhat节点和服务器进程"
    return 0
}

stop_server() {
    echo "🛑 停止服务器..."
    
    if [ -f "$PID_FILE" ]; then
        PID=$(cat "$PID_FILE")
        if ps -p $PID > /dev/null 2>&1; then
            echo "⏳ 正在停止服务器 (PID: $PID)..."
            kill $PID
            
            # 等待进程结束
            for i in {1..10}; do
                if ! ps -p $PID > /dev/null 2>&1; then
                    break
                fi
                sleep 1
            done
            
            if ps -p $PID > /dev/null 2>&1; then
                echo "⚠️  强制杀死进程..."
                kill -9 $PID
            fi
            
            echo "✅ 服务器已停止 (PID: $PID)"
        else
            echo "⚠️  服务器未运行"
        fi
        rm -f "$PID_FILE"
    else
        echo "⚠️  未找到PID文件，服务器可能未运行"
        
        # 尝试通过端口查找进程
        PORT_PID=$(lsof -ti:$PORT 2>/dev/null)
        if [ ! -z "$PORT_PID" ]; then
            echo "🔍 发现占用端口 $PORT 的进程: $PORT_PID"
            read -p "是否杀死这些进程? [y/N] " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                kill -9 $PORT_PID
                echo "✅ 已杀死进程: $PORT_PID"
            fi
        fi
    fi
}

status_server() {
    echo "📊 服务器状态检查..."
    
    # 检查进程
    if [ -f "$PID_FILE" ]; then
        PID=$(cat "$PID_FILE")
        if ps -p $PID > /dev/null 2>&1; then
            echo "✅ 服务器运行中 (PID: $PID)"
            
            # 检查端口
            if lsof -ti:$PORT > /dev/null 2>&1; then
                echo "✅ 端口 $PORT 监听正常"
                
                # 检查健康端点
                if curl -s http://localhost:$PORT/health > /dev/null 2>&1; then
                    echo "✅ 健康检查通过"
                else
                    echo "⚠️  健康检查失败"
                fi
            else
                echo "❌ 端口 $PORT 未监听"
            fi
            
            echo "📝 查看日志: tail -f $LOG_FILE"
            echo "🌐 访问地址: http://localhost:$PORT"
        else
            echo "❌ PID文件存在但进程未运行"
            rm -f "$PID_FILE"
        fi
    else
        echo "❌ 服务器未运行"
    fi
    
    # 检查区块链连接
    echo ""
    check_blockchain
}

show_logs() {
    if [ -f "$LOG_FILE" ]; then
        tail -f "$LOG_FILE"
    else
        echo "❌ 日志文件不存在: $LOG_FILE"
    fi
}

create_config() {
    cat > "$CONFIG_FILE" << EOF
# 数据共享服务器配置
ETH_RPC_URL="http://localhost:8545"
ADMIN_PRIVATE_KEY="ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
CONTRACT_ADDRESS="0x5FbDB2315678afecb367f032d93F642f64180aa3"
STORAGE_PATH="./storage"
PORT="8080"
GIN_MODE="release"

# Gas 配置
GAS_PRICE="1000000000"
GAS_LIMIT="1000000"
CHAIN_ID="31337"
EOF
    echo "✅ 配置文件已创建: $CONFIG_FILE"
    echo "💡 请根据实际情况修改配置"
}

case "$1" in
    start)
        start_server
        ;;
    stop)
        stop_server
        ;;
    restart)
        stop_server
        sleep 2
        start_server
        ;;
    status)
        status_server
        ;;
    logs)
        show_logs
        ;;
    config)
        create_config
        ;;
    *)
        echo "用法: $0 {start|stop|restart|status|logs|config}"
        echo ""
        echo "命令说明:"
        echo "  start   启动服务器"
        echo "  stop    停止服务器"
        echo "  restart 重启服务器"
        echo "  status  查看服务器状态"
        echo "  logs    查看服务器日志"
        echo "  config  创建配置文件"
        exit 1
        ;;
esac