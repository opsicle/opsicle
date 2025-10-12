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
	Accept       Verb = "accept"
	Connect      Verb = "connect"
	Create       Verb = "create"
	Delete       Verb = "delete"
	Execute      Verb = "execute"
	ForcedLogout Verb = "forced_logout"
	Get          Verb = "get"
	List         Verb = "list"
	Login        Verb = "login"
	LoginWithMfa Verb = "login_with_mfa"
	Logout       Verb = "logout"
	Reject       Verb = "reject"
	Start        Verb = "start"
	Stop         Verb = "stop"
	Terminate    Verb = "terminate"
	Update       Verb = "update"
	Verify       Verb = "verify"
	VerifyEmail  Verb = "verify_email"
)

type EntityType string

const (
	UserEntity        EntityType = "user"
	OrgEntity         EntityType = "org"
	ControllerEntity  EntityType = "controller"
	CoordinatorEntity EntityType = "coordinator"
	WorkerEntity      EntityType = "worker"
)

type ResourceType string

const (
	AutomationTemplateResource        ResourceType = "autotmpl"
	AutomationResource                ResourceType = "automation"
	CacheResource                     ResourceType = "cache"
	ConfigResource                    ResourceType = "config"
	DbResource                        ResourceType = "db"
	OrgResource                       ResourceType = "org"
	OrgConfigResource                 ResourceType = "org_config"
	OrgMemberTypesResource            ResourceType = "org_member_types"
	OrgUserResource                   ResourceType = "org_user"
	OrgUserInvitationResource         ResourceType = "org_user_invitation"
	QueueResource                     ResourceType = "queue"
	SessionResource                   ResourceType = "session"
	TemplateUserInvitationResource    ResourceType = "template_user_invitation"
	UserResource                      ResourceType = "user"
	UserConfigResource                ResourceType = "user_config"
	UserEmailVerificationCodeResource ResourceType = "user_email_verification_code"
	UserMfaResource                   ResourceType = "user_mfa"
	UserPasswordResource              ResourceType = "user_password"
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
	EntityId     string         `json:"entityId" bson:"entityId"`
	EntityType   EntityType     `json:"entityType" bson:"entityType"`
	Verb         Verb           `json:"verb" bson:"verb"`
	ResourceId   string         `json:"resourceId,omitempty" bson:"resourceId,omitempty"`
	ResourceType ResourceType   `json:"resourceType,omitempty" bson:"resourceType,omitempty"`
	Status       Status         `json:"status,omitempty" bson:"status,omitempty"`
	FieldId      *string        `json:"fieldId,omitempty" bson:"fieldId,omitempty"`
	FieldType    ResourceType   `json:"fieldType,omitempty" bson:"fieldType,omitempty"`
	SrcIp        *string        `json:"srcIp,omitempty" bson:"srcIp,omitempty"`
	SrcUa        *string        `json:"srcUa,omitempty" bson:"srcUa,omitempty"`
	DstHost      *string        `json:"dstHost,omitempty" bson:"dstHost,omitempty"`
	Timestamp    time.Time      `json:"timestamp" bson:"timestamp"`
	Data         map[string]any `json:"data,omitempty" bson:"data,omitempty"`
}

type Logger interface {
	Log(log LogEntry) error
	GetByEntity(entityId string, entityType EntityType, cursor time.Time, limit int64, reverseTimeOrder bool) (LogEntries, error)
}
