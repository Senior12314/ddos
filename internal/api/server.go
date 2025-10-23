package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/cloudnordsp/minecraft-protection/internal/config"
	"github.com/cloudnordsp/minecraft-protection/internal/monitoring"
	"github.com/cloudnordsp/minecraft-protection/internal/node"
	"github.com/cloudnordsp/minecraft-protection/internal/proxy"
	"github.com/cloudnordsp/minecraft-protection/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Server represents the API server
type Server struct {
	config       *config.APIConfig
	store        storage.Storage
	nodeManager  *node.Manager
	proxyManager *proxy.Manager
	monitor      *monitoring.Monitoring
}

// NewServer creates a new API server
func NewServer(cfg *config.APIConfig, store storage.Storage, nodeManager *node.Manager, proxyManager *proxy.Manager, monitor *monitoring.Monitoring) *Server {
	return &Server{
		config:       cfg,
		store:        store,
		nodeManager:  nodeManager,
		proxyManager: proxyManager,
		monitor:      monitor,
	}
}

// SetupRoutes sets up API routes
func (s *Server) SetupRoutes(rg *gin.RouterGroup) {
	// Authentication middleware
	rg.Use(s.authMiddleware())

	// Endpoint management
	endpoints := rg.Group("/endpoints")
	{
		endpoints.POST("", s.createEndpoint)
		endpoints.GET("", s.listEndpoints)
		endpoints.GET("/:id", s.getEndpoint)
		endpoints.PUT("/:id", s.updateEndpoint)
		endpoints.DELETE("/:id", s.deleteEndpoint)
		endpoints.GET("/:id/metrics", s.getEndpointMetrics)
		endpoints.POST("/:id/whitelist", s.addToWhitelist)
		endpoints.DELETE("/:id/whitelist/:ip", s.removeFromWhitelist)
		endpoints.GET("/:id/whitelist", s.getWhitelist)
	}

	// Node management
	nodes := rg.Group("/nodes")
	{
		nodes.GET("", s.listNodes)
		nodes.GET("/:id", s.getNode)
		nodes.GET("/:id/status", s.getNodeStatus)
	}

	// Blacklist management
	blacklist := rg.Group("/blacklist")
	{
		blacklist.POST("", s.addToBlacklist)
		blacklist.DELETE("/:ip", s.removeFromBlacklist)
		blacklist.GET("", s.getBlacklist)
	}

	// User management
	users := rg.Group("/users")
	{
		users.GET("/profile", s.getProfile)
		users.PUT("/profile", s.updateProfile)
	}

	// Organization management
	orgs := rg.Group("/organizations")
	{
		orgs.GET("", s.listOrganizations)
		orgs.GET("/:id", s.getOrganization)
		orgs.PUT("/:id", s.updateOrganization)
	}

	// System endpoints
	system := rg.Group("/system")
	{
		system.GET("/status", s.getSystemStatus)
		system.GET("/stats", s.getSystemStats)
	}
}

// authMiddleware provides authentication middleware
func (s *Server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// For now, we'll implement a simple token-based auth
		// In production, this would use JWT tokens
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Simple token validation (replace with proper JWT validation)
		if token != "Bearer valid-token" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// Set user context (in production, extract from JWT)
		c.Set("user_id", "user-123")
		c.Set("organization_id", "org-123")
		c.Next()
	}
}

