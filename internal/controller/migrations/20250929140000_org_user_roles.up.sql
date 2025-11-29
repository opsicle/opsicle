CREATE TABLE IF NOT EXISTS `org_user_roles` (
    `user_id` VARCHAR(36) NOT NULL,
    `org_id` VARCHAR(36) NOT NULL,
    `org_role_id` VARCHAR(36) NOT NULL,
    `assigned_by` VARCHAR(36) NULL,
    `assigned_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`user_id`, `org_id`, `org_role_id`),
    CONSTRAINT `fk_org_user_roles_user_org` FOREIGN KEY (`user_id`, `org_id`) REFERENCES `org_users`(`user_id`, `org_id`) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT `fk_org_user_roles_role` FOREIGN KEY (`org_role_id`) REFERENCES `org_roles`(`id`) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT `fk_org_user_roles_assigned_by` FOREIGN KEY (`assigned_by`) REFERENCES `users`(`id`) ON DELETE SET NULL ON UPDATE CASCADE
);
