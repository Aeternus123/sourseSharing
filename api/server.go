package api

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"soursesharing/core"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
)

type Server struct {
	ethClient *core.EthereumClient
	storage   *core.FileStorage
	userDB    *core.UserDB // 🔥 新增用户数据库
	router    *gin.Engine
}

type UserRegisterRequest struct {
	Address  string `json:"address" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
}

type UserLoginRequest struct {
	// 密码字段已移除，现在直接使用地址登录
}

type AdminRegisterRequest struct {
	UserAddress string `json:"user_address" binding:"required"`
	Password    string `json:"password" binding:"required,min=6"`
}

func (s *Server) registerUser(c *gin.Context) {
	var req UserRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !common.IsHexAddress(req.Address) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的以太坊地址"})
		return
	}

	address := common.HexToAddress(req.Address)

	// 检查用户是否已在数据库中存在
	if s.userDB.UserExists(address) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户已注册"})
		return
	}

	// 在本地数据库注册用户
	if err := s.userDB.RegisterUser(address, req.Password); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "注册失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "用户注册成功，请联系管理员在区块链上激活",
		"address": address.Hex(),
	})
}

// 用户登录API
func (s *Server) userLogin(c *gin.Context) {
	address := c.Param("address")

	if !common.IsHexAddress(address) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的以太坊地址"})
		return
	}

	userAddress := common.HexToAddress(address)

	// 检查用户是否在区块链上激活
	blockchainRegistered, err := s.ethClient.IsUserRegistered(userAddress)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "检查区块链状态失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":           "登录成功",
		"address":           address,
		"blockchain_active": blockchainRegistered,
	})
}

// 管理员注册用户（带密码）
func (s *Server) adminRegisterUserWithPassword(c *gin.Context) {
	var req AdminRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !common.IsHexAddress(req.UserAddress) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的以太坊地址"})
		return
	}

	userAddress := common.HexToAddress(req.UserAddress)

	// 🔥 添加详细日志
	log.Printf("开始注册用户（带密码）: %s", userAddress.Hex())

	// 1. 在区块链上注册用户
	result, err := s.ethClient.RegisterUser(userAddress)
	if err != nil {
		log.Printf("区块链注册失败 - 地址: %s, 错误: %v", userAddress.Hex(), err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "区块链注册失败: " + err.Error()})
		return
	}

	// 2. 在本地数据库注册用户（设置密码）
	if err := s.userDB.RegisterUser(userAddress, req.Password); err != nil {
		log.Printf("用户数据保存失败 - 地址: %s, 错误: %v", userAddress.Hex(), err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "用户数据保存失败: " + err.Error()})
		return
	}

	log.Printf("用户注册成功: %s, 交易哈希: %s", userAddress.Hex(), result.TxHash)

	c.JSON(http.StatusOK, gin.H{
		"message": "用户注册成功",
		"result":  result,
		"address": userAddress.Hex(),
	})
}

// 修改原有的管理员注册函数名，避免冲突
func (s *Server) adminRegisterUser(c *gin.Context) {
	var req core.RegisterUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !common.IsHexAddress(req.UserAddress) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的以太坊地址"})
		return
	}

	userAddress := common.HexToAddress(req.UserAddress)

	log.Printf("开始注册用户: %s", userAddress.Hex())

	result, err := s.ethClient.RegisterUser(userAddress)
	if err != nil {
		log.Printf("注册用户失败 - 地址: %s, 错误: %v", userAddress.Hex(), err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "注册用户失败: " + err.Error()})
		return
	}

	log.Printf("用户注册成功: %s, 交易哈希: %s", userAddress.Hex(), result.TxHash)

	c.JSON(http.StatusOK, gin.H{
		"message": "用户注册成功",
		"result":  result,
	})
}

func NewServer(ethClient *core.EthereumClient, storage *core.FileStorage) (*Server, error) {
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 初始化用户数据库
	userDB, err := core.NewUserDB()
	if err != nil {
		return nil, fmt.Errorf("初始化用户数据库失败: %v", err)
	}

	// 🔥 异步自动创建Hardhat测试用户的本地账户，避免阻塞服务器启动
	go func() {
		if err := userDB.AutoCreateHardhatUsers(); err != nil {
			log.Printf("自动创建用户账户失败: %v", err)
		}
	}()

	server := &Server{
		ethClient: ethClient,
		storage:   storage,
		userDB:    userDB,
		router:    gin.Default(),
	}

	server.setupRoutes()
	return server, nil
}

func (s *Server) setupRoutes() {
	s.router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-User-ID, X-Admin-Address")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	api := s.router.Group("/api/v1")
	{
		// 🔥 新增用户认证路由
		api.POST("/users/register", s.registerUser)
		api.POST("/users/:address/login", s.userLogin)

		// 原有路由
		api.POST("/users/initialize", s.initializeUser)
// 原有路由
		api.POST("/files/upload", s.uploadFile)
		api.DELETE("/files/:file_hash", s.deleteFile)
		api.GET("/files/download/:owner/:file_hash", s.downloadFile)
		api.GET("/users/:address/info", s.getUserInfo)
		api.GET("/files/:file_hash/info", s.getFileInfo)
		api.GET("/files/shared", s.getSharedFiles)
		api.GET("/users/:address/files", s.getUserFiles)
		api.GET("/storage/info", s.getStorageInfo)
		// 账本相关API
		api.GET("/ledger/records", s.getLedgerRecords)
		api.GET("/ledger/records/user/:address", s.getLedgerRecordsByUser)
		api.GET("/ledger/records/type/:type", s.getLedgerRecordsByType)
		api.GET("/ledger/sync-info", s.getLedgerSyncInfo)
	}

	admin := s.router.Group("/api/v1/admin")
	admin.Use(s.adminMiddleware())
	{
		// 🔥 修改管理员注册，支持密码
		admin.POST("/register", s.adminRegisterUserWithPassword)
		admin.POST("/register/batch", s.adminBatchRegisterUsers)
		admin.POST("/registration/status", s.adminSetRegistrationStatus)
		admin.GET("/system/info", s.adminGetSystemInfo)
		admin.GET("/users", s.adminGetAllUsers)
	}

	s.router.GET("/health", s.healthCheck)
	s.router.GET("/", s.rootHandler)
}

func (s *Server) adminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		adminAddress := c.GetHeader("X-Admin-Address")
		if adminAddress == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "需要管理员权限"})
			c.Abort()
			return
		}

		log.Printf("管理员中间件: 收到管理员地址=%s", adminAddress)

		if !common.IsHexAddress(adminAddress) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的管理员地址格式"})
			c.Abort()
			return
		}

		address := common.HexToAddress(adminAddress)

		// 检查是否是合约管理员
		isAdmin := s.ethClient.IsAdmin(address)
		log.Printf("管理员检查结果: 地址=%s, 是管理员=%v", adminAddress, isAdmin)

		if !isAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "权限不足: 不是合约管理员"})
			c.Abort()
			return
		}

		c.Next()
	}
}

func (s *Server) rootHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "数据共享区块链系统 API",
		"version": "1.0.0",
		"endpoints": []string{
			"POST /api/v1/users/register - 用户注册（设置密码）",
			"POST /api/v1/users/:address/login - 用户登录（验证密码）",
			"POST /api/v1/users/initialize",
			"POST /api/v1/files/upload",
		"DELETE /api/v1/files/:file_hash",
		"GET /api/v1/files/download/:owner/:file_hash",
		"GET /api/v1/users/:address/info",
		"GET /api/v1/files/:file_hash/info",
		"GET /api/v1/files/shared",
		"GET /api/v1/users/:address/files",
		"GET /api/v1/storage/info",
			"POST /api/v1/admin/register - 管理员注册用户（带密码）",
			"POST /api/v1/admin/register/batch",
			"POST /api/v1/admin/registration/status",
			"GET /api/v1/admin/system/info",
			"GET /health",
		},
	})
}

func (s *Server) healthCheck(c *gin.Context) {
	log.Println("🩺 收到健康检查请求")
	log.Printf("   请求方法: %s", c.Request.Method)
	log.Printf("   请求路径: %s", c.Request.URL.Path)
	log.Printf("   客户端IP: %s", c.ClientIP())

	// 检查以太坊客户端状态
	ethStatus := "connected"
	if s.ethClient == nil {
		ethStatus = "disconnected"
	}
	log.Printf("   以太坊客户端状态: %s", ethStatus)

	// 发送响应
	response := gin.H{
		"status":    "healthy",
		"timestamp": core.GetCurrentTimestamp(),
		"server":    "DataShare Server",
		"version":   "1.0.0",
		"ethStatus": ethStatus,
	}

	log.Println("✅ 健康检查响应准备完成")
	c.JSON(http.StatusOK, response)
}

func (s *Server) initializeUser(c *gin.Context) {
	var req struct {
		Address string `json:"address" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !common.IsHexAddress(req.Address) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的以太坊地址"})
		return
	}

	address := common.HexToAddress(req.Address)
	result, err := s.ethClient.InitializeUser(address)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "用户初始化成功",
		"result":  result,
	})
}

