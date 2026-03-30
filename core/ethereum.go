package core

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type EthereumClient struct {
	client          *ethclient.Client
	auth            *bind.TransactOpts
	contract        *DataShare
	contractAddress common.Address
	adminAddress    common.Address
	ledger          *Ledger
}

// 移除重复的类型声明，使用 models.go 中的定义

func NewEthereumClient() (*EthereumClient, error) {
	rpcURL := os.Getenv("ETH_RPC_URL")
	if rpcURL == "" {
		rpcURL = "http://localhost:8545"
	}

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("连接以太坊节点失败: %v", err)
	}

	privateKeyHex := os.Getenv("ADMIN_PRIVATE_KEY")
	if privateKeyHex == "" {
		privateKeyHex = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	}

	// 移除可能的0x前缀
	privateKeyHex = strings.TrimPrefix(privateKeyHex, "0x")
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("加载管理员私钥失败: %v", err)
	}

	chainID, err := client.ChainID(context.Background())
	if err != nil {
		chainID = big.NewInt(31337) // Hardhat 默认链ID
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		return nil, fmt.Errorf("创建交易认证失败: %v", err)
	}

	// 设置Gas配置
	gasPrice, gasLimit := getGasConfig(client)
	auth.GasPrice = gasPrice
	auth.GasLimit = gasLimit

	log.Printf("Gas 配置 - 价格: %d wei, 限制: %d", auth.GasPrice, auth.GasLimit)

	adminAddress := crypto.PubkeyToAddress(privateKey.PublicKey)

	contractAddressHex := os.Getenv("CONTRACT_ADDRESS")
	var contractAddress common.Address
	var contract *DataShare

	if contractAddressHex == "" {
		maxUsers := big.NewInt(50) // 设置更大的用户限制
		contractAddress, _, contract, err = DeployDataShare(auth, client, maxUsers)
		if err != nil {
			return nil, fmt.Errorf("部署合约失败: %v", err)
		}
		log.Printf("✅ 新合约已部署，地址: %s，最大用户数: %d", contractAddress.Hex(), maxUsers)
	} else {
		contractAddress = common.HexToAddress(contractAddressHex)
		contract, err = NewDataShare(contractAddress, client)
		if err != nil {
			return nil, fmt.Errorf("加载合约失败: %v", err)
		}
		log.Printf("✅ 合约已加载，地址: %s", contractAddress.Hex())
	}

	// 创建账本目录
	ledgerPath := os.Getenv("LEDGER_PATH")
	if ledgerPath == "" {
		ledgerPath = "./ledger"
	}

	// 创建账本管理器
	ledger := NewLedger(client, contract, ledgerPath)
	if err := ledger.Init(); err != nil {
		log.Printf("警告: 账本初始化失败: %v", err)
	}

	// 启动账本事件监听
	if err := ledger.Start(); err != nil {
		log.Printf("警告: 账本事件监听启动失败: %v", err)
	}

	ethClient := &EthereumClient{
		client:          client,
		auth:            auth,
		contract:        contract,
		contractAddress: contractAddress,
		adminAddress:    adminAddress,
		ledger:          ledger,
	}

	// 创建一个goroutine异步处理自动注册，避免阻塞主流程
	go func() {
		// 短暂延迟，确保服务已经启动
		time.Sleep(5 * time.Second)

		// 自动注册Hardhat测试账户
		if err := ethClient.autoRegisterHardhatAccounts(); err != nil {
			log.Printf("自动注册测试账户失败: %v", err)
		} else {
			log.Printf("✅ 自动注册Hardhat测试账户完成")
		}
	}()

	return ethClient, nil
}

