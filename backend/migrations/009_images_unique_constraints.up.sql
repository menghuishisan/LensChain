-- 009_images_unique_constraints.up.sql
-- 为镜像同步 API（POST /api/v1/admin/images/sync）提供幂等 upsert 基础。
--
-- 背景：deploy/images/<name>/manifest.yaml 是镜像元数据的真相源，
-- seed-images.sh 通过 sync API 解析并写入 DB。upsert 必须按业务键
-- (images.name) 和 (image_versions.image_id, version) 幂等，
-- 因此需要数据库层 UNIQUE 约束兜底，避免并发同步插入重复行。
--
-- 影响范围：纯加约束，不动数据。
-- 兼容性：执行前已通过 ON CONFLICT (id) 跳过的种子数据若有同名重复，
-- 本迁移会报错；首次部署或 dev 重置数据库无此风险。

-- images.name 全局唯一（一个逻辑镜像名只对应一行 images 记录）
CREATE UNIQUE INDEX IF NOT EXISTS uk_images_name ON images(name) WHERE deleted_at IS NULL;

-- (image_id, version) 唯一（同一镜像下版本号不重复）
CREATE UNIQUE INDEX IF NOT EXISTS uk_image_versions_image_version ON image_versions(image_id, version);
