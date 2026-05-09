-- 000_seed_image_categories.sql
-- 镜像分类的种子数据。
--
-- 必须先于 cmd/seed-manifests CLI 执行：sync 流程会通过 manifest.category 字段
-- 反查 image_categories.code → category_id，因此分类必须先存在。
--
-- 这是唯一一个允许跑在镜像 manifest 同步前的 seed 文件；其他 seed
-- 都通过 (image_name, version) 子查询绑定 image_version_id，必须排在
-- 同步之后。

INSERT INTO image_categories (id, name, code, description, sort_order, created_at, updated_at)
VALUES
    (910000000000004001, '基础开发环境', 'base',        '开发环境与工具基础镜像',                    1, NOW(), NOW()),
    (910000000000004002, '链节点',       'chain-nodes', '链节点与协议运行镜像',                      2, NOW(), NOW()),
    (910000000000004003, '区块链中间件', 'middleware',  '部署、索引与链上调试中间件镜像',            3, NOW(), NOW()),
    (910000000000004004, '工具镜像',     'tools',       '浏览器、IDE、CLI 与调试分析工具镜像',       4, NOW(), NOW())
ON CONFLICT (code) DO NOTHING;
