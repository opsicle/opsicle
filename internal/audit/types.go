package audit

import (
	"errors"
	"time"
)

var (
	ErrorNotInitialized = errors.New("not_nitialized")
)

type Verb string

const (
	Create       Verb = "create"
	Delete       Verb = "delete"
	ForcedLogout Verb = "forced_logout"
	Update       Verb = "update"
	Get          Verb = "get"
	List         Verb = "list"
	Login        Verb = "login"
	LoginWithMfa Verb = "login_with_mfa"
	Logout       Verb = "logout"
	Execute      Verb = "execute"
	VerifyEmail  Verb = "verify_email"
	Verify       Verb = "verify"
	Terminate    Verb = "terminate"
	Connect      Verb = "connect"
	Start        Verb = "start"
	Stop         Verb = "stop"
)

type EntityType string

const (
	UserEntity       EntityType = "user"
	OrgEntity        EntityType = "org"
	ControllerEntity EntityType = "controller"
	WorkerEntity     EntityType = "worker"
)

type ResourceType string

const (
	AutomationTemplateResource        ResourceType = "autotmpl"
	AutomationResource                ResourceType = "automation"
	UserResource                      ResourceType = "user"
	UserConfigResource                ResourceType = "user_config"
	UserEmailVerificationCodeResource ResourceType = "user_email_verification_code"
	UserMfaResource                   ResourceType = "user_mfa"
	UserPasswordResource              ResourceType = "user_password"
	SessionResource                   ResourceType = "session"
	OrgResource                       ResourceType = "org"
	OrgConfigResource                 ResourceType = "org_config"
	OrgMemberTypesResource            ResourceType = "org_member_types"
	OrgUserResource                   ResourceType = "org_user"
	OrgUserInvitationResource         ResourceType = "org_user_invitation"
	ConfigResource                    ResourceType = "config"
	CacheResource                     ResourceType = "cache"
	DbResource                        ResourceType = "db"
	QueueResource                     ResourceType = "queue"
)

type FieldType string

const (
	BoolField        FieldType = "bool"
	BoolArrayField   FieldType = "boolArray"
	FloatField       FieldType = "float"
	FloatArrayField  FieldType = "floatArray"
	JsonField        FieldType = "json"
	IntField         FieldType = "int"
	IntArrayField    FieldType = "intArray"
	MapField         FieldType = "map"
	StringField      FieldType = "string"
	StringArrayField FieldType = "stringArray"
	StructField      FieldType = "struct"
	StructArrayField FieldType = "structArray"
)

type Status string

const (
	Success Status = "success"
	Failed  Status = "failed"
)

type LogEntries []LogEntry

type LogEntry struct {
	EntityId     string         `bson:"entityId"`
	EntityType   EntityType     `bson:"entityType"`
	Verb         Verb           `bson:"verb"`
	ResourceId   string         `bson:"resourceId,omitempty"`
	ResourceType ResourceType   `bson:"resourceType,omitempty"`
	Status       Status         `bson:"status,omitempty"`
	FieldId      *string        `bson:"fieldId,omitempty"`
	FieldType    ResourceType   `bson:"fieldType,omitempty"`
	SrcIp        *string        `bson:"srcIp,omitempty"`
	SrcUa        *string        `bson:"srcUa,omitempty"`
	DstHost      *string        `bson:"dstHost,omitempty"`
	Timestamp    time.Time      `bson:"timestamp"`
	Data         map[string]any `bson:"data,omitempty"`
}

type Logger interface {
	Log(log LogEntry) error
	GetByEntity(entityId string, entityType EntityType, cursor time.Time, limit int64) (LogEntries, error)
}
