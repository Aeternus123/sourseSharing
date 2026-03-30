package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

var (
	userLoginCmd = &cli.Command{
		Name:    "login",
		Usage:   "用户登录",
		Action:  userLoginCommand,
	}
	userLogoutCmd = &cli.Command{
		Name:    "logout",
		Usage:   "用户退出登录",
		Action:  userLogoutCommand,
	}
	infoCmd = &cli.Command{
		Name:    "info",
		Usage:   "查看用户信息",
		Action:  userInfoCommand,
	}
	adminLoginCmd = &cli.Command{
		Name:    "admin-login",
		Usage:   "管理员登录",
		Action:  adminLoginCommand,
	}
	adminLogoutCmd = &cli.Command{
		Name:    "admin-logout",
		Usage:   "管理员退出登录",
		Action:  adminLogoutCommand,
	}
	adminOpenRegistrationCmd = &cli.Command{
		Name:    "admin-open-registration",
		Usage:   "管理员开启注册功能",
		Action:  adminOpenRegistrationCommand,
	}
	adminRegisterCmd = &cli.Command{
		Name:    "admin-register",
		Usage:   "管理员注册/激活用户账号",
		Action:  adminRegisterUserCommand,
	}
)

func main() {
	app := &cli.App{
    Name:    "soursesharing",
    Usage:   "数据共享区块链系统客户端",
    Version: "1.0.0",
    Commands: []*cli.Command{
        // 配置命令
        {
            Name:  "config",
            Usage: "查看和修改配置",
            Action: configCommand,
        },
        
        // 用户命令
        {
            Name:  "login",
            Usage: "用户登录（自动使用默认密码）",
            Action: userLoginCommand,
        },
        {
            Name:  "logout",
            Usage: "用户退出登录",
            Action: userLogoutCommand,
        },
        {
            Name:  "info",
            Usage: "查看用户信息",
            Action: userInfoCommand,
        },
        
        // 文件操作
        {
            Name:  "upload",
            Usage: "上传文件（默认共享）",
            Action: uploadCommand,
            Flags: []cli.Flag{
                &cli.BoolFlag{
                    Name:  "private",
                    Usage: "将文件设为私有",
                    Value: false,
                },
            },
        },
        {
            Name:  "delete",
            Usage: "删除文件",
            Action: deleteFileCommand,
        },
        {
            Name:  "download",
            Usage: "下载文件",
            Action: downloadCommand,
        },
        {
            Name:  "list",
            Usage: "列出用户文件",
            Action: listFilesCommand,
        },
        {            Name:  "shared",            Usage: "查看共享文件",            Action: sharedFilesCommand,        },        
        // 账本命令        
        {            Name:  "ledger",            Usage: "查看账本记录",            Action: ledgerRecordsCommand,        },        {            Name:  "ledger-user",            Usage: "查看用户账本记录 (可选参数: 用户地址，默认当前用户)",            Action: ledgerRecordsByUserCommand,        },        {            Name:  "ledger-sync",            Usage: "查看账本同步状态",            Action: ledgerSyncInfoCommand,        },
        
        // 管理员命令（可选）
        {
            Name:  "admin-login",
            Usage: "管理员登录",
            Action: adminLoginCommand,
        },
        {
            Name:  "admin-logout", 
            Usage: "管理员退出登录",
            Action: adminLogoutCommand,
        },
        {
            Name:  "admin-open-registration", 
            Usage: "管理员开启注册功能",
            Action: adminOpenRegistrationCommand,
        },
        {
            Name:  "admin-register", 
            Usage: "管理员注册/激活用户账号",
            Action: adminRegisterUserCommand,
        },
    },
}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}