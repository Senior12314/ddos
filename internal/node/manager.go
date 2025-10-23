package node

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/cloudnordsp/minecraft-protection/internal/config"
	"github.com/cloudnordsp/minecraft-protection/internal/monitoring"
	"github.com/cloudnordsp/minecraft-protection/internal/storage"
	"go.uber.org/zap"
)

// Manager manages edge nodes running XDP programs
type Manager struct {
	config  *config.NodeConfig
	store   storage.Storage
	monitor *monitoring.Monitoring

	nodes   map[string]*Node
	nodesMu sync.RWMutex

	updateTicker *time.Ticker
	healthTicker *time.Ticker
	stopCh       chan struct{}
}

// Node represents an edge node
type Node struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	IP          string    `json:"ip"`
	Port        int       `json:"port"`
	Interface   string    `json:"interface"`
	Status      string    `json:"status"`
	LastSeen    time.Time `json:"last_seen"`
	CPUUsage    float64   `json:"cpu_usage"`
	MemoryUsage float64   `json:"memory_usage"`
	PacketRate  int64     `json:"packet_rate"`
	Endpoints   []string  `json:"endpoints"`
	client      *http.Client
}

// NodeStatus represents the status of a node
type NodeStatus struct {
	Status      string    `json:"status"`
	LastSeen    time.Time `json:"last_seen"`
	CPUUsage    float64   `json:"cpu_usage"`
	MemoryUsage float64   `json:"memory_usage"`
	PacketRate  int64     `json:"packet_rate"`
	Endpoints   []string  `json:"endpoints"`
}

// EndpointUpdate represents an endpoint update for a node
type EndpointUpdate struct {
	Action   string                 `json:"action"` // "add", "remove", "update"
	Endpoint *storage.ProtectedEndpoint `json:"endpoint"`
}

// NewManager creates a new node manager
func NewManager(cfg *config.NodeConfig, store storage.Storage, monitor *monitoring.Monitoring) *Manager {
	return &Manager{
		config:  cfg,
		store:   store,
		monitor: monitor,
		nodes:   make(map[string]*Node),
		stopCh:  make(chan struct{}),
	}
}

// Start starts the node manager
func (m *Manager) Start(ctx context.Context) error {
	m.monitor.LogInfo("Starting node manager")

	// Load existing nodes from database
	if err := m.loadNodes(ctx); err != nil {
		return fmt.Errorf("failed to load nodes: %w", err)
	}

	// Start update ticker
	m.updateTicker = time.NewTicker(m.config.UpdateInterval)
	go m.updateLoop(ctx)

	// Start health check ticker
	m.healthTicker = time.NewTicker(m.config.HealthCheckInterval)
	go m.healthCheckLoop(ctx)

	return nil
}

// Stop stops the node manager
func (m *Manager) Stop() {
	m.monitor.LogInfo("Stopping node manager")

	if m.updateTicker != nil {
		m.updateTicker.Stop()
	}
	if m.healthTicker != nil {
		m.healthTicker.Stop()
	}

	close(m.stopCh)
}

// RegisterNode registers a new node
func (m *Manager) RegisterNode(ctx context.Context, node *Node) error {
	m.nodesMu.Lock()
	defer m.nodesMu.Unlock()

	// Create HTTP client for the node
	node.client = &http.Client{
		Timeout: m.config.NodeTimeout,
	}

	// Store in memory
	m.nodes[node.ID] = node

	// Store in database
	dbNode := &storage.Node{
		ID:          node.ID,
		Name:        node.Name,
		IP:          node.IP,
		Port:        node.Port,
		Interface:   node.Interface,
		Status:      node.Status,
		LastSeen:    node.LastSeen,
		CPUUsage:    node.CPUUsage,
		MemoryUsage: node.MemoryUsage,
		PacketRate:  node.PacketRate,
	}

	if err := m.store.CreateNode(ctx, dbNode); err != nil {
		return fmt.Errorf("failed to create node in database: %w", err)
	}

	m.monitor.LogInfo("Node registered", zap.String("node_id", node.ID), zap.String("node_name", node.Name))
	return nil
}

