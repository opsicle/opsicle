CREATE TABLE IF NOT EXISTS `users` (
    id            VARCHAR(36) PRIMARY KEY,
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash TEXT,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME DEFAULT NULL
);

CREATE TABLE IF NOT EXISTS `orgs` (
    id            VARCHAR(36) PRIMARY KEY,
    name          VARCHAR(255) NOT NULL,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME DEFAULT NULL,
    is_deleted    BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at    DATETIME,
    is_disabled   BOOLEAN NOT NULL DEFAULT FALSE,
    disabled_at   DATETIME
);

CREATE TABLE IF NOT EXISTS `groups` (
    id                VARCHAR(36) PRIMARY KEY,
    name              VARCHAR(255) NOT NULL,
    description       TEXT,
    org_id            VARCHAR(36),
    created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (org_id) REFERENCES `orgs`(id)
);

CREATE TABLE IF NOT EXISTS `approval_policies` (
    id              VARCHAR(36) PRIMARY KEY,
    org_id          VARCHAR(36) NOT NULL,
    name            VARCHAR(255) NOT NULL,
    policy_json     JSON NOT NULL,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT NULL,
    is_deleted  BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at  DATETIME,
    FOREIGN KEY (org_id) REFERENCES `orgs`(id)
);

CREATE TABLE IF NOT EXISTS `automation_templates` (
    id              VARCHAR(36) PRIMARY KEY,
    org_id          VARCHAR(36) NOT NULL,
    name            VARCHAR(255) NOT NULL,
    description     TEXT,
    content         TEXT NOT NULL,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT NULL,
    FOREIGN KEY (org_id) REFERENCES `orgs`(id)
);

CREATE TABLE IF NOT EXISTS `automation_runs` (
    id                 VARCHAR(36) PRIMARY KEY,
    template_id        VARCHAR(36) NOT NULL,
    org_id             VARCHAR(36) NOT NULL,
    triggered_by       VARCHAR(36) NOT NULL,
    input              JSON,
    last_known_status  VARCHAR(20),
    logs               TEXT,
    created_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (template_id) REFERENCES `automation_templates`(id),
    FOREIGN KEY (org_id) REFERENCES `orgs`(id),
    FOREIGN KEY (triggered_by) REFERENCES `users`(id)
);

CREATE TABLE IF NOT EXISTS `groups_users` (
    group_id     VARCHAR(36) NOT NULL,
    user_id      VARCHAR(36) NOT NULL,
    added_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (group_id, user_id),
    FOREIGN KEY (group_id) REFERENCES `groups`(id),
    FOREIGN KEY (user_id) REFERENCES `users`(id)
);

CREATE TABLE IF NOT EXISTS `org_users` (
    user_id         VARCHAR(36) NOT NULL,
    org_id          VARCHAR(36) NOT NULL,
    joined_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, org_id),
    FOREIGN KEY (user_id) REFERENCES `users`(id),
    FOREIGN KEY (org_id) REFERENCES `orgs`(id)
);

CREATE TABLE IF NOT EXISTS `org_roles` (
    id              VARCHAR(36) PRIMARY KEY,
    org_id          VARCHAR(36) NOT NULL,
    name            VARCHAR(255) NOT NULL,
    permissions     JSON,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (org_id) REFERENCES `orgs`(id)
);

CREATE TABLE IF NOT EXISTS `org_role_users` (
    user_id    VARCHAR(36) NOT NULL,
    role_id    VARCHAR(36) NOT NULL,
    assigned_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, role_id),
    FOREIGN KEY (user_id) REFERENCES `users`(id),
    FOREIGN KEY (role_id) REFERENCES `org_roles`(id)
);

CREATE TABLE IF NOT EXISTS `org_config_sso` (
    id                VARCHAR(36) PRIMARY KEY,
    org_id            VARCHAR(36) NOT NULL UNIQUE,
    idp_name          VARCHAR(255) NOT NULL,
    config_json       JSON NOT NULL,
    updated_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (org_id) REFERENCES `orgs`(id)
);


CREATE TABLE IF NOT EXISTS `org_config_sso_mapping` (
    id           VARCHAR(36) PRIMARY KEY,
    org_id       VARCHAR(36) NOT NULL,
    idp_group    VARCHAR(255) NOT NULL,
    role_id      VARCHAR(36) NOT NULL,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (org_id) REFERENCES `orgs`(id),
    FOREIGN KEY (role_id) REFERENCES `org_roles`(id)
);

CREATE TABLE IF NOT EXISTS `org_config_security` (
    id                           VARCHAR(36) PRIMARY KEY,
    org_id                       VARCHAR(36) NOT NULL UNIQUE,
    force_mfa                    BOOLEAN NOT NULL DEFAULT FALSE,
    session_timeout              INT NOT NULL,
    blanket_approval_policy_id   VARCHAR(36),
    updated_at                   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (org_id) REFERENCES `orgs`(id),
    FOREIGN KEY (blanket_approval_policy_id) REFERENCES `approval_policies`(id)
);

CREATE TABLE IF NOT EXISTS `sessions` (
    id             VARCHAR(36) PRIMARY KEY,
    session_token  VARCHAR(512) NOT NULL,
    user_agent     TEXT,
    ip_address     VARCHAR(64),
    location       VARCHAR(255),
    login_method   VARCHAR(64),
    issued_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at     DATETIME,
    revoked_at     DATETIME
);

CREATE TABLE IF NOT EXISTS `user_sessions` (
    user_id    VARCHAR(36) NOT NULL,
    session_id VARCHAR(36) NOT NULL,
    PRIMARY KEY (user_id, session_id),
    FOREIGN KEY (user_id) REFERENCES `users`(id),
    FOREIGN KEY (session_id) REFERENCES `sessions`(id)
);

CREATE TABLE IF NOT EXISTS `user_slack` (
    id            VARCHAR(36) PRIMARY KEY,
    user_id       VARCHAR(36) NOT NULL,
    slack_id      VARCHAR(255) NOT NULL,
    slack_team_id VARCHAR(255),
    linked_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES `users`(id)
);

CREATE TABLE IF NOT EXISTS `user_sso` (
    id              VARCHAR(36) PRIMARY KEY,
    user_id         VARCHAR(36) NOT NULL,
    idp_name        VARCHAR(255) NOT NULL,
    idp_subject     VARCHAR(255) NOT NULL,
    raw_attributes  JSON,
    provisioned_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES `users`(id),
    UNIQUE KEY uniq_user_idp (idp_name, idp_subject)
);

CREATE TABLE IF NOT EXISTS `user_sso_group` (
    id              VARCHAR(36) PRIMARY KEY,
    user_sso_id     VARCHAR(36) NOT NULL,
    group_name      VARCHAR(255) NOT NULL,
    source          VARCHAR(255),
    assigned_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_sso_id) REFERENCES `user_sso`(id)
);

CREATE TABLE IF NOT EXISTS `user_telegram` (
    id              VARCHAR(36) PRIMARY KEY,
    user_id         VARCHAR(36) NOT NULL,
    telegram_id     BIGINT NOT NULL,
    telegram_handle VARCHAR(255),
    linked_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES `users`(id)
);

CREATE TABLE IF NOT EXISTS `user_permissions` (
    id              VARCHAR(36) PRIMARY KEY,
    user_id         VARCHAR(36) NOT NULL,
    org_id          VARCHAR(36),
    permission      VARCHAR(255) NOT NULL,
    granted_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    granted_by      VARCHAR(36),
    FOREIGN KEY (user_id) REFERENCES `users`(id),
    FOREIGN KEY (org_id) REFERENCES `orgs`(id),
    FOREIGN KEY (granted_by) REFERENCES `users`(id),
    UNIQUE KEY uniq_user_org_permission (user_id, org_id, permission)
);

CREATE TABLE IF NOT EXISTS `user_profiles` (
    user_id           VARCHAR(36) PRIMARY KEY,
    full_name         VARCHAR(255),
    avatar_url        TEXT,
    bio               TEXT,
    public_email      VARCHAR(255),
    public_phone      VARCHAR(64),
    slack_messaging_id VARCHAR(255),
    company           VARCHAR(255),
    department        VARCHAR(255),
    team              VARCHAR(255),
    created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        DATETIME DEFAULT NULL,
    FOREIGN KEY (user_id) REFERENCES `users`(id)
);

CREATE TABLE IF NOT EXISTS `user_mfa` (
    id              VARCHAR(36) PRIMARY KEY,
    user_id         VARCHAR(36) NOT NULL,
    secret          TEXT,
    config_json     JSON,
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT NULL,
    FOREIGN KEY (user_id) REFERENCES `users`(id)
);
