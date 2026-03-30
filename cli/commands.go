package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
)

type APIResponse struct {
	Message string      `json:"message"`
	Result  interface{} `json:"result"`
}

type FileInfo struct {
	FileName string `json:"file_name"`
}

func makeRequest(method, url string, headers map[string]string, body io.Reader) ([]byte, error) {
	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errorResp map[string]interface{}
		if err := json.Unmarshal(responseBody, &errorResp); err == nil {
			if errorMsg, ok := errorResp["error"].(string); ok {
				return nil, fmt.Errorf("服务器返回错误: %s", errorMsg)
			}
		}
		return nil, fmt.Errorf("HTTP错误 %d: %s", resp.StatusCode, resp.Status)
	}

	return responseBody, nil
}

func validateConfig(config *Config) error {
	// 允许使用localhost作为服务器地址
	return nil
}

func testConnection(config *Config) error {
	url := config.ServerURL + "/health"
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("无法连接到服务器 %s: %v", config.ServerURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("服务器返回错误状态: %s", resp.Status)
	}

	return nil
}

func configCommand(c *cli.Context) error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	if c.NArg() == 0 {
		fmt.Println("═══════════════════════════════════════")
		fmt.Println("           当前配置信息")
		fmt.Println("═══════════════════════════════════════")
		fmt.Printf("服务器地址: %s\n", config.ServerURL)

		fmt.Print("连接状态: ")
		if err := testConnection(config); err != nil {
			fmt.Printf("❌ 连接失败\n")
			fmt.Printf("   错误: %v\n", err)
		} else {
			fmt.Printf("✅ 连接正常\n")
		}

		fmt.Printf("用户地址: %s\n", config.UserAddress)
		fmt.Printf("管理员地址: %s\n", config.AdminAddress)
		fmt.Println("═══════════════════════════════════════")
		return nil
	}

	switch c.Args().Get(0) {
	case "server":
		if c.NArg() < 2 {
			return fmt.Errorf("请提供服务器地址")
		}
		url := c.Args().Get(1)
		if !strings.HasPrefix(url, "http") {
			url = "http://" + url
		}
		if err := config.SetServerURL(url); err != nil {
			return err
		}
		fmt.Printf("✅ 服务器地址已设置为: %s\n", url)

		fmt.Print("测试连接... ")
		if err := testConnection(config); err != nil {
			fmt.Printf("❌ 连接失败: %v\n", err)
			fmt.Println("请检查服务器地址和网络连接")
		} else {
			fmt.Printf("✅ 连接成功!\n")
		}
	case "user":
		if c.NArg() < 2 {
			return fmt.Errorf("请提供用户地址")
		}
		address := c.Args().Get(1)
		if err := config.SetUserAddress(address); err != nil {
			return err
		}
		fmt.Printf("✅ 用户地址已设置为: %s\n", address)
	case "admin":
		if c.NArg() < 2 {
			return fmt.Errorf("请提供管理员地址")
		}
		address := c.Args().Get(1)
		if err := config.SetAdminAddress(address); err != nil {
			return err
		}
		fmt.Printf("✅ 管理员地址已设置为: %s\n", address)
	default:
		return fmt.Errorf("未知配置项，可用选项: server, user, admin")
	}

	return nil
}

// 🔥 删除文件命令
func deleteFileCommand(c *cli.Context) error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	if err := validateConfig(config); err != nil {
		return err
	}

	if config.UserAddress == "" {
		return fmt.Errorf("请先使用 login 命令登录")
	}

	fileHash := c.Args().Get(0)
	if fileHash == "" {
		return fmt.Errorf("请提供文件哈希")
	}

	fmt.Printf("正在删除文件...\n")
	fmt.Printf("文件哈希: %s\n", fileHash)

	// 确认删除
	var confirm string
	fmt.Print("确认删除此文件？(y/N): ")
	fmt.Scanln(&confirm)
	if strings.ToLower(confirm) != "y" {
		return fmt.Errorf("删除操作已取消")
	}

	url := fmt.Sprintf("%s/api/v1/files/%s", config.ServerURL, fileHash)
	headers := map[string]string{
		"X-User-ID":     config.UserAddress,
		"X-Private-Key": config.PrivateKey,
		"Content-Type":  "application/json",
	}

	// 检查是否已提供私钥
	if config.PrivateKey == "" {
		return fmt.Errorf("未找到用户私钥，请重新登录")
	}

	response, err := makeRequest("DELETE", url, headers, nil)
	if err != nil {
		return err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(response, &result); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}

	fmt.Println("✅ 文件删除成功!")
	fmt.Printf("文件哈希: %s\n", result["file_hash"])

	return nil
}

