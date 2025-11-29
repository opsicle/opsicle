CREATE TABLE IF NOT EXISTS `automations` (
    id VARCHAR(36) PRIMARY KEY,
    input_vars JSON,
    last_known_status ENUM(
        'created',
        'accepted',
        'rejected',
        'pending-approval',
        'pending-execution',
        'executing',
        'completed-success',
        'completed-failed'
    ) DEFAULT 'created',
    logs TEXT,
    org_id VARCHAR(36),
    template_content TEXT,
    template_id VARCHAR(36),
    template_version BIGINT,
    triggered_by VARCHAR(36),
    triggered_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    triggerer_comment TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (template_id) REFERENCES `templates`(id) ON DELETE SET NULL ON UPDATE CASCADE,
    FOREIGN KEY (org_id) REFERENCES `orgs`(id) ON DELETE SET NULL ON UPDATE CASCADE,
    FOREIGN KEY (triggered_by) REFERENCES `users`(id) ON DELETE SET NULL ON UPDATE CASCADE
);
