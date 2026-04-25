-- 模块04 实验环境
-- 文档依据：
-- 1. docs/modules/04-实验环境/02-数据库设计.md
-- 2. docs/standards/数据库规范.md
-- 负责范围：
-- 1. 镜像、实验模板、仿真场景、标签
-- 2. 实验实例、检查点结果、快照、日志
-- 3. 分组协作、资源配额、实验报告
-- 不负责：
-- 1. 课程课时主数据，仅通过外键引用
-- 2. SimEngine 运行时缓存与消息通道

-- image_categories：镜像分类表。
CREATE TABLE image_categories (
    id BIGINT PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    code VARCHAR(50) NOT NULL UNIQUE,
    description VARCHAR(200) NULL,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- images：镜像主表。
CREATE TABLE images (
    id BIGINT PRIMARY KEY,
    category_id BIGINT NOT NULL,
    name VARCHAR(100) NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    description TEXT NULL,
    icon_url VARCHAR(500) NULL,
    ecosystem VARCHAR(50) NULL,
    source_type SMALLINT NOT NULL DEFAULT 1,
    uploaded_by BIGINT NULL,
    school_id BIGINT NULL,
    status SMALLINT NOT NULL DEFAULT 1,
    review_comment VARCHAR(500) NULL,
    reviewed_by BIGINT NULL,
    reviewed_at TIMESTAMP NULL,
    default_ports JSONB NULL,
    default_env_vars JSONB NULL,
    default_volumes JSONB NULL,
    typical_companions JSONB NULL,
    required_dependencies JSONB NULL,
    resource_recommendation JSONB NULL,
    documentation_url VARCHAR(500) NULL,
    usage_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP NULL,
    CONSTRAINT fk_images_category_id FOREIGN KEY (category_id) REFERENCES image_categories(id),
    CONSTRAINT fk_images_uploaded_by FOREIGN KEY (uploaded_by) REFERENCES users(id),
    CONSTRAINT fk_images_school_id FOREIGN KEY (school_id) REFERENCES schools(id),
    CONSTRAINT fk_images_reviewed_by FOREIGN KEY (reviewed_by) REFERENCES users(id)
);
CREATE INDEX idx_images_category_id ON images(category_id);
CREATE INDEX idx_images_ecosystem ON images(ecosystem);
CREATE INDEX idx_images_source_type ON images(source_type);
CREATE INDEX idx_images_status ON images(status);
CREATE INDEX idx_images_school_id ON images(school_id);
CREATE INDEX idx_images_uploaded_by ON images(uploaded_by);

-- image_versions：镜像版本表。
CREATE TABLE image_versions (
    id BIGINT PRIMARY KEY,
    image_id BIGINT NOT NULL,
    version VARCHAR(50) NOT NULL,
    registry_url VARCHAR(500) NOT NULL,
    image_size BIGINT NULL,
    digest VARCHAR(200) NULL,
    min_cpu VARCHAR(20) NULL,
    min_memory VARCHAR(20) NULL,
    min_disk VARCHAR(20) NULL,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    status SMALLINT NOT NULL DEFAULT 1,
    scan_result JSONB NULL,
    scanned_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_image_versions_image_id FOREIGN KEY (image_id) REFERENCES images(id)
);
CREATE INDEX idx_image_versions_image_id ON image_versions(image_id);
CREATE UNIQUE INDEX uk_image_versions_image_version ON image_versions(image_id, version);

-- experiment_templates：实验模板主表。
CREATE TABLE experiment_templates (
    id BIGINT PRIMARY KEY,
    school_id BIGINT NOT NULL,
    teacher_id BIGINT NOT NULL,
    title VARCHAR(200) NOT NULL,
    description TEXT NULL,
    objectives TEXT NULL,
    instructions TEXT NULL,
    reference_materials TEXT NULL,
    experiment_type SMALLINT NOT NULL DEFAULT 2,
    topology_mode SMALLINT NULL,
    judge_mode SMALLINT NOT NULL DEFAULT 1,
    auto_weight DECIMAL(5,2) NULL,
    manual_weight DECIMAL(5,2) NULL,
    total_score DECIMAL(6,2) NOT NULL DEFAULT 100,
    max_duration INT NULL,
    idle_timeout INT NOT NULL DEFAULT 30,
    cpu_limit VARCHAR(20) NULL,
    memory_limit VARCHAR(20) NULL,
    disk_limit VARCHAR(20) NULL,
    score_strategy SMALLINT NOT NULL DEFAULT 1,
    is_shared BOOLEAN NOT NULL DEFAULT FALSE,
    cloned_from_id BIGINT NULL,
    status SMALLINT NOT NULL DEFAULT 1,
    sim_layout JSONB NULL,
    k8s_config JSONB NULL,
    network_config JSONB NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP NULL,
    CONSTRAINT fk_experiment_templates_school_id FOREIGN KEY (school_id) REFERENCES schools(id),
    CONSTRAINT fk_experiment_templates_teacher_id FOREIGN KEY (teacher_id) REFERENCES users(id),
    CONSTRAINT fk_experiment_templates_cloned_from_id FOREIGN KEY (cloned_from_id) REFERENCES experiment_templates(id)
);
CREATE INDEX idx_experiment_templates_school_id ON experiment_templates(school_id);
CREATE INDEX idx_experiment_templates_teacher_id ON experiment_templates(teacher_id);
CREATE INDEX idx_experiment_templates_experiment_type ON experiment_templates(experiment_type);
CREATE INDEX idx_experiment_templates_topology_mode ON experiment_templates(topology_mode);
CREATE INDEX idx_experiment_templates_status ON experiment_templates(status);
CREATE INDEX idx_experiment_templates_is_shared ON experiment_templates(is_shared) WHERE is_shared = TRUE;

-- template_containers：模板容器配置表。
CREATE TABLE template_containers (
    id BIGINT PRIMARY KEY,
    template_id BIGINT NOT NULL,
    image_version_id BIGINT NOT NULL,
    container_name VARCHAR(100) NOT NULL,
    deployment_scope SMALLINT NOT NULL DEFAULT 1,
    role_id BIGINT NULL,
    env_vars JSONB NULL,
    ports JSONB NULL,
    volumes JSONB NULL,
    cpu_limit VARCHAR(20) NULL,
    memory_limit VARCHAR(20) NULL,
    depends_on JSONB NULL,
    startup_order INT NOT NULL DEFAULT 0,
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_template_containers_template_id FOREIGN KEY (template_id) REFERENCES experiment_templates(id),
    CONSTRAINT fk_template_containers_image_version_id FOREIGN KEY (image_version_id) REFERENCES image_versions(id)
);
CREATE INDEX idx_template_containers_template_id ON template_containers(template_id);
CREATE INDEX idx_template_containers_image_version_id ON template_containers(image_version_id);
CREATE INDEX idx_template_containers_deployment_scope ON template_containers(deployment_scope);

-- template_checkpoints：检查点定义表。
CREATE TABLE template_checkpoints (
    id BIGINT PRIMARY KEY,
    template_id BIGINT NOT NULL,
    title VARCHAR(200) NOT NULL,
    description TEXT NULL,
    check_type SMALLINT NOT NULL,
    script_content TEXT NULL,
    script_language VARCHAR(20) NULL,
    target_container VARCHAR(100) NULL,
    assertion_config JSONB NULL,
    score DECIMAL(6,2) NOT NULL,
    scope SMALLINT NOT NULL DEFAULT 1,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_template_checkpoints_template_id FOREIGN KEY (template_id) REFERENCES experiment_templates(id)
);
CREATE INDEX idx_template_checkpoints_template_id ON template_checkpoints(template_id);

-- template_init_scripts：初始化脚本表。
CREATE TABLE template_init_scripts (
    id BIGINT PRIMARY KEY,
    template_id BIGINT NOT NULL,
    target_container VARCHAR(100) NOT NULL,
    script_content TEXT NOT NULL,
    script_language VARCHAR(20) NOT NULL DEFAULT 'bash',
    execution_order INT NOT NULL DEFAULT 0,
    timeout INT NOT NULL DEFAULT 300,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_template_init_scripts_template_id FOREIGN KEY (template_id) REFERENCES experiment_templates(id)
);
CREATE INDEX idx_template_init_scripts_template_id ON template_init_scripts(template_id);

-- sim_scenarios：仿真场景库表。
CREATE TABLE sim_scenarios (
    id BIGINT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    code VARCHAR(100) NOT NULL,
    category VARCHAR(50) NOT NULL,
    description TEXT NULL,
    icon_url VARCHAR(500) NULL,
    thumbnail_url VARCHAR(500) NULL,
    source_type SMALLINT NOT NULL DEFAULT 1,
    uploaded_by BIGINT NULL,
    school_id BIGINT NULL,
    status SMALLINT NOT NULL DEFAULT 1,
    review_comment VARCHAR(500) NULL,
    reviewed_by BIGINT NULL,
    reviewed_at TIMESTAMP NULL,
    algorithm_type VARCHAR(100) NOT NULL,
    time_control_mode VARCHAR(20) NOT NULL DEFAULT 'process',
    container_image_url VARCHAR(500) NULL,
    container_image_size BIGINT NULL,
    default_params JSONB NULL,
    interaction_schema JSONB NULL,
    data_source_mode SMALLINT NOT NULL DEFAULT 1,
    default_size JSONB NULL,
    delivery_phase SMALLINT NOT NULL DEFAULT 1,
    version VARCHAR(50) NOT NULL DEFAULT '1.0.0',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP NULL,
    CONSTRAINT fk_sim_scenarios_uploaded_by FOREIGN KEY (uploaded_by) REFERENCES users(id),
    CONSTRAINT fk_sim_scenarios_school_id FOREIGN KEY (school_id) REFERENCES schools(id),
    CONSTRAINT fk_sim_scenarios_reviewed_by FOREIGN KEY (reviewed_by) REFERENCES users(id)
);
CREATE UNIQUE INDEX uk_sim_scenarios_code ON sim_scenarios(code) WHERE deleted_at IS NULL;
CREATE INDEX idx_sim_scenarios_category ON sim_scenarios(category);
CREATE INDEX idx_sim_scenarios_source_type ON sim_scenarios(source_type);
CREATE INDEX idx_sim_scenarios_status ON sim_scenarios(status);
CREATE INDEX idx_sim_scenarios_algorithm_type ON sim_scenarios(algorithm_type);
CREATE INDEX idx_sim_scenarios_delivery_phase ON sim_scenarios(delivery_phase);
CREATE INDEX idx_sim_scenarios_time_control_mode ON sim_scenarios(time_control_mode);

-- sim_link_groups：联动组定义表。
CREATE TABLE sim_link_groups (
    id BIGINT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    code VARCHAR(100) NOT NULL,
    description TEXT NULL,
    shared_state_schema JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX uk_sim_link_groups_code ON sim_link_groups(code);

-- sim_link_group_scenes：联动组场景关联表。
CREATE TABLE sim_link_group_scenes (
    id BIGINT PRIMARY KEY,
    link_group_id BIGINT NOT NULL,
    scenario_id BIGINT NOT NULL,
    role_in_group VARCHAR(50) NULL,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_sim_link_group_scenes_link_group_id FOREIGN KEY (link_group_id) REFERENCES sim_link_groups(id),
    CONSTRAINT fk_sim_link_group_scenes_scenario_id FOREIGN KEY (scenario_id) REFERENCES sim_scenarios(id)
);
CREATE UNIQUE INDEX uk_link_group_scenes ON sim_link_group_scenes(link_group_id, scenario_id);
CREATE INDEX idx_link_group_scenes_scenario_id ON sim_link_group_scenes(scenario_id);

-- template_sim_scenes：模板仿真场景配置表。
CREATE TABLE template_sim_scenes (
    id BIGINT PRIMARY KEY,
    template_id BIGINT NOT NULL,
    scenario_id BIGINT NOT NULL,
    link_group_id BIGINT NULL,
    config JSONB NULL,
    layout_position JSONB NULL,
    data_source_config JSONB NULL,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_template_sim_scenes_template_id FOREIGN KEY (template_id) REFERENCES experiment_templates(id),
    CONSTRAINT fk_template_sim_scenes_scenario_id FOREIGN KEY (scenario_id) REFERENCES sim_scenarios(id),
    CONSTRAINT fk_template_sim_scenes_link_group_id FOREIGN KEY (link_group_id) REFERENCES sim_link_groups(id)
);
CREATE INDEX idx_template_sim_scenes_template_id ON template_sim_scenes(template_id);
CREATE INDEX idx_template_sim_scenes_scenario_id ON template_sim_scenes(scenario_id);
CREATE INDEX idx_template_sim_scenes_link_group_id ON template_sim_scenes(link_group_id);

-- tags：标签表。
CREATE TABLE tags (
    id BIGINT PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    category VARCHAR(50) NOT NULL,
    color VARCHAR(20) NULL,
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX uk_tags_name_category ON tags(name, category);
CREATE INDEX idx_tags_category ON tags(category);

-- template_tags：模板标签关联表。
CREATE TABLE template_tags (
    id BIGINT PRIMARY KEY,
    template_id BIGINT NOT NULL,
    tag_id BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_template_tags_template_id FOREIGN KEY (template_id) REFERENCES experiment_templates(id),
    CONSTRAINT fk_template_tags_tag_id FOREIGN KEY (tag_id) REFERENCES tags(id)
);
CREATE UNIQUE INDEX uk_template_tags ON template_tags(template_id, tag_id);
CREATE INDEX idx_template_tags_tag_id ON template_tags(tag_id);

-- template_roles：多人实验角色定义表。
CREATE TABLE template_roles (
    id BIGINT PRIMARY KEY,
    template_id BIGINT NOT NULL,
    role_name VARCHAR(100) NOT NULL,
    description TEXT NULL,
    max_members INT NOT NULL DEFAULT 1,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_template_roles_template_id FOREIGN KEY (template_id) REFERENCES experiment_templates(id)
);
CREATE INDEX idx_template_roles_template_id ON template_roles(template_id);

-- experiment_instances：实验实例表。
CREATE TABLE experiment_instances (
    id BIGINT PRIMARY KEY,
    template_id BIGINT NOT NULL,
    student_id BIGINT NOT NULL,
    school_id BIGINT NOT NULL,
    course_id BIGINT NULL,
    lesson_id BIGINT NULL,
    assignment_id BIGINT NULL,
    group_id BIGINT NULL,
    experiment_type SMALLINT NOT NULL,
    status SMALLINT NOT NULL DEFAULT 1,
    attempt_no INT NOT NULL DEFAULT 1,
    namespace VARCHAR(100) NULL,
    access_url VARCHAR(500) NULL,
    total_score DECIMAL(6,2) NULL,
    auto_score DECIMAL(6,2) NULL,
    manual_score DECIMAL(6,2) NULL,
    started_at TIMESTAMP NULL,
    paused_at TIMESTAMP NULL,
    submitted_at TIMESTAMP NULL,
    destroyed_at TIMESTAMP NULL,
    last_active_at TIMESTAMP NULL,
    error_message TEXT NULL,
    sim_session_id VARCHAR(100) NULL,
    sim_websocket_url VARCHAR(500) NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_experiment_instances_template_id FOREIGN KEY (template_id) REFERENCES experiment_templates(id),
    CONSTRAINT fk_experiment_instances_student_id FOREIGN KEY (student_id) REFERENCES users(id),
    CONSTRAINT fk_experiment_instances_school_id FOREIGN KEY (school_id) REFERENCES schools(id),
    CONSTRAINT fk_experiment_instances_course_id FOREIGN KEY (course_id) REFERENCES courses(id),
    CONSTRAINT fk_experiment_instances_lesson_id FOREIGN KEY (lesson_id) REFERENCES lessons(id),
    CONSTRAINT fk_experiment_instances_assignment_id FOREIGN KEY (assignment_id) REFERENCES assignments(id)
);
CREATE INDEX idx_experiment_instances_template_id ON experiment_instances(template_id);
CREATE INDEX idx_experiment_instances_student_id ON experiment_instances(student_id);
CREATE INDEX idx_experiment_instances_school_id ON experiment_instances(school_id);
CREATE INDEX idx_experiment_instances_course_id ON experiment_instances(course_id);
CREATE INDEX idx_experiment_instances_group_id ON experiment_instances(group_id);
CREATE INDEX idx_experiment_instances_experiment_type ON experiment_instances(experiment_type);
CREATE INDEX idx_experiment_instances_status ON experiment_instances(status);
CREATE INDEX idx_experiment_instances_last_active_at ON experiment_instances(last_active_at) WHERE status = 3;

-- instance_containers：实例容器表。
CREATE TABLE instance_containers (
    id BIGINT PRIMARY KEY,
    instance_id BIGINT NOT NULL,
    template_container_id BIGINT NOT NULL,
    container_name VARCHAR(100) NOT NULL,
    pod_name VARCHAR(200) NULL,
    container_id VARCHAR(200) NULL,
    internal_ip VARCHAR(45) NULL,
    status SMALLINT NOT NULL DEFAULT 1,
    cpu_usage VARCHAR(20) NULL,
    memory_usage VARCHAR(20) NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_instance_containers_instance_id FOREIGN KEY (instance_id) REFERENCES experiment_instances(id),
    CONSTRAINT fk_instance_containers_template_container_id FOREIGN KEY (template_container_id) REFERENCES template_containers(id)
);
CREATE INDEX idx_instance_containers_instance_id ON instance_containers(instance_id);

-- checkpoint_results：检查点结果表。
CREATE TABLE checkpoint_results (
    id BIGINT PRIMARY KEY,
    instance_id BIGINT NOT NULL,
    checkpoint_id BIGINT NOT NULL,
    student_id BIGINT NOT NULL,
    is_passed BOOLEAN NULL,
    score DECIMAL(6,2) NULL,
    check_output TEXT NULL,
    assertion_result JSONB NULL,
    teacher_comment TEXT NULL,
    graded_by BIGINT NULL,
    graded_at TIMESTAMP NULL,
    checked_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_checkpoint_results_instance_id FOREIGN KEY (instance_id) REFERENCES experiment_instances(id),
    CONSTRAINT fk_checkpoint_results_checkpoint_id FOREIGN KEY (checkpoint_id) REFERENCES template_checkpoints(id),
    CONSTRAINT fk_checkpoint_results_student_id FOREIGN KEY (student_id) REFERENCES users(id),
    CONSTRAINT fk_checkpoint_results_graded_by FOREIGN KEY (graded_by) REFERENCES users(id)
);
CREATE INDEX idx_checkpoint_results_instance_id ON checkpoint_results(instance_id);
CREATE INDEX idx_checkpoint_results_checkpoint_id ON checkpoint_results(checkpoint_id);
CREATE INDEX idx_checkpoint_results_student_id ON checkpoint_results(student_id);
CREATE UNIQUE INDEX uk_checkpoint_results ON checkpoint_results(instance_id, checkpoint_id);

-- instance_snapshots：实例快照表。
CREATE TABLE instance_snapshots (
    id BIGINT PRIMARY KEY,
    instance_id BIGINT NOT NULL,
    snapshot_type SMALLINT NOT NULL,
    snapshot_data_url VARCHAR(500) NOT NULL,
    snapshot_size BIGINT NULL,
    container_states JSONB NULL,
    sim_engine_state JSONB NULL,
    description VARCHAR(200) NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_instance_snapshots_instance_id FOREIGN KEY (instance_id) REFERENCES experiment_instances(id)
);
CREATE INDEX idx_instance_snapshots_instance_id ON instance_snapshots(instance_id);
CREATE INDEX idx_instance_snapshots_snapshot_type ON instance_snapshots(snapshot_type);

-- instance_operation_logs：实例操作日志表。
CREATE TABLE instance_operation_logs (
    id BIGINT PRIMARY KEY,
    instance_id BIGINT NOT NULL,
    student_id BIGINT NOT NULL,
    action VARCHAR(50) NOT NULL,
    target_container VARCHAR(100) NULL,
    target_scene VARCHAR(100) NULL,
    command TEXT NULL,
    command_output TEXT NULL,
    detail JSONB NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_instance_operation_logs_instance_id FOREIGN KEY (instance_id) REFERENCES experiment_instances(id),
    CONSTRAINT fk_instance_operation_logs_student_id FOREIGN KEY (student_id) REFERENCES users(id)
);
CREATE INDEX idx_instance_operation_logs_instance_id ON instance_operation_logs(instance_id);
CREATE INDEX idx_instance_operation_logs_student_id ON instance_operation_logs(student_id);
CREATE INDEX idx_instance_operation_logs_action ON instance_operation_logs(action);
CREATE INDEX idx_instance_operation_logs_created_at ON instance_operation_logs(created_at);

-- experiment_groups：实验分组表。
CREATE TABLE experiment_groups (
    id BIGINT PRIMARY KEY,
    template_id BIGINT NOT NULL,
    course_id BIGINT NOT NULL,
    group_name VARCHAR(100) NOT NULL,
    group_method SMALLINT NOT NULL,
    max_members INT NOT NULL,
    status SMALLINT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_experiment_groups_template_id FOREIGN KEY (template_id) REFERENCES experiment_templates(id),
    CONSTRAINT fk_experiment_groups_course_id FOREIGN KEY (course_id) REFERENCES courses(id)
);
CREATE INDEX idx_experiment_groups_template_id ON experiment_groups(template_id);
CREATE INDEX idx_experiment_groups_course_id ON experiment_groups(course_id);
CREATE INDEX idx_experiment_groups_status ON experiment_groups(status);

-- group_members：分组成员表。
CREATE TABLE group_members (
    id BIGINT PRIMARY KEY,
    group_id BIGINT NOT NULL,
    student_id BIGINT NOT NULL,
    role_id BIGINT NULL,
    instance_id BIGINT NULL,
    joined_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_group_members_group_id FOREIGN KEY (group_id) REFERENCES experiment_groups(id),
    CONSTRAINT fk_group_members_student_id FOREIGN KEY (student_id) REFERENCES users(id),
    CONSTRAINT fk_group_members_role_id FOREIGN KEY (role_id) REFERENCES template_roles(id),
    CONSTRAINT fk_group_members_instance_id FOREIGN KEY (instance_id) REFERENCES experiment_instances(id)
);
CREATE UNIQUE INDEX uk_group_members ON group_members(group_id, student_id);
CREATE INDEX idx_group_members_student_id ON group_members(student_id);

-- group_messages：组内消息表。
CREATE TABLE group_messages (
    id BIGINT PRIMARY KEY,
    group_id BIGINT NOT NULL,
    sender_id BIGINT NOT NULL,
    content TEXT NOT NULL,
    message_type SMALLINT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_group_messages_group_id FOREIGN KEY (group_id) REFERENCES experiment_groups(id),
    CONSTRAINT fk_group_messages_sender_id FOREIGN KEY (sender_id) REFERENCES users(id)
);
CREATE INDEX idx_group_messages_group_id ON group_messages(group_id);
CREATE INDEX idx_group_messages_created_at ON group_messages(created_at);

-- resource_quotas：资源配额表。
CREATE TABLE resource_quotas (
    id BIGINT PRIMARY KEY,
    quota_level SMALLINT NOT NULL,
    school_id BIGINT NOT NULL,
    course_id BIGINT NULL,
    max_cpu VARCHAR(20) NULL,
    max_memory VARCHAR(20) NULL,
    max_storage VARCHAR(20) NULL,
    max_concurrency INT NULL,
    max_per_student INT NOT NULL DEFAULT 2,
    used_cpu VARCHAR(20) NOT NULL DEFAULT '0',
    used_memory VARCHAR(20) NOT NULL DEFAULT '0',
    used_storage VARCHAR(20) NOT NULL DEFAULT '0',
    current_concurrency INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_resource_quotas_school_id FOREIGN KEY (school_id) REFERENCES schools(id),
    CONSTRAINT fk_resource_quotas_course_id FOREIGN KEY (course_id) REFERENCES courses(id)
);
CREATE INDEX idx_resource_quotas_school_id ON resource_quotas(school_id);
CREATE INDEX idx_resource_quotas_course_id ON resource_quotas(course_id);
CREATE UNIQUE INDEX uk_resource_quotas_school ON resource_quotas(school_id) WHERE quota_level = 1;
CREATE UNIQUE INDEX uk_resource_quotas_course ON resource_quotas(school_id, course_id) WHERE quota_level = 2 AND course_id IS NOT NULL;

-- experiment_reports：实验报告表。
CREATE TABLE experiment_reports (
    id BIGINT PRIMARY KEY,
    instance_id BIGINT NOT NULL,
    student_id BIGINT NOT NULL,
    content TEXT NULL,
    file_url VARCHAR(500) NULL,
    file_name VARCHAR(200) NULL,
    file_size BIGINT NULL,
    submitted_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_experiment_reports_instance_id FOREIGN KEY (instance_id) REFERENCES experiment_instances(id),
    CONSTRAINT fk_experiment_reports_student_id FOREIGN KEY (student_id) REFERENCES users(id)
);
CREATE INDEX idx_experiment_reports_instance_id ON experiment_reports(instance_id);
CREATE INDEX idx_experiment_reports_student_id ON experiment_reports(student_id);

-- 补充需要延后声明的外键，避免同文件内前后依赖冲突。
ALTER TABLE template_containers
    ADD CONSTRAINT fk_template_containers_role_id FOREIGN KEY (role_id) REFERENCES template_roles(id);

ALTER TABLE experiment_instances
    ADD CONSTRAINT fk_experiment_instances_group_id FOREIGN KEY (group_id) REFERENCES experiment_groups(id);

ALTER TABLE lessons
    ADD CONSTRAINT fk_lessons_experiment_id FOREIGN KEY (experiment_id) REFERENCES experiment_templates(id);

ALTER TABLE course_experiments
    ADD CONSTRAINT fk_course_experiments_experiment_id FOREIGN KEY (experiment_id) REFERENCES experiment_templates(id);