func (s *Server) uploadFile(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	// 获取用户私钥
	userPrivateKey := c.GetHeader("X-Private-Key")
	if userPrivateKey == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未提供用户私钥"})
		return
	}

	if !common.IsHexAddress(userID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户地址"})
		return
	}

	address := common.HexToAddress(userID)
	initialized, err := s.ethClient.IsUserRegistered(address)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "检查用户状态失败: " + err.Error()})
		return
	}
	if !initialized {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户未注册或未激活，请联系管理员"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件上传失败: " + err.Error()})
		return
	}

	if file.Size == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件不能为空"})
		return
	}

	// 默认共享，除非显式指定为私有
	shareStr := c.PostForm("share")
	share := true // 默认共享
	if shareStr != "" {
		shareVal, err := strconv.ParseBool(shareStr)
		if err == nil {
			share = shareVal
		}
	}

	fileInfo, err := s.storage.SaveFile(userID, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "文件保存失败: " + err.Error()})
		return
	}

	// 传递用户私钥给UploadFile方法
	result, err := s.ethClient.UploadFile(address, fileInfo.FileHash, fileInfo.FileName, fileInfo.FileSize, share, userPrivateKey)
	if err != nil {
		s.storage.DeleteFile(userID, fileInfo.FileHash)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "区块链记录失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "文件上传成功",
		"file_hash": fileInfo.FileHash,
		"file_name": fileInfo.FileName,
		"file_size": fileInfo.FileSize,
		"shared":    share,
		"result":    result,
	})
}

