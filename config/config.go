package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// Config 服务器配置
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Network  NetworkConfig  `mapstructure:"network"`
	Storage  StorageConfig  `mapstructure:"storage"`
	Security SecurityConfig `mapstructure:"security"`
	Logger   LoggerConfig   `mapstructure:"logger"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	BindAddress string `mapstructure:"bind_address"`
	Port        int    `mapstructure:"port"`
	DataDir     string `mapstructure:"data_dir"`
	BaseDir     string `mapstructure:"base_dir"`
	User        string `mapstructure:"user"`
	ProfilePort int    `mapstructure:"profile_port"`
}

// NetworkConfig 网络配置
type NetworkConfig struct {
	TCPKeepAlive      bool   `mapstructure:"tcp_keep_alive"`
	KeepAlivePeriod   string `mapstructure:"keep_alive_period"`
	TCPReadTimeout    string `mapstructure:"tcp_read_timeout"`
	TCPWriteTimeout   string `mapstructure:"tcp_write_timeout"`
	MaxMsgLen         int    `mapstructure:"max_msg_len"`
	CompressEncoding  bool   `mapstructure:"compress_encoding"`
	MaxConnections    int    `mapstructure:"max_connections"`
	ConnectionTimeout string `mapstructure:"connection_timeout"`
}

// StorageConfig 存储配置
type StorageConfig struct {
	Engine          string `mapstructure:"engine"`
	JournalEnabled  bool   `mapstructure:"journal_enabled"`
	OplogSizeMB     int    `mapstructure:"oplog_size_mb"`
	CacheSizeGB     int    `mapstructure:"cache_size_gb"`
	DirectoryForDB  string `mapstructure:"directory_for_db"`
	SyncPeriodSecs  int    `mapstructure:"sync_period_secs"`
	CheckpointSecs  int    `mapstructure:"checkpoint_secs"`
	WiredTigerCache int    `mapstructure:"wired_tiger_cache"`
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	Authorization   bool   `mapstructure:"authorization"`
	AuthMechanism   string `mapstructure:"auth_mechanism"`
	KeyFile         string `mapstructure:"key_file"`
	ClusterAuthMode string `mapstructure:"cluster_auth_mode"`
	SSLMode         string `mapstructure:"ssl_mode"`
	SSLPEMKeyFile   string `mapstructure:"ssl_pem_key_file"`
	SSLCAFile       string `mapstructure:"ssl_ca_file"`
}

// LoggerConfig 日志配置
type LoggerConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	Output     string `mapstructure:"output"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
	Compress   bool   `mapstructure:"compress"`
}

// LoadConfig 加载配置文件
func LoadConfig(configPath string) (*Config, error) {
	// 设置默认值
	setDefaults()

	viper.SetConfigFile(configPath)
	viper.SetConfigType("toml")

	// 如果配置文件存在则读取
	if _, err := os.Stat(configPath); err == nil {
		if err := viper.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("读取配置文件失败: %w", err)
		}
	} else {
		// 配置文件不存在，创建默认配置文件
		if err := createDefaultConfig(configPath); err != nil {
			return nil, fmt.Errorf("创建默认配置文件失败: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	return &config, nil
}

// setDefaults 设置默认配置值
func setDefaults() {
	// Server defaults
	viper.SetDefault("server.bind_address", "127.0.0.1")
	viper.SetDefault("server.port", 27017)
	viper.SetDefault("server.data_dir", "./data")
	viper.SetDefault("server.base_dir", "./")
	viper.SetDefault("server.user", "mongodb")
	viper.SetDefault("server.profile_port", 6060)

	// Network defaults
	viper.SetDefault("network.tcp_keep_alive", true)
	viper.SetDefault("network.keep_alive_period", "180s")
	viper.SetDefault("network.tcp_read_timeout", "30s")
	viper.SetDefault("network.tcp_write_timeout", "30s")
	viper.SetDefault("network.max_msg_len", 67108864) // 64MB
	viper.SetDefault("network.compress_encoding", false)
	viper.SetDefault("network.max_connections", 1000)
	viper.SetDefault("network.connection_timeout", "30s")

	// Storage defaults
	viper.SetDefault("storage.engine", "wiredTiger")
	viper.SetDefault("storage.journal_enabled", true)
	viper.SetDefault("storage.oplog_size_mb", 1024)
	viper.SetDefault("storage.cache_size_gb", 1)
	viper.SetDefault("storage.directory_for_db", "./data/db")
	viper.SetDefault("storage.sync_period_secs", 60)
	viper.SetDefault("storage.checkpoint_secs", 60)
	viper.SetDefault("storage.wired_tiger_cache", 1073741824) // 1GB

	// Security defaults
	viper.SetDefault("security.authorization", false)
	viper.SetDefault("security.auth_mechanism", "SCRAM-SHA-256")
	viper.SetDefault("security.key_file", "")
	viper.SetDefault("security.cluster_auth_mode", "keyFile")
	viper.SetDefault("security.ssl_mode", "disabled")
	viper.SetDefault("security.ssl_pem_key_file", "")
	viper.SetDefault("security.ssl_ca_file", "")

	// Logger defaults
	viper.SetDefault("logger.level", "info")
	viper.SetDefault("logger.format", "json")
	viper.SetDefault("logger.output", "stdout")
	viper.SetDefault("logger.max_size", 100)
	viper.SetDefault("logger.max_backups", 3)
	viper.SetDefault("logger.max_age", 30)
	viper.SetDefault("logger.compress", true)
}

// createDefaultConfig 创建默认配置文件
func createDefaultConfig(configPath string) error {
	configContent := `# XMongoDB Server 配置文件

[server]
bind_address = "127.0.0.1"
port = 27017
data_dir = "./data"
base_dir = "./"
user = "mongodb"
profile_port = 6060

[network]
tcp_keep_alive = true
keep_alive_period = "180s"
tcp_read_timeout = "30s"
tcp_write_timeout = "30s"
max_msg_len = 67108864
compress_encoding = false
max_connections = 1000
connection_timeout = "30s"

[storage]
engine = "wiredTiger"
journal_enabled = true
oplog_size_mb = 1024
cache_size_gb = 1
directory_for_db = "./data/db"
sync_period_secs = 60
checkpoint_secs = 60
wired_tiger_cache = 1073741824

[security]
authorization = false
auth_mechanism = "SCRAM-SHA-256"
key_file = ""
cluster_auth_mode = "keyFile"
ssl_mode = "disabled"
ssl_pem_key_file = ""
ssl_ca_file = ""

[logger]
level = "info"
format = "json"
output = "stdout"
max_size = 100
max_backups = 3
max_age = 30
compress = true
`

	return os.WriteFile(configPath, []byte(configContent), 0644)
}
