// main.go
// 数据库迁移工具入口
// 使用 golang-migrate 管理 SQL 迁移文件
// 支持 up（执行迁移）、down（回滚迁移）、version（查看版本）等命令
// 迁移文件存放在 backend/migrations/ 目录

package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/lenschain/backend/internal/config"
)

func main() {
	// 命令行参数
	configPath := flag.String("config", "", "配置文件路径")
	migrationsPath := flag.String("path", "file://migrations", "迁移文件目录路径")
	steps := flag.Int("steps", 0, "迁移步数（用于 up-steps/down-steps）")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	// 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 构建数据库连接 URL
	dbURL := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		url.QueryEscape(cfg.Database.User),
		url.QueryEscape(cfg.Database.Password),
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.DBName,
		cfg.Database.SSLMode,
	)

	// 创建迁移实例
	m, err := migrate.New(*migrationsPath, dbURL)
	if err != nil {
		fmt.Printf("创建迁移实例失败: %v\n", err)
		os.Exit(1)
	}
	defer m.Close()

	// 执行命令
	command := args[0]
	switch command {
	case "up":
		err = m.Up()
		if err != nil && err != migrate.ErrNoChange {
			fmt.Printf("执行迁移失败: %v\n", err)
			os.Exit(1)
		}
		if err == migrate.ErrNoChange {
			fmt.Println("数据库已是最新版本，无需迁移")
		} else {
			fmt.Println("迁移执行成功")
		}

	case "down":
		err = m.Down()
		if err != nil && err != migrate.ErrNoChange {
			fmt.Printf("回滚迁移失败: %v\n", err)
			os.Exit(1)
		}
		if err == migrate.ErrNoChange {
			fmt.Println("已回滚到初始状态")
		} else {
			fmt.Println("回滚执行成功")
		}

	case "up-steps":
		if *steps <= 0 {
			fmt.Println("请指定迁移步数: -steps N")
			os.Exit(1)
		}
		err = m.Steps(*steps)
		if err != nil && err != migrate.ErrNoChange {
			fmt.Printf("执行迁移失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("已执行 %d 步迁移\n", *steps)

	case "down-steps":
		if *steps <= 0 {
			fmt.Println("请指定回滚步数: -steps N")
			os.Exit(1)
		}
		err = m.Steps(-*steps)
		if err != nil && err != migrate.ErrNoChange {
			fmt.Printf("回滚迁移失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("已回滚 %d 步迁移\n", *steps)

	case "version":
		version, dirty, err := m.Version()
		if err != nil {
			fmt.Printf("获取版本失败: %v\n", err)
			os.Exit(1)
		}
		dirtyStr := ""
		if dirty {
			dirtyStr = " (dirty)"
		}
		fmt.Printf("当前版本: %d%s\n", version, dirtyStr)

	case "force":
		if *steps < 0 {
			fmt.Println("请指定强制版本号: -steps N")
			os.Exit(1)
		}
		err = m.Force(*steps)
		if err != nil {
			fmt.Printf("强制设置版本失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("已强制设置版本为: %d\n", *steps)

	case "drop":
		fmt.Print("警告：此操作将删除所有表！确认请输入 'yes': ")
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "yes" {
			fmt.Println("操作已取消")
			os.Exit(0)
		}
		err = m.Drop()
		if err != nil {
			fmt.Printf("删除所有表失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("所有表已删除")

	default:
		fmt.Printf("未知命令: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

// printUsage 打印使用说明
func printUsage() {
	fmt.Println(`链镜平台 — 数据库迁移工具

用法:
  go run cmd/migrate/main.go [选项] <命令>

命令:
  up          执行所有未执行的迁移
  down        回滚所有迁移
  up-steps    执行指定步数的迁移（需配合 -steps）
  down-steps  回滚指定步数的迁移（需配合 -steps）
  version     查看当前迁移版本
  force       强制设置迁移版本（需配合 -steps 指定版本号）
  drop        删除所有表（危险操作）

选项:
  -config     配置文件路径（默认自动查找 configs/config.yaml）
  -path       迁移文件目录（默认 file://migrations）
  -steps      迁移/回滚步数

示例:
  go run cmd/migrate/main.go up                    # 执行所有迁移
  go run cmd/migrate/main.go down                  # 回滚所有迁移
  go run cmd/migrate/main.go up-steps -steps 1     # 执行1步迁移
  go run cmd/migrate/main.go down-steps -steps 1   # 回滚1步迁移
  go run cmd/migrate/main.go version               # 查看当前版本
  go run cmd/migrate/main.go force -steps 3        # 强制设置版本为3`)
}
