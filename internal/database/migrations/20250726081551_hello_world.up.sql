CREATE SCHEMA IF NOT EXISTS `opsicle`;
CREATE TABLE IF NOT EXISTS `users` (
    `id` VARCHAR(36) PRIMARY KEY,
    `email` VARCHAR(255) NOT NULL,
    `email_verification_code` TEXT NOT NULL,
    `is_email_verified` BOOLEAN NOT NULL DEFAULT FALSE,
    `email_verified_at` DATETIME,
    `email_verified_by_user_agent` TEXT,
    `email_verified_by_ip_address` TEXT,
    `password_hash` TEXT,
    `type` VARCHAR(64) NOT NULL DEFAULT 'user',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `last_updated_at` DATETIME DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
    `is_deleted` BOOLEAN NOT NULL DEFAULT FALSE,
    `deleted_at` DATETIME,
    `is_disabled` BOOLEAN NOT NULL DEFAULT FALSE,
    `disabled_at` DATETIME
);
CREATE TABLE IF NOT EXISTS `orgs` (
    `id` VARCHAR(36) PRIMARY KEY,
    `name` VARCHAR(255) NOT NULL,
    `code` VARCHAR(32) NOT NULL UNIQUE,
    `type` VARCHAR(32) NOT NULL DEFAULT 'tenant',
    `icon` TEXT,
    `logo` TEXT,
    `motd` TEXT,
    `created_by` VARCHAR(36) NOT NULL,
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `last_updated_at` DATETIME DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
    `is_scheduled_for_deletion` BOOLEAN NOT NULL DEFAULT FALSE,
    `is_deleted` BOOLEAN NOT NULL DEFAULT FALSE,
    `deleted_at` DATETIME,
    `is_disabled` BOOLEAN NOT NULL DEFAULT FALSE,
    `disabled_at` DATETIME,
    FOREIGN KEY (created_by) REFERENCES `users`(id)
);
CREATE TABLE IF NOT EXISTS `groups` (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    org_id VARCHAR(36),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (org_id) REFERENCES `orgs`(id)
);
CREATE TABLE IF NOT EXISTS `approval_policies` (
    id VARCHAR(36) PRIMARY KEY,
    org_id VARCHAR(36) NOT NULL,
    name VARCHAR(255) NOT NULL,
    policy_json JSON NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_updated_at DATETIME DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at DATETIME,
    FOREIGN KEY (org_id) REFERENCES `orgs`(id)
);
CREATE TABLE IF NOT EXISTS `automation_templates` (
    id VARCHAR(36) PRIMARY KEY,
    org_id VARCHAR(36) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    content TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_updated_at DATETIME DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (org_id) REFERENCES `orgs`(id)
);
CREATE TABLE IF NOT EXISTS `automation_runs` (
    id VARCHAR(36) PRIMARY KEY,
    template_id VARCHAR(36) NOT NULL,
    org_id VARCHAR(36) NOT NULL,
    triggered_by VARCHAR(36) NOT NULL,
    input JSON,
    last_known_status VARCHAR(20),
    logs TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (template_id) REFERENCES `automation_templates`(id),
    FOREIGN KEY (org_id) REFERENCES `orgs`(id),
    FOREIGN KEY (triggered_by) REFERENCES `users`(id)
);
CREATE TABLE IF NOT EXISTS `groups_users` (
    group_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    added_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (group_id, user_id),
    FOREIGN KEY (group_id) REFERENCES `groups`(id),
    FOREIGN KEY (user_id) REFERENCES `users`(id)
);
CREATE TABLE IF NOT EXISTS `org_users` (
    user_id VARCHAR(36) NOT NULL,
    org_id VARCHAR(36) NOT NULL,
    `type` VARCHAR(64) NOT NULL DEFAULT 'member',
    joined_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, org_id),
    FOREIGN KEY (user_id) REFERENCES `users`(id),
    FOREIGN KEY (org_id) REFERENCES `orgs`(id)
);
CREATE TABLE IF NOT EXISTS `org_roles` (
    id VARCHAR(36) PRIMARY KEY,
    org_id VARCHAR(36) NOT NULL,
    name VARCHAR(255) NOT NULL,
    permissions JSON,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (org_id) REFERENCES `orgs`(id)
);
CREATE TABLE IF NOT EXISTS `org_role_users` (
    user_id VARCHAR(36) NOT NULL,
    role_id VARCHAR(36) NOT NULL,
    assigned_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, role_id),
    FOREIGN KEY (user_id) REFERENCES `users`(id),
    FOREIGN KEY (role_id) REFERENCES `org_roles`(id)
);
CREATE TABLE IF NOT EXISTS `org_config_db` (
    `id` VARCHAR(36) PRIMARY KEY,
    `db_hostname` TEXT NOT NULL,
    `db_port` TEXT NOT NULL,
    `db_username` TEXT NOT NULL,
    `db_password` TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS `org_config_sso` (
    `id` VARCHAR(36) PRIMARY KEY,
    `org_id` VARCHAR(36) NOT NULL,
    `type` ENUM('saml', 'oidc') NOT NULL,
    `idp_entity_id` TEXT,
    `idp_sso_url` TEXT,
    `idp_cert` TEXT,
    `idp_metadata_url` TEXT,
    `client_id` VARCHAR(255),
    `client_secret` TEXT,
    `issuer` TEXT,
    `authorization_url` TEXT,
    `token_url` TEXT,
    `userinfo_url` TEXT,
    `scopes` TEXT,
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_updated_at DATETIME DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (`org_id`) REFERENCES `orgs` (`id`) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS `org_config_sso_mapping` (
    `id` VARCHAR(36) PRIMARY KEY,
    `org_id` VARCHAR(36) NOT NULL,
    `field` VARCHAR(64) NOT NULL,
    `source` VARCHAR(255) NOT NULL,
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_updated_at DATETIME DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (`org_id`) REFERENCES `orgs` (`id`) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS `org_config_security` (
    id VARCHAR(36) PRIMARY KEY,
    org_id VARCHAR(36) NOT NULL UNIQUE,
    force_mfa BOOLEAN NOT NULL DEFAULT FALSE,
    session_timeout INT NOT NULL,
    blanket_approval_policy_id VARCHAR(36),
    last_updated_at DATETIME DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (org_id) REFERENCES `orgs`(id),
    FOREIGN KEY (blanket_approval_policy_id) REFERENCES `approval_policies`(id)
);
CREATE TABLE IF NOT EXISTS `sessions` (
    `id` VARCHAR(36) PRIMARY KEY,
    `user_id` VARCHAR(36) NOT NULL,
    `org_id` VARCHAR(36),
    `token_hash` TEXT NOT NULL,
    `user_agent` TEXT,
    `ip_address` VARCHAR(45),
    `location` VARCHAR(255),
    `login_method` VARCHAR(64),
    `issued_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `expires_at` DATETIME,
    `revoked_at` DATETIME,
    FOREIGN KEY (org_id) REFERENCES `orgs`(id)
);
CREATE TABLE IF NOT EXISTS `user_slack` (
    `id` VARCHAR(36) PRIMARY KEY,
    `user_id` VARCHAR(36) NOT NULL,
    `slack_id` VARCHAR(255) NOT NULL,
    `linked_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES `users`(id)
);
CREATE TABLE IF NOT EXISTS `user_sso` (
    `id` VARCHAR(36) PRIMARY KEY,
    `user_id` VARCHAR(36) NOT NULL,
    `provider_id` VARCHAR(255) NOT NULL,
    `raw_attributes` JSON,
    `subject` VARCHAR(255) NOT NULL,
    provisioned_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_login DATETIME,
    FOREIGN KEY (user_id) REFERENCES `users`(id),
    UNIQUE KEY `user_provider_subject` (`user_id`, `provider_id`, `subject`)
);
CREATE TABLE IF NOT EXISTS `user_sso_group` (
    id VARCHAR(36) PRIMARY KEY,
    user_sso_id VARCHAR(36) NOT NULL,
    group_name VARCHAR(255) NOT NULL,
    source VARCHAR(255),
    assigned_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_sso_id) REFERENCES `user_sso`(id)
);
CREATE TABLE IF NOT EXISTS `user_telegram` (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    telegram_id BIGINT NOT NULL,
    telegram_handle VARCHAR(255),
    linked_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES `users`(id)
);
CREATE TABLE IF NOT EXISTS `user_permissions` (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    org_id VARCHAR(36),
    permission TEXT NOT NULL,
    granted_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    granted_by VARCHAR(36),
    FOREIGN KEY (user_id) REFERENCES `users`(id),
    FOREIGN KEY (org_id) REFERENCES `orgs`(id),
    FOREIGN KEY (granted_by) REFERENCES `users`(id)
);
CREATE TABLE IF NOT EXISTS `user_profiles` (
    user_id VARCHAR(36) PRIMARY KEY,
    full_name VARCHAR(255),
    avatar_url TEXT,
    bio TEXT,
    public_email VARCHAR(255),
    public_phone VARCHAR(64),
    slack_id VARCHAR(255),
    telegram_id VARCHAR(255),
    company VARCHAR(255),
    department VARCHAR(255),
    team VARCHAR(255),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_updated_at DATETIME DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES `users`(id)
);
CREATE TABLE IF NOT EXISTS `user_mfa` (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    `secret` TEXT,
    `type` TEXT,
    `is_verified` BOOLEAN NOT NULL DEFAULT false,
    `verified_at` DATETIME,
    `config_json` JSON,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_updated_at DATETIME DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES `users`(id)
);
CREATE TABLE IF NOT EXISTS `user_login` (
    `id` VARCHAR(36) NOT NULL,
    `user_id` VARCHAR(36) NULL,
    `attempted_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `ip_address` VARCHAR(45) NULL,
    `user_agent` TEXT NULL,
    `is_pending_mfa` BOOLEAN NOT NULL DEFAULT 0,
    `expires_at` TIMESTAMP NOT NULL,
    PRIMARY KEY (`id`),
    CONSTRAINT `fk_user_login_user` FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE SET NULL ON UPDATE CASCADE
);
