数据共享区块链系统 - Windows客户端
==================================

📋 简介
这是一个基于区块链的数据共享系统命令行客户端，专为Windows系统设计。

🚀 快速开始

1. 编译程序
   双击运行 build.bat 文件，会自动编译生成 soursesharing.exe

2. 配置服务器
   soursesharing config server http://您的服务器IP:8080

3. 用户登录
   soursesharing login 0x您的以太坊地址

4. 上传文件
   soursesharing upload C:\Users\您的文件.pdf --share

5. 下载文件
   soursesharing download 0x文件所有者地址 文件哈希

🛠️ 可用命令

- config     配置客户端设置
- login      用户登录
- info       查看用户信息  
- upload     上传文件
- download   下载文件
- list       列出文件
- register   注册用户 (管理员)
- system     系统信息 (管理员)

📁 配置文件位置
Windows: C:\Users\[用户名]\AppData\Local\SourseSharing\config.json

🌐 网络要求
- 确保能访问服务器IP和端口
- 如果服务器在内网，需要配置端口转发
- 防火墙需要允许程序访问网络

❓ 帮助
使用 soursesharing --help 查看详细帮助信息