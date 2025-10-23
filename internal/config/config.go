package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Debug      bool           `yaml:"debug"`
	API        APIConfig      `yaml:"api"`
	Database   DatabaseConfig `yaml:"database"`
	Node       NodeConfig     `yaml:"node"`
	Proxy      ProxyConfig    `yaml:"proxy"`
	Monitoring MonitoringConfig `yaml:"monitoring"`
	Security   SecurityConfig `yaml:"security"`
}

// APIConfig represents API server configuration
type APIConfig struct {
	Address         string        `yaml:"address"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	IdleTimeout     time.Duration `yaml:"idle_timeout"`
	MaxHeaderBytes  int           `yaml:"max_header_bytes"`
	EnableCORS      bool          `yaml:"enable_cors"`
	EnableMetrics   bool          `yaml:"enable_metrics"`
	RateLimitRPS    int           `yaml:"rate_limit_rps"`
	RateLimitBurst  int           `yaml:"rate_limit_burst"`
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Type         string `yaml:"type"`
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	Database     string `yaml:"database"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
	SSLMode      string `yaml:"ssl_mode"`
	MaxOpenConns int    `yaml:"max_open_conns"`
	MaxIdleConns int    `yaml:"max_idle_conns"`
	MaxLifetime  time.Duration `yaml:"max_lifetime"`
}

// NodeConfig represents node management configuration
type NodeConfig struct {
	UpdateInterval    time.Duration `yaml:"update_interval"`
	HealthCheckInterval time.Duration `yaml:"health_check_interval"`
	MaxNodes          int           `yaml:"max_nodes"`
	NodeTimeout       time.Duration `yaml:"node_timeout"`
	RetryAttempts     int           `yaml:"retry_attempts"`
	RetryDelay        time.Duration `yaml:"retry_delay"`
}

// ProxyConfig represents proxy configuration
type ProxyConfig struct {
	EnableTCPProxy    bool          `yaml:"enable_tcp_proxy"`
	EnableUDPProxy    bool          `yaml:"enable_udp_proxy"`
	TCPTimeout        time.Duration `yaml:"tcp_timeout"`
	UDPTimeout        time.Duration `yaml:"udp_timeout"`
	MaxConnections    int           `yaml:"max_connections"`
	BufferSize        int           `yaml:"buffer_size"`
	EnableAFXDP       bool          `yaml:"enable_af_xdp"`
	XDPInterface      string        `yaml:"xdp_interface"`
	XDPQueueID        int           `yaml:"xdp_queue_id"`
}

// MonitoringConfig represents monitoring configuration
type MonitoringConfig struct {
	EnablePrometheus bool          `yaml:"enable_prometheus"`
	MetricsPath      string        `yaml:"metrics_path"`
	EnableLogging    bool          `yaml:"enable_logging"`
	LogLevel         string        `yaml:"log_level"`
	LogFormat        string        `yaml:"log_format"`
	EnableTracing    bool          `yaml:"enable_tracing"`
	TraceEndpoint    string        `yaml:"trace_endpoint"`
	SampleRate       float64       `yaml:"sample_rate"`
}

// SecurityConfig represents security configuration
type SecurityConfig struct {
	EnableTLS        bool   `yaml:"enable_tls"`
	TLSCertFile      string `yaml:"tls_cert_file"`
	TLSKeyFile       string `yaml:"tls_key_file"`
	EnableJWT        bool   `yaml:"enable_jwt"`
	JWTSecret        string `yaml:"jwt_secret"`
	JWTExpiry        time.Duration `yaml:"jwt_expiry"`
	EnableRBAC       bool   `yaml:"enable_rbac"`
	EnableRateLimit  bool   `yaml:"enable_rate_limit"`
	RateLimitRPS     int    `yaml:"rate_limit_rps"`
	RateLimitBurst   int    `yaml:"rate_limit_burst"`
	EnableIPWhitelist bool  `yaml:"enable_ip_whitelist"`
	AllowedIPs       []string `yaml:"allowed_ips"`
}

