package storage

import (
	"time"
	"gorm.io/gorm"
)

// User represents a customer user
type User struct {
	gorm.Model
	ID        string `json:"id" gorm:"primaryKey"`
	Email     string `json:"email" gorm:"uniqueIndex"`
	Username  string `json:"username" gorm:"uniqueIndex"`
	Password  string `json:"-" gorm:"not null"` // Never expose password
	Role      string `json:"role" gorm:"default:user"`
	Active    bool   `json:"active" gorm:"default:true"`
}

// Organization represents a customer organization
type Organization struct {
	gorm.Model
	ID          string `json:"id" gorm:"primaryKey"`
	Name        string `json:"name" gorm:"not null"`
	OwnerID     string `json:"owner_id" gorm:"not null"`
	Plan        string `json:"plan" gorm:"default:free"`
	MaxServers  int    `json:"max_servers" gorm:"default:1"`
	Active      bool   `json:"active" gorm:"default:true"`
}

// ProtectedEndpoint represents a protected Minecraft server endpoint
type ProtectedEndpoint struct {
	gorm.Model
	ID              string `json:"id" gorm:"primaryKey"`
	OrganizationID  string `json:"organization_id" gorm:"not null"`
	Name            string `json:"name" gorm:"not null"`
	FrontIP         string `json:"front_ip" gorm:"not null"`
	FrontPort       int    `json:"front_port" gorm:"not null"`
	OriginIP        string `json:"origin_ip" gorm:"not null"`
	OriginPort      int    `json:"origin_port" gorm:"not null"`
	Protocol        string `json:"protocol" gorm:"not null"` // "java" or "bedrock"
	RateLimit       int    `json:"rate_limit" gorm:"default:1000"`
	BurstLimit      int    `json:"burst_limit" gorm:"default:2000"`
	MaintenanceMode bool   `json:"maintenance_mode" gorm:"default:false"`
	Active          bool   `json:"active" gorm:"default:true"`
}

// Node represents an edge node running XDP programs
type Node struct {
	gorm.Model
	ID          string  `json:"id" gorm:"primaryKey"`
	Name        string  `json:"name" gorm:"not null"`
	IP          string  `json:"ip" gorm:"not null"`
	Port        int     `json:"port" gorm:"not null"`
	Interface   string  `json:"interface" gorm:"not null"`
	Status      string  `json:"status" gorm:"default:inactive"` // "active", "inactive", "maintenance"
	LastSeen    time.Time `json:"last_seen"`
	CPUUsage    float64 `json:"cpu_usage" gorm:"default:0"`
	MemoryUsage float64 `json:"memory_usage" gorm:"default:0"`
	PacketRate  int64   `json:"packet_rate" gorm:"default:0"`
}

// IPWhitelist represents IP whitelist entries
type IPWhitelist struct {
	gorm.Model
	ID         string `json:"id" gorm:"primaryKey"`
	EndpointID string `json:"endpoint_id" gorm:"not null"`
	IP         string `json:"ip" gorm:"not null"`
	Description string `json:"description"`
}

// IPBlacklist represents IP blacklist entries
type IPBlacklist struct {
	gorm.Model
	ID        string    `json:"id" gorm:"primaryKey"`
	IP        string    `json:"ip" gorm:"not null;uniqueIndex"`
	Reason    string    `json:"reason"`
	Duration  int       `json:"duration" gorm:"default:3600"` // seconds
	ExpiresAt time.Time `json:"expires_at"`
	Active    bool      `json:"active" gorm:"default:true"`
}

// IPRule represents IP-based rules for endpoints
type IPRule struct {
	gorm.Model
	ID         string `json:"id" gorm:"primaryKey"`
	EndpointID string `json:"endpoint_id" gorm:"not null"`
	Type       string `json:"type" gorm:"not null"` // "whitelist", "blacklist", "rate_limit"
	IP         string `json:"ip" gorm:"not null"`
	Action     string `json:"action" gorm:"not null"` // "allow", "deny", "rate_limit"
	Value      int    `json:"value"` // For rate limiting
	Active     bool   `json:"active" gorm:"default:true"`
}

