package config

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

type Config struct {
	// 数据库配置
	DatabasePath string `json:"database_path"`

	// 服务器配置
	Port        string `json:"port"`
	Environment string `json:"environment"`

	// JWT认证配置
	JWTSecret    string        `json:"jwt_secret"`
	JWTExpiresIn time.Duration `json:"jwt_expires_in"`

	// 监控配置
	MonitorInterval time.Duration `json:"monitor_interval"`
	PingTimeout     time.Duration `json:"ping_timeout"`
	MaxConcurrent   int           `json:"max_concurrent"`

	// 日志配置
	LogLevel  string `json:"log_level"`
	LogFormat string `json:"log_format"`

	// CORS配置
	AllowOrigins     []string `json:"allow_origins"`
	AllowCredentials bool     `json:"allow_credentials"`
}

// ConfigFile 配置文件结构
type ConfigFile struct {
	Server struct {
		Port        string `json:"port"`
		Environment string `json:"environment"`
	} `json:"server"`

	Database struct {
		Path string `json:"path"`
	} `json:"database"`

	JWT struct {
		Secret    string `json:"secret"`
		ExpiresIn string `json:"expires_in"`
	} `json:"jwt"`

	Monitor struct {
		Interval      string `json:"interval"`
		PingTimeout   string `json:"ping_timeout"`
		MaxConcurrent int    `json:"max_concurrent"`
	} `json:"monitor"`

	Logging struct {
		Level  string `json:"level"`
		Format string `json:"format"`
	} `json:"logging"`

	CORS struct {
		AllowOrigins     []string `json:"allow_origins"`
		AllowCredentials bool     `json:"allow_credentials"`
	} `json:"cors"`
}

// Load 加载配置，优先级：环境变量 > 配置文件 > 默认值
func Load(configPath string) *Config {
	config := &Config{}

	// 1. 设置默认值
	setDefaults(config)

	// 2. 加载配置文件
	if configPath == "" {
		configPath = getEnv("CONFIG_PATH", "./config.json")
	}
	loadConfigFile(config, configPath)

	// 3. 环境变量覆盖
	loadEnvironmentVariables(config)

	// 4. 验证配置
	validateConfig(config)

	// 5. 打印配置信息（隐藏敏感信息）
	printConfig(config)

	return config
}

// setDefaults 设置默认配置
func setDefaults(config *Config) {
	config.DatabasePath = "./data/etamonitor.db"
	config.Port = "11451"
	config.Environment = "release"
	config.JWTSecret = "your-secret-key-change-in-production"
	config.JWTExpiresIn = 24 * time.Hour
	config.MonitorInterval = 10 * time.Second
	config.PingTimeout = 10 * time.Second
	config.MaxConcurrent = 10
	config.LogLevel = "info"
	config.LogFormat = "json"
	config.AllowOrigins = []string{"*"}
	config.AllowCredentials = true
}

// loadConfigFile 从配置文件加载配置
func loadConfigFile(config *Config, configPath string) {
	// 检查配置文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Printf("配置文件不存在: %s，使用默认配置", configPath)
		return
	}

	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Printf("读取配置文件失败: %v，使用默认配置", err)
		return
	}

	// 解析配置文件
	var configFile ConfigFile
	if err := json.Unmarshal(data, &configFile); err != nil {
		log.Printf("解析配置文件失败: %v，使用默认配置", err)
		return
	}

	// 应用配置文件设置
	applyConfigFile(config, &configFile)
	log.Printf("成功加载配置文件: %s", configPath)
}

// applyConfigFile 应用配置文件设置
func applyConfigFile(config *Config, configFile *ConfigFile) {
	if configFile.Server.Port != "" {
		config.Port = configFile.Server.Port
	}
	if configFile.Server.Environment != "" {
		config.Environment = configFile.Server.Environment
	}

	if configFile.Database.Path != "" {
		config.DatabasePath = configFile.Database.Path
	}

	if configFile.JWT.Secret != "" {
		config.JWTSecret = configFile.JWT.Secret
	}
	if configFile.JWT.ExpiresIn != "" {
		if duration, err := time.ParseDuration(configFile.JWT.ExpiresIn); err == nil {
			config.JWTExpiresIn = duration
		}
	}

	if configFile.Monitor.Interval != "" {
		if duration, err := time.ParseDuration(configFile.Monitor.Interval); err == nil {
			config.MonitorInterval = duration
		}
	}
	if configFile.Monitor.PingTimeout != "" {
		if duration, err := time.ParseDuration(configFile.Monitor.PingTimeout); err == nil {
			config.PingTimeout = duration
		}
	}
	if configFile.Monitor.MaxConcurrent > 0 {
		config.MaxConcurrent = configFile.Monitor.MaxConcurrent
	}

	if configFile.Logging.Level != "" {
		config.LogLevel = configFile.Logging.Level
	}
	if configFile.Logging.Format != "" {
		config.LogFormat = configFile.Logging.Format
	}

	if len(configFile.CORS.AllowOrigins) > 0 {
		config.AllowOrigins = configFile.CORS.AllowOrigins
	}
	config.AllowCredentials = configFile.CORS.AllowCredentials
}