func userInfoCommand(c *cli.Context) error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	if validateErr := validateConfig(config); validateErr != nil {
		return validateErr
	}

	if config.UserAddress == "" {
		return fmt.Errorf("请先使用 login 命令登录")
	}

	fmt.Printf("正在获取用户信息: %s\n", config.UserAddress)

	url := fmt.Sprintf("%s/api/v1/users/%s/info", config.ServerURL, config.UserAddress)
	response, err := makeRequest("GET", url, nil, nil)
	if err != nil {
		return err
	}

	var userInfo map[string]interface{}
	if err := json.Unmarshal(response, &userInfo); err != nil {
		return fmt.Errorf("解析用户信息失败: %v", err)
	}

	fmt.Println("═══════════════════════════════════════")
	fmt.Println("             用户信息")
	fmt.Println("═══════════════════════════════════════")
	fmt.Printf("地址: %s\n", userInfo["address"])
	fmt.Printf("余额: %.0f 币\n", userInfo["balance"])

	usedStorage := userInfo["used_storage"].(float64)
	maxStorage := userInfo["max_storage"].(float64)
	usagePercent := (usedStorage / maxStorage) * 100

	fmt.Printf("存储使用: %.2f MB / %.2f MB (%.1f%%)\n",
		usedStorage/1024/1024,
		maxStorage/1024/1024,
		usagePercent)
	fmt.Printf("总收入: %.0f 币\n", userInfo["total_earned"])
	fmt.Printf("总支出: %.0f 币\n", userInfo["total_spent"])

	if isActive, ok := userInfo["is_active"].(bool); ok {
		status := "激活"
		if !isActive {
			status = "未激活"
		}
		fmt.Printf("账户状态: %s\n", status)
	}
	fmt.Println("═══════════════════════════════════════")

	return nil
}

func uploadCommand(c *cli.Context) error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	if err := validateConfig(config); err != nil {
		return err
	}

	if config.UserAddress == "" {
		return fmt.Errorf("请先使用 login 命令登录")
	}

	filePath := c.Args().Get(0)
	if filePath == "" {
		return fmt.Errorf("请提供文件路径")
	}

	filePath = filepath.Clean(filePath)

	// 默认共享，除非显式指定--private标志
	share := true
	if c.Bool("private") {
		share = false
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("文件不存在: %s", filePath)
	}

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("获取文件信息失败: %v", err)
	}

	fileSizeMB := float64(fileInfo.Size()) / 1024 / 1024
	fmt.Printf("正在上传文件: %s (大小: %.2f MB)\n",
		filepath.Base(filePath),
		fileSizeMB)

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("打开文件失败: %v", err)
	}
	defer file.Close()

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("创建表单文件失败: %v", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return fmt.Errorf("复制文件内容失败: %v", err)
	}

	if err := writer.WriteField("share", strconv.FormatBool(share)); err != nil {
		return fmt.Errorf("添加分享参数失败: %v", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("关闭writer失败: %v", err)
	}

	url := fmt.Sprintf("%s/api/v1/files/upload", config.ServerURL)
	headers := map[string]string{
		"X-User-ID":     config.UserAddress,
		"X-Private-Key": config.PrivateKey,
		"Content-Type":  writer.FormDataContentType(),
	}

	// 检查是否已提供私钥
	if config.PrivateKey == "" {
		return fmt.Errorf("未找到用户私钥，请重新登录")
	}

	response, err := makeRequest("POST", url, headers, &requestBody)
	if err != nil {
		return err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(response, &result); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}

	fmt.Println("✅ 文件上传成功!")
	fmt.Printf("文件哈希: %s\n", result["file_hash"])
	fmt.Printf("文件名称: %s\n", result["file_name"])
	fmt.Printf("文件大小: %.2f MB\n", result["file_size"].(float64)/1024/1024)
	fmt.Printf("分享状态: %v\n", result["shared"])

	return nil
}