func (ec *EthereumClient) autoRegisterHardhatAccounts() error {
	log.Printf("开始自动注册Hardhat测试账户...")

	// 检查是否需要自动注册
	skipAutoRegister := os.Getenv("SKIP_AUTO_REGISTER") 
	if skipAutoRegister == "true" {
		log.Printf("✅ 自动注册已跳过")
		return nil
	}

	// 完整的Hardhat默认测试账户列表（20个账户）
	hardhatAccounts := []string{
		"0x70997970C51812dc3A010C7d01b50e0d17dc79C8", // 账户1
		"0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC", // 账户2
		"0x90F79bf6EB2c4f870365E785982E1f101E93b906", // 账户3
		"0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65", // 账户4
		"0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc", // 账户5
		"0x976EA74026E726554dB657fA54763abd0C3a0aa9", // 账户6
		"0x14dC79964da2C08b23698B3D3cc7Ca32193d9955", // 账户7
		"0x23618e81E3f5cdF7f54C3d65f7FBc0aBf5B21E8f", // 账户8
		"0xa0Ee7A142d267C1f36714E4a8F75612F20a79720", // 账户9
		"0xBcd4042DE499D14e55001CcbB24a551F3b954096", // 账户10
		"0x71bE63f3384f5fb98995898A86B02Fb2426c5788", // 账户11
		"0xFABB0ac9d68B0B445fB7357272Ff202C5651694a", // 账户12
		"0x1CBd3b2770909D4e10f157cABC84C7264073C9Ec", // 账户13
		"0xdF3e18d64BC6A983f673Ab319CCaE4f1a57C7097", // 账户14
		"0xcd3B766CCDd6AE721141F452C550Ca635964ce71", // 账户15
		"0x2546BcD3c84621e976D8185a91A922aE77ECEc30", // 账户16
		"0xbDA5747bFD65F08deb54cb465eB87D40e51B197E", // 账户17
		"0xdD2FD4581271e230360230F9337D5c0430Bf44C0", // 账户18
		"0x8626f6940E2eb28930eFb4CeF49B2d1F2C9C1199", // 账户19
		"0xF09Fb4818Df43D378a3B16426C53c1053d1Ce009", // 账户20
	}

	// 开启注册功能
	if _, err := ec.contract.SetRegistrationStatus(ec.auth, true); err != nil {
		return fmt.Errorf("开启注册功能失败: %v", err)
	}

	registeredCount := 0
	for _, account := range hardhatAccounts {
		address := common.HexToAddress(account)

		// 检查用户是否已注册
		isRegistered, err := ec.IsUserRegistered(address)
		if err == nil && isRegistered {
			log.Printf("用户已存在: %s", account)
			continue
		}

		// 注册用户
		if _, err := ec.contract.RegisterUser(ec.auth, address); err != nil {
			log.Printf("注册用户失败 %s: %v", account, err)
			continue
		}

		log.Printf("✅ 自动注册用户: %s", account)
		registeredCount++

		// 增加短暂延迟，避免交易拥堵
		time.Sleep(500 * time.Millisecond)
	}

	// 关闭注册功能（可选）
	if _, err := ec.contract.SetRegistrationStatus(ec.auth, false); err != nil {
		log.Printf("关闭注册功能失败: %v", err)
	}

	log.Printf("自动注册完成，成功注册 %d 个用户", registeredCount)
	return nil
}

// getGasConfig 获取 Gas 配置
func getGasConfig(client *ethclient.Client) (*big.Int, uint64) {
	// 尝试从环境变量读取
	gasPriceStr := os.Getenv("GAS_PRICE")
	gasLimitStr := os.Getenv("GAS_LIMIT")

	var gasPrice *big.Int
	var gasLimit uint64

	// 设置 Gas 价格
	if gasPriceStr != "" {
		gasPrice = new(big.Int)
		gasPrice.SetString(gasPriceStr, 10)
	} else {
		// 尝试获取网络建议的 Gas 价格
		suggestedPrice, err := client.SuggestGasPrice(context.Background())
		if err != nil {
			// 使用合理的默认值
			gasPrice = big.NewInt(1000000000) // 1 Gwei
		} else {
			gasPrice = suggestedPrice
		}
	}

	// 设置 Gas 限制
	if gasLimitStr != "" {
		fmt.Sscanf(gasLimitStr, "%d", &gasLimit)
	} else {
		gasLimit = 1000000 // 100万 Gas 默认值
	}

	return gasPrice, gasLimit
}

