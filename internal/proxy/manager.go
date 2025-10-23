package proxy

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/cloudnordsp/minecraft-protection/internal/config"
	"github.com/cloudnordsp/minecraft-protection/internal/monitoring"
	"github.com/cloudnordsp/minecraft-protection/internal/node"
	"github.com/cloudnordsp/minecraft-protection/internal/storage"
	"go.uber.org/zap"
)

// Manager manages proxy connections
type Manager struct {
	config       *config.ProxyConfig
	nodeManager  *node.Manager
	monitor      *monitoring.Monitoring

	// Connection tracking
	connections map[string]*Connection
	connMu      sync.RWMutex

	// Proxy servers
	tcpServers map[string]*TCPServer
	udpServers map[string]*UDPServer
	serversMu  sync.RWMutex

	stopCh chan struct{}
}

// Connection represents a proxy connection
type Connection struct {
	ID         string
	EndpointID string
	ClientAddr net.Addr
	ServerAddr net.Addr
	Protocol   string
	StartTime  time.Time
	LastSeen   time.Time
	BytesIn    int64
	BytesOut   int64
}

// TCPServer represents a TCP proxy server
type TCPServer struct {
	Endpoint   *storage.ProtectedEndpoint
	Listener   net.Listener
	Connections map[string]*Connection
	connMu     sync.RWMutex
	stopCh     chan struct{}
}

// UDPServer represents a UDP proxy server
type UDPServer struct {
	Endpoint   *storage.ProtectedEndpoint
	Conn       *net.UDPConn
	Connections map[string]*Connection
	connMu     sync.RWMutex
	stopCh     chan struct{}
}

// NewManager creates a new proxy manager
func NewManager(cfg *config.ProxyConfig, nodeManager *node.Manager, monitor *monitoring.Monitoring) *Manager {
	return &Manager{
		config:      cfg,
		nodeManager: nodeManager,
		monitor:     monitor,
		connections: make(map[string]*Connection),
		tcpServers:  make(map[string]*TCPServer),
		udpServers:  make(map[string]*UDPServer),
		stopCh:      make(chan struct{}),
	}
}

// Start starts the proxy manager
func (m *Manager) Start(ctx context.Context) error {
	m.monitor.LogInfo("Starting proxy manager")

	// Start existing endpoints - we'll need to get these from storage
	// For now, we'll start with an empty list
	var endpoints []*storage.ProtectedEndpoint

	for _, endpoint := range endpoints {
		if err := m.startEndpoint(ctx, endpoint); err != nil {
			m.monitor.LogError("Failed to start endpoint",
				zap.String("endpoint_id", endpoint.ID),
				zap.Error(err))
		}
	}

	return nil
}

// Stop stops the proxy manager
func (m *Manager) Stop() {
	m.monitor.LogInfo("Stopping proxy manager")

	// Stop all servers
	m.serversMu.Lock()
	for _, server := range m.tcpServers {
		server.stopCh <- struct{}{}
		server.Listener.Close()
	}
	for _, server := range m.udpServers {
		server.stopCh <- struct{}{}
		server.Conn.Close()
	}
	m.serversMu.Unlock()

	close(m.stopCh)
}

// AddEndpoint adds a new endpoint and starts its proxy
func (m *Manager) AddEndpoint(ctx context.Context, endpoint *storage.ProtectedEndpoint) error {
	m.monitor.LogInfo("Adding endpoint",
		zap.String("endpoint_id", endpoint.ID),
		zap.String("front_endpoint", fmt.Sprintf("%s:%d", endpoint.FrontIP, endpoint.FrontPort)))

	return m.startEndpoint(ctx, endpoint)
}

// RemoveEndpoint removes an endpoint and stops its proxy
func (m *Manager) RemoveEndpoint(ctx context.Context, endpointID string) error {
	m.monitor.LogInfo("Removing endpoint", zap.String("endpoint_id", endpointID))

	m.serversMu.Lock()
	defer m.serversMu.Unlock()

	// Stop TCP server if exists
	if server, exists := m.tcpServers[endpointID]; exists {
		server.stopCh <- struct{}{}
		server.Listener.Close()
		delete(m.tcpServers, endpointID)
	}

	// Stop UDP server if exists
	if server, exists := m.udpServers[endpointID]; exists {
		server.stopCh <- struct{}{}
		server.Conn.Close()
		delete(m.udpServers, endpointID)
	}

	return nil
}

// UpdateEndpoint updates an existing endpoint
func (m *Manager) UpdateEndpoint(ctx context.Context, endpoint *storage.ProtectedEndpoint) error {
	m.monitor.LogInfo("Updating endpoint",
		zap.String("endpoint_id", endpoint.ID))

	// Remove existing endpoint
	if err := m.RemoveEndpoint(ctx, endpoint.ID); err != nil {
		return err
	}

	// Add updated endpoint
	return m.AddEndpoint(ctx, endpoint)
}

