# 数据共享区块链系统 (SourseSharing)

一个基于以太坊区块链的去中心化数据共享平台，支持文件上传、下载、交易和权限管理。

## 📋 项目简介

SourseSharing 是一个完整的区块链数据共享解决方案，包含：
- **智能合约**: 基于 Solidity 的数据共享合约
- **后端 API 服务**: Go 语言编写的 RESTful API 服务器
- **命令行客户端**: Windows 平台的命令行工具
- **本地区块链网络**: 基于 Hardhat 的测试环境

## 🏗️ 系统架构

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│    CLI 客户端   │────│   API 服务器    │────│   区块链网络    │
│  (Windows CLI)  │    │   (Go + Gin)    │    │  (Hardhat)     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                              │
                      ┌─────────────────┐
                      │   本地文件存储  │
                      │   (文件系统)    │
                      └─────────────────┘
```

## ✨ 核心功能

### 智能合约功能
- 用户注册与账户管理
- 文件上传与元数据存储
- 文件下载与交易机制
- 存储空间管理
- 收益统计与结算

### 后端服务功能
- 用户认证与授权
- 文件上传/下载处理
- 区块链交互接口
- 本地文件存储管理

### 客户端功能
- 用户登录/登出
- 文件上传/下载
- 账户信息查询
- 系统配置管理

## 🚀 快速开始

### 环境要求

- **Go 1.25.3+**
- **Node.js 16+**
- **Hardhat** (用于本地区块链)
- **Windows 系统** (客户端)

### 1. 安装依赖

```bash
# 安装 Go 依赖
go mod download

# 安装 Node.js 依赖
npm install
```

### 2. 配置环境变量

创建 `.env` 文件：

```env
# 服务器配置
PORT=8080
STORAGE_PATH=./storage

# 区块链配置
ETH_RPC_URL=http://localhost:8545
CONTRACT_ADDRESS=0x5FbDB2315678afecb367f032d93F642f64180aa3
ADMIN_PRIVATE_KEY=0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80

# 自动注册设置
SKIP_AUTO_REGISTER=false
```

### 3. 启动本地区块链

```bash
# 启动 Hardhat 本地网络
npx hardhat node
```

### 4. 部署智能合约

```bash
# 部署合约到本地网络
npx hardhat run deploy.js --network localhost
```

### 5. 启动 API 服务器

```bash
# 启动 Go API 服务器
go run main.go
```

### 6. 使用命令行客户端

```bash
# 编译客户端
cd cli && build.bat

# 配置服务器
sscli.exe config server http://localhost:8080

# 用户登录
sscli.exe login 0x70997970C51812dc3A010C7d01b50e0d17dc79C8

# 上传文件
sscli.exe upload ./example.pdf --share

# 下载文件
sscli.exe download 0x文件所有者地址 文件哈希
```

## 📁 项目结构

```
SourseSharing/
├── api/                 # API 服务器模块
│   └── server.go        # HTTP 服务器实现
├── cli/                 # 命令行客户端
│   ├── main.go          # 客户端主程序
│   ├── commands.go      # 命令定义
│   ├── auth.go          # 认证模块
│   └── config.go        # 配置管理
├── contracts/           # 智能合约
│   └── DataShare.sol    # 主合约文件
├── core/                # 核心业务逻辑
│   ├── ethereum.go      # 以太坊客户端
│   ├── datashare.go     # 数据共享逻辑
│   ├── storage.go       # 文件存储
│   └── models.go        # 数据模型
├── scripts/             # 部署脚本
│   ├── deploy.sh        # 部署脚本
│   └── generate_bindings.sh # 绑定生成
├── build/               # 编译输出
├── chaindata/           # 区块链数据
└── artifacts/           # 合约编译产物
```

## 🔧 核心组件说明

### 智能合约 (DataShare.sol)

主要数据结构：
- `FileInfo`: 文件信息（哈希、名称、大小、所有者等）
- `User`: 用户信息（余额、存储使用、收益统计等）

核心功能：
- 用户注册与激活
- 文件上传与下载
- 存储空间管理
- 交易结算

### API 服务器

基于 Gin 框架的 RESTful API：
- 用户认证接口
- 文件上传/下载接口
- 区块链交互接口
- 账户管理接口

### 命令行客户端

支持的命令：
- `config`: 配置管理
- `login/logout`: 用户登录/登出
- `upload/download`: 文件上传/下载
- `list`: 文件列表查看
- `info`: 用户信息查询

## ⚙️ 配置说明

### 服务器配置

配置文件位置：`./.env`

```env
# 基本配置
PORT=8080                    # 服务器端口
STORAGE_PATH=./storage       # 文件存储路径