// Metric represents system metrics
type Metric struct {
	gorm.Model
	ID         string    `json:"id" gorm:"primaryKey"`
	EndpointID string    `json:"endpoint_id"`
	NodeID     string    `json:"node_id"`
	Type       string    `json:"type" gorm:"not null"` // "packets", "connections", "bandwidth"
	Value      float64   `json:"value" gorm:"not null"`
	Timestamp  time.Time `json:"timestamp" gorm:"not null"`
	Labels     string    `json:"labels"` // JSON string of labels
}

// AuditLog represents audit trail entries
type AuditLog struct {
	gorm.Model
	ID           string    `json:"id" gorm:"primaryKey"`
	UserID       string    `json:"user_id"`
	Action       string    `json:"action" gorm:"not null"`
	Resource     string    `json:"resource" gorm:"not null"`
	ResourceID   string    `json:"resource_id"`
	Details      string    `json:"details"` // JSON string of additional details
	IPAddress    string    `json:"ip_address"`
	UserAgent    string    `json:"user_agent"`
	Timestamp    time.Time `json:"timestamp" gorm:"not null"`
}

// Request/Response types for API
type CreateEndpointRequest struct {
	Name            string `json:"name" binding:"required"`
	OriginIP        string `json:"origin_ip" binding:"required,ip"`
	OriginPort      int    `json:"origin_port" binding:"required,min=1,max=65535"`
	Protocol        string `json:"protocol" binding:"required,oneof=java bedrock"`
	RateLimit       int    `json:"rate_limit"`
	BurstLimit      int    `json:"burst_limit"`
	MaintenanceMode bool   `json:"maintenance_mode"`
}

type UpdateEndpointRequest struct {
	Name            *string `json:"name"`
	RateLimit       *int    `json:"rate_limit"`
	BurstLimit      *int    `json:"burst_limit"`
	MaintenanceMode *bool   `json:"maintenance_mode"`
	Active          *bool   `json:"active"`
}

type EndpointResponse struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	FrontIP         string    `json:"front_ip"`
	FrontPort       int       `json:"front_port"`
	OriginIP        string    `json:"origin_ip"`
	OriginPort      int       `json:"origin_port"`
	Protocol        string    `json:"protocol"`
	RateLimit       int       `json:"rate_limit"`
	BurstLimit      int       `json:"burst_limit"`
	MaintenanceMode bool      `json:"maintenance_mode"`
	Active          bool      `json:"active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type NodeStatus struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	IP          string   `json:"ip"`
	Port        int      `json:"port"`
	Status      string   `json:"status"`
	LastSeen    time.Time `json:"last_seen"`
	CPUUsage    float64  `json:"cpu_usage"`
	MemoryUsage float64  `json:"memory_usage"`
	PacketRate  int64    `json:"packet_rate"`
	Endpoints   []string `json:"endpoints"`
}

type MetricsResponse struct {
	EndpointID            string    `json:"endpoint_id"`
	Timestamp             time.Time `json:"timestamp"`
	AllowedPackets        int64     `json:"allowed_packets"`
	BlockedRateLimit      int64     `json:"blocked_rate_limit"`
	BlockedBlacklist      int64     `json:"blocked_blacklist"`
	BlockedInvalidProto   int64     `json:"blocked_invalid_proto"`
	BlockedChallenge      int64     `json:"blocked_challenge"`
	BlockedMaintenance    int64     `json:"blocked_maintenance"`
	TotalPackets          int64     `json:"total_packets"`
	UDPChallengesSent     int64     `json:"udp_challenges_sent"`
	UDPChallengesPassed   int64     `json:"udp_challenges_passed"`
	TopAttackers          []string  `json:"top_attackers"`
}