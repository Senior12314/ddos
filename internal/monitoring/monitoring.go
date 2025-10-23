package monitoring

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudnordsp/minecraft-protection/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

// Monitoring handles metrics and logging
type Monitoring struct {
	config *config.MonitoringConfig
	logger *zap.Logger

	// Prometheus metrics
	httpRequestsTotal     *prometheus.CounterVec
	httpRequestDuration   *prometheus.HistogramVec
	httpRequestSize       *prometheus.HistogramVec
	activeConnections     prometheus.Gauge
	endpointsTotal        prometheus.Gauge
	nodesTotal            prometheus.Gauge
	packetsProcessed      *prometheus.CounterVec
	packetsBlocked        *prometheus.CounterVec
	rateLimitHits         *prometheus.CounterVec
	blacklistHits         *prometheus.CounterVec
	challengeHits         *prometheus.CounterVec
	udpChallengesSent     *prometheus.CounterVec
	udpChallengesPassed   *prometheus.CounterVec
	nodeCPUUsage          *prometheus.GaugeVec
	nodeMemoryUsage       *prometheus.GaugeVec
	nodePacketRate        *prometheus.GaugeVec
}

// New creates a new monitoring instance
func New(cfg *config.MonitoringConfig) *Monitoring {
	// Initialize logger
	var logger *zap.Logger
	var err error

	if cfg.EnableLogging {
		if cfg.LogFormat == "json" {
			logger, err = zap.NewProduction()
		} else {
			logger, err = zap.NewDevelopment()
		}
		if err != nil {
			panic(fmt.Sprintf("Failed to initialize logger: %v", err))
		}
	} else {
		logger = zap.NewNop()
	}

	// Set log level
	switch cfg.LogLevel {
	case "debug":
		logger = logger.WithOptions(zap.IncreaseLevel(zap.DebugLevel))
	case "info":
		logger = logger.WithOptions(zap.IncreaseLevel(zap.InfoLevel))
	case "warn":
		logger = logger.WithOptions(zap.IncreaseLevel(zap.WarnLevel))
	case "error":
		logger = logger.WithOptions(zap.IncreaseLevel(zap.ErrorLevel))
	}

	m := &Monitoring{
		config: cfg,
		logger: logger,
	}

	// Initialize Prometheus metrics
	if cfg.EnablePrometheus {
		m.initMetrics()
	}

	return m
}

// initMetrics initializes Prometheus metrics
func (m *Monitoring) initMetrics() {
	m.httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cloudnordsp_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status_code"},
	)

	m.httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "cloudnordsp_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	m.httpRequestSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "cloudnordsp_http_request_size_bytes",
			Help:    "HTTP request size in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "endpoint"},
	)

	m.activeConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "cloudnordsp_active_connections",
			Help: "Number of active connections",
		},
	)

	m.endpointsTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "cloudnordsp_endpoints_total",
			Help: "Total number of protected endpoints",
		},
	)

	m.nodesTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "cloudnordsp_nodes_total",
			Help: "Total number of nodes",
		},
	)

	m.packetsProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cloudnordsp_packets_processed_total",
			Help: "Total number of packets processed",
		},
		[]string{"endpoint_id", "protocol", "action"},
	)

	m.packetsBlocked = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cloudnordsp_packets_blocked_total",
			Help: "Total number of packets blocked",
		},
		[]string{"endpoint_id", "reason"},
	)

	m.rateLimitHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cloudnordsp_rate_limit_hits_total",
			Help: "Total number of rate limit hits",
		},
		[]string{"endpoint_id", "source_ip"},
	)

	m.blacklistHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cloudnordsp_blacklist_hits_total",
			Help: "Total number of blacklist hits",
		},
		[]string{"endpoint_id", "source_ip"},
	)

	m.challengeHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cloudnordsp_challenge_hits_total",
			Help: "Total number of challenge hits",
		},
		[]string{"endpoint_id", "source_ip"},
	)

	m.udpChallengesSent = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cloudnordsp_udp_challenges_sent_total",
			Help: "Total number of UDP challenges sent",
		},
		[]string{"endpoint_id", "source_ip"},
	)

	m.udpChallengesPassed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cloudnordsp_udp_challenges_passed_total",
			Help: "Total number of UDP challenges passed",
		},
		[]string{"endpoint_id", "source_ip"},
	)

	m.nodeCPUUsage = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cloudnordsp_node_cpu_usage_percent",
			Help: "Node CPU usage percentage",
		},
		[]string{"node_id", "node_name"},
	)

	m.nodeMemoryUsage = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cloudnordsp_node_memory_usage_percent",
			Help: "Node memory usage percentage",
		},
		[]string{"node_id", "node_name"},
	)

	m.nodePacketRate = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cloudnordsp_node_packet_rate",
			Help: "Node packet processing rate",
		},
		[]string{"node_id", "node_name"},
	)
}

// Logger returns the logger instance
func (m *Monitoring) Logger() *zap.Logger {
	return m.logger
}