// 🔥 创建用户特定的交易选项
func (ec *EthereumClient) createUserTransactOpts() *bind.TransactOpts {
	// 复制管理员的配置，但使用不同的 From 地址
	return &bind.TransactOpts{
		From:     ec.auth.From,
		Signer:   ec.auth.Signer,
		GasPrice: ec.auth.GasPrice,
		GasLimit: ec.auth.GasLimit,
		Value:    big.NewInt(0),
		Context:  context.Background(),
	}
}

// 🔥 为文件操作创建特定的交易选项
func (ec *EthereumClient) createFileTransactOpts() *bind.TransactOpts {
	opts := ec.createUserTransactOpts()
	opts.GasLimit = 500000 // 文件操作需要更多 Gas
	return opts
}

// 🔥 为简单操作创建交易选项
func (ec *EthereumClient) createSimpleTransactOpts() *bind.TransactOpts {
	opts := ec.createUserTransactOpts()
	opts.GasLimit = 200000 // 简单操作需要较少 Gas
	return opts
}

func (ec *EthereumClient) RegisterUser(userAddress common.Address) (*TransactionResponse, error) {
	// 🔥 先检查余额
	balance, err := ec.client.BalanceAt(context.Background(), ec.auth.From, nil)
	if err != nil {
		return nil, fmt.Errorf("检查余额失败: %v", err)
	}

	log.Printf("交易发送者余额: %s WEI", balance.String())
	log.Printf("交易发送者地址: %s", ec.auth.From.Hex())

	// 估算Gas费用
	gasPrice := ec.auth.GasPrice
	gasLimit := ec.auth.GasLimit
	gasCost := new(big.Int).Mul(gasPrice, new(big.Int).SetUint64(gasLimit))

	log.Printf("预估Gas费用: 价格=%s, 限制=%d, 总费用=%s",
		gasPrice.String(), gasLimit, gasCost.String())

	if balance.Cmp(gasCost) < 0 {
		return nil, fmt.Errorf("余额不足。余额: %s WEI, 需要: %s WEI", balance.String(), gasCost.String())
	}

	opts := ec.createSimpleTransactOpts()

	tx, err := ec.contract.RegisterUser(opts, userAddress)
	if err != nil {
		return nil, fmt.Errorf("注册用户失败: %v", err)
	}

	log.Printf("用户注册交易已发送，用户: %s, 交易哈希: %s", userAddress.Hex(), tx.Hash().Hex())

	return &TransactionResponse{
		TxHash:    tx.Hash().Hex(),
		Status:    "pending",
		Message:   "用户注册交易已发送",
		Timestamp: time.Now().Unix(),
	}, nil
}

func (ec *EthereumClient) BatchRegisterUsers(userAddresses []common.Address) (*TransactionResponse, error) {
	opts := ec.createSimpleTransactOpts()
	opts.GasLimit = 300000 // 批量操作需要更多 Gas

	tx, err := ec.contract.BatchRegisterUsers(opts, userAddresses)
	if err != nil {
		return nil, fmt.Errorf("批量注册用户失败: %v", err)
	}

	log.Printf("批量用户注册交易已发送，用户数量: %d, 交易哈希: %s", len(userAddresses), tx.Hash().Hex())

	return &TransactionResponse{
		TxHash:    tx.Hash().Hex(),
		Status:    "pending",
		Message:   "批量用户注册交易已发送",
		Timestamp: time.Now().Unix(),
	}, nil
}

func (ec *EthereumClient) SetRegistrationStatus(open bool) (*TransactionResponse, error) {
	opts := ec.createSimpleTransactOpts()

	tx, err := ec.contract.SetRegistrationStatus(opts, open)
	if err != nil {
		return nil, fmt.Errorf("设置注册状态失败: %v", err)
	}

	log.Printf("注册状态更新交易已发送，开放注册: %v, 交易哈希: %s", open, tx.Hash().Hex())

	return &TransactionResponse{
		TxHash:    tx.Hash().Hex(),
		Status:    "pending",
		Message:   "注册状态更新交易已发送",
		Timestamp: time.Now().Unix(),
	}, nil
}