func (s *Server) downloadFile(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	// 获取用户私钥
	userPrivateKey := c.GetHeader("X-Private-Key")
	if userPrivateKey == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未提供用户私钥"})
		return
	}

	owner := c.Param("owner")
	fileHash := c.Param("file_hash")

	if !common.IsHexAddress(userID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户地址"})
		return
	}

	downloaderAddress := common.HexToAddress(userID)
	initialized, err := s.ethClient.IsUserRegistered(downloaderAddress)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "检查用户状态失败: " + err.Error()})
		return
	}
	if !initialized {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户未注册或未激活"})
		return
	}

	// 传递用户私钥给DownloadFile方法
	result, err := s.ethClient.DownloadFile(downloaderAddress, fileHash, userPrivateKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "下载授权失败: " + err.Error()})
		return
	}

	filePath, err := s.storage.GetFile(owner, fileHash)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在: " + err.Error()})
		return
	}

	fileInfo, err := s.ethClient.GetFileInfo(fileHash)
	fileName := fileHash
	if err == nil && fileInfo.FileName != "" {
		fileName = fileInfo.FileName
	}

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", "attachment; filename="+fileName)
	c.Header("Content-Type", "application/octet-stream")

	c.File(filePath)

	log.Printf("文件下载成功: 用户=%s, 文件=%s, 交易哈希=%s", userID, fileHash, result.TxHash)
}

