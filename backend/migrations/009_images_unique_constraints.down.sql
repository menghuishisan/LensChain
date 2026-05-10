-- 009_images_unique_constraints.down.sql
-- 回滚：移除镜像同步幂等 upsert 所需的 UNIQUE 约束。

DROP INDEX IF EXISTS uk_image_versions_image_version;
DROP INDEX IF EXISTS uk_images_name;