func (ec *EthereumClient) GetSystemInfo() (*SystemInfo, error) {
	result, err := ec.contract.GetSystemInfo(&bind.CallOpts{})
	if err != nil {
		return nil, fmt.Errorf("获取系统信息失败: %v", err)
	}

	return &SystemInfo{
		MaxUsers:         result.MaxUsers.Uint64(),
		CurrentUserCount: result.CurrentUserCount.Uint64(),
		RegistrationOpen: result.RegistrationOpen,
		AdminAddress:     result.Admin.Hex(),
	}, nil
}

func (ec *EthereumClient) IsAdmin(address common.Address) bool {
	// 从区块链合约获取实际的管理员地址
	systemInfo, err := ec.GetSystemInfo()
	if err != nil {
		log.Printf("获取系统信息失败: %v", err)
		return false
	}

	contractAdmin := common.HexToAddress(systemInfo.AdminAddress)
	log.Printf("检查管理员权限: 输入地址=%s, 合约管理员=%s", address.Hex(), contractAdmin.Hex())

	return address.Hex() == contractAdmin.Hex()
}

func (ec *EthereumClient) GetAdminAddress() string {
	return ec.adminAddress.Hex()
}

func (ec *EthereumClient) IsUserRegistered(userAddress common.Address) (bool, error) {
	result, err := ec.contract.GetUserInfo(&bind.CallOpts{}, userAddress)
	if err != nil {
		return false, err
	}
	return result.IsActive, nil
}

func (ec *EthereumClient) InitializeUser(userAddress common.Address) (*TransactionResponse, error) {
	// 使用 RegisterUser 方法
	opts := ec.createSimpleTransactOpts()

	tx, err := ec.contract.RegisterUser(opts, userAddress)
	if err != nil {
		return nil, fmt.Errorf("初始化用户失败: %v", err)
	}

	log.Printf("用户初始化交易已发送，用户: %s, 交易哈希: %s", userAddress.Hex(), tx.Hash().Hex())

	return &TransactionResponse{
		TxHash:    tx.Hash().Hex(),
		Status:    "pending",
		Message:   "用户初始化交易已发送",
		Timestamp: time.Now().Unix(),
	}, nil
}

func (ec *EthereumClient) UploadFile(userAddress common.Address, fileHash, fileName string, fileSize int64, share bool, userPrivateKeyHex string) (*TransactionResponse, error) {
	// 检查用户是否已在区块链上激活
	isRegistered, err := ec.IsUserRegistered(userAddress)
	if err != nil {
		return nil, fmt.Errorf("检查用户状态失败: %v", err)
	}
	if !isRegistered {
		return nil, fmt.Errorf("用户未在区块链上注册或未激活")
	}

	// 使用用户的私钥创建交易选项
	log.Printf("准备上传文件，使用用户地址: %s 的私钥创建交易选项", userAddress.Hex())

	// 解析用户私钥
	// 移除可能的0x前缀
	userPrivateKeyHex = strings.TrimPrefix(userPrivateKeyHex, "0x")
	userPrivateKey, err := crypto.HexToECDSA(userPrivateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("解析用户私钥失败: %v", err)
	}

	// 获取链ID
	chainID, err := ec.client.ChainID(context.Background())
	if err != nil {
		chainID = big.NewInt(31337) // Hardhat 默认链ID
	}

	// 创建用户交易选项
	userAuth, err := bind.NewKeyedTransactorWithChainID(userPrivateKey, chainID)
	if err != nil {
		return nil, fmt.Errorf("创建用户交易认证失败: %v", err)
	}

	// 设置Gas配置
	gasPrice, _ := getGasConfig(ec.client)
	userAuth.GasPrice = gasPrice
	userAuth.GasLimit = 500000 // 文件操作需要更多 Gas
	userAuth.Value = big.NewInt(0)
	userAuth.Context = context.Background()

	// 检查余额
	balance, err := ec.client.BalanceAt(context.Background(), userAddress, nil)
	if err != nil {
		return nil, fmt.Errorf("检查用户余额失败: %v", err)
	}

	gasCost := new(big.Int).Mul(gasPrice, new(big.Int).SetUint64(userAuth.GasLimit))
	if balance.Cmp(gasCost) < 0 {
		return nil, fmt.Errorf("用户余额不足。余额: %s WEI, 需要: %s WEI", balance.String(), gasCost.String())
	}

	// 记录交易信息用于调试
	log.Printf("使用用户地址 %s 发送文件上传交易", userAuth.From.Hex())

	// 直接调用用户的uploadFile方法，使用用户自己的私钥签名
	tx, err := ec.contract.UploadFile(userAuth, fileHash, fileName, big.NewInt(fileSize), share)
	if err != nil {
		log.Printf("文件上传交易失败: %v", err)
		return nil, fmt.Errorf("上传文件记录失败: %v", err)
	}

	log.Printf("文件上传交易已发送，文件: %s, 交易哈希: %s, Gas 限制: %d", fileName, tx.Hash().Hex(), userAuth.GasLimit)

	return &TransactionResponse{
		TxHash:    tx.Hash().Hex(),
		Status:    "pending",
		Message:   "文件上传记录交易已发送",
		Timestamp: time.Now().Unix(),
	}, nil
}

