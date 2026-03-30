package core

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/crypto/bcrypt"
)

// UserDB 用户数据库
type UserDB struct {
	mu    sync.RWMutex
	users map[string]*UserAccount // key: 以太坊地址
	file  string
}

// UserAccount 用户账户信息（存储在服务器）
type UserAccount struct {
	Address  string `json:"address"`
	Password string `json:"password"` // bcrypt 哈希
	IsActive bool   `json:"is_active"`
}

// NewUserDB 创建用户数据库
func NewUserDB() (*UserDB, error) {
	db := &UserDB{
		users: make(map[string]*UserAccount),
		file:  "data/users.json",
	}

	// 创建数据目录
	if err := os.MkdirAll(filepath.Dir(db.file), 0755); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %v", err)
	}

	// 加载现有数据
	if err := db.load(); err != nil {
		return nil, fmt.Errorf("加载用户数据失败: %v", err)
	}

	return db, nil
}

// load 从文件加载用户数据
func (db *UserDB) load() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, err := os.Stat(db.file); os.IsNotExist(err) {
		return nil // 文件不存在，使用空数据库
	}

	data, err := os.ReadFile(db.file)
	if err != nil {
		return err
	}

	var users map[string]*UserAccount
	if err := json.Unmarshal(data, &users); err != nil {
		return err
	}

	db.users = users
	return nil
}

// save 保存用户数据到文件
func (db *UserDB) save() error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	data, err := json.MarshalIndent(db.users, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(db.file, data, 0644)
}

// RegisterUser 注册用户（设置密码）
func (db *UserDB) RegisterUser(address common.Address, password string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	addressHex := address.Hex()
	
	// 检查用户是否已存在
	if _, exists := db.users[addressHex]; exists {
		return fmt.Errorf("用户已存在")
	}

	// 哈希密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("密码哈希失败: %v", err)
	}

	// 创建用户账户
	db.users[addressHex] = &UserAccount{
		Address:  addressHex,
		Password: string(hashedPassword),
		IsActive: true,
	}

	// 保存到文件
	if err := db.save(); err != nil {
		delete(db.users, addressHex) // 回滚
		return fmt.Errorf("保存用户数据失败: %v", err)
	}

	return nil
}

// VerifyPassword 验证密码
func (db *UserDB) VerifyPassword(address common.Address, password string) (bool, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	addressHex := address.Hex()
	user, exists := db.users[addressHex]
	if !exists {
		return false, fmt.Errorf("用户不存在")
	}

	if !user.IsActive {
		return false, fmt.Errorf("用户未激活")
	}

	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return false, fmt.Errorf("密码错误")
	}

	return true, nil
}

// UserExists 检查用户是否存在
func (db *UserDB) UserExists(address common.Address) bool {
	db.mu.RLock()
	defer db.mu.RUnlock()

	_, exists := db.users[address.Hex()]
	return exists
}

// GetUser 获取用户信息
func (db *UserDB) GetUser(address common.Address) (*UserAccount, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	user, exists := db.users[address.Hex()]
	if !exists {
		return nil, fmt.Errorf("用户不存在")
	}

	return user, nil
}

// UpdatePassword 更新密码
func (db *UserDB) UpdatePassword(address common.Address, newPassword string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	addressHex := address.Hex()
	user, exists := db.users[addressHex]
	if !exists {
		return fmt.Errorf("用户不存在")
	}

	// 哈希新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("密码哈希失败: %v", err)
	}

	user.Password = string(hashedPassword)

	// 保存到文件
	if err := db.save(); err != nil {
		return fmt.Errorf("保存用户数据失败: %v", err)
	}

	return nil
}

// AutoCreateHardhatUsers 自动创建Hardhat测试用户的本地账户
func (db *UserDB) AutoCreateHardhatUsers() error {
    hardhatAccounts := []string{
        "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266", // 账户0（管理员）
        "0x70997970C51812dc3A010C7d01b50e0d17dc79C8",
        "0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC",
        "0x90F79bf6EB2c4f870365E785982E1f101E93b906",
        "0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65",
        "0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc",
        "0x976EA74026E726554dB657fA54763abd0C3a0aa9",
        "0x14dC79964da2C08b23698B3D3cc7Ca32193d9955",
        "0x23618e81E3f5cdF7f54C3d65f7FBc0aBf5B21E8f",
        "0xa0Ee7A142d267C1f36714E4a8F75612F20a79720",
        "0xBcd4042DE499D14e55001CcbB24a551F3b954096",
        "0x71bE63f3384f5fb98995898A86B02Fb2426c5788",
        "0xFABB0ac9d68B0B445fB7357272Ff202C5651694a",
        "0x1CBd3b2770909D4e10f157cABC84C7264073C9Ec",
        "0xdF3e18d64BC6A983f673Ab319CCaE4f1a57C7097",
        "0xcd3B766CCDd6AE721141F452C550Ca635964ce71",
        "0x2546BcD3c84621e976D8185a91A922aE77ECEc30",
        "0xbDA5747bFD65F08deb54cb465eB87D40e51B197E",
        "0xdD2FD4581271e230360230F9337D5c0430Bf44C0",
        "0x8626f6940E2eb28930eFb4CeF49B2d1F2C9C1199",
    }

    createdCount := 0
    for i, account := range hardhatAccounts {
        address := common.HexToAddress(account)
        
        // 检查用户是否已存在
        if db.UserExists(address) {
            continue
        }

        // 为每个用户设置默认密码（账户索引+123456）
        defaultPassword := fmt.Sprintf("%d123456", i)
        
        if err := db.RegisterUser(address, defaultPassword); err != nil {
            log.Printf("创建用户账户失败 %s: %v", account, err)
            continue
        }

        createdCount++
    }

    log.Printf("自动创建 %d 个本地用户账户", createdCount)
    return nil
}