package models

import (
	"encoding/json"
	"time"
)

// Server 服务器模型
type Server struct {
	ID             uint            `json:"id" gorm:"primaryKey"`
	Name           string          `json:"name" gorm:"not null"`
	Address        string          `json:"address" gorm:"not null"`
	Port           int             `json:"port" gorm:"not null;default:25565"`
	Type           string          `json:"type" gorm:"not null"` // "java", "bedrock"
	Status         string          `json:"status" gorm:"default:offline"`
	PlayersOnline  int             `json:"players_online" gorm:"default:0"`
	MaxPlayers     int             `json:"max_players" gorm:"default:0"`
	AnonymousCount int             `json:"anonymous_count" gorm:"default:0"` // 匿名玩家数量
	Ping           int             `json:"ping" gorm:"default:0"`
	Version        string          `json:"version"`
	MOTD           string          `json:"motd"`
	Description    string          `json:"description"`
	LastChecked    *time.Time      `json:"last_checked"`
	LastOnlineData json.RawMessage `json:"-" gorm:"type:json"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// ServerStat 服务器状态历史
type ServerStat struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	ServerID      uint      `json:"server_id" gorm:"not null"`
	PlayersOnline int       `json:"players_online"`
	MaxPlayers    int       `json:"max_players"`
	Ping          int       `json:"ping"`
	Version       string    `json:"version"`
	MOTD          string    `json:"motd"`
	Timestamp     time.Time `json:"timestamp"`
	Server        Server    `json:"server" gorm:"foreignKey:ServerID"`
}

// Player 玩家模型
type Player struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	Username      string    `json:"username" gorm:"not null;uniqueIndex"`
	UUID          string    `json:"uuid" gorm:"uniqueIndex"`
	FirstSeen     time.Time `json:"first_seen"`
	LastSeen      time.Time `json:"last_seen"`
	TotalPlaytime int       `json:"total_playtime"` // 秒
	Rank          string    `json:"rank" gorm:"default:Newcomer"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// PlayerSession 玩家会话
type PlayerSession struct {
	ID        uint       `json:"id" gorm:"primaryKey"`
	PlayerID  uint       `json:"player_id" gorm:"not null"`
	ServerID  uint       `json:"server_id" gorm:"not null"`
	JoinTime  time.Time  `json:"join_time"`
	LeaveTime *time.Time `json:"leave_time"`
	Duration  int        `json:"duration"` // 秒
	Player    Player     `json:"player" gorm:"foreignKey:PlayerID"`
	Server    Server     `json:"server" gorm:"foreignKey:ServerID"`
}

// PlayerActivity 玩家活动记录
type PlayerActivity struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	PlayerID        uint      `json:"player_id" gorm:"not null;index"`
	ServerID        uint      `json:"server_id" gorm:"not null;index"`
	ActivityType    string    `json:"activity_type" gorm:"not null"` // "join" 或 "leave"
	Timestamp       time.Time `json:"timestamp" gorm:"not null;index"`
	SessionDuration int       `json:"session_duration,omitempty"` // 仅在 leave 时有值，单位：秒
	Player          Player    `json:"player" gorm:"foreignKey:PlayerID"`
	Server          Server    `json:"server" gorm:"foreignKey:ServerID"`
}

// PlayerTitle 玩家称号
type PlayerTitle struct {
	ID       uint      `json:"id" gorm:"primaryKey"`
	PlayerID uint      `json:"player_id" gorm:"not null"`
	Title    string    `json:"title" gorm:"not null"`
	EarnedAt time.Time `json:"earned_at"`
	Player   Player    `json:"player" gorm:"foreignKey:PlayerID"`
}

// User 用户管理
type User struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	Username    string     `json:"username" gorm:"not null;uniqueIndex"`
	Password    string     `json:"-" gorm:"column:password_hash;not null"`
	Role        string     `json:"role" gorm:"default:admin"`
	LastLoginAt *time.Time `json:"last_login_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