func (ec *EthereumClient) DownloadFile(userAddress common.Address, fileHash string, userPrivateKeyHex string) (*TransactionResponse, error) {
	// 解析用户私钥
	// 移除可能的0x前缀
	userPrivateKeyHex = strings.TrimPrefix(userPrivateKeyHex, "0x")
	userPrivateKey, err := crypto.HexToECDSA(userPrivateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("解析用户私钥失败: %v", err)
	}

	// 获取链ID
	chainID, err := ec.client.ChainID(context.Background())
	if err != nil {
		chainID = big.NewInt(31337) // Hardhat 默认链ID
	}

	// 创建用户交易选项
	userAuth, err := bind.NewKeyedTransactorWithChainID(userPrivateKey, chainID)
	if err != nil {
		return nil, fmt.Errorf("创建用户交易认证失败: %v", err)
	}

	// 设置Gas配置
	gasPrice, _ := getGasConfig(ec.client)
	userAuth.GasPrice = gasPrice
	userAuth.GasLimit = 300000 // 下载操作Gas限制
	userAuth.Value = big.NewInt(0)
	userAuth.Context = context.Background()

	// 检查余额
	balance, err := ec.client.BalanceAt(context.Background(), userAddress, nil)
	if err != nil {
		return nil, fmt.Errorf("检查用户余额失败: %v", err)
	}

	gasCost := new(big.Int).Mul(gasPrice, new(big.Int).SetUint64(userAuth.GasLimit))
	if balance.Cmp(gasCost) < 0 {
		return nil, fmt.Errorf("用户余额不足。余额: %s WEI, 需要: %s WEI", balance.String(), gasCost.String())
	}

	tx, err := ec.contract.DownloadFile(userAuth, fileHash)
	if err != nil {
		return nil, fmt.Errorf("下载文件记录失败: %v", err)
	}

	log.Printf("文件下载交易已发送，文件哈希: %s, 交易哈希: %s, 用户地址: %s", fileHash, tx.Hash().Hex(), userAddress.Hex())

	return &TransactionResponse{
		TxHash:    tx.Hash().Hex(),
		Status:    "pending",
		Message:   "文件下载记录交易已发送",
		Timestamp: time.Now().Unix(),
	}, nil
}

func (ec *EthereumClient) GetUserInfo(userAddress common.Address) (*UserInfo, error) {
	result, err := ec.contract.GetUserInfo(&bind.CallOpts{}, userAddress)
	if err != nil {
		return nil, fmt.Errorf("获取用户信息失败: %v", err)
	}

	return &UserInfo{
		Address:      userAddress.Hex(),
		Balance:      result.Balance,
		UsedStorage:  result.UsedStorage.Uint64(),
		MaxStorage:   result.MaxStorage.Uint64(),
		TotalEarned:  result.TotalEarned,
		TotalSpent:   result.TotalSpent,
		IsActive:     result.IsActive,
		RegisterTime: result.RegisterTime.Uint64(),
	}, nil
}

