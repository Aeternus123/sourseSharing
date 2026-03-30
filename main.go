package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"soursesharing/api"
	"soursesharing/core"

	"github.com/joho/godotenv"
)

func main() {
	// 加载环境变量
	if err := godotenv.Load(); err != nil {
		log.Println("未找到.env文件，使用默认配置")
	}

	// 初始化以太坊客户端
	log.Println("正在初始化以太坊客户端...")
	ethClient, err := core.NewEthereumClient()
	if err != nil {
		log.Fatalf("初始化以太坊客户端失败: %v", err)
	}
	log.Printf("✅ 以太坊客户端初始化成功，合约地址: %s", ethClient.GetContractAddress())
	log.Printf("🔑 管理员地址: %s", ethClient.GetAdminAddress())

	// 初始化文件存储
	storagePath := os.Getenv("STORAGE_PATH")
	if storagePath == "" {
		storagePath = "./storage"
	}
	fileStorage := core.NewFileStorage(storagePath)
	log.Printf("📁 文件存储路径: %s", storagePath)

	// 创建存储目录
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		log.Fatalf("创建存储目录失败: %v", err)
	}

	// 启动API服务器
	server, err := api.NewServer(ethClient, fileStorage)
	if err != nil {
		log.Fatalf("启动API服务器失败: %v", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}



	// 准备绑定地址
	bindAddress := ":" + port
	log.Printf("🚀 准备启动API服务器，绑定地址: %s", bindAddress)
	log.Println("═══════════════════════════════════════")
	log.Println("           服务状态监控")
	log.Println("═══════════════════════════════════════")
	log.Println("健康检查: http://localhost:" + port + "/health")
	log.Println("API文档: http://localhost:" + port)
	log.Println("以太坊节点: " + os.Getenv("ETH_RPC_URL"))
	log.Println("═══════════════════════════════════════")
	log.Println("正在启动HTTP服务器...")

	// 添加goroutine来验证服务器是否成功启动
	go func() {
		time.Sleep(3 * time.Second) // 等待服务器启动
		healthCheckURL := "http://localhost:" + port + "/health"
		log.Printf("🔍 尝试内部健康检查: %s", healthCheckURL)
		
		// 尝试简单的HTTP请求来验证服务是否正常
		resp, err := http.Get(healthCheckURL)
		if err != nil {
			log.Printf("⚠️  内部健康检查失败: %v", err)
			log.Println("🔍 请检查服务器是否正确绑定到端口")
		} else {
			defer resp.Body.Close()
			log.Printf("✅ 内部健康检查成功! 状态码: %d", resp.StatusCode)
		}
	}()

	log.Println("等待请求中...")
	if err := server.Run(bindAddress); err != nil {
		log.Fatalf("服务器运行失败: %v", err)
	}
}