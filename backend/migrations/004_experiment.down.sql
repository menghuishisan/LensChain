-- 模块04 实验环境回滚

ALTER TABLE IF EXISTS course_experiments DROP CONSTRAINT IF EXISTS fk_course_experiments_experiment_id;
ALTER TABLE IF EXISTS lessons DROP CONSTRAINT IF EXISTS fk_lessons_experiment_id;
ALTER TABLE IF EXISTS experiment_instances DROP CONSTRAINT IF EXISTS fk_experiment_instances_group_id;
ALTER TABLE IF EXISTS template_containers DROP CONSTRAINT IF EXISTS fk_template_containers_role_id;

DROP TABLE IF EXISTS experiment_reports;
DROP TABLE IF EXISTS resource_quotas;
DROP TABLE IF EXISTS group_messages;
DROP TABLE IF EXISTS group_members;
DROP TABLE IF EXISTS experiment_groups;
DROP TABLE IF EXISTS instance_operation_logs;
DROP TABLE IF EXISTS instance_snapshots;
DROP TABLE IF EXISTS checkpoint_results;
DROP TABLE IF EXISTS instance_containers;
DROP TABLE IF EXISTS experiment_instances;
DROP TABLE IF EXISTS template_roles;
DROP TABLE IF EXISTS template_tags;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS template_sim_scenes;
DROP TABLE IF EXISTS sim_link_group_scenes;
DROP TABLE IF EXISTS sim_link_groups;
DROP TABLE IF EXISTS sim_scenarios;
DROP TABLE IF EXISTS template_init_scripts;
DROP TABLE IF EXISTS template_checkpoints;
DROP TABLE IF EXISTS template_containers;
DROP TABLE IF EXISTS experiment_templates;
DROP TABLE IF EXISTS image_versions;
DROP TABLE IF EXISTS images;
DROP TABLE IF EXISTS image_categories;
