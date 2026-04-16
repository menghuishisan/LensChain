-- 053_create_experiment_groups.up.sql
-- 模块04 — 实验环境：实验分组表
-- 对照 docs/modules/04-实验环境/02-数据库设计.md 2.20节

CREATE TABLE experiment_groups (
    id           BIGINT       PRIMARY KEY,
    template_id  BIGINT       NOT NULL REFERENCES experiment_templates(id),
    course_id    BIGINT       NOT NULL,
    school_id    BIGINT       NOT NULL,
    group_name   VARCHAR(100) NOT NULL,
    group_method SMALLINT     NOT NULL DEFAULT 1,
    max_members  INT          NOT NULL DEFAULT 4,
    status       SMALLINT     NOT NULL DEFAULT 1,
    namespace    VARCHAR(100) NULL,
    created_at   TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMP    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_experiment_groups_template_id ON experiment_groups(template_id);
CREATE INDEX idx_experiment_groups_course_id ON experiment_groups(course_id);
CREATE INDEX idx_experiment_groups_school_id ON experiment_groups(school_id);
CREATE INDEX idx_experiment_groups_status ON experiment_groups(status);