func (ec *EthereumClient) GetFileInfo(fileHash string) (*BlockchainFileInfo, error) {
	result, err := ec.contract.GetFileInfo(&bind.CallOpts{}, fileHash)
	if err != nil {
		return nil, fmt.Errorf("获取文件信息失败: %v", err)
	}

	return &BlockchainFileInfo{
		FileHash:      fileHash,
		FileName:      result.FileName,
		FileSize:      result.FileSize.Uint64(),
		Owner:         result.Owner.Hex(),
		UploadTime:    result.UploadTime.Uint64(),
		DownloadPrice: result.DownloadPrice,
		IsShared:      result.IsShared,
		DownloadCount: result.DownloadCount.Uint64(),
	}, nil
}

func (ec *EthereumClient) GetUserFiles(userAddress common.Address) ([]string, error) {
	files, err := ec.contract.GetUserFiles(&bind.CallOpts{}, userAddress)
	if err != nil {
		return nil, fmt.Errorf("获取用户文件列表失败: %v", err)
	}
	return files, nil
}

func (ec *EthereumClient) GetContractAddress() string {
	return ec.contractAddress.Hex()
}

// 🔥 检查交易状态
func (ec *EthereumClient) CheckTransactionStatus(txHash common.Hash) (string, error) {
	receipt, err := ec.client.TransactionReceipt(context.Background(), txHash)
	if err != nil {
		return "pending", nil // 交易可能还在等待中
	}

	if receipt.Status == 1 {
		return "success", nil
	} else {
		return "failed", nil
	}
}

// 🔥 获取最新区块号
func (ec *EthereumClient) GetBlockNumber() (uint64, error) {
	header, err := ec.client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		return 0, err
	}
	return header.Number.Uint64(), nil
}

// 🔥 获取网络信息
func (ec *EthereumClient) GetNetworkInfo() (string, *big.Int, error) {
	chainID, err := ec.client.ChainID(context.Background())
	if err != nil {
		return "", nil, err
	}

	networkName := "Unknown"
	switch chainID.Uint64() {
	case 1:
		networkName = "Mainnet"
	case 5:
		networkName = "Goerli"
	case 11155111:
		networkName = "Sepolia"
	case 31337:
		networkName = "Hardhat"
	default:
		networkName = fmt.Sprintf("Chain %d", chainID)
	}

	return networkName, chainID, nil
}

// 🔥 获取账户余额
func (ec *EthereumClient) GetBalance(address common.Address) (*big.Int, error) {
	balance, err := ec.client.BalanceAt(context.Background(), address, nil)
	if err != nil {
		return nil, err
	}
	return balance, nil
}

// GetLedger 获取账本管理器
func (ec *EthereumClient) GetLedger() *Ledger {
	return ec.ledger
}

// 🔥 获取共享文件列表
func (ec *EthereumClient) GetSharedFiles() ([]*BlockchainFileInfo, error) {
	log.Println("🔍 开始获取共享文件列表...")
	
	// 使用Hardhat测试账户列表作为基础用户集
	hardhatAccounts := []string{
		"0x70997970C51812dc3A010C7d01b50e0d17dc79C8", // 账户1
		"0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC", // 账户2
		"0x90F79bf6EB2c4f870365E785982E1f101E93b906", // 账户3
		"0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65", // 账户4
		"0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc", // 账户5
		"0x976EA74026E726554dB657fA54763abd0C3a0aa9", // 账户6
		"0x14dC79964da2C08b23698B3D3cc7Ca32193d9955", // 账户7
		"0x23618e81E3f5cdF7f54C3d65f7FBc0aBf5B21E8f", // 账户8
		"0xa0Ee7A142d267C1f36714E4a8F75612F20a79720", // 账户9
		"0xBcd4042DE499D14e55001CcbB24a551F3b954096", // 账户10
	}
	
	var sharedFiles []*BlockchainFileInfo
	processedFiles := make(map[string]bool)
	
	// 遍历每个用户，获取他们的文件并检查是否共享
	for _, accountStr := range hardhatAccounts {
		userAddress := common.HexToAddress(accountStr)
		
		// 检查用户是否已注册
		isRegistered, err := ec.IsUserRegistered(userAddress)
		if err != nil || !isRegistered {
			continue // 跳过未注册的用户
		}
		
		// 获取用户的所有文件
		fileHashes, err := ec.GetUserFiles(userAddress)
		if err != nil {
			log.Printf("获取用户 %s 的文件列表失败: %v", accountStr, err)
			continue
		}
		
		// 检查每个文件是否为共享文件
		for _, fileHash := range fileHashes {
			// 避免处理重复文件
			if processedFiles[fileHash] {
				continue
			}
			processedFiles[fileHash] = true
			
			// 获取文件信息
			fileInfo, err := ec.GetFileInfo(fileHash)
			if err != nil {
				log.Printf("获取文件 %s 信息失败: %v", fileHash, err)
				continue
			}
			
			// 如果文件是共享的，添加到结果列表
			if fileInfo.IsShared {
				sharedFiles = append(sharedFiles, fileInfo)
				log.Printf("✅ 找到共享文件: %s (所有者: %s)", fileInfo.FileName, fileInfo.Owner)
			}
		}
	}
	
	log.Printf("📋 共享文件列表获取完成，共找到 %d 个共享文件", len(sharedFiles))
	return sharedFiles, nil
}