// Middleware returns Gin middleware for HTTP metrics
func (m *Monitoring) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath()
		method := c.Request.Method

		// Record request size
		if m.httpRequestSize != nil {
			m.httpRequestSize.WithLabelValues(method, path).Observe(float64(c.Request.ContentLength))
		}

		// Process request
		c.Next()

		// Record metrics
		if m.httpRequestsTotal != nil {
			m.httpRequestsTotal.WithLabelValues(method, path, fmt.Sprintf("%d", c.Writer.Status())).Inc()
		}

		if m.httpRequestDuration != nil {
			m.httpRequestDuration.WithLabelValues(method, path).Observe(time.Since(start).Seconds())
		}
	}
}

// RecordPacketProcessed records a processed packet
func (m *Monitoring) RecordPacketProcessed(endpointID, protocol, action string) {
	if m.packetsProcessed != nil {
		m.packetsProcessed.WithLabelValues(endpointID, protocol, action).Inc()
	}
}

// RecordPacketBlocked records a blocked packet
func (m *Monitoring) RecordPacketBlocked(endpointID, reason string) {
	if m.packetsBlocked != nil {
		m.packetsBlocked.WithLabelValues(endpointID, reason).Inc()
	}
}

// RecordRateLimitHit records a rate limit hit
func (m *Monitoring) RecordRateLimitHit(endpointID, sourceIP string) {
	if m.rateLimitHits != nil {
		m.rateLimitHits.WithLabelValues(endpointID, sourceIP).Inc()
	}
}

// RecordBlacklistHit records a blacklist hit
func (m *Monitoring) RecordBlacklistHit(endpointID, sourceIP string) {
	if m.blacklistHits != nil {
		m.blacklistHits.WithLabelValues(endpointID, sourceIP).Inc()
	}
}

// RecordChallengeHit records a challenge hit
func (m *Monitoring) RecordChallengeHit(endpointID, sourceIP string) {
	if m.challengeHits != nil {
		m.challengeHits.WithLabelValues(endpointID, sourceIP).Inc()
	}
}

// RecordUDPChallengeSent records a UDP challenge sent
func (m *Monitoring) RecordUDPChallengeSent(endpointID, sourceIP string) {
	if m.udpChallengesSent != nil {
		m.udpChallengesSent.WithLabelValues(endpointID, sourceIP).Inc()
	}
}

// RecordUDPChallengePassed records a UDP challenge passed
func (m *Monitoring) RecordUDPChallengePassed(endpointID, sourceIP string) {
	if m.udpChallengesPassed != nil {
		m.udpChallengesPassed.WithLabelValues(endpointID, sourceIP).Inc()
	}
}

// UpdateActiveConnections updates the active connections gauge
func (m *Monitoring) UpdateActiveConnections(count float64) {
	if m.activeConnections != nil {
		m.activeConnections.Set(count)
	}
}

// UpdateEndpointsTotal updates the total endpoints gauge
func (m *Monitoring) UpdateEndpointsTotal(count float64) {
	if m.endpointsTotal != nil {
		m.endpointsTotal.Set(count)
	}
}

// UpdateNodesTotal updates the total nodes gauge
func (m *Monitoring) UpdateNodesTotal(count float64) {
	if m.nodesTotal != nil {
		m.nodesTotal.Set(count)
	}
}

// UpdateNodeMetrics updates node-specific metrics
func (m *Monitoring) UpdateNodeMetrics(nodeID, nodeName string, cpuUsage, memoryUsage, packetRate float64) {
	if m.nodeCPUUsage != nil {
		m.nodeCPUUsage.WithLabelValues(nodeID, nodeName).Set(cpuUsage)
	}
	if m.nodeMemoryUsage != nil {
		m.nodeMemoryUsage.WithLabelValues(nodeID, nodeName).Set(memoryUsage)
	}
	if m.nodePacketRate != nil {
		m.nodePacketRate.WithLabelValues(nodeID, nodeName).Set(packetRate)
	}
}

// LogInfo logs an info message
func (m *Monitoring) LogInfo(msg string, fields ...zap.Field) {
	m.logger.Info(msg, fields...)
}

// LogError logs an error message
func (m *Monitoring) LogError(msg string, fields ...zap.Field) {
	m.logger.Error(msg, fields...)
}

// LogWarn logs a warning message
func (m *Monitoring) LogWarn(msg string, fields ...zap.Field) {
	m.logger.Warn(msg, fields...)
}

// LogDebug logs a debug message
func (m *Monitoring) LogDebug(msg string, fields ...zap.Field) {
	m.logger.Debug(msg, fields...)
}

// WithContext returns a logger with context
func (m *Monitoring) WithContext(ctx context.Context) *zap.Logger {
	return m.logger.With(
		zap.String("trace_id", getTraceID(ctx)),
		zap.String("span_id", getSpanID(ctx)),
	)
}

// getTraceID extracts trace ID from context
func getTraceID(ctx context.Context) string {
	if traceID := ctx.Value("trace_id"); traceID != nil {
		if id, ok := traceID.(string); ok {
			return id
		}
	}
	return ""
}

// getSpanID extracts span ID from context
func getSpanID(ctx context.Context) string {
	if spanID := ctx.Value("span_id"); spanID != nil {
		if id, ok := spanID.(string); ok {
			return id
		}
	}
	return ""
}