// createEndpoint creates a new protected endpoint
func (s *Server) createEndpoint(c *gin.Context) {
	var req storage.CreateEndpointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user context
	userID := c.GetString("user_id")
	orgID := c.GetString("organization_id")

	// Create endpoint
	endpoint := &storage.ProtectedEndpoint{
		OrganizationID:  orgID,
		Name:            req.Name,
		OriginIP:        req.OriginIP,
		OriginPort:      req.OriginPort,
		Protocol:        req.Protocol,
		RateLimit:       req.RateLimit,
		BurstLimit:      req.BurstLimit,
		MaintenanceMode: req.MaintenanceMode,
		Active:          true,
	}

	// Set default values
	if endpoint.RateLimit == 0 {
		endpoint.RateLimit = 1000
	}
	if endpoint.BurstLimit == 0 {
		endpoint.BurstLimit = 5000
	}

	// Assign front IP and port (in production, this would be managed by a pool)
	endpoint.FrontIP = "198.51.100.10" // This would come from a pool
	endpoint.FrontPort = 25565 + len(s.nodeManager.GetAllNodes()) // Simple port assignment

	// Save to database
	if err := s.store.CreateEndpoint(c.Request.Context(), endpoint); err != nil {
		s.monitor.LogError("Failed to create endpoint", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create endpoint"})
		return
	}

	// Add to node manager
	if err := s.nodeManager.AddEndpoint(c.Request.Context(), endpoint); err != nil {
		s.monitor.LogError("Failed to add endpoint to nodes", zap.Error(err))
	}

	// Add to proxy manager
	if err := s.proxyManager.AddEndpoint(c.Request.Context(), endpoint); err != nil {
		s.monitor.LogError("Failed to add endpoint to proxy", zap.Error(err))
	}

	// Log audit event
	auditLog := &storage.AuditLog{
		UserID:     userID,
		Action:     "create_endpoint",
		Resource:   "endpoint",
		ResourceID: endpoint.ID,
		Details:    fmt.Sprintf(`{"name": "%s", "origin": "%s:%d", "protocol": "%s"}`, endpoint.Name, endpoint.OriginIP, endpoint.OriginPort, endpoint.Protocol),
		IPAddress:  c.ClientIP(),
		UserAgent:  c.GetHeader("User-Agent"),
	}
	s.store.LogAuditEvent(c.Request.Context(), auditLog)

	// Return response
	response := &storage.EndpointResponse{
		ID:              endpoint.ID,
		Name:            endpoint.Name,
		FrontIP:         endpoint.FrontIP,
		FrontPort:       endpoint.FrontPort,
		OriginIP:        endpoint.OriginIP,
		OriginPort:      endpoint.OriginPort,
		Protocol:        endpoint.Protocol,
		RateLimit:       endpoint.RateLimit,
		BurstLimit:      endpoint.BurstLimit,
		MaintenanceMode: endpoint.MaintenanceMode,
		Active:          endpoint.Active,
		CreatedAt:       endpoint.CreatedAt,
		UpdatedAt:       endpoint.UpdatedAt,
	}

	c.JSON(http.StatusCreated, response)
}

// listEndpoints lists all endpoints for the user's organization
func (s *Server) listEndpoints(c *gin.Context) {
	orgID := c.GetString("organization_id")

	endpoints, err := s.store.GetEndpointsByOrganization(c.Request.Context(), orgID)
	if err != nil {
		s.monitor.LogError("Failed to list endpoints", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list endpoints"})
		return
	}

	// Convert to response format
	responses := make([]*storage.EndpointResponse, len(endpoints))
	for i, endpoint := range endpoints {
		responses[i] = &storage.EndpointResponse{
			ID:              endpoint.ID,
			Name:            endpoint.Name,
			FrontIP:         endpoint.FrontIP,
			FrontPort:       endpoint.FrontPort,
			OriginIP:        endpoint.OriginIP,
			OriginPort:      endpoint.OriginPort,
			Protocol:        endpoint.Protocol,
			RateLimit:       endpoint.RateLimit,
			BurstLimit:      endpoint.BurstLimit,
			MaintenanceMode: endpoint.MaintenanceMode,
			Active:          endpoint.Active,
			CreatedAt:       endpoint.CreatedAt,
			UpdatedAt:       endpoint.UpdatedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{"endpoints": responses})
}

// getEndpoint gets a specific endpoint
func (s *Server) getEndpoint(c *gin.Context) {
	endpointID := c.Param("id")

	endpoint, err := s.store.GetEndpointByID(c.Request.Context(), endpointID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Endpoint not found"})
		return
	}

	// Check if user has access to this endpoint
	orgID := c.GetString("organization_id")
	if endpoint.OrganizationID != orgID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	response := &storage.EndpointResponse{
		ID:              endpoint.ID,
		Name:            endpoint.Name,
		FrontIP:         endpoint.FrontIP,
		FrontPort:       endpoint.FrontPort,
		OriginIP:        endpoint.OriginIP,
		OriginPort:      endpoint.OriginPort,
		Protocol:        endpoint.Protocol,
		RateLimit:       endpoint.RateLimit,
		BurstLimit:      endpoint.BurstLimit,
		MaintenanceMode: endpoint.MaintenanceMode,
		Active:          endpoint.Active,
		CreatedAt:       endpoint.CreatedAt,
		UpdatedAt:       endpoint.UpdatedAt,
	}

	c.JSON(http.StatusOK, response)
}

// updateEndpoint updates an endpoint
func (s *Server) updateEndpoint(c *gin.Context) {
	endpointID := c.Param("id")

	// Get existing endpoint
	endpoint, err := s.store.GetEndpointByID(c.Request.Context(), endpointID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Endpoint not found"})
		return
	}

	// Check if user has access to this endpoint
	orgID := c.GetString("organization_id")
	if endpoint.OrganizationID != orgID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	var req storage.UpdateEndpointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update fields
	if req.Name != nil {
		endpoint.Name = *req.Name
	}
	if req.RateLimit != nil {
		endpoint.RateLimit = *req.RateLimit
	}
	if req.BurstLimit != nil {
		endpoint.BurstLimit = *req.BurstLimit
	}
	if req.MaintenanceMode != nil {
		endpoint.MaintenanceMode = *req.MaintenanceMode
	}
	if req.Active != nil {
		endpoint.Active = *req.Active
	}

	// Save to database
	if err := s.store.UpdateEndpoint(c.Request.Context(), endpoint); err != nil {
		s.monitor.LogError("Failed to update endpoint", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update endpoint"})
		return
	}

	// Update in node manager
	if err := s.nodeManager.UpdateEndpoint(c.Request.Context(), endpoint); err != nil {
		s.monitor.LogError("Failed to update endpoint in nodes", zap.Error(err))
	}

	// Update in proxy manager
	if err := s.proxyManager.UpdateEndpoint(c.Request.Context(), endpoint); err != nil {
		s.monitor.LogError("Failed to update endpoint in proxy", zap.Error(err))
	}

	// Log audit event
	userID := c.GetString("user_id")
	auditLog := &storage.AuditLog{
		UserID:     userID,
		Action:     "update_endpoint",
		Resource:   "endpoint",
		ResourceID: endpoint.ID,
		Details:    fmt.Sprintf(`{"name": "%s", "rate_limit": %d, "burst_limit": %d, "maintenance_mode": %t, "active": %t}`, endpoint.Name, endpoint.RateLimit, endpoint.BurstLimit, endpoint.MaintenanceMode, endpoint.Active),
		IPAddress:  c.ClientIP(),
		UserAgent:  c.GetHeader("User-Agent"),
	}
	s.store.LogAuditEvent(c.Request.Context(), auditLog)

	response := &storage.EndpointResponse{
		ID:              endpoint.ID,
		Name:            endpoint.Name,
		FrontIP:         endpoint.FrontIP,
		FrontPort:       endpoint.FrontPort,
		OriginIP:        endpoint.OriginIP,
		OriginPort:      endpoint.OriginPort,
		Protocol:        endpoint.Protocol,
		RateLimit:       endpoint.RateLimit,
		BurstLimit:      endpoint.BurstLimit,
		MaintenanceMode: endpoint.MaintenanceMode,
		Active:          endpoint.Active,
		CreatedAt:       endpoint.CreatedAt,
		UpdatedAt:       endpoint.UpdatedAt,
	}

	c.JSON(http.StatusOK, response)
}

// deleteEndpoint deletes an endpoint
func (s *Server) deleteEndpoint(c *gin.Context) {
	endpointID := c.Param("id")

	// Get existing endpoint
	endpoint, err := s.store.GetEndpointByID(c.Request.Context(), endpointID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Endpoint not found"})
		return
	}

	// Check if user has access to this endpoint
	orgID := c.GetString("organization_id")
	if endpoint.OrganizationID != orgID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Remove from node manager
	if err := s.nodeManager.RemoveEndpoint(c.Request.Context(), endpointID); err != nil {
		s.monitor.LogError("Failed to remove endpoint from nodes", zap.Error(err))
	}

	// Remove from proxy manager
	if err := s.proxyManager.RemoveEndpoint(c.Request.Context(), endpointID); err != nil {
		s.monitor.LogError("Failed to remove endpoint from proxy", zap.Error(err))
	}

	// Delete from database
	if err := s.store.DeleteEndpoint(c.Request.Context(), endpointID); err != nil {
		s.monitor.LogError("Failed to delete endpoint", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete endpoint"})
		return
	}

	// Log audit event
	userID := c.GetString("user_id")
	auditLog := &storage.AuditLog{
		UserID:     userID,
		Action:     "delete_endpoint",
		Resource:   "endpoint",
		ResourceID: endpointID,
		Details:    fmt.Sprintf(`{"name": "%s"}`, endpoint.Name),
		IPAddress:  c.ClientIP(),
		UserAgent:  c.GetHeader("User-Agent"),
	}
	s.store.LogAuditEvent(c.Request.Context(), auditLog)

	c.JSON(http.StatusOK, gin.H{"message": "Endpoint deleted successfully"})
}

// getEndpointMetrics gets metrics for an endpoint
func (s *Server) getEndpointMetrics(c *gin.Context) {
	endpointID := c.Param("id")

	// Get time range from query parameters
	sinceStr := c.DefaultQuery("since", "1h")
	since, err := time.ParseDuration(sinceStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid since parameter"})
		return
	}

	// Get metrics from database
	metrics, err := s.store.GetMetrics(c.Request.Context(), endpointID, time.Now().Add(-since))
	if err != nil {
		s.monitor.LogError("Failed to get endpoint metrics", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get metrics"})
		return
	}

	// Convert to response format
	responses := make([]*storage.MetricsResponse, len(metrics))
	for i, metric := range metrics {
		responses[i] = &storage.MetricsResponse{
			EndpointID:            metric.EndpointID,
			Timestamp:             metric.Timestamp,
			AllowedPackets:        int64(metric.Value), // Using Value field for now
			BlockedRateLimit:      0, // These would need to be separate metrics
			BlockedBlacklist:      0,
			BlockedInvalidProto:   0,
			BlockedChallenge:      0,
			BlockedMaintenance:    0,
			TotalPackets:          int64(metric.Value),
			UDPChallengesSent:     0,
			UDPChallengesPassed:   0,
			TopAttackers:          []string{}, // Parse from JSON if needed
		}
	}

	c.JSON(http.StatusOK, gin.H{"metrics": responses})
}

// addToWhitelist adds an IP to the whitelist
func (s *Server) addToWhitelist(c *gin.Context) {
	endpointID := c.Param("id")

	// Check if user has access to this endpoint
	endpoint, err := s.store.GetEndpointByID(c.Request.Context(), endpointID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Endpoint not found"})
		return
	}

	orgID := c.GetString("organization_id")
	if endpoint.OrganizationID != orgID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	var req struct {
		IP          string `json:"ip" binding:"required,ip"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Add to whitelist
	entry := &storage.IPWhitelist{
		EndpointID:  endpointID,
		IP:          req.IP,
		Description: req.Description,
	}

	if err := s.store.AddToWhitelist(c.Request.Context(), entry); err != nil {
		s.monitor.LogError("Failed to add IP to whitelist", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add IP to whitelist"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "IP added to whitelist"})
}

// removeFromWhitelist removes an IP from the whitelist
func (s *Server) removeFromWhitelist(c *gin.Context) {
	endpointID := c.Param("id")
	ip := c.Param("ip")

	// Check if user has access to this endpoint
	endpoint, err := s.store.GetEndpointByID(c.Request.Context(), endpointID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Endpoint not found"})
		return
	}

	orgID := c.GetString("organization_id")
	if endpoint.OrganizationID != orgID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if err := s.store.RemoveFromWhitelist(c.Request.Context(), endpointID, ip); err != nil {
		s.monitor.LogError("Failed to remove IP from whitelist", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove IP from whitelist"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "IP removed from whitelist"})
}

// getWhitelist gets the whitelist for an endpoint
func (s *Server) getWhitelist(c *gin.Context) {
	endpointID := c.Param("id")

	// Check if user has access to this endpoint
	endpoint, err := s.store.GetEndpointByID(c.Request.Context(), endpointID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Endpoint not found"})
		return
	}

	orgID := c.GetString("organization_id")
	if endpoint.OrganizationID != orgID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	whitelist, err := s.store.GetWhitelist(c.Request.Context(), endpointID)
	if err != nil {
		s.monitor.LogError("Failed to get whitelist", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get whitelist"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"whitelist": whitelist})
}

// listNodes lists all nodes
func (s *Server) listNodes(c *gin.Context) {
	nodes := s.nodeManager.GetAllNodes()

	// Convert to response format
	responses := make([]*storage.NodeStatus, len(nodes))
	for i, node := range nodes {
		responses[i] = &storage.NodeStatus{
			ID:          node.ID,
			Name:        node.Name,
			IP:          node.IP,
			Port:        node.Port,
			Status:      node.Status,
			LastSeen:    node.LastSeen,
			CPUUsage:    node.CPUUsage,
			MemoryUsage: node.MemoryUsage,
			PacketRate:  node.PacketRate,
			Endpoints:   node.Endpoints,
		}
	}

	c.JSON(http.StatusOK, gin.H{"nodes": responses})
}

// getNode gets a specific node
func (s *Server) getNode(c *gin.Context) {
	nodeID := c.Param("id")

	node, exists := s.nodeManager.GetNode(nodeID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Node not found"})
		return
	}

	response := &storage.NodeStatus{
		ID:          node.ID,
		Name:        node.Name,
		IP:          node.IP,
		Port:        node.Port,
		Status:      node.Status,
		LastSeen:    node.LastSeen,
		CPUUsage:    node.CPUUsage,
		MemoryUsage: node.MemoryUsage,
		PacketRate:  node.PacketRate,
		Endpoints:   node.Endpoints,
	}

	c.JSON(http.StatusOK, response)
}

// getNodeStatus gets the status of a specific node
func (s *Server) getNodeStatus(c *gin.Context) {
	nodeID := c.Param("id")

	node, exists := s.nodeManager.GetNode(nodeID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Node not found"})
		return
	}

	// Get connections for this node
	connections := s.proxyManager.GetConnectionsForEndpoint(nodeID)

	c.JSON(http.StatusOK, gin.H{
		"node":        node,
		"connections": len(connections),
	})
}

// addToBlacklist adds an IP to the global blacklist
func (s *Server) addToBlacklist(c *gin.Context) {
	var req struct {
		IP       string `json:"ip" binding:"required,ip"`
		Reason   string `json:"reason" binding:"required"`
		Duration int    `json:"duration" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Add to blacklist
	entry := &storage.IPBlacklist{
		IP:       req.IP,
		Reason:   req.Reason,
		Duration: req.Duration,
	}

	if err := s.store.AddToBlacklist(c.Request.Context(), entry); err != nil {
		s.monitor.LogError("Failed to add IP to blacklist", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add IP to blacklist"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "IP added to blacklist"})
}

// removeFromBlacklist removes an IP from the global blacklist
func (s *Server) removeFromBlacklist(c *gin.Context) {
	ip := c.Param("ip")

	if err := s.store.RemoveFromBlacklist(c.Request.Context(), ip); err != nil {
		s.monitor.LogError("Failed to remove IP from blacklist", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove IP from blacklist"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "IP removed from blacklist"})
}

// getBlacklist gets the global blacklist
func (s *Server) getBlacklist(c *gin.Context) {
	blacklist, err := s.store.GetBlacklist(c.Request.Context())
	if err != nil {
		s.monitor.LogError("Failed to get blacklist", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get blacklist"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"blacklist": blacklist})
}

// getProfile gets the user's profile
func (s *Server) getProfile(c *gin.Context) {
	userID := c.GetString("user_id")

	user, err := s.store.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":       user.ID,
		"email":    user.Email,
		"username": user.Username,
		"role":     user.Role,
		"active":   user.Active,
	})
}

// updateProfile updates the user's profile
func (s *Server) updateProfile(c *gin.Context) {
	userID := c.GetString("user_id")

	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get existing user
	user, err := s.store.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Update fields
	if req.Username != "" {
		user.Username = req.Username
	}
	if req.Email != "" {
		user.Email = req.Email
	}

	// Save to database
	if err := s.store.UpdateUser(c.Request.Context(), user); err != nil {
		s.monitor.LogError("Failed to update user profile", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profile updated successfully"})
}

// listOrganizations lists the user's organizations
func (s *Server) listOrganizations(c *gin.Context) {
	userID := c.GetString("user_id")

	organizations, err := s.store.GetOrganizationsByOwner(c.Request.Context(), userID)
	if err != nil {
		s.monitor.LogError("Failed to list organizations", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list organizations"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"organizations": organizations})
}

// getOrganization gets a specific organization
func (s *Server) getOrganization(c *gin.Context) {
	orgID := c.Param("id")

	organization, err := s.store.GetOrganizationByID(c.Request.Context(), orgID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Organization not found"})
		return
	}

	// Check if user has access to this organization
	userID := c.GetString("user_id")
	if organization.OwnerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	c.JSON(http.StatusOK, organization)
}

// updateOrganization updates an organization
func (s *Server) updateOrganization(c *gin.Context) {
	orgID := c.Param("id")

	// Get existing organization
	organization, err := s.store.GetOrganizationByID(c.Request.Context(), orgID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Organization not found"})
		return
	}

	// Check if user has access to this organization
	userID := c.GetString("user_id")
	if organization.OwnerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	var req struct {
		Name       string `json:"name"`
		Plan       string `json:"plan"`
		MaxServers int    `json:"max_servers"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update fields
	if req.Name != "" {
		organization.Name = req.Name
	}
	if req.Plan != "" {
		organization.Plan = req.Plan
	}
	if req.MaxServers > 0 {
		organization.MaxServers = req.MaxServers
	}

	// Save to database
	if err := s.store.UpdateOrganization(c.Request.Context(), organization); err != nil {
		s.monitor.LogError("Failed to update organization", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update organization"})
		return
	}

	c.JSON(http.StatusOK, organization)
}

// getSystemStatus gets the system status
func (s *Server) getSystemStatus(c *gin.Context) {
	nodes := s.nodeManager.GetAllNodes()
	connections := s.proxyManager.GetConnections()

	activeNodes := 0
	for _, node := range nodes {
		if node.Status == "active" {
			activeNodes++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":         "healthy",
		"active_nodes":   activeNodes,
		"total_nodes":    len(nodes),
		"connections":    len(connections),
		"timestamp":      time.Now().UTC(),
	})
}

// getSystemStats gets system statistics
func (s *Server) getSystemStats(c *gin.Context) {
	nodes := s.nodeManager.GetAllNodes()
	connections := s.proxyManager.GetConnections()

	totalCPU := 0.0
	totalMemory := 0.0
	totalPacketRate := int64(0)

	for _, node := range nodes {
		if node.Status == "active" {
			totalCPU += node.CPUUsage
			totalMemory += node.MemoryUsage
			totalPacketRate += node.PacketRate
		}
	}

	avgCPU := 0.0
	avgMemory := 0.0
	if len(nodes) > 0 {
		avgCPU = totalCPU / float64(len(nodes))
		avgMemory = totalMemory / float64(len(nodes))
	}

	c.JSON(http.StatusOK, gin.H{
		"nodes": gin.H{
			"total":  len(nodes),
			"active": 0, // Count active nodes
		},
		"performance": gin.H{
			"avg_cpu_usage":    avgCPU,
			"avg_memory_usage": avgMemory,
			"total_packet_rate": totalPacketRate,
		},
		"connections": len(connections),
		"timestamp":   time.Now().UTC(),
	})
}
