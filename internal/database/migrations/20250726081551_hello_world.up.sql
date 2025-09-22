CREATE SCHEMA IF NOT EXISTS `opsicle`;
CREATE TABLE IF NOT EXISTS `users` (
    `id` VARCHAR(36) PRIMARY KEY,
    `email` VARCHAR(255) NOT NULL UNIQUE,
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
    `created_by` VARCHAR(36),
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `last_updated_at` DATETIME DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
    `is_scheduled_for_deletion` BOOLEAN NOT NULL DEFAULT FALSE,
    `is_deleted` BOOLEAN NOT NULL DEFAULT FALSE,
    `deleted_at` DATETIME,
    `is_disabled` BOOLEAN NOT NULL DEFAULT FALSE,
    `disabled_at` DATETIME,
    FOREIGN KEY (created_by) REFERENCES `users`(id) ON DELETE SET NULL ON UPDATE CASCADE
);
CREATE TABLE IF NOT EXISTS `org_ca` (
    `id` VARCHAR(36) PRIMARY KEY,
    `org_id` VARCHAR(36) NOT NULL,
    `cert_b64` TEXT NOT NULL,
    `private_key_b64` TEXT NOT NULL,
    `is_deactivated` BOOLEAN NOT NULL DEFAULT FALSE,
    `created_at`      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `expires_at` TIMESTAMP NOT NULL,
    FOREIGN KEY (org_id) REFERENCES `orgs`(id) ON DELETE DELETE ON UPDATE CASCADE
);
CREATE TABLE IF NOT EXISTS `org_worker_tokens` (
    `id` VARCHAR(36) PRIMARY KEY,
    `org_id` VARCHAR(36) NOT NULL,
    `token` TEXT NOT NULL,
    `cert_b64` TEXT NOT NULL,
    `private_key_b64` TEXT NOT NULL,
    `is_deactivated` BOOLEAN NOT NULL DEFAULT FALSE,
    `created_at`      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `expires_at` TIMESTAMP NOT NULL,
    FOREIGN KEY (org_id) REFERENCES `orgs`(id) ON DELETE DELETE ON UPDATE CASCADE
);
CREATE TABLE `org_user_invitations` (
    `id`              VARCHAR(36) NOT NULL,
    `inviter_id`      VARCHAR(36) NOT NULL,
    `acceptor_id`     VARCHAR(36) NULL,
    `acceptor_email`  VARCHAR(255) NULL,
    `org_id`          VARCHAR(36) NOT NULL,
    `join_code`       VARCHAR(32) NOT NULL,
    `type`            VARCHAR(64) NOT NULL DEFAULT 'member',
    `created_at`      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `last_updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (`id`),
    CONSTRAINT `fk_org_user_invitations_inviter` FOREIGN KEY (`inviter_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
    CONSTRAINT `fk_org_user_invitations_acceptor` FOREIGN KEY (`acceptor_id`) REFERENCES `users` (`id`) ON DELETE SET NULL,
    CONSTRAINT `fk_org_user_invitations_org` FOREIGN KEY (`org_id`) REFERENCES `orgs` (`id`) ON DELETE CASCADE,
    UNIQUE KEY `uk_org_user_invitations_join_code` (`join_code`),
    UNIQUE KEY `uk_org_user_invitations_email_org` (`acceptor_email`, `org_id`),
    UNIQUE KEY `uk_org_user_invitations_user_org` (`acceptor_id`, `org_id`)
);
CREATE TABLE IF NOT EXISTS `groups` (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    org_id VARCHAR(36),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (org_id) REFERENCES `orgs`(id) ON DELETE CASCADE ON UPDATE CASCADE
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
    FOREIGN KEY (org_id) REFERENCES `orgs`(id) ON DELETE CASCADE ON UPDATE CASCADE
);
CREATE TABLE IF NOT EXISTS `automation_templates` (
    `id` VARCHAR(36) PRIMARY KEY,
    `name` VARCHAR(255) NOT NULL,
    `description` TEXT,
    `version` BIGINT NOT NULL,
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `created_by` VARCHAR(36),
    `last_updated_at` DATETIME DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
    `last_updated_by` VARCHAR(36),
    FOREIGN KEY (created_by) REFERENCES `users`(id) ON DELETE SET NULL ON UPDATE CASCADE,
    FOREIGN KEY (last_updated_by) REFERENCES `users`(id) ON DELETE SET NULL ON UPDATE CASCADE
);
CREATE TABLE IF NOT EXISTS `automation_template_versions` (
    `automation_template_id` VARCHAR(36),
    `version` BIGINT NOT NULL,
    `content` TEXT NOT NULL,
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `created_by` VARCHAR(36),
    FOREIGN KEY (automation_template_id) REFERENCES `automation_templates`(id) ON DELETE CASCADE ON UPDATE CASCADE,
    FOREIGN KEY (created_by) REFERENCES `users`(id) ON DELETE SET NULL ON UPDATE CASCADE
);
CREATE TABLE IF NOT EXISTS `automation_template_users` (
    `automation_template_id` VARCHAR(36) NOT NULL,
    `user_id` VARCHAR(36) NOT NULL,
    `can_view` BOOLEAN NOT NULL DEFAULT FALSE,
    `can_execute` BOOLEAN NOT NULL DEFAULT FALSE,
    `can_update` BOOLEAN NOT NULL DEFAULT FALSE,
    `can_delete` BOOLEAN NOT NULL DEFAULT FALSE,
    `can_invite` BOOLEAN NOT NULL DEFAULT FALSE,
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `created_by` VARCHAR(36),
    `last_updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `last_updated_by` VARCHAR(36),
    PRIMARY KEY (automation_template_id, user_id),
    FOREIGN KEY (user_id) REFERENCES `users`(id) ON DELETE CASCADE ON UPDATE CASCADE,
    FOREIGN KEY (created_by) REFERENCES `users`(id) ON DELETE SET NULL ON UPDATE CASCADE,
    FOREIGN KEY (last_updated_by) REFERENCES `users`(id) ON DELETE SET NULL ON UPDATE CASCADE,
    FOREIGN KEY (automation_template_id) REFERENCES `automation_templates`(id) ON DELETE CASCADE ON UPDATE CASCADE,
);
CREATE TABLE IF NOT EXISTS `automation_template_user_invitations` (
    `id`              VARCHAR(36) NOT NULL,
    `inviter_id`      VARCHAR(36) NOT NULL,
    `acceptor_id`     VARCHAR(36) NULL,
    `acceptor_email`  VARCHAR(255) NULL,
    `automation_template_id`          VARCHAR(36) NOT NULL,
    `join_code`       VARCHAR(32) NOT NULL,
    `can_view` BOOLEAN NOT NULL DEFAULT FALSE,
    `can_execute` BOOLEAN NOT NULL DEFAULT FALSE,
    `can_update` BOOLEAN NOT NULL DEFAULT FALSE,
    `can_delete` BOOLEAN NOT NULL DEFAULT FALSE,
    `can_invite` BOOLEAN NOT NULL DEFAULT FALSE,
    `created_at`      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `last_updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (`id`),
    CONSTRAINT `fk_automation_template_user_invitations_inviter` FOREIGN KEY (`inviter_id`) REFERENCES `users` (`id`) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT `fk_automation_template_user_invitations_acceptor` FOREIGN KEY (`acceptor_id`) REFERENCES `users` (`id`) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT `fk_automation_template_user_invitations_automation_template_id` FOREIGN KEY (`automation_template_id`) REFERENCES `automation_templates` (`id`) ON DELETE CASCADE ON UPDATE CASCADE,
    UNIQUE KEY `uk_automation_template_user_invitations_join_code` (`join_code`),
    UNIQUE KEY `uk_automation_template_user_invitations_email_org` (`acceptor_email`, `automation_template_id`),
    UNIQUE KEY `uk_automation_template_user_invitations_user_org` (`acceptor_id`, `automation_template_id`)
);
CREATE TABLE IF NOT EXISTS `automations` (
    id VARCHAR(36) PRIMARY KEY,
    template_id VARCHAR(36),
    template_content TEXT,
    template_version BIGINT,
    org_id VARCHAR(36),
    triggered_by VARCHAR(36),
    triggered_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    triggerer_comment TEXT,
    FOREIGN KEY (template_id) REFERENCES `automation_templates`(id) ON DELETE SET NULL ON UPDATE CASCADE,
    FOREIGN KEY (org_id) REFERENCES `orgs`(id) ON DELETE SET NULL ON UPDATE CASCADE,
    FOREIGN KEY (triggered_by) REFERENCES `users`(id) ON DELETE SET NULL ON UPDATE CASCADE
);
CREATE TABLE IF NOT EXISTS `automation_runs` (
    automation_id VARCHAR(36) NOT NULL,
    input_vars JSON,
    last_known_status VARCHAR(20),
    logs TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (automation_id) REFERENCES `automations`(id) ON DELETE CASCADE ON UPDATE CASCADE
);
CREATE TABLE IF NOT EXISTS `groups_users` (
    group_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    added_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (group_id, user_id),
    FOREIGN KEY (group_id) REFERENCES `groups`(id) ON DELETE CASCADE ON UPDATE CASCADE,
    FOREIGN KEY (user_id) REFERENCES `users`(id) ON DELETE CASCADE ON UPDATE CASCADE
);
CREATE TABLE IF NOT EXISTS `org_users` (
    user_id VARCHAR(36) NOT NULL,
    org_id VARCHAR(36) NOT NULL,
    `type` VARCHAR(64) NOT NULL DEFAULT 'member',
    joined_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, org_id),
    FOREIGN KEY (user_id) REFERENCES `users`(id) ON DELETE CASCADE ON UPDATE CASCADE,
    FOREIGN KEY (org_id) REFERENCES `orgs`(id) ON DELETE CASCADE ON UPDATE CASCADE
);
CREATE TABLE IF NOT EXISTS `org_roles` (
    id VARCHAR(36) PRIMARY KEY,
    org_id VARCHAR(36) NOT NULL,
    name VARCHAR(255) NOT NULL,
    permissions JSON,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (org_id) REFERENCES `orgs`(id) ON DELETE CASCADE ON UPDATE CASCADE
);
CREATE TABLE IF NOT EXISTS `org_role_users` (
    user_id VARCHAR(36) NOT NULL,
    role_id VARCHAR(36) NOT NULL,
    assigned_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, role_id),
    FOREIGN KEY (user_id) REFERENCES `users`(id) ON DELETE CASCADE ON UPDATE CASCADE,
    FOREIGN KEY (role_id) REFERENCES `org_roles`(id) ON DELETE CASCADE ON UPDATE CASCADE
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
    FOREIGN KEY (`org_id`) REFERENCES `orgs` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
);
CREATE TABLE IF NOT EXISTS `org_config_sso_mapping` (
    `id` VARCHAR(36) PRIMARY KEY,
    `org_id` VARCHAR(36) NOT NULL,
    `field` VARCHAR(64) NOT NULL,
    `source` VARCHAR(255) NOT NULL,
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_updated_at DATETIME DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (`org_id`) REFERENCES `orgs` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
);
CREATE TABLE IF NOT EXISTS `org_config_security` (
    id VARCHAR(36) PRIMARY KEY,
    org_id VARCHAR(36) NOT NULL UNIQUE,
    force_mfa BOOLEAN NOT NULL DEFAULT FALSE,
    session_timeout INT NOT NULL,
    blanket_approval_policy_id VARCHAR(36),
    last_updated_at DATETIME DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (org_id) REFERENCES `orgs`(id) ON DELETE CASCADE ON UPDATE CASCADE,
    FOREIGN KEY (blanket_approval_policy_id) REFERENCES `approval_policies`(id)  ON DELETE CASCADE
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
    FOREIGN KEY (org_id) REFERENCES `orgs`(id) ON DELETE CASCADE ON UPDATE CASCADE
);
CREATE TABLE IF NOT EXISTS `user_slack` (
    `id` VARCHAR(36) PRIMARY KEY,
    `user_id` VARCHAR(36) NOT NULL,
    `slack_id` VARCHAR(255) NOT NULL,
    `linked_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES `users`(id) ON DELETE CASCADE ON UPDATE CASCADE
);
CREATE TABLE IF NOT EXISTS `user_sso` (
    `id` VARCHAR(36) PRIMARY KEY,
    `user_id` VARCHAR(36) NOT NULL,
    `provider_id` VARCHAR(255) NOT NULL,
    `raw_attributes` JSON,
    `subject` VARCHAR(255) NOT NULL,
    provisioned_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_login DATETIME,
    FOREIGN KEY (user_id) REFERENCES `users`(id) ON DELETE CASCADE ON UPDATE CASCADE,
    UNIQUE KEY `user_provider_subject` (`user_id`, `provider_id`, `subject`)
);
CREATE TABLE IF NOT EXISTS `user_sso_group` (
    id VARCHAR(36) PRIMARY KEY,
    user_sso_id VARCHAR(36) NOT NULL,
    group_name VARCHAR(255) NOT NULL,
    source VARCHAR(255),
    assigned_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_sso_id) REFERENCES `user_sso`(id) ON DELETE CASCADE ON UPDATE CASCADE
);
CREATE TABLE IF NOT EXISTS `user_telegram` (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    telegram_id BIGINT NOT NULL,
    telegram_handle VARCHAR(255),
    linked_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES `users`(id) ON DELETE CASCADE ON UPDATE CASCADE
);
CREATE TABLE IF NOT EXISTS `user_permissions` (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    org_id VARCHAR(36),
    permission TEXT NOT NULL,
    granted_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    granted_by VARCHAR(36),
    FOREIGN KEY (user_id) REFERENCES `users`(id) ON DELETE CASCADE ON UPDATE CASCADE,
    FOREIGN KEY (org_id) REFERENCES `orgs`(id) ON DELETE CASCADE ON UPDATE CASCADE,
    FOREIGN KEY (granted_by) REFERENCES `users`(id) ON DELETE SET NULL ON UPDATE CASCADE
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
    FOREIGN KEY (user_id) REFERENCES `users`(id) ON DELETE CASCADE ON UPDATE CASCADE
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
    FOREIGN KEY (user_id) REFERENCES `users`(id) ON DELETE CASCADE ON UPDATE CASCADE
);
CREATE TABLE IF NOT EXISTS `user_login` (
    `id` VARCHAR(36) NOT NULL,
    `user_id` VARCHAR(36) NULL,
    `attempted_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `ip_address` VARCHAR(45) NULL,
    `user_agent` TEXT NULL,
    `is_pending_mfa` BOOLEAN NOT NULL DEFAULT 0,
    `expires_at` TIMESTAMP NOT NULL,
    `status` TEXT NOT NULL,
    PRIMARY KEY (`id`),
    CONSTRAINT `fk_user_login_user` FOREIGN KEY (`user_id`) REFERENCES `users`(`id`)  ON DELETE CASCADE ON UPDATE CASCADE
);
CREATE TABLE IF NOT EXISTS `user_password_reset` (
    `id` VARCHAR(36) NOT NULL,
    `user_id` VARCHAR(36) NULL,
    `attempted_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `ip_address` VARCHAR(45) NULL,
    `user_agent` TEXT NULL,
    `verification_code` TEXT,
    `expires_at` TIMESTAMP NOT NULL,
    `status` TEXT NOT NULL,
    PRIMARY KEY (`id`),
    CONSTRAINT `fk_user_password_reset_user` FOREIGN KEY (`user_id`) REFERENCES `users`(`id`)  ON DELETE CASCADE ON UPDATE CASCADE
);
