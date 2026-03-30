package main

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"strings"
	"syscall"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/urfave/cli/v2"
	"golang.org/x/term"
)

// 安全的读取密码输入
func readPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println() // 换行
	if err != nil {
		return "", fmt.Errorf("读取密码失败: %v", err)
	}
	return string(bytePassword), nil
}

// 从私钥获取地址
func getAddressFromPrivateKey(privateKeyHex string) (common.Address, error) {
	// 清理私钥字符串
	privateKeyHex = strings.TrimSpace(privateKeyHex)
	privateKeyHex = strings.TrimPrefix(privateKeyHex, "0x")
	
	// 验证私钥长度
	if len(privateKeyHex) != 64 {
		return common.Address{}, fmt.Errorf("私钥长度不正确，应该是64位十六进制字符")
	}
	
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return common.Address{}, fmt.Errorf("私钥格式错误: %v", err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return common.Address{}, fmt.Errorf("无法生成公钥")
	}

	return crypto.PubkeyToAddress(*publicKeyECDSA), nil
}

// 管理员登录命令
func adminLoginCommand(c *cli.Context) error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	if err := validateConfig(config); err != nil {
		return err
	}

	address := c.Args().First()
	if address == "" {
		return fmt.Errorf("请提供管理员地址")
	}

	if !common.IsHexAddress(address) {
		return fmt.Errorf("无效的以太坊地址")
	}

	fmt.Println("正在登录管理员...")

	// 读取私钥
	privateKey, err := readPassword("请输入管理员私钥: ")
	if err != nil {
		return fmt.Errorf("读取私钥失败: %v", err)
	}

	// 验证私钥和地址匹配
	derivedAddress, err := getAddressFromPrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("私钥验证失败: %v", err)
	}

	expectedAddress := common.HexToAddress(address)
	if derivedAddress.Hex() != expectedAddress.Hex() {
		return fmt.Errorf("私钥与地址不匹配\n期望地址: %s\n实际地址: %s", 
			expectedAddress.Hex(), derivedAddress.Hex())
	}

	// 保存管理员信息
	if err := config.SetAdminAddress(address); err != nil {
		return err
	}
	if err := config.SetPrivateKey(privateKey); err != nil {
		return err
	}

	fmt.Println("✅ 管理员登录成功!")
	return nil
}


// 用户登录命令（使用私钥登录）
func userLoginCommand(c *cli.Context) error {
    config, err := LoadConfig()
    if err != nil {
        return err
    }

    if err := validateConfig(config); err != nil {
        return err
    }

    address := c.Args().First()
    if address == "" {
        return fmt.Errorf("请提供用户地址")
    }

    if !common.IsHexAddress(address) {
        return fmt.Errorf("无效的以太坊地址格式")
    }

    fmt.Printf("正在登录用户: %s\n", address)

    // 读取用户私钥
    privateKey, err := readPassword("请输入用户私钥: ")
    if err != nil {
        return fmt.Errorf("读取私钥失败: %v", err)
    }

    // 验证私钥和地址匹配
    derivedAddress, err := getAddressFromPrivateKey(privateKey)
    if err != nil {
        return fmt.Errorf("私钥验证失败: %v", err)
    }

    expectedAddress := common.HexToAddress(address)
    if derivedAddress.Hex() != expectedAddress.Hex() {
        return fmt.Errorf("私钥与地址不匹配\n期望地址: %s\n实际地址: %s", 
            expectedAddress.Hex(), derivedAddress.Hex())
    }

    // 调用服务器API验证用户登录
    url := fmt.Sprintf("%s/api/v1/users/%s/login", config.ServerURL, address)
    headers := map[string]string{
        "Content-Type": "application/json",
    }

    // 不需要发送密码，只需要发送空的JSON对象
    requestBody, _ := json.Marshal(map[string]interface{}{})

    response, err := makeRequest("POST", url, headers, bytes.NewBuffer(requestBody))
    if err != nil {
        return fmt.Errorf("登录失败: %v", err)
    }

    var result map[string]interface{}
    if err := json.Unmarshal(response, &result); err != nil {
        return fmt.Errorf("解析响应失败: %v", err)
    }

    // 保存用户信息和私钥
    if err := config.SetUserAddress(address); err != nil {
        return err
    }
    if err := config.SetPrivateKey(privateKey); err != nil {
        return err
    }

    fmt.Printf("✅ 用户登录成功! 地址: %s\n", address)
    return nil
}