// 🔥 删除文件处理函数
func (s *Server) deleteFile(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	// 获取用户私钥
	userPrivateKey := c.GetHeader("X-Private-Key")
	if userPrivateKey == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未提供用户私钥"})
		return
	}

	fileHash := c.Param("file_hash")
	if fileHash == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件哈希不能为空"})
		return
	}

	if !common.IsHexAddress(userID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户地址"})
		return
	}

	userAddress := common.HexToAddress(userID)
	initialized, err := s.ethClient.IsUserRegistered(userAddress)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "检查用户状态失败: " + err.Error()})
		return
	}
	if !initialized {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户未注册或未激活"})
		return
	}

	// 调用区块链删除文件
	result, err := s.ethClient.DeleteFile(userAddress, fileHash, userPrivateKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "区块链删除失败: " + err.Error()})
		return
	}

	// 删除本地文件
	err = s.storage.DeleteFile(userID, fileHash)
	if err != nil {
		log.Printf("警告: 区块链删除成功但本地文件删除失败: %v", err)
		// 不返回错误，因为区块链删除是主要操作
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "文件删除成功",
		"file_hash": fileHash,
		"result":    result,
	})
}

func (s *Server) getUserInfo(c *gin.Context) {
	addressStr := c.Param("address")

	if !common.IsHexAddress(addressStr) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的以太坊地址"})
		return
	}

	address := common.HexToAddress(addressStr)
	userInfo, err := s.ethClient.GetUserInfo(address)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, userInfo)
}

func (s *Server) getFileInfo(c *gin.Context) {
	fileHash := c.Param("file_hash")

	fileInfo, err := s.ethClient.GetFileInfo(fileHash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, fileInfo)
}

func (s *Server) getSharedFiles(c *gin.Context) {
	log.Println("📤 收到获取共享文件列表请求")
	
	// 调用ethClient的GetSharedFiles方法获取共享文件列表
	sharedFiles, err := s.ethClient.GetSharedFiles()
	if err != nil {
		log.Printf("❌ 获取共享文件列表失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取共享文件列表失败: " + err.Error(),
		})
		return
	}
	
	log.Printf("✅ 成功获取共享文件列表，共 %d 个文件", len(sharedFiles))
	
	// 构造响应数据
	responseFiles := make([]gin.H, len(sharedFiles))
	for i, file := range sharedFiles {
		responseFiles[i] = gin.H{
			"file_hash":     file.FileHash,
			"file_name":     file.FileName,
			"file_size":     file.FileSize,
			"owner":         file.Owner,
			"upload_time":   file.UploadTime,
			"download_price": file.DownloadPrice.String(),
			"is_shared":     file.IsShared,
			"download_count": file.DownloadCount,
		}
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "获取共享文件列表成功",
		"files":   responseFiles,
		"count":   len(sharedFiles),
	})
}

func (s *Server) getUserFiles(c *gin.Context) {
	userID := c.Param("address")

	if !common.IsHexAddress(userID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的以太坊地址"})
		return
	}

	address := common.HexToAddress(userID)

	fileHashes, err := s.ethClient.GetUserFiles(address)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var files []*core.BlockchainFileInfo
	for _, fileHash := range fileHashes {
		fileInfo, err := s.ethClient.GetFileInfo(fileHash)
		if err == nil {
			files = append(files, fileInfo)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id": userID,
		"files":   files,
		"count":   len(files),
	})
}

func (s *Server) getStorageInfo(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	if !common.IsHexAddress(userID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户地址"})
		return
	}

	address := common.HexToAddress(userID)
	userInfo, err := s.ethClient.GetUserInfo(address)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	localFiles, err := s.storage.GetUserFiles(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取本地文件失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_info":         userInfo,
		"local_files_count": len(localFiles),
		"storage_path":      s.storage.BasePath,
	})
}

