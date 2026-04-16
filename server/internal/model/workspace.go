// server/internal/model/workspace.go
package model

import "time"

// Workspace 工作空间表，支持个人和团队两种类型
type Workspace struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:100;not null" json:"name"`
	Type        string    `gorm:"size:20;not null;default:personal" json:"type"` // personal, team
	OwnerID     uint      `gorm:"index;not null" json:"owner_id"`
	Description string    `gorm:"type:text" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// WorkspaceMember 工作空间成员表，管理用户与工作空间的关联关系
type WorkspaceMember struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	WorkspaceID uint      `gorm:"uniqueIndex:idx_ws_user;not null" json:"workspace_id"`
	UserID      uint      `gorm:"uniqueIndex:idx_ws_user;not null" json:"user_id"`
	Role        string    `gorm:"size:20;not null;default:viewer" json:"role"` // owner, editor, viewer
	CreatedAt   time.Time `json:"created_at"`
}

// WorkspaceType 工作空间类型枚举
const (
	WorkspaceTypePersonal = "personal"
	WorkspaceTypeTeam     = "team"
)

// WorkspaceRole 工作空间角色枚举
const (
	WorkspaceRoleOwner  = "owner"
	WorkspaceRoleEditor = "editor"
	WorkspaceRoleViewer = "viewer"
)

// ValidWorkspaceTypes 合法的工作空间类型白名单
var ValidWorkspaceTypes = map[string]bool{
	WorkspaceTypePersonal: true,
	WorkspaceTypeTeam:     true,
}

// ValidWorkspaceRoles 合法的工作空间角色白名单
var ValidWorkspaceRoles = map[string]bool{
	WorkspaceRoleOwner:  true,
	WorkspaceRoleEditor: true,
	WorkspaceRoleViewer: true,
}
