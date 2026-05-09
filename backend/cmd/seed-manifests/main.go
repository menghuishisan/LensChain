// main.go
// 镜像 manifest 批量同步 CLI。
//
// 用途：在 init-db 流程中（或任何无需启动 backend 的 bootstrap 场景）扫描
// deploy/images/<category>/<name>/manifest.yaml，复用 service.SyncImageFromManifest
// 直接 upsert 到 images / image_versions 表。与运行期 admin HTTP 接口
// (POST /api/v1/admin/images/sync) 共享同一份业务逻辑，确保唯一真相源。
//
// 与 deploy/scripts/bash/seed-images.sh 的区别：
//   - shell 脚本要求 backend 已启动 + 拥有 admin token，适合生产环境管理员操作；
//   - 本 CLI 直接连数据库，适合首次部署 / 测试环境重置 / CI 流水线，
//     不依赖 backend HTTP 服务存活。
//
// 用法：
//   go run cmd/seed-manifests/main.go -images-dir ../deploy/images
//   或：DEPLOY_IMAGES_DIR=../deploy/images go run cmd/seed-manifests/main.go

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/lenschain/backend/internal/config"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/logger"
	experimentrepo "github.com/lenschain/backend/internal/repository/experiment"
	experimentsvc "github.com/lenschain/backend/internal/service/experiment"
)

func main() {
	configPath := flag.String("config", "", "配置文件路径（留空则按 config.Load 默认搜索逻辑）")
	imagesDirFlag := flag.String("images-dir", "", "deploy/images 根目录（留空则读取 DEPLOY_IMAGES_DIR 环境变量）")
	flag.Parse()

	imagesDir := *imagesDirFlag
	if imagesDir == "" {
		imagesDir = os.Getenv("DEPLOY_IMAGES_DIR")
	}
	if imagesDir == "" {
		fmt.Println("缺少 deploy/images 路径：请通过 -images-dir 参数或 DEPLOY_IMAGES_DIR 环境变量指定")
		os.Exit(1)
	}
	absImagesDir, err := filepath.Abs(imagesDir)
	if err != nil {
		fmt.Printf("解析 deploy/images 绝对路径失败: %v\n", err)
		os.Exit(1)
	}
	info, err := os.Stat(absImagesDir)
	if err != nil || !info.IsDir() {
		fmt.Printf("deploy/images 目录不存在或不可访问: %s\n", absImagesDir)
		os.Exit(1)
	}

	// 与 cmd/server 保持同一初始化顺序，避免 logger/database 内部空指针。
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		os.Exit(1)
	}
	if err := logger.Init(&cfg.Log); err != nil {
		fmt.Printf("初始化日志失败: %v\n", err)
		os.Exit(1)
	}
	if err := database.Init(&cfg.Database); err != nil {
		fmt.Printf("初始化数据库失败: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	manifests, err := scanManifests(absImagesDir)
	if err != nil {
		fmt.Printf("扫描 manifest 失败: %v\n", err)
		os.Exit(1)
	}
	if len(manifests) == 0 {
		fmt.Printf("未在 %s 下找到任何 manifest.yaml，跳过\n", absImagesDir)
		return
	}
	fmt.Printf("==> 找到 %d 份 manifest.yaml，开始同步\n", len(manifests))

	db := database.Get()
	imgSvc := experimentsvc.NewImageService(
		db,
		experimentrepo.NewImageCategoryRepository(db),
		experimentrepo.NewImageRepository(db),
		experimentrepo.NewImageVersionRepository(db),
		nil, // userNameQuerier：sync 流程不涉及上传者展示
		nil, // k8sSvc：sync 流程不需要 K8s
	)

	ctx := context.Background()
	var (
		ok      int
		failed  int
		failMsg []string
	)
	for _, path := range manifests {
		raw, err := os.ReadFile(path)
		if err != nil {
			failed++
			failMsg = append(failMsg, fmt.Sprintf("%s: 读文件失败 %v", path, err))
			continue
		}
		result, err := imgSvc.SyncImageFromManifest(ctx, raw)
		if err != nil {
			failed++
			failMsg = append(failMsg, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		ok++
		action := "updated"
		if result.ImageCreated {
			action = "created"
		}
		fmt.Printf("    [%s] %s (versions: +%d ~%d =%d)\n",
			action, result.Name,
			result.VersionsCreated, result.VersionsUpdated, result.VersionsSkipped,
		)
	}

	fmt.Printf("==> 同步完成：成功 %d，失败 %d\n", ok, failed)
	if failed > 0 {
		fmt.Println("失败明细：")
		for _, m := range failMsg {
			fmt.Println("  -", m)
		}
		os.Exit(1)
	}
}

// scanManifests 递归收集 deploy/images 下所有 manifest.yaml，
// 按路径排序确保不同环境执行顺序一致（便于排错与对账）。
func scanManifests(root string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Base(path) == "manifest.yaml" {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	return paths, nil
}