func (s *Server) adminBatchRegisterUsers(c *gin.Context) {
	var req core.BatchRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var addresses []common.Address
	for _, addrStr := range req.UserAddresses {
		if !common.IsHexAddress(addrStr) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的以太坊地址: " + addrStr})
			return
		}
		addresses = append(addresses, common.HexToAddress(addrStr))
	}

	result, err := s.ethClient.BatchRegisterUsers(addresses)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "批量用户注册成功",
		"result":  result,
	})
}

func (s *Server) adminSetRegistrationStatus(c *gin.Context) {
	var req core.RegistrationStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := s.ethClient.SetRegistrationStatus(req.Open)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	statusText := "关闭"
	if req.Open {
		statusText = "开启"
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "注册状态已" + statusText,
		"result":  result,
	})
}

func (s *Server) adminGetSystemInfo(c *gin.Context) {
	systemInfo, err := s.ethClient.GetSystemInfo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, systemInfo)
}

func (s *Server) adminGetAllUsers(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "获取用户列表功能待完善",
		"users":   []interface{}{},
	})
}

// 获取所有账本记录
func (s *Server) getLedgerRecords(c *gin.Context) {
	if s.ethClient == nil || s.ethClient.GetLedger() == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "账本系统未初始化",
		})
		return
	}

	records := s.ethClient.GetLedger().GetAllRecords()
	c.JSON(http.StatusOK, gin.H{
		"message": "获取账本记录成功",
		"records": records,
		"count":   len(records),
	})
}

// 根据用户获取账本记录
func (s *Server) getLedgerRecordsByUser(c *gin.Context) {
	if s.ethClient == nil || s.ethClient.GetLedger() == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "账本系统未初始化",
		})
		return
	}

	address := c.Param("address")
	if !common.IsHexAddress(address) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的以太坊地址",
		})
		return
	}

	records := s.ethClient.GetLedger().GetRecordsByUser(address)
	c.JSON(http.StatusOK, gin.H{
		"message": "获取用户账本记录成功",
		"user":    address,
		"records": records,
		"count":   len(records),
	})
}

// 根据类型获取账本记录
func (s *Server) getLedgerRecordsByType(c *gin.Context) {
	if s.ethClient == nil || s.ethClient.GetLedger() == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "账本系统未初始化",
		})
		return
	}

	recordType := c.Param("type")
	// 验证类型是否有效
	validTypes := map[string]bool{
		"upload":         true,
		"download":       true,
		"balance_update": true,
		"user_register":  true,
	}

	if !validTypes[recordType] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的记录类型",
			"valid_types": []string{"upload", "download", "balance_update", "user_register"},
		})
		return
	}

	records := s.ethClient.GetLedger().GetRecordsByType(recordType)
	c.JSON(http.StatusOK, gin.H{
		"message": "获取类型账本记录成功",
		"type":    recordType,
		"records": records,
		"count":   len(records),
	})
}

// 获取账本同步状态
func (s *Server) getLedgerSyncInfo(c *gin.Context) {
	if s.ethClient == nil || s.ethClient.GetLedger() == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "账本系统未初始化",
		})
		return
	}

	syncInfo := s.ethClient.GetLedger().GetSyncInfo()
	c.JSON(http.StatusOK, gin.H{
		"message":   "获取账本同步状态成功",
		"sync_info": syncInfo,
	})
}

func (s *Server) Run(addr string) error {
	log.Printf("启动服务器在 %s", addr)

	// 🔥 重要：确保绑定到 0.0.0.0 而不是 127.0.0.1
	if addr == "" {
		addr = "0.0.0.0:8080"
	} else if len(addr) > 0 && addr[0] == ':' {
		// 如果地址以 : 开头，如 ":8080"，改为 "0.0.0.0:8080"
		addr = "0.0.0.0" + addr
	} else if !contains(addr, ":") {
		// 如果只有端口号，如 "8080"，改为 "0.0.0.0:8080"
		addr = "0.0.0.0:" + addr
	}

	log.Printf("实际绑定地址: %s", addr)
	return s.router.Run(addr)
}

// 辅助函数：检查字符串是否包含子串
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