// 🔥 等待交易确认（可选使用）
func (ec *EthereumClient) WaitForTransaction(txHash common.Hash, timeout time.Duration) (*TransactionResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	receipt, err := bind.WaitMined(ctx, ec.client, &types.Transaction{})
	if err != nil {
		return nil, fmt.Errorf("等待交易确认失败: %v", err)
	}

	status := "success"
	if receipt.Status == 0 {
		status = "failed"
	}

	return &TransactionResponse{
		TxHash:    txHash.Hex(),
		Status:    status,
		Message:   fmt.Sprintf("交易已确认，区块: %d", receipt.BlockNumber.Uint64()),
		Timestamp: time.Now().Unix(),
	}, nil
}

// 🔥 删除文件功能
func (ec *EthereumClient) DeleteFile(userAddress common.Address, fileHash string, userPrivateKeyHex string) (*TransactionResponse, error) {
	// 解析用户私钥
	// 移除可能的0x前缀
	userPrivateKeyHex = strings.TrimPrefix(userPrivateKeyHex, "0x")
	userPrivateKey, err := crypto.HexToECDSA(userPrivateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("解析用户私钥失败: %v", err)
	}

	// 获取链ID
	chainID, err := ec.client.ChainID(context.Background())
	if err != nil {
		chainID = big.NewInt(31337) // Hardhat 默认链ID
	}

	// 创建用户交易选项
	userAuth, err := bind.NewKeyedTransactorWithChainID(userPrivateKey, chainID)
	if err != nil {
		return nil, fmt.Errorf("创建用户交易认证失败: %v", err)
	}

	// 设置Gas配置
	gasPrice, _ := getGasConfig(ec.client)
	userAuth.GasPrice = gasPrice
	userAuth.GasLimit = 300000 // 删除操作Gas限制
	userAuth.Value = big.NewInt(0)
	userAuth.Context = context.Background()

	// 检查余额
	balance, err := ec.client.BalanceAt(context.Background(), userAddress, nil)
	if err != nil {
		return nil, fmt.Errorf("检查用户余额失败: %v", err)
	}

	gasCost := new(big.Int).Mul(gasPrice, new(big.Int).SetUint64(userAuth.GasLimit))
	if balance.Cmp(gasCost) < 0 {
		return nil, fmt.Errorf("用户余额不足。余额: %s WEI, 需要: %s WEI", balance.String(), gasCost.String())
	}
	
	tx, err := ec.contract.DeleteFile(userAuth, fileHash)
	if err != nil {
		return nil, fmt.Errorf("删除文件记录失败: %v", err)
	}

	log.Printf("文件删除交易已发送，文件哈希: %s, 交易哈希: %s, 用户地址: %s", fileHash, tx.Hash().Hex(), userAddress.Hex())

	return &TransactionResponse{
		TxHash:    tx.Hash().Hex(),
		Status:    "pending",
		Message:   "文件删除记录交易已发送",
		Timestamp: time.Now().Unix(),
	}, nil
}