// GetConnections returns all active connections
func (m *Manager) GetConnections() []*Connection {
	m.connMu.RLock()
	defer m.connMu.RUnlock()

	connections := make([]*Connection, 0, len(m.connections))
	for _, conn := range m.connections {
		connections = append(connections, conn)
	}
	return connections
}

// GetConnectionsForEndpoint returns connections for a specific endpoint
func (m *Manager) GetConnectionsForEndpoint(endpointID string) []*Connection {
	m.connMu.RLock()
	defer m.connMu.RUnlock()

	var connections []*Connection
	for _, conn := range m.connections {
		if conn.EndpointID == endpointID {
			connections = append(connections, conn)
		}
	}
	return connections
}

// startEndpoint starts a proxy for an endpoint
func (m *Manager) startEndpoint(ctx context.Context, endpoint *storage.ProtectedEndpoint) error {
	m.serversMu.Lock()
	defer m.serversMu.Unlock()

	// Start TCP server for Java Minecraft
	if endpoint.Protocol == "java" && m.config.EnableTCPProxy {
		if err := m.startTCPServer(ctx, endpoint); err != nil {
			return fmt.Errorf("failed to start TCP server: %w", err)
		}
	}

	// Start UDP server for Bedrock Minecraft
	if endpoint.Protocol == "bedrock" && m.config.EnableUDPProxy {
		if err := m.startUDPServer(ctx, endpoint); err != nil {
			return fmt.Errorf("failed to start UDP server: %w", err)
		}
	}

	return nil
}

// startTCPServer starts a TCP proxy server
func (m *Manager) startTCPServer(ctx context.Context, endpoint *storage.ProtectedEndpoint) error {
	addr := fmt.Sprintf("%s:%d", endpoint.FrontIP, endpoint.FrontPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	server := &TCPServer{
		Endpoint:    endpoint,
		Listener:    listener,
		Connections: make(map[string]*Connection),
		stopCh:      make(chan struct{}),
	}

	m.tcpServers[endpoint.ID] = server

	// Start accepting connections
	go m.handleTCPConnections(ctx, server)

	m.monitor.LogInfo("TCP server started",
		zap.String("endpoint_id", endpoint.ID),
		zap.String("address", addr))

	return nil
}

// startUDPServer starts a UDP proxy server
func (m *Manager) startUDPServer(ctx context.Context, endpoint *storage.ProtectedEndpoint) error {
	addr := fmt.Sprintf("%s:%d", endpoint.FrontIP, endpoint.FrontPort)
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to resolve UDP address %s: %w", addr, err)
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on UDP %s: %w", addr, err)
	}

	server := &UDPServer{
		Endpoint:    endpoint,
		Conn:        conn,
		Connections: make(map[string]*Connection),
		stopCh:      make(chan struct{}),
	}

	m.udpServers[endpoint.ID] = server

	// Start handling UDP packets
	go m.handleUDPPackets(ctx, server)

	m.monitor.LogInfo("UDP server started",
		zap.String("endpoint_id", endpoint.ID),
		zap.String("address", addr))

	return nil
}

// handleTCPConnections handles TCP connections
func (m *Manager) handleTCPConnections(ctx context.Context, server *TCPServer) {
	for {
		select {
		case <-server.stopCh:
			return
		case <-ctx.Done():
			return
		default:
			// Set deadline for accept
			server.Listener.(*net.TCPListener).SetDeadline(time.Now().Add(1 * time.Second))
			
			conn, err := server.Listener.Accept()
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				m.monitor.LogError("Failed to accept TCP connection",
					zap.String("endpoint_id", server.Endpoint.ID),
					zap.Error(err))
				continue
			}

			// Handle connection in goroutine
			go m.handleTCPConnection(ctx, server, conn)
		}
	}
}

// handleTCPConnection handles a single TCP connection
func (m *Manager) handleTCPConnection(ctx context.Context, server *TCPServer, clientConn net.Conn) {
	defer clientConn.Close()

	// Create connection record
	connID := fmt.Sprintf("%s-%d", clientConn.RemoteAddr().String(), time.Now().UnixNano())
	connection := &Connection{
		ID:         connID,
		EndpointID: server.Endpoint.ID,
		ClientAddr: clientConn.RemoteAddr(),
		Protocol:   "tcp",
		StartTime:  time.Now(),
		LastSeen:   time.Now(),
	}

	// Add to connection tracking
	m.connMu.Lock()
	m.connections[connID] = connection
	m.connMu.Unlock()

	server.connMu.Lock()
	server.Connections[connID] = connection
	server.connMu.Unlock()

	defer func() {
		m.connMu.Lock()
		delete(m.connections, connID)
		m.connMu.Unlock()

		server.connMu.Lock()
		delete(server.Connections, connID)
		server.connMu.Unlock()
	}()

	// Connect to origin server
	originAddr := fmt.Sprintf("%s:%d", server.Endpoint.OriginIP, server.Endpoint.OriginPort)
	serverConn, err := net.DialTimeout("tcp", originAddr, m.config.TCPTimeout)
	if err != nil {
		m.monitor.LogError("Failed to connect to origin server",
			zap.String("endpoint_id", server.Endpoint.ID),
			zap.String("origin_addr", originAddr),
			zap.Error(err))
		return
	}
	defer serverConn.Close()

	connection.ServerAddr = serverConn.RemoteAddr()

	// Start proxying data
	m.proxyTCPData(ctx, connection, clientConn, serverConn)
}