func downloadCommand(c *cli.Context) error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	if err := validateConfig(config); err != nil {
		return err
	}

	if config.UserAddress == "" {
		return fmt.Errorf("请先使用 login 命令登录")
	}

	if c.NArg() < 2 {
		return fmt.Errorf("请提供文件所有者和文件哈希\n用法: download <所有者地址> <文件哈希> [保存路径]")
	}

	owner := c.Args().Get(0)
	fileHash := c.Args().Get(1)
	outputPath := c.Args().Get(2)

	if outputPath == "" {
		outputPath = fileHash
	}

	fmt.Printf("正在下载文件...\n")
	fmt.Printf("所有者: %s\n", owner)
	fmt.Printf("文件哈希: %s\n", fileHash)
	fmt.Printf("保存路径: %s\n", outputPath)

	url := fmt.Sprintf("%s/api/v1/files/download/%s/%s", config.ServerURL, owner, fileHash)
	headers := map[string]string{
		"X-User-ID":     config.UserAddress,
		"X-Private-Key": config.PrivateKey,
	}

	// 检查是否已提供私钥
	if config.PrivateKey == "" {
		return fmt.Errorf("未找到用户私钥，请重新登录")
	}

	response, err := makeRequest("GET", url, headers, nil)
	if err != nil {
		return err
	}

	if err := os.WriteFile(outputPath, response, 0644); err != nil {
		return fmt.Errorf("保存文件失败: %v", err)
	}

	fileInfo, _ := getFileInfo(config.ServerURL, fileHash)
	actualFileName := fileHash
	if fileInfo != nil && fileInfo.FileName != "" {
		actualFileName = fileInfo.FileName
		if outputPath == fileHash {
			os.Rename(fileHash, actualFileName)
			outputPath = actualFileName
		}
	}

	fileStat, _ := os.Stat(outputPath)
	fmt.Printf("✅ 下载成功!\n")
	fmt.Printf("文件名称: %s\n", actualFileName)
	fmt.Printf("文件大小: %.2f MB\n", float64(fileStat.Size())/1024/1024)
	fmt.Printf("保存位置: %s\n", outputPath)

	return nil
}

func listFilesCommand(c *cli.Context) error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	if err := validateConfig(config); err != nil {
		return err
	}

	address := c.Args().First()
	if address == "" {
		if config.UserAddress == "" {
			return fmt.Errorf("请提供用户地址或先登录")
		}
		address = config.UserAddress
	}

	fmt.Printf("正在获取用户文件列表: %s\n", address)

	url := fmt.Sprintf("%s/api/v1/users/%s/files", config.ServerURL, address)
	response, err := makeRequest("GET", url, nil, nil)
	if err != nil {
		return err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(response, &result); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}

	files, ok := result["files"].([]interface{})
	if !ok || len(files) == 0 {
		fmt.Println("该用户没有文件")
		return nil
	}

	fmt.Println("═══════════════════════════════════════")
	fmt.Printf("       用户 %s 的文件列表\n", address)
	fmt.Printf("           共 %d 个文件\n", len(files))
	fmt.Println("═══════════════════════════════════════")

	for i, file := range files {
		fileInfo := file.(map[string]interface{})
		fileName := fileInfo["file_name"].(string)
		fileSize := fileInfo["file_size"].(float64)
		isShared := fileInfo["is_shared"].(bool)
		downloadCount := fileInfo["download_count"].(float64)

		fmt.Printf("%2d. %s\n", i+1, fileName)
		fmt.Printf("    大小: %.2f MB | ", fileSize/1024/1024)

		shareStatus := "私有"
		if isShared {
			shareStatus = "共享"
		}
		fmt.Printf("状态: %s | ", shareStatus)
		fmt.Printf("下载: %.0f 次\n", downloadCount)
		fmt.Printf("    哈希: %s\n", fileInfo["file_hash"])
		fmt.Println()
	}

	return nil
}

func getFileInfo(serverURL, fileHash string) (*FileInfo, error) {
	url := fmt.Sprintf("%s/api/v1/files/%s/info", serverURL, fileHash)
	response, err := makeRequest("GET", url, nil, nil)
	if err != nil {
		return nil, err
	}

	var fileInfo FileInfo
	if err := json.Unmarshal(response, &fileInfo); err != nil {
		return nil, err
	}

	return &fileInfo, nil
}

func sharedFilesCommand(c *cli.Context) error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	if err := validateConfig(config); err != nil {
		return err
	}

	fmt.Printf("正在获取共享文件列表...\n")

	url := fmt.Sprintf("%s/api/v1/files/shared", config.ServerURL)
	response, err := makeRequest("GET", url, nil, nil)
	if err != nil {
		return err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(response, &result); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}

	files, ok := result["files"].([]interface{})
	if !ok || len(files) == 0 {
		fmt.Println("当前没有共享文件")
		return nil
	}

	fmt.Println("═══════════════════════════════════════")
	fmt.Println("           共享文件市场")
	fmt.Printf("           共 %d 个文件\n", len(files))
	fmt.Println("═══════════════════════════════════════")

	for i, file := range files {
		fileInfo := file.(map[string]interface{})
		fileName := fileInfo["file_name"].(string)
		owner := fileInfo["owner"].(string)
		fileHash := fileInfo["file_hash"].(string)
		
		// 安全处理file_size
		var fileSize float64
		if sizeStr, ok := fileInfo["file_size"].(string); ok {
			if size, err := strconv.ParseFloat(sizeStr, 64); err == nil {
				fileSize = size
			}
		} else if sizeNum, ok := fileInfo["file_size"].(float64); ok {
			fileSize = sizeNum
		}
		
		// 安全处理download_count
		var downloadCount float64
		if countStr, ok := fileInfo["download_count"].(string); ok {
			if count, err := strconv.ParseFloat(countStr, 64); err == nil {
				downloadCount = count
			}
		} else if countNum, ok := fileInfo["download_count"].(float64); ok {
			downloadCount = countNum
		}
		
		// 安全处理download_price
		var downloadPrice float64
		if priceStr, ok := fileInfo["download_price"].(string); ok {
			if price, err := strconv.ParseFloat(priceStr, 64); err == nil {
				downloadPrice = price
			}
		} else if priceNum, ok := fileInfo["download_price"].(float64); ok {
			downloadPrice = priceNum
		}

		fmt.Printf("%2d. %s\n", i+1, fileName)
		fmt.Printf("    大小: %.2f MB | ", fileSize/1024/1024)
		fmt.Printf("价格: %.0f 币 | ", downloadPrice)
		fmt.Printf("下载: %.0f 次\n", downloadCount)
		fmt.Printf("    所有者: %s\n", owner)
		fmt.Printf("    哈希: %s\n", fileHash)
		fmt.Println()
	}

	return nil
}

// 账本记录结构
type LedgerRecord struct {
	ID            string    `json:"id"`
	Type          string    `json:"type"`
	TxHash        string    `json:"tx_hash"`
	BlockNumber   uint64    `json:"block_number"`
	Timestamp     uint64    `json:"timestamp"`
	From          string    `json:"from"`
	To            string    `json:"to,omitempty"`
	Amount        string    `json:"amount,omitempty"`
	FileHash      string    `json:"file_hash,omitempty"`
	FileName      string    `json:"file_name,omitempty"`
	FileSize      uint64    `json:"file_size,omitempty"`
	UserAddress   string    `json:"user_address,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// 获取账本记录命令
func ledgerRecordsCommand(c *cli.Context) error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	if err := validateConfig(config); err != nil {
		return err
	}

	fmt.Printf("正在获取账本记录...\n")

	url := fmt.Sprintf("%s/ledger/records", config.ServerURL)
	response, err := makeRequestWithAuth("GET", url, nil, config.UserAddress, config.PrivateKey)
	if err != nil {
		return fmt.Errorf("获取账本记录失败: %v", err)
	}

	var result struct {
		Message string         `json:"message"`
		Records []LedgerRecord `json:"records"`
		Count   int            `json:"count"`
	}

	if err := json.Unmarshal(response, &result); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}

	fmt.Println("\n📒 账本记录")
	fmt.Println("====================================================================")
	fmt.Printf("共有 %d 条交易记录\n\n", result.Count)

	if result.Count > 0 {
		for _, record := range result.Records {
			// 显示记录信息
			fmt.Printf("类型: %s | 时间: %s\n", record.Type, formatTime(record.CreatedAt))
			fmt.Printf("交易哈希: %s\n", record.TxHash)
			fmt.Printf("区块: %d\n", record.BlockNumber)

			switch record.Type {
			case "upload":
				fmt.Printf("上传者: %s\n", record.From)
				fmt.Printf("文件: %s (%s)\n", record.FileName, formatFileSize(record.FileSize))
				fmt.Printf("奖励: %s 虚拟币\n", record.Amount)
			case "download":
				fmt.Printf("下载者: %s\n", record.From)
				fmt.Printf("收款者: %s\n", record.To)
				fmt.Printf("费用: %s 虚拟币\n", record.Amount)
			case "balance_update":
				fmt.Printf("用户: %s\n", record.UserAddress)
				fmt.Printf("余额: %s 虚拟币\n", record.Amount)
			case "user_register":
				fmt.Printf("用户: %s\n", record.UserAddress)
				fmt.Printf("操作: 注册成功\n")
			}

			fmt.Println("--------------------------------------------------------------------")
		}

		// 保存到本地文件
		saveToLocalLedger(result.Records)
	} else {
		fmt.Println("暂无交易记录")
	}

	return nil
}

// 获取用户账本记录命令
func ledgerRecordsByUserCommand(c *cli.Context) error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	if err := validateConfig(config); err != nil {
		return err
	}

	userAddress := c.Args().First()
	if userAddress == "" {
		userAddress = config.UserAddress // 默认查看当前用户
	}

	fmt.Printf("正在获取用户 %s 的账本记录...\n", userAddress)

	url := fmt.Sprintf("%s/ledger/records/user/%s", config.ServerURL, userAddress)
	response, err := makeRequestWithAuth("GET", url, nil, config.UserAddress, config.PrivateKey)
	if err != nil {
		return fmt.Errorf("获取用户账本记录失败: %v", err)
	}

	var result struct {
		Message string         `json:"message"`
		User    string         `json:"user"`
		Records []LedgerRecord `json:"records"`
		Count   int            `json:"count"`
	}

	if err := json.Unmarshal(response, &result); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}

	fmt.Printf("\n📒 用户 %s 的账本记录\n", result.User)
	fmt.Println("====================================================================")
	fmt.Printf("共有 %d 条交易记录\n\n", result.Count)

	if result.Count > 0 {
		for _, record := range result.Records {
			// 显示记录信息
			fmt.Printf("类型: %s | 时间: %s\n", record.Type, formatTime(record.CreatedAt))
			fmt.Printf("交易哈希: %s\n", record.TxHash)

			switch record.Type {
			case "upload":
				fmt.Printf("文件: %s (%s)\n", record.FileName, formatFileSize(record.FileSize))
				fmt.Printf("奖励: %s 虚拟币\n", record.Amount)
			case "download":
				if record.From == userAddress {
					fmt.Printf("操作: 下载文件\n")
					fmt.Printf("支付给: %s\n", record.To)
					fmt.Printf("费用: %s 虚拟币\n", record.Amount)
				} else {
					fmt.Printf("操作: 文件被下载\n")
					fmt.Printf("下载者: %s\n", record.From)
					fmt.Printf("收入: %s 虚拟币\n", record.Amount)
				}
			case "balance_update":
				fmt.Printf("余额: %s 虚拟币\n", record.Amount)
			case "user_register":
				fmt.Printf("操作: 注册成功\n")
			}

			fmt.Println("--------------------------------------------------------------------")
		}
	} else {
		fmt.Println("暂无交易记录")
	}

	return nil
}

// 获取账本同步状态命令
func ledgerSyncInfoCommand(c *cli.Context) error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	if err := validateConfig(config); err != nil {
		return err
	}

	fmt.Printf("正在获取账本同步状态...\n")

	url := fmt.Sprintf("%s/ledger/sync-info", config.ServerURL)
	response, err := makeRequestWithAuth("GET", url, nil, config.UserAddress, config.PrivateKey)
	if err != nil {
		return fmt.Errorf("获取账本同步状态失败: %v", err)
	}

	var result struct {
		Message  string                 `json:"message"`
		SyncInfo map[string]interface{} `json:"sync_info"`
	}

	if err := json.Unmarshal(response, &result); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}

	fmt.Println("\n🔄 账本同步状态")
	fmt.Println("====================================")
	fmt.Printf("同步状态: %v\n", result.SyncInfo["is_syncing"])
	fmt.Printf("当前区块: %v\n", result.SyncInfo["current_block"])
	fmt.Printf("最新区块: %v\n", result.SyncInfo["latest_block"])
	fmt.Printf("记录数量: %v\n", result.SyncInfo["records_count"])
	fmt.Printf("同步进度: %v\n", result.SyncInfo["sync_progress"])

	return nil
}

// 获取配置目录
func getConfigDir() (string, error) {
	// 在Windows上使用AppData目录
	if os.Getenv("APPDATA") != "" {
		return filepath.Join(os.Getenv("APPDATA"), "SourceSharing"), nil
	}
	// 其他平台使用当前目录
	return filepath.Join(".", ".config"), nil
}

// 保存账本到本地文件
func saveToLocalLedger(records []LedgerRecord) {
	// 创建本地账本目录
	configDir, err := getConfigDir()
	if err != nil {
		fmt.Printf("⚠️  获取配置目录失败: %v\n", err)
		return
	}

	ledgerDir := filepath.Join(configDir, "ledger")
	if err := os.MkdirAll(ledgerDir, 0755); err != nil {
		fmt.Printf("⚠️  创建本地账本目录失败: %v\n", err)
		return
	}

	// 保存到文件
	fileName := filepath.Join(ledgerDir, fmt.Sprintf("ledger_%s.json", time.Now().Format("20060102_150405")))
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		fmt.Printf("⚠️  序列化账本记录失败: %v\n", err)
		return
	}

	if err := os.WriteFile(fileName, data, 0644); err != nil {
		fmt.Printf("⚠️  保存本地账本失败: %v\n", err)
		return
	}

	fmt.Printf("✅ 账本记录已保存到本地: %s\n", fileName)
}

// 格式化时间
func formatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// 格式化文件大小
func formatFileSize(size uint64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%.2f KB", float64(size)/1024)
	} else if size < 1024*1024*1024 {
		return fmt.Sprintf("%.2f MB", float64(size)/(1024*1024))
	} else {
		return fmt.Sprintf("%.2f GB", float64(size)/(1024*1024*1024))
	}
}

// 带认证的请求函数
func makeRequestWithAuth(method, url string, body []byte, userAddress, privateKey string) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	// 添加认证头
	req.Header.Set("X-User-ID", userAddress)
	req.Header.Set("X-Private-Key", privateKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("请求失败: HTTP %d", resp.StatusCode)
	}

	return ioutil.ReadAll(resp.Body)
}