// Load loads configuration from file
func Load(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
	config.setDefaults()

	// Validate configuration
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// setDefaults sets default values for configuration
func (c *Config) setDefaults() {
	if c.API.Address == "" {
		c.API.Address = ":8080"
	}
	if c.API.ReadTimeout == 0 {
		c.API.ReadTimeout = 30 * time.Second
	}
	if c.API.WriteTimeout == 0 {
		c.API.WriteTimeout = 30 * time.Second
	}
	if c.API.IdleTimeout == 0 {
		c.API.IdleTimeout = 120 * time.Second
	}
	if c.API.MaxHeaderBytes == 0 {
		c.API.MaxHeaderBytes = 1 << 20 // 1MB
	}
	if c.API.RateLimitRPS == 0 {
		c.API.RateLimitRPS = 100
	}
	if c.API.RateLimitBurst == 0 {
		c.API.RateLimitBurst = 200
	}

	if c.Database.Type == "" {
		c.Database.Type = "postgres"
	}
	if c.Database.Host == "" {
		c.Database.Host = "localhost"
	}
	if c.Database.Port == 0 {
		c.Database.Port = 5432
	}
	if c.Database.Database == "" {
		c.Database.Database = "cloudnordsp"
	}
	if c.Database.SSLMode == "" {
		c.Database.SSLMode = "disable"
	}
	if c.Database.MaxOpenConns == 0 {
		c.Database.MaxOpenConns = 25
	}
	if c.Database.MaxIdleConns == 0 {
		c.Database.MaxIdleConns = 5
	}
	if c.Database.MaxLifetime == 0 {
		c.Database.MaxLifetime = 5 * time.Minute
	}

	if c.Node.UpdateInterval == 0 {
		c.Node.UpdateInterval = 30 * time.Second
	}
	if c.Node.HealthCheckInterval == 0 {
		c.Node.HealthCheckInterval = 10 * time.Second
	}
	if c.Node.MaxNodes == 0 {
		c.Node.MaxNodes = 100
	}
	if c.Node.NodeTimeout == 0 {
		c.Node.NodeTimeout = 30 * time.Second
	}
	if c.Node.RetryAttempts == 0 {
		c.Node.RetryAttempts = 3
	}
	if c.Node.RetryDelay == 0 {
		c.Node.RetryDelay = 5 * time.Second
	}

	if c.Proxy.TCPTimeout == 0 {
		c.Proxy.TCPTimeout = 30 * time.Second
	}
	if c.Proxy.UDPTimeout == 0 {
		c.Proxy.UDPTimeout = 10 * time.Second
	}
	if c.Proxy.MaxConnections == 0 {
		c.Proxy.MaxConnections = 10000
	}
	if c.Proxy.BufferSize == 0 {
		c.Proxy.BufferSize = 4096
	}
	if c.Proxy.XDPInterface == "" {
		c.Proxy.XDPInterface = "eth0"
	}

	if c.Monitoring.MetricsPath == "" {
		c.Monitoring.MetricsPath = "/metrics"
	}
	if c.Monitoring.LogLevel == "" {
		c.Monitoring.LogLevel = "info"
	}
	if c.Monitoring.LogFormat == "" {
		c.Monitoring.LogFormat = "json"
	}
	if c.Monitoring.SampleRate == 0 {
		c.Monitoring.SampleRate = 0.1
	}

	if c.Security.JWTExpiry == 0 {
		c.Security.JWTExpiry = 24 * time.Hour
	}
	if c.Security.RateLimitRPS == 0 {
		c.Security.RateLimitRPS = 100
	}
	if c.Security.RateLimitBurst == 0 {
		c.Security.RateLimitBurst = 200
	}
}

// validate validates the configuration
func (c *Config) validate() error {
	if c.API.Address == "" {
		return fmt.Errorf("API address is required")
	}

	if c.Database.Type == "" {
		return fmt.Errorf("database type is required")
	}

	if c.Security.EnableTLS {
		if c.Security.TLSCertFile == "" || c.Security.TLSKeyFile == "" {
			return fmt.Errorf("TLS certificate and key files are required when TLS is enabled")
		}
	}

	if c.Security.EnableJWT && c.Security.JWTSecret == "" {
		return fmt.Errorf("JWT secret is required when JWT is enabled")
	}

	return nil
}
