CREATE TABLE IF NOT EXISTS `templates` (
    `id` VARCHAR(36) PRIMARY KEY,
    `name` VARCHAR(255) NOT NULL,
    `org_id` VARCHAR(36),
    `description` TEXT,
    `version` BIGINT NOT NULL,
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `created_by` VARCHAR(36),
    `last_updated_at` DATETIME DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
    `last_updated_by` VARCHAR(36),
    FOREIGN KEY (org_id) REFERENCES `orgs`(id) ON DELETE CASCADE ON UPDATE CASCADE,
    FOREIGN KEY (created_by) REFERENCES `users`(id) ON DELETE SET NULL ON UPDATE CASCADE,
    FOREIGN KEY (last_updated_by) REFERENCES `users`(id) ON DELETE SET NULL ON UPDATE CASCADE
);
CREATE TABLE IF NOT EXISTS `template_users` (
    `template_id` VARCHAR(36) NOT NULL,
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
    PRIMARY KEY (template_id, user_id),
    FOREIGN KEY (user_id) REFERENCES `users`(id) ON DELETE CASCADE ON UPDATE CASCADE,
    FOREIGN KEY (created_by) REFERENCES `users`(id) ON DELETE SET NULL ON UPDATE CASCADE,
    FOREIGN KEY (last_updated_by) REFERENCES `users`(id) ON DELETE SET NULL ON UPDATE CASCADE,
    FOREIGN KEY (template_id) REFERENCES `templates`(id) ON DELETE CASCADE ON UPDATE CASCADE
);
CREATE TABLE IF NOT EXISTS `template_user_invitations` (
    `id`              VARCHAR(36) NOT NULL,
    `inviter_id`      VARCHAR(36) NOT NULL,
    `acceptor_id`     VARCHAR(36) NULL,
    `acceptor_email`  VARCHAR(255) NULL,
    `template_id`     VARCHAR(36) NOT NULL,
    `join_code`       VARCHAR(32) NOT NULL,
    `can_view` BOOLEAN NOT NULL DEFAULT FALSE,
    `can_execute` BOOLEAN NOT NULL DEFAULT FALSE,
    `can_update` BOOLEAN NOT NULL DEFAULT FALSE,
    `can_delete` BOOLEAN NOT NULL DEFAULT FALSE,
    `can_invite` BOOLEAN NOT NULL DEFAULT FALSE,
    `created_at`      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `last_updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    CONSTRAINT `fk_template_user_invitations_inviter` FOREIGN KEY (`inviter_id`) REFERENCES `users` (`id`) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT `fk_template_user_invitations_acceptor` FOREIGN KEY (`acceptor_id`) REFERENCES `users` (`id`) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT `fk_template_user_invitations_template_id` FOREIGN KEY (`template_id`) REFERENCES `templates` (`id`) ON DELETE CASCADE ON UPDATE CASCADE,
    UNIQUE KEY `uk_template_user_invitations_join_code` (`join_code`),
    UNIQUE KEY `uk_template_user_invitations_email_org` (`acceptor_email`, `template_id`),
    UNIQUE KEY `uk_template_user_invitations_user_org` (`acceptor_id`, `template_id`)
);

CREATE TABLE IF NOT EXISTS `template_versions` (
    `template_id` VARCHAR(36),
    `version` BIGINT NOT NULL,
    `content` TEXT NOT NULL,
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `created_by` VARCHAR(36),
    FOREIGN KEY (template_id) REFERENCES `templates`(id) ON DELETE CASCADE ON UPDATE CASCADE,
    FOREIGN KEY (created_by) REFERENCES `users`(id) ON DELETE SET NULL ON UPDATE CASCADE
);
