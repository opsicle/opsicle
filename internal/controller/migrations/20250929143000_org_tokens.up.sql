CREATE TABLE IF NOT EXISTS `org_tokens` (
    `id` VARCHAR(36) PRIMARY KEY,
    `org_id` VARCHAR(36) NOT NULL,
    `name` VARCHAR(255) NOT NULL,
    `description` TEXT,
    `api_key` TEXT NOT NULL,
    `created_by` VARCHAR(36),
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `last_updated_by` VARCHAR(36),
    `last_updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT `fk_org_tokens_org` FOREIGN KEY (`org_id`) REFERENCES `orgs`(`id`) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT `fk_org_tokens_created_by` FOREIGN KEY (`created_by`) REFERENCES `users`(`id`) ON DELETE SET NULL ON UPDATE CASCADE,
    CONSTRAINT `fk_org_tokens_last_updated_by` FOREIGN KEY (`last_updated_by`) REFERENCES `users`(`id`) ON DELETE SET NULL ON UPDATE CASCADE
);

CREATE TABLE IF NOT EXISTS `org_token_roles` (
    `id` VARCHAR(36) PRIMARY KEY,
    `org_token_id` VARCHAR(36) NOT NULL,
    `org_role_id` VARCHAR(36) NOT NULL,
    `created_by` VARCHAR(36),
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `last_updated_by` VARCHAR(36),
    `last_updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY `uk_org_token_roles_token_role` (`org_token_id`, `org_role_id`),
    CONSTRAINT `fk_org_token_roles_token` FOREIGN KEY (`org_token_id`) REFERENCES `org_tokens`(`id`) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT `fk_org_token_roles_role` FOREIGN KEY (`org_role_id`) REFERENCES `org_roles`(`id`) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT `fk_org_token_roles_created_by` FOREIGN KEY (`created_by`) REFERENCES `users`(`id`) ON DELETE SET NULL ON UPDATE CASCADE,
    CONSTRAINT `fk_org_token_roles_last_updated_by` FOREIGN KEY (`last_updated_by`) REFERENCES `users`(`id`) ON DELETE SET NULL ON UPDATE CASCADE
);
