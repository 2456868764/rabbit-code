package types

// Role identifies who produced a message in a transcript.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
	// RoleProgress is internal UI/session progress; stripped by API normalization.
	RoleProgress Role = "progress"
)

func (r Role) String() string { return string(r) }

// APIRoles are roles accepted by the Messages API after normalization.
func APIRoles() []Role {
	return []Role{RoleUser, RoleAssistant, RoleSystem}
}