// 生成默认密码（基于Hardhat账户索引）
func generateDefaultPassword(address string) string {
    // Hardhat测试账户列表
    hardhatAccounts := []string{
        "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266", // 账户0
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
    }

    // 查找地址索引
    for i, acc := range hardhatAccounts {
        if strings.EqualFold(acc, address) {
            return fmt.Sprintf("%d123456", i) // 索引 + 123456
        }
    }

    // 如果不是Hardhat测试账户，使用简单密码
    return "123456"
}

// 管理员退出登录
func adminLogoutCommand(c *cli.Context) error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	if config.AdminAddress == "" {
		fmt.Println("管理员未登录")
		return nil
	}

	if err := config.ClearPrivateKey(); err != nil {
		return err
	}
	
	config.AdminAddress = ""
	if err := config.Save(); err != nil {
		return err
	}

	fmt.Println("✅ 管理员已退出登录")
	return nil
}

// 管理员开启注册功能命令
func adminOpenRegistrationCommand(c *cli.Context) error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	if err := validateConfig(config); err != nil {
		return err
	}

	// 检查管理员是否已登录
	if config.AdminAddress == "" {
		return fmt.Errorf("请先使用 admin-login 命令登录管理员账号")
	}

	fmt.Println("正在开启注册功能...")

	// 调用服务器API开启注册功能
	url := fmt.Sprintf("%s/api/v1/admin/registration/status", config.ServerURL)
	headers := map[string]string{
		"Content-Type": "application/json",
		"X-Admin-Address": config.AdminAddress,
	}

	requestBody, _ := json.Marshal(map[string]bool{
		"Open": true,
	})

	response, err := makeRequest("POST", url, headers, bytes.NewBuffer(requestBody))
	if err != nil {
		return fmt.Errorf("开启注册功能失败: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(response, &result); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}

	fmt.Println("✅ 注册功能已成功开启!")
	return nil
}

// 管理员注册用户命令（用于激活用户账号）
func adminRegisterUserCommand(c *cli.Context) error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	if err := validateConfig(config); err != nil {
		return err
	}

	// 检查管理员是否已登录
	if config.AdminAddress == "" {
		return fmt.Errorf("请先使用 admin-login 命令登录管理员账号")
	}

	// 获取要注册的用户地址
	userAddress := c.Args().First()
	if userAddress == "" {
		return fmt.Errorf("请提供要注册的用户地址")
	}

	if !common.IsHexAddress(userAddress) {
		return fmt.Errorf("无效的以太坊地址格式")
	}

	fmt.Printf("正在注册用户: %s\n", userAddress)

	// 调用服务器API注册用户
	url := fmt.Sprintf("%s/api/v1/admin/register", config.ServerURL)
	headers := map[string]string{
		"Content-Type": "application/json",
		"X-Admin-Address": config.AdminAddress,
	}

	requestBody, _ := json.Marshal(map[string]string{
		"user_address": userAddress,
		"password": "123456", // 设置默认密码
	})

	response, err := makeRequest("POST", url, headers, bytes.NewBuffer(requestBody))
	if err != nil {
		return fmt.Errorf("注册用户失败: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(response, &result); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}

	fmt.Printf("✅ 用户注册成功! 地址: %s\n", userAddress)
	return nil
}

// 用户退出登录
func userLogoutCommand(c *cli.Context) error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	if config.UserAddress == "" {
		fmt.Println("用户未登录")
		return nil
	}

	if err := config.ClearPassword(); err != nil {
		return err
	}

	oldAddress := config.UserAddress
	config.UserAddress = ""
	if err := config.Save(); err != nil {
		return err
	}

	fmt.Printf("✅ 用户已退出登录: %s\n", oldAddress)
	return nil
}