// GetNode returns a node by ID
func (m *Manager) GetNode(id string) (*Node, bool) {
	m.nodesMu.RLock()
	defer m.nodesMu.RUnlock()

	node, exists := m.nodes[id]
	return node, exists
}

// GetAllNodes returns all nodes
func (m *Manager) GetAllNodes() []*Node {
	m.nodesMu.RLock()
	defer m.nodesMu.RUnlock()

	nodes := make([]*Node, 0, len(m.nodes))
	for _, node := range m.nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// UpdateEndpoint updates an endpoint on all nodes
func (m *Manager) UpdateEndpoint(ctx context.Context, endpoint *storage.ProtectedEndpoint) error {
	m.nodesMu.RLock()
	nodes := make([]*Node, 0, len(m.nodes))
	for _, node := range m.nodes {
		if node.Status == "active" {
			nodes = append(nodes, node)
		}
	}
	m.nodesMu.RUnlock()

	update := &EndpointUpdate{
		Action:   "update",
		Endpoint: endpoint,
	}

	for _, node := range nodes {
		if err := m.sendEndpointUpdate(ctx, node, update); err != nil {
			m.monitor.LogError("Failed to update endpoint on node",
				zap.String("node_id", node.ID),
				zap.String("endpoint_id", endpoint.ID),
				zap.Error(err))
		}
	}

	return nil
}

// AddEndpoint adds an endpoint to all nodes
func (m *Manager) AddEndpoint(ctx context.Context, endpoint *storage.ProtectedEndpoint) error {
	m.nodesMu.RLock()
	nodes := make([]*Node, 0, len(m.nodes))
	for _, node := range m.nodes {
		if node.Status == "active" {
			nodes = append(nodes, node)
		}
	}
	m.nodesMu.RUnlock()

	update := &EndpointUpdate{
		Action:   "add",
		Endpoint: endpoint,
	}

	for _, node := range nodes {
		if err := m.sendEndpointUpdate(ctx, node, update); err != nil {
			m.monitor.LogError("Failed to add endpoint to node",
				zap.String("node_id", node.ID),
				zap.String("endpoint_id", endpoint.ID),
				zap.Error(err))
		}
	}

	return nil
}

// RemoveEndpoint removes an endpoint from all nodes
func (m *Manager) RemoveEndpoint(ctx context.Context, endpointID string) error {
	m.nodesMu.RLock()
	nodes := make([]*Node, 0, len(m.nodes))
	for _, node := range m.nodes {
		if node.Status == "active" {
			nodes = append(nodes, node)
		}
	}
	m.nodesMu.RUnlock()

	update := &EndpointUpdate{
		Action: "remove",
		Endpoint: &storage.ProtectedEndpoint{
			ID: endpointID,
		},
	}

	for _, node := range nodes {
		if err := m.sendEndpointUpdate(ctx, node, update); err != nil {
			m.monitor.LogError("Failed to remove endpoint from node",
				zap.String("node_id", node.ID),
				zap.String("endpoint_id", endpointID),
				zap.Error(err))
		}
	}

	return nil
}

// loadNodes loads nodes from database
func (m *Manager) loadNodes(ctx context.Context) error {
	dbNodes, err := m.store.GetAllNodes(ctx)
	if err != nil {
		return fmt.Errorf("failed to get nodes from database: %w", err)
	}

	m.nodesMu.Lock()
	defer m.nodesMu.Unlock()

	for _, dbNode := range dbNodes {
		node := &Node{
			ID:          dbNode.ID,
			Name:        dbNode.Name,
			IP:          dbNode.IP,
			Port:        dbNode.Port,
			Interface:   dbNode.Interface,
			Status:      dbNode.Status,
			LastSeen:    dbNode.LastSeen,
			CPUUsage:    dbNode.CPUUsage,
			MemoryUsage: dbNode.MemoryUsage,
			PacketRate:  dbNode.PacketRate,
			client: &http.Client{
				Timeout: m.config.NodeTimeout,
			},
		}
		m.nodes[node.ID] = node
	}

	m.monitor.LogInfo("Loaded nodes from database", zap.Int("count", len(m.nodes)))
	return nil
}

// updateLoop runs the update loop
func (m *Manager) updateLoop(ctx context.Context) {
	for {
		select {
		case <-m.updateTicker.C:
			m.updateNodes(ctx)
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		}
	}
}

// healthCheckLoop runs the health check loop
func (m *Manager) healthCheckLoop(ctx context.Context) {
	for {
		select {
		case <-m.healthTicker.C:
			m.healthCheckNodes(ctx)
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		}
	}
}

// updateNodes updates all nodes
func (m *Manager) updateNodes(ctx context.Context) {
	m.nodesMu.RLock()
	nodes := make([]*Node, 0, len(m.nodes))
	for _, node := range m.nodes {
		nodes = append(nodes, node)
	}
	m.nodesMu.RUnlock()

	for _, node := range nodes {
		if err := m.updateNode(ctx, node); err != nil {
			m.monitor.LogError("Failed to update node",
				zap.String("node_id", node.ID),
				zap.Error(err))
		}
	}
}

// healthCheckNodes performs health checks on all nodes
func (m *Manager) healthCheckNodes(ctx context.Context) {
	m.nodesMu.RLock()
	nodes := make([]*Node, 0, len(m.nodes))
	for _, node := range m.nodes {
		nodes = append(nodes, node)
	}
	m.nodesMu.RUnlock()

	for _, node := range nodes {
		if err := m.healthCheckNode(ctx, node); err != nil {
			m.monitor.LogError("Failed to health check node",
				zap.String("node_id", node.ID),
				zap.Error(err))
		}
	}
}

// updateNode updates a single node
func (m *Manager) updateNode(ctx context.Context, node *Node) error {
	// Get node status
	status, err := m.getNodeStatus(ctx, node)
	if err != nil {
		// Mark node as inactive if we can't reach it
		node.Status = "inactive"
		m.updateNodeInDatabase(ctx, node)
		return err
	}

	// Update node with new status
	node.Status = status.Status
	node.LastSeen = status.LastSeen
	node.CPUUsage = status.CPUUsage
	node.MemoryUsage = status.MemoryUsage
	node.PacketRate = status.PacketRate
	node.Endpoints = status.Endpoints

	// Update in database
	return m.updateNodeInDatabase(ctx, node)
}

// healthCheckNode performs a health check on a node
func (m *Manager) healthCheckNode(ctx context.Context, node *Node) error {
	// Check if node is reachable
	url := fmt.Sprintf("http://%s:%d/health", node.IP, node.Port)
	resp, err := node.client.Get(url)
	if err != nil {
		node.Status = "inactive"
		m.updateNodeInDatabase(ctx, node)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		node.Status = "inactive"
		m.updateNodeInDatabase(ctx, node)
		return fmt.Errorf("health check failed with status %d", resp.StatusCode)
	}

	// Update last seen time
	node.LastSeen = time.Now()
	node.Status = "active"
	return m.updateNodeInDatabase(ctx, node)
}

// getNodeStatus gets the status of a node
func (m *Manager) getNodeStatus(ctx context.Context, node *Node) (*NodeStatus, error) {
	url := fmt.Sprintf("http://%s:%d/api/v1/status", node.IP, node.Port)
	resp, err := node.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status request failed with status %d", resp.StatusCode)
	}

	var status NodeStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, err
	}

	return &status, nil
}

// sendEndpointUpdate sends an endpoint update to a node
func (m *Manager) sendEndpointUpdate(ctx context.Context, node *Node, update *EndpointUpdate) error {
	url := fmt.Sprintf("http://%s:%d/api/v1/endpoint", node.IP, node.Port)
	
	data, err := json.Marshal(update)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := node.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("endpoint update failed with status %d", resp.StatusCode)
	}

	return nil
}

// updateNodeInDatabase updates a node in the database
func (m *Manager) updateNodeInDatabase(ctx context.Context, node *Node) error {
	dbNode := &storage.Node{
		ID:          node.ID,
		Name:        node.Name,
		IP:          node.IP,
		Port:        node.Port,
		Interface:   node.Interface,
		Status:      node.Status,
		LastSeen:    node.LastSeen,
		CPUUsage:    node.CPUUsage,
		MemoryUsage: node.MemoryUsage,
		PacketRate:  node.PacketRate,
	}

	return m.store.UpdateNode(ctx, dbNode)
}
