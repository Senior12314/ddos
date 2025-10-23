package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Storage defines the storage interface
type Storage interface {
	// User management
	CreateUser(ctx context.Context, user *User) error
	GetUserByID(ctx context.Context, id string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	UpdateUser(ctx context.Context, user *User) error
	DeleteUser(ctx context.Context, id string) error

	// Organization management
	CreateOrganization(ctx context.Context, org *Organization) error
	GetOrganizationByID(ctx context.Context, id string) (*Organization, error)
	GetOrganizationsByOwner(ctx context.Context, ownerID string) ([]*Organization, error)
	UpdateOrganization(ctx context.Context, org *Organization) error
	DeleteOrganization(ctx context.Context, id string) error

	// Endpoint management
	CreateEndpoint(ctx context.Context, endpoint *ProtectedEndpoint) error
	GetEndpointByID(ctx context.Context, id string) (*ProtectedEndpoint, error)
	GetEndpointsByOrganization(ctx context.Context, orgID string) ([]*ProtectedEndpoint, error)
	GetAllActiveEndpoints(ctx context.Context) ([]*ProtectedEndpoint, error)
	UpdateEndpoint(ctx context.Context, endpoint *ProtectedEndpoint) error
	DeleteEndpoint(ctx context.Context, id string) error

	// Node management
	CreateNode(ctx context.Context, node *Node) error
	GetNodeByID(ctx context.Context, id string) (*Node, error)
	GetAllNodes(ctx context.Context) ([]*Node, error)
	UpdateNode(ctx context.Context, node *Node) error
	DeleteNode(ctx context.Context, id string) error

	// IP management
	AddToWhitelist(ctx context.Context, entry *IPWhitelist) error
	RemoveFromWhitelist(ctx context.Context, endpointID, ip string) error
	GetWhitelist(ctx context.Context, endpointID string) ([]*IPWhitelist, error)
	AddToBlacklist(ctx context.Context, entry *IPBlacklist) error
	RemoveFromBlacklist(ctx context.Context, ip string) error
	GetBlacklist(ctx context.Context) ([]*IPBlacklist, error)

	// Metrics
	StoreMetrics(ctx context.Context, metrics *Metric) error
	GetMetrics(ctx context.Context, endpointID string, since time.Time) ([]*Metric, error)
	GetLatestMetrics(ctx context.Context, endpointID string) (*Metric, error)

	// Audit logging
	LogAuditEvent(ctx context.Context, log *AuditLog) error
	GetAuditLogs(ctx context.Context, userID string, limit int) ([]*AuditLog, error)

	// Cleanup
	CleanupExpiredBlacklist(ctx context.Context) error
	CleanupOldMetrics(ctx context.Context, olderThan time.Time) error
}

// storage implements the Storage interface
type storage struct {
	db *gorm.DB
}

// New creates a new storage instance
func New(db *gorm.DB) Storage {
	return &storage{db: db}
}

// CreateUser creates a new user
func (s *storage) CreateUser(ctx context.Context, user *User) error {
	user.ID = uuid.New().String()
	return s.db.WithContext(ctx).Create(user).Error
}

// GetUserByID retrieves a user by ID
func (s *storage) GetUserByID(ctx context.Context, id string) (*User, error) {
	var user User
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByEmail retrieves a user by email
func (s *storage) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	err := s.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateUser updates a user
func (s *storage) UpdateUser(ctx context.Context, user *User) error {
	return s.db.WithContext(ctx).Save(user).Error
}

// DeleteUser deletes a user
func (s *storage) DeleteUser(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Delete(&User{}, "id = ?", id).Error
}

// CreateOrganization creates a new organization
func (s *storage) CreateOrganization(ctx context.Context, org *Organization) error {
	org.ID = uuid.New().String()
	return s.db.WithContext(ctx).Create(org).Error
}

// GetOrganizationByID retrieves an organization by ID
func (s *storage) GetOrganizationByID(ctx context.Context, id string) (*Organization, error) {
	var org Organization
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&org).Error
	if err != nil {
		return nil, err
	}
	return &org, nil
}

// GetOrganizationsByOwner retrieves organizations by owner ID
func (s *storage) GetOrganizationsByOwner(ctx context.Context, ownerID string) ([]*Organization, error) {
	var orgs []*Organization
	err := s.db.WithContext(ctx).Where("owner_id = ?", ownerID).Order("created_at DESC").Find(&orgs).Error
	return orgs, err
}

// UpdateOrganization updates an organization
func (s *storage) UpdateOrganization(ctx context.Context, org *Organization) error {
	return s.db.WithContext(ctx).Save(org).Error
}

// DeleteOrganization deletes an organization
func (s *storage) DeleteOrganization(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Delete(&Organization{}, "id = ?", id).Error
}

// CreateEndpoint creates a new protected endpoint
func (s *storage) CreateEndpoint(ctx context.Context, endpoint *ProtectedEndpoint) error {
	endpoint.ID = uuid.New().String()
	return s.db.WithContext(ctx).Create(endpoint).Error
}

// GetEndpointByID retrieves an endpoint by ID
func (s *storage) GetEndpointByID(ctx context.Context, id string) (*ProtectedEndpoint, error) {
	var endpoint ProtectedEndpoint
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&endpoint).Error
	if err != nil {
		return nil, err
	}
	return &endpoint, nil
}

