-- ============================================
-- AI-Curton 数据库初始化脚本
-- 生成时间: 2026-04-01
-- 说明: 根据 GORM 模型定义生成的完整建库建表 SQL
-- ============================================

-- 创建数据库
DROP DATABASE IF EXISTS ai_curton;
CREATE DATABASE ai_curton CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE ai_curton;

-- ============================================
-- 1. 用户表 (users)
-- 说明: 存储系统用户信息，支持用户名和邮箱唯一性约束
-- ============================================
CREATE TABLE `users` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '用户ID',
  `username` VARCHAR(50) NOT NULL COMMENT '用户名',
  `email` VARCHAR(100) NOT NULL COMMENT '邮箱地址',
  `password_hash` VARCHAR(255) NOT NULL COMMENT '密码哈希值',
  `role` VARCHAR(20) NOT NULL DEFAULT 'creator' COMMENT '用户角色: admin, creator, viewer',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_username` (`username`),
  UNIQUE KEY `idx_email` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户表';

-- ============================================
-- 2. 工作空间表 (workspaces)
-- 说明: 支持个人和团队两种类型的工作空间
-- ============================================
CREATE TABLE `workspaces` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '工作空间ID',
  `name` VARCHAR(100) NOT NULL COMMENT '工作空间名称',
  `type` VARCHAR(20) NOT NULL DEFAULT 'personal' COMMENT '类型: personal, team',
  `owner_id` BIGINT UNSIGNED NOT NULL COMMENT '所有者用户ID',
  `description` TEXT COMMENT '工作空间描述',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_owner_id` (`owner_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='工作空间表';

-- ============================================
-- 3. 工作空间成员表 (workspace_members)
-- 说明: 管理用户与工作空间的关联关系及权限
-- ============================================
CREATE TABLE `workspace_members` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '成员关系ID',
  `workspace_id` BIGINT UNSIGNED NOT NULL COMMENT '工作空间ID',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  `role` VARCHAR(20) NOT NULL DEFAULT 'viewer' COMMENT '角色: owner, editor, viewer',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '加入时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_ws_user` (`workspace_id`, `user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='工作空间成员表';

-- ============================================
-- 4. 作品集表 (portfolios)
-- 说明: 归属于某个工作空间的作品集
-- ============================================
CREATE TABLE `portfolios` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '作品集ID',
  `workspace_id` BIGINT UNSIGNED NOT NULL COMMENT '所属工作空间ID',
  `name` VARCHAR(100) NOT NULL COMMENT '作品集名称',
  `description` TEXT COMMENT '作品集描述',
  `cover_image` VARCHAR(500) COMMENT '封面图片路径',
  `status` VARCHAR(20) NOT NULL DEFAULT 'draft' COMMENT '状态: draft, published',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_workspace_id` (`workspace_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='作品集表';

-- ============================================
-- 5. 角色模型表 (characters)
-- 说明: 用于人物一致性管理，支持参考图和属性存储
-- ============================================
CREATE TABLE `characters` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '角色ID',
  `portfolio_id` BIGINT UNSIGNED NOT NULL COMMENT '所属作品集ID',
  `name` VARCHAR(100) NOT NULL COMMENT '角色名称',
  `description` TEXT COMMENT '角色描述',
  `reference_images` JSON COMMENT '参考图路径列表(JSON数组)',
  `lora_path` VARCHAR(500) COMMENT 'LoRA模型路径',
  `attributes` JSON COMMENT '角色属性(JSON对象: 发型、服装等)',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_portfolio_id` (`portfolio_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='角色模型表';

-- ============================================
-- 6. 资源表 (assets)
-- 说明: 存储生成的图片、文本、脚本等文件
-- ============================================
CREATE TABLE `assets` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '资源ID',
  `portfolio_id` BIGINT UNSIGNED NOT NULL COMMENT '所属作品集ID',
  `type` VARCHAR(20) NOT NULL COMMENT '资源类型: image, text, script',
  `file_path` VARCHAR(500) NOT NULL COMMENT '文件存储路径',
  `metadata` JSON COMMENT '生成参数、提示词等元数据',
  `created_by` BIGINT UNSIGNED NOT NULL COMMENT '创建者用户ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  KEY `idx_portfolio_id` (`portfolio_id`),
  KEY `idx_created_by` (`created_by`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='资源表';

-- ============================================
-- 7. AI任务表 (ai_tasks)
-- 说明: 记录每次AI调用的完整生命周期
-- ============================================
CREATE TABLE `ai_tasks` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '任务ID',
  `user_id` BIGINT UNSIGNED COMMENT '用户ID',
  `portfolio_id` BIGINT UNSIGNED COMMENT '作品集ID',
  `task_type` VARCHAR(50) COMMENT '任务类型: text_gen, image_gen, character_adjust',
  `model_name` VARCHAR(50) COMMENT '模型名称: kimi, claude, copilot',
  `prompt` TEXT COMMENT '提示词',
  `status` VARCHAR(20) DEFAULT 'pending' COMMENT '状态: pending, running, completed, failed, cancelled',
  `result` JSON COMMENT '任务结果',
  `error_msg` TEXT COMMENT '错误信息',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_portfolio_id` (`portfolio_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='AI任务表';

-- ============================================
-- 8. API密钥表 (api_keys)
-- 说明: 用户API Key管理，支持用户自有Key和平台默认Key
-- ============================================
CREATE TABLE `api_keys` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'API密钥ID',
  `user_id` BIGINT UNSIGNED COMMENT '用户ID',
  `provider` VARCHAR(50) COMMENT '服务提供商: kimi, claude, copilot',
  `key_value` VARCHAR(500) COMMENT 'API密钥值(加密存储)',
  `is_default` TINYINT(1) DEFAULT 0 COMMENT '是否为该Provider的默认Key',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='API密钥表';

-- ============================================
-- 索引说明
-- ============================================
-- users 表:
--   - idx_username: 用户名唯一索引，用于登录验证
--   - idx_email: 邮箱唯一索引，用于登录验证和找回密码
--
-- workspaces 表:
--   - idx_owner_id: 所有者索引，用于查询用户创建的工作空间
--
-- workspace_members 表:
--   - idx_ws_user: 工作空间+用户联合唯一索引，防止重复加入
--
-- portfolios 表:
--   - idx_workspace_id: 工作空间索引，用于查询空间下的所有作品集
--
-- characters 表:
--   - idx_portfolio_id: 作品集索引，用于查询作品集下的所有角色
--
-- assets 表:
--   - idx_portfolio_id: 作品集索引，用于查询作品集下的所有资源
--   - idx_created_by: 创建者索引，用于查询用户创建的所有资源
--
-- ai_tasks 表:
--   - idx_user_id: 用户索引，用于查询用户的所有任务
--   - idx_portfolio_id: 作品集索引，用于查询作品集相关的所有任务
--
-- api_keys 表:
--   - idx_user_id: 用户索引，用于查询用户的所有API密钥
--
-- ============================================
-- 数据库初始化完成
-- ============================================