// loadEnvironmentVariables 从环境变量加载配置
func loadEnvironmentVariables(config *Config) {
	config.DatabasePath = getEnv("DB_PATH", config.DatabasePath)
	config.Port = getEnv("PORT", config.Port)
	config.Environment = getEnv("GIN_MODE", config.Environment)
	config.JWTSecret = getEnv("JWT_SECRET", config.JWTSecret)
	config.LogLevel = getEnv("LOG_LEVEL", config.LogLevel)
	config.LogFormat = getEnv("LOG_FORMAT", config.LogFormat)

	if expiresIn := os.Getenv("JWT_EXPIRES_IN"); expiresIn != "" {
		if duration, err := time.ParseDuration(expiresIn); err == nil {
			config.JWTExpiresIn = duration
		}
	}

	if interval := os.Getenv("MONITOR_INTERVAL"); interval != "" {
		if duration, err := time.ParseDuration(interval); err == nil {
			config.MonitorInterval = duration
		}
	}

	if timeout := os.Getenv("PING_TIMEOUT"); timeout != "" {
		if duration, err := time.ParseDuration(timeout); err == nil {
			config.PingTimeout = duration
		}
	}

	config.MaxConcurrent = getEnvInt("MAX_CONCURRENT", config.MaxConcurrent)
}

// generateRandomSecret 生成随机JWT密钥
func generateRandomSecret() (string, error) {
	// 生成32字节的随机数
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("生成随机密钥失败: %w", err)
	}
	// 转换为base64编码
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// saveConfigFile 保存配置到文件
func saveConfigFile(configPath string, config *Config) error {
	// 转换为ConfigFile格式
	configFile := ConfigFile{}
	configFile.Server.Port = config.Port
	configFile.Server.Environment = config.Environment
	configFile.Database.Path = config.DatabasePath
	configFile.JWT.Secret = config.JWTSecret
	configFile.JWT.ExpiresIn = config.JWTExpiresIn.String()
	configFile.Monitor.Interval = config.MonitorInterval.String()
	configFile.Monitor.PingTimeout = config.PingTimeout.String()
	configFile.Monitor.MaxConcurrent = config.MaxConcurrent
	configFile.Logging.Level = config.LogLevel
	configFile.Logging.Format = config.LogFormat
	configFile.CORS.AllowOrigins = config.AllowOrigins
	configFile.CORS.AllowCredentials = config.AllowCredentials

	// 格式化JSON
	data, err := json.MarshalIndent(configFile, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("保存配置文件失败: %w", err)
	}

	return nil
}

// validateConfig 验证配置并自动生成JWT密钥
func validateConfig(config *Config) {
	// 检查JWT Secret
	if config.JWTSecret == "your-secret-key-change-in-production" || config.JWTSecret == "114514" || config.JWTSecret == "" {
		// 生成新的随机密钥
		if secret, err := generateRandomSecret(); err == nil {
			config.JWTSecret = secret
			// 尝试保存到配置文件
			if err := saveConfigFile("./config.json", config); err == nil {
				log.Println("已生成并保存新的JWT密钥")
			} else {
				log.Printf("警告: 无法保存新生成的JWT密钥: %v", err)
			}
		} else {
			log.Printf("警告: 无法生成随机JWT密钥: %v", err)
		}
	}

	if config.MonitorInterval < 5*time.Second {
		log.Println("警告: 监控间隔过短，设置为5秒")
		config.MonitorInterval = 5 * time.Second
	}

	if config.PingTimeout > 30*time.Second {
		log.Println("警告: Ping超时时间过长，设置为30秒")
		config.PingTimeout = 30 * time.Second
	}
}

// printConfig 打印配置信息（隐藏敏感信息）
func printConfig(config *Config) {
	fmt.Println("=== etaMonitor 配置信息 ===")
	fmt.Printf("服务端口: %s\n", config.Port)
	fmt.Printf("运行模式: %s\n", config.Environment)
	fmt.Printf("数据库路径: %s\n", config.DatabasePath)
	fmt.Printf("监控间隔: %v\n", config.MonitorInterval)
	fmt.Printf("Ping超时: %v\n", config.PingTimeout)
	fmt.Printf("最大并发: %d\n", config.MaxConcurrent)
	fmt.Printf("日志级别: %s\n", config.LogLevel)
	fmt.Printf("JWT过期时间: %v\n", config.JWTExpiresIn)
	fmt.Println("========================")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