// GetEndpointsByOrganization retrieves endpoints by organization ID
func (s *storage) GetEndpointsByOrganization(ctx context.Context, orgID string) ([]*ProtectedEndpoint, error) {
	var endpoints []*ProtectedEndpoint
	err := s.db.WithContext(ctx).Where("organization_id = ?", orgID).Order("created_at DESC").Find(&endpoints).Error
	return endpoints, err
}

// GetAllActiveEndpoints retrieves all active endpoints
func (s *storage) GetAllActiveEndpoints(ctx context.Context) ([]*ProtectedEndpoint, error) {
	var endpoints []*ProtectedEndpoint
	err := s.db.WithContext(ctx).Where("active = ?", true).Order("created_at DESC").Find(&endpoints).Error
	return endpoints, err
}

// UpdateEndpoint updates an endpoint
func (s *storage) UpdateEndpoint(ctx context.Context, endpoint *ProtectedEndpoint) error {
	return s.db.WithContext(ctx).Save(endpoint).Error
}

// DeleteEndpoint deletes an endpoint
func (s *storage) DeleteEndpoint(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Delete(&ProtectedEndpoint{}, "id = ?", id).Error
}

// CreateNode creates a new node
func (s *storage) CreateNode(ctx context.Context, node *Node) error {
	node.ID = uuid.New().String()
	return s.db.WithContext(ctx).Create(node).Error
}

// GetNodeByID retrieves a node by ID
func (s *storage) GetNodeByID(ctx context.Context, id string) (*Node, error) {
	var node Node
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&node).Error
	if err != nil {
		return nil, err
	}
	return &node, nil
}

// GetAllNodes retrieves all nodes
func (s *storage) GetAllNodes(ctx context.Context) ([]*Node, error) {
	var nodes []*Node
	err := s.db.WithContext(ctx).Order("created_at DESC").Find(&nodes).Error
	return nodes, err
}

// UpdateNode updates a node
func (s *storage) UpdateNode(ctx context.Context, node *Node) error {
	return s.db.WithContext(ctx).Save(node).Error
}

// DeleteNode deletes a node
func (s *storage) DeleteNode(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Delete(&Node{}, "id = ?", id).Error
}

// AddToWhitelist adds an IP to the whitelist
func (s *storage) AddToWhitelist(ctx context.Context, entry *IPWhitelist) error {
	entry.ID = uuid.New().String()
	return s.db.WithContext(ctx).Create(entry).Error
}

