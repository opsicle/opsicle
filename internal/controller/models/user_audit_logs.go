package models

import (
	"fmt"
	"opsicle/internal/audit"
	"time"
)

type ListAuditLogsV1Opts struct {
	Timestamp time.Time
	Limit     int64
	Reverse   bool
}

func (u *User) ListAuditLogsV1(opts ListAuditLogsV1Opts) (AuditLogs, error) {
	rawAuditLogs, err := audit.GetByEntity(u.GetId(), audit.UserEntity, opts.Timestamp, opts.Limit, opts.Reverse)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs: %w", err)
	}
	auditLogs := AuditLogs{}
	for _, auditLog := range rawAuditLogs {
		auditLogs = append(auditLogs, AuditLog{
			EntityId:     auditLog.EntityId,
			EntityType:   string(auditLog.EntityType),
			Verb:         string(auditLog.Verb),
			ResourceId:   auditLog.ResourceId,
			ResourceType: string(auditLog.ResourceType),
			Status:       string(auditLog.Status),
			FieldId:      auditLog.FieldId,
			FieldType:    string(auditLog.FieldType),
			SrcIp:        auditLog.SrcIp,
			SrcUa:        auditLog.SrcUa,
			DstHost:      auditLog.DstHost,
			Timestamp:    auditLog.Timestamp,
			Data:         auditLog.Data,
		})
	}
	return auditLogs, nil
}
