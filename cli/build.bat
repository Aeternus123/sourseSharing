@echo off
chcp 65001 >nul
echo ========================================
echo   数据共享区块链系统 - Windows客户端编译
echo ========================================
echo.

echo 正在初始化Go模块...
go mod init sscli

echo 正在下载依赖...
go mod tidy

echo 正在编译Windows可执行文件...
go build -o sscli.exe

if %errorlevel% neq 0 (
    echo.
    echo ❌ 编译失败!
    echo 可能的原因:
    echo   - 缺少依赖包
    echo   - 代码语法错误
    echo   - 网络连接问题
    echo.
    echo 请检查上面的错误信息
    pause
    exit /b 1
)

echo.
echo ✅ 编译成功!
echo 📁 生成的可执行文件: sscli.exe
echo.
echo 🚀 使用步骤:
echo   1. 首次运行会自动引导配置
echo   2. 或手动配置: sscli config server http://服务器IP:8080
echo   3. 用户登录: sscli login 您的地址
echo.
pause