package models

import "time"

type AuditLog struct {
	ID            string                 `json:"id"`
	Timestamp     time.Time              `json:"timestamp"`
	ActorType     string                 `json:"actor_type"` // "user" or "service"
	ActorID       string                 `json:"actor_id"`
	ActorUsername string                 `json:"actor_username,omitempty"`
	Action        string                 `json:"action"`
	Resource      string                 `json:"resource"`
	ResourceID    string                 `json:"resource_id,omitempty"`
	IPAddress     string                 `json:"ip_address"`
	UserAgent     string                 `json:"user_agent,omitempty"`
	Result        string                 `json:"result"` // "success" or "failure"
	Reason        string                 `json:"reason,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Signature     string                 `json:"signature"` // HMAC signature for tamper-proof audit
}

type AuditLogEntry struct {
	Timestamp     time.Time
	ActorType     string
	ActorID       string
	ActorName     string
	Action        string
	ResourceType  string
	ResourceID    string
	IPAddress     string
	UserAgent     string
	Result        string
	ErrorMessage  string
	Metadata      map[string]interface{}
}

const (
	ActionLogin            = "login"
	ActionLogout           = "logout"
	ActionRegister         = "register"
	ActionTokenRefresh     = "token_refresh"
	ActionTokenValidate    = "token_validate"
	ActionTokenRevoke      = "token_revoke"
	ActionHECTokenCreate   = "hec_token_create"
	ActionHECTokenRevoke   = "hec_token_revoke"
	ActionHECTokenValidate = "hec_token_validate"
	ActionUserUpdate       = "user_update"
	ActionUserDelete       = "user_delete"
	ActionPasswordChange   = "password_change"
)

const (
	ResultSuccess = "success"
	ResultFailure = "failure"
)

const (
	ActorTypeUser    = "user"
	ActorTypeService = "service"
	ActorTypeSystem  = "system"
)