// RemoveFromWhitelist removes an IP from the whitelist
func (s *storage) RemoveFromWhitelist(ctx context.Context, endpointID, ip string) error {
	return s.db.WithContext(ctx).Where("endpoint_id = ? AND ip = ?", endpointID, ip).Delete(&IPWhitelist{}).Error
}

// GetWhitelist retrieves the whitelist for an endpoint
func (s *storage) GetWhitelist(ctx context.Context, endpointID string) ([]*IPWhitelist, error) {
	var entries []*IPWhitelist
	err := s.db.WithContext(ctx).Where("endpoint_id = ?", endpointID).Order("created_at DESC").Find(&entries).Error
	return entries, err
}

// AddToBlacklist adds an IP to the blacklist
func (s *storage) AddToBlacklist(ctx context.Context, entry *IPBlacklist) error {
	entry.ID = uuid.New().String()
	entry.ExpiresAt = time.Now().Add(time.Duration(entry.Duration) * time.Second)
	return s.db.WithContext(ctx).Create(entry).Error
}

// RemoveFromBlacklist removes an IP from the blacklist
func (s *storage) RemoveFromBlacklist(ctx context.Context, ip string) error {
	return s.db.WithContext(ctx).Where("ip = ?", ip).Delete(&IPBlacklist{}).Error
}

// GetBlacklist retrieves all blacklist entries
func (s *storage) GetBlacklist(ctx context.Context) ([]*IPBlacklist, error) {
	var entries []*IPBlacklist
	err := s.db.WithContext(ctx).Where("expires_at > ?", time.Now()).Order("created_at DESC").Find(&entries).Error
	return entries, err
}

// StoreMetrics stores endpoint metrics
func (s *storage) StoreMetrics(ctx context.Context, metrics *Metric) error {
	metrics.ID = uuid.New().String()
	metrics.Timestamp = time.Now()
	return s.db.WithContext(ctx).Create(metrics).Error
}

// GetMetrics retrieves metrics for an endpoint
func (s *storage) GetMetrics(ctx context.Context, endpointID string, since time.Time) ([]*Metric, error) {
	var metrics []*Metric
	err := s.db.WithContext(ctx).Where("endpoint_id = ? AND timestamp >= ?", endpointID, since).Order("timestamp DESC").Find(&metrics).Error
	return metrics, err
}

// GetLatestMetrics retrieves the latest metrics for an endpoint
func (s *storage) GetLatestMetrics(ctx context.Context, endpointID string) (*Metric, error) {
	var metric Metric
	err := s.db.WithContext(ctx).Where("endpoint_id = ?", endpointID).Order("timestamp DESC").First(&metric).Error
	if err != nil {
		return nil, err
	}
	return &metric, nil
}

// LogAuditEvent logs an audit event
func (s *storage) LogAuditEvent(ctx context.Context, log *AuditLog) error {
	log.ID = uuid.New().String()
	log.Timestamp = time.Now()
	return s.db.WithContext(ctx).Create(log).Error
}

// GetAuditLogs retrieves audit logs for a user
func (s *storage) GetAuditLogs(ctx context.Context, userID string, limit int) ([]*AuditLog, error) {
	var logs []*AuditLog
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).Order("timestamp DESC").Limit(limit).Find(&logs).Error
	return logs, err
}

// CleanupExpiredBlacklist removes expired blacklist entries
func (s *storage) CleanupExpiredBlacklist(ctx context.Context) error {
	return s.db.WithContext(ctx).Where("expires_at <= ?", time.Now()).Delete(&IPBlacklist{}).Error
}

// CleanupOldMetrics removes old metrics
func (s *storage) CleanupOldMetrics(ctx context.Context, olderThan time.Time) error {
	return s.db.WithContext(ctx).Where("timestamp < ?", olderThan).Delete(&Metric{}).Error
}