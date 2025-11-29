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

CREATE TABLE IF NOT EXISTS `org_roles` (
    `id` VARCHAR(36) PRIMARY KEY,
    `org_id` VARCHAR(36) NOT NULL,
    `name` VARCHAR(255) NOT NULL,
    `created_by` VARCHAR(36),
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `last_updated_at` DATETIME DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (created_by) REFERENCES `users`(id) ON DELETE SET NULL ON UPDATE CASCADE,
    FOREIGN KEY (org_id) REFERENCES `orgs`(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS `org_role_permissions` (
    `id` VARCHAR(36) PRIMARY KEY,
    `org_role_id` VARCHAR(36) NOT NULL,
    `resource` VARCHAR(64) NOT NULL,
    `allows` BIGINT UNSIGNED NOT NULL,
    `denys` BIGINT UNSIGNED NOT NULL,
    FOREIGN KEY (org_role_id) REFERENCES `org_roles`(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS `org_users` (
    `user_id` VARCHAR(36) NOT NULL,
    `org_id` VARCHAR(36) NOT NULL,
    `type` VARCHAR(64) NOT NULL DEFAULT 'member',
    `joined_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, org_id),
    FOREIGN KEY (user_id) REFERENCES `users`(id) ON DELETE CASCADE ON UPDATE CASCADE,
    FOREIGN KEY (org_id) REFERENCES `orgs`(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS `org_user_invitations` (
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
CREATE TABLE IF NOT EXISTS `org_ca` (
    `id` VARCHAR(36) PRIMARY KEY,
    `org_id` VARCHAR(36),
    `cert_b64` TEXT NOT NULL,
    `private_key_b64` TEXT NOT NULL,
    `is_deactivated` BOOLEAN NOT NULL DEFAULT FALSE,
    `created_at`      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `expires_at` TIMESTAMP NOT NULL,
    FOREIGN KEY (org_id) REFERENCES `orgs`(id) ON DELETE SET NULL ON UPDATE CASCADE
);
CREATE TABLE IF NOT EXISTS `org_workers` (
    `id` VARCHAR(36) PRIMARY KEY,
    `org_id` VARCHAR(36),
    `token` TEXT NOT NULL,
    `cert_b64` TEXT NOT NULL,
    `private_key_b64` TEXT NOT NULL,
    `is_deactivated` BOOLEAN NOT NULL DEFAULT FALSE,
    `tags` TEXT NULL,
    `created_at`      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `expires_at` TIMESTAMP NOT NULL,
    FOREIGN KEY (org_id) REFERENCES `orgs`(id) ON DELETE SET NULL ON UPDATE CASCADE
);