// handleUDPPackets handles UDP packets
func (m *Manager) handleUDPPackets(ctx context.Context, server *UDPServer) {
	buffer := make([]byte, m.config.BufferSize)

	for {
		select {
		case <-server.stopCh:
			return
		case <-ctx.Done():
			return
		default:
			// Set read deadline
			server.Conn.SetReadDeadline(time.Now().Add(1 * time.Second))

			n, clientAddr, err := server.Conn.ReadFromUDP(buffer)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				m.monitor.LogError("Failed to read UDP packet",
					zap.String("endpoint_id", server.Endpoint.ID),
					zap.Error(err))
				continue
			}

			// Handle packet in goroutine
			go m.handleUDPPacket(ctx, server, buffer[:n], clientAddr)
		}
	}
}

// handleUDPPacket handles a single UDP packet
func (m *Manager) handleUDPPacket(ctx context.Context, server *UDPServer, data []byte, clientAddr *net.UDPAddr) {
	// Create connection record
	connID := fmt.Sprintf("%s-%d", clientAddr.String(), time.Now().UnixNano())
	connection := &Connection{
		ID:         connID,
		EndpointID: server.Endpoint.ID,
		ClientAddr: clientAddr,
		Protocol:   "udp",
		StartTime:  time.Now(),
		LastSeen:   time.Now(),
	}

	// Add to connection tracking
	m.connMu.Lock()
	m.connections[connID] = connection
	m.connMu.Unlock()

	server.connMu.Lock()
	server.Connections[connID] = connection
	server.connMu.Unlock()

	defer func() {
		m.connMu.Lock()
		delete(m.connections, connID)
		m.connMu.Unlock()

		server.connMu.Lock()
		delete(server.Connections, connID)
		server.connMu.Unlock()
	}()

	// Connect to origin server
	originAddr := fmt.Sprintf("%s:%d", server.Endpoint.OriginIP, server.Endpoint.OriginPort)
	serverAddr, err := net.ResolveUDPAddr("udp", originAddr)
	if err != nil {
		m.monitor.LogError("Failed to resolve origin UDP address",
			zap.String("endpoint_id", server.Endpoint.ID),
			zap.String("origin_addr", originAddr),
			zap.Error(err))
		return
	}

	serverConn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		m.monitor.LogError("Failed to connect to origin UDP server",
			zap.String("endpoint_id", server.Endpoint.ID),
			zap.String("origin_addr", originAddr),
			zap.Error(err))
		return
	}
	defer serverConn.Close()

	connection.ServerAddr = serverConn.RemoteAddr()

	// Forward packet to origin
	if _, err := serverConn.Write(data); err != nil {
		m.monitor.LogError("Failed to forward UDP packet to origin",
			zap.String("endpoint_id", server.Endpoint.ID),
			zap.Error(err))
		return
	}

	connection.BytesOut += int64(len(data))

	// Read response from origin
	responseBuffer := make([]byte, m.config.BufferSize)
	serverConn.SetReadDeadline(time.Now().Add(m.config.UDPTimeout))
	n, err := serverConn.Read(responseBuffer)
	if err != nil {
		m.monitor.LogError("Failed to read UDP response from origin",
			zap.String("endpoint_id", server.Endpoint.ID),
			zap.Error(err))
		return
	}

	connection.BytesIn += int64(n)

	// Forward response to client
	if _, err := server.Conn.WriteToUDP(responseBuffer[:n], clientAddr); err != nil {
		m.monitor.LogError("Failed to forward UDP response to client",
			zap.String("endpoint_id", server.Endpoint.ID),
			zap.Error(err))
		return
	}
}

// proxyTCPData proxies data between client and server connections
func (m *Manager) proxyTCPData(ctx context.Context, connection *Connection, clientConn, serverConn net.Conn) {
	// Channel to signal when copying is done
	done := make(chan struct{}, 2)

	// Copy from client to server
	go func() {
		defer func() { done <- struct{}{} }()
		buffer := make([]byte, m.config.BufferSize)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, err := clientConn.Read(buffer)
				if err != nil {
					return
				}
				connection.BytesIn += int64(n)
				connection.LastSeen = time.Now()

				if _, err := serverConn.Write(buffer[:n]); err != nil {
					return
				}
			}
		}
	}()

	// Copy from server to client
	go func() {
		defer func() { done <- struct{}{} }()
		buffer := make([]byte, m.config.BufferSize)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, err := serverConn.Read(buffer)
				if err != nil {
					return
				}
				connection.BytesOut += int64(n)
				connection.LastSeen = time.Now()

				if _, err := clientConn.Write(buffer[:n]); err != nil {
					return
				}
			}
		}
	}()

	// Wait for either direction to finish
	<-done
}
