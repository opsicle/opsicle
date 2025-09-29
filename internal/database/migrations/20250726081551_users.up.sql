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
CREATE TABLE IF NOT EXISTS `user_sessions` (
    `id` VARCHAR(36) PRIMARY KEY,
    `user_id` VARCHAR(36) NOT NULL,
    `token_hash` TEXT NOT NULL,
    `user_agent` TEXT,
    `ip_address` VARCHAR(45),
    `location` VARCHAR(255),
    `login_method` VARCHAR(64),
    `issued_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `expires_at` DATETIME,
    `revoked_at` DATETIME,
    FOREIGN KEY (user_id) REFERENCES `users`(id) ON DELETE CASCADE ON UPDATE CASCADE
);
CREATE TABLE IF NOT EXISTS `user_slack` (
    `id` VARCHAR(36) PRIMARY KEY,
    `user_id` VARCHAR(36) NOT NULL,
    `slack_id` VARCHAR(255) NOT NULL,
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `last_updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES `users`(id) ON DELETE CASCADE ON UPDATE CASCADE
);
CREATE TABLE IF NOT EXISTS `user_telegram` (
    `id` VARCHAR(36) PRIMARY KEY,
    `user_id` VARCHAR(36) NOT NULL,
    `telegram_id` BIGINT NOT NULL,
    `telegram_handle` VARCHAR(255),
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `last_updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES `users`(id) ON DELETE CASCADE ON UPDATE CASCADE
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