# 区块链配置
ETH_RPC_URL=http://localhost:8545          # 以太坊节点URL
CONTRACT_ADDRESS=0x...                     # 合约地址
ADMIN_PRIVATE_KEY=0x...                    # 管理员私钥

# 功能开关
SKIP_AUTO_REGISTER=false                   # 是否跳过自动注册
```

### 客户端配置

Windows 客户端配置文件位置：
`C:\Users\[用户名]\AppData\Local\SourseSharing\config.json`

## 🔒 安全特性

- **私钥安全**: 私钥本地存储，不传输到服务器
- **文件加密**: 文件哈希验证，确保数据完整性
- **权限控制**: 基于区块链的权限管理
- **交易安全**: 智能合约确保交易透明可追溯

## 💰 经济模型

系统采用代币经济模型：
- **上传奖励**: 用户上传文件获得奖励
- **下载费用**: 下载文件需要支付费用
- **存储费用**: 超出免费额度需支付存储费
- **删除费用**: 删除文件需支付手续费

## 🧪 测试

### 智能合约测试

```bash
# 运行合约测试
npx hardhat test
```

### API 测试

使用 curl 或 Postman 测试 API 接口：

```bash
# 测试用户登录
curl -X POST http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{"address":"0x70997970C51812dc3A010C7d01b50e0d17dc79C8"}'

# 测试文件上传
curl -X POST http://localhost:8080/api/upload \
  -F "file=@example.pdf" \
  -F "address=0x70997970C51812dc3A010C7d01b50e0d17dc79C8"
```

## 🐛 故障排除

### 常见问题

1. **连接区块链失败**
   - 检查 Hardhat 节点是否启动
   - 确认 RPC URL 配置正确

2. **合约部署失败**
   - 检查合约编译是否成功
   - 确认管理员私钥有足够余额

3. **文件上传失败**
   - 检查存储目录权限
   - 确认用户已登录且账户激活

4. **客户端连接失败**
   - 检查服务器地址配置
   - 确认防火墙设置

## 📈 性能优化

- **缓存优化**: 使用内存缓存减少区块链查询
- **批量操作**: 支持批量文件操作
- **异步处理**: 文件上传下载异步处理
- **压缩传输**: 支持文件压缩传输

## 🤝 贡献指南

欢迎提交 Issue 和 Pull Request！

### 开发流程

1. Fork 项目
2. 创建功能分支
3. 提交更改
4. 推送到分支
5. 创建 Pull Request

### 代码规范

- Go 代码遵循标准格式
- Solidity 代码使用最新版本
- 添加必要的注释和文档
- 编写单元测试

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 🔗 相关链接

- [以太坊官方文档](https://ethereum.org/)
- [Hardhat 文档](https://hardhat.org/docs)
- [Go 语言文档](https://golang.org/doc/)
- [Gin 框架文档](https://gin-gonic.com/docs/)

## 📞 联系方式

如有问题或建议，请通过以下方式联系：
- 提交 GitHub Issue
- 发送邮件至项目维护者

---

**注意**: 本项目为教育演示用途，生产环境使用前请进行充分的安全测试和审计。