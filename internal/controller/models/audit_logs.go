package models

import "time"

type AuditLogs []AuditLog

type AuditLog struct {
	EntityId     string         `bson:"entityId"`
	EntityType   string         `bson:"entityType"`
	Verb         string         `bson:"verb"`
	ResourceId   string         `bson:"resourceId,omitempty"`
	ResourceType string         `bson:"resourceType,omitempty"`
	Status       string         `bson:"status,omitempty"`
	FieldId      *string        `bson:"fieldId,omitempty"`
	FieldType    string         `bson:"fieldType,omitempty"`
	SrcIp        *string        `bson:"srcIp,omitempty"`
	SrcUa        *string        `bson:"srcUa,omitempty"`
	DstHost      *string        `bson:"dstHost,omitempty"`
	Timestamp    time.Time      `bson:"timestamp"`
	Data         map[string]any `bson:"data,omitempty"`
}
