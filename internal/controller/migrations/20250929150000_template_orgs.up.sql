CREATE TABLE IF NOT EXISTS `template_orgs` (
    `id` VARCHAR(36) PRIMARY KEY,
    `template_id` VARCHAR(36) NOT NULL,
    `org_id` VARCHAR(36) NOT NULL,
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `created_by` VARCHAR(36),
    `last_updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `last_updated_by` VARCHAR(36),
    UNIQUE KEY `uk_template_orgs_template_org` (`template_id`, `org_id`),
    CONSTRAINT `fk_template_orgs_template` FOREIGN KEY (`template_id`) REFERENCES `templates`(`id`) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT `fk_template_orgs_org` FOREIGN KEY (`org_id`) REFERENCES `orgs`(`id`) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT `fk_template_orgs_created_by` FOREIGN KEY (`created_by`) REFERENCES `users`(`id`) ON DELETE SET NULL ON UPDATE CASCADE,
    CONSTRAINT `fk_template_orgs_updated_by` FOREIGN KEY (`last_updated_by`) REFERENCES `users`(`id`) ON DELETE SET NULL ON UPDATE CASCADE
);
