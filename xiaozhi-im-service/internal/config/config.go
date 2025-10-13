package config

import (
	"os"
	"strconv"
)

// Config 应用配置
type Config struct {
	Server ServerConfig `yaml:"server"`
	GRPC   GRPCConfig   `yaml:"grpc"`
	JWT    JWTConfig    `yaml:"jwt"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port  int  `yaml:"port"`
	Debug bool `yaml:"debug"`
}

// GRPCConfig gRPC配置
type GRPCConfig struct {
	AIServiceAddr    string `yaml:"ai_service_addr"`
	MaxConnections   int    `yaml:"max_connections"`
	HeartbeatSeconds int    `yaml:"heartbeat_seconds"`
	ReconnectSeconds int    `yaml:"reconnect_seconds"`
}

// JWTConfig JWT配置
type JWTConfig struct {
	Secret string `yaml:"secret"`
}

// LoadConfig 加载配置
func LoadConfig() (*Config, error) {
	config := &Config{
		Server: ServerConfig{
			Port:  getEnvAsInt("IM_SERVER_PORT", 8081),
			Debug: getEnvAsBool("IM_SERVER_DEBUG", true),
		},
		GRPC: GRPCConfig{
			AIServiceAddr:    getEnv("AI_SERVICE_GRPC_ADDR", "localhost:50051"),
			MaxConnections:   getEnvAsInt("GRPC_MAX_CONNECTIONS", 10),
			HeartbeatSeconds: getEnvAsInt("GRPC_HEARTBEAT_SECONDS", 30),
			ReconnectSeconds: getEnvAsInt("GRPC_RECONNECT_SECONDS", 5),
		},
		JWT: JWTConfig{
			Secret: getEnv("JWT_SECRET", "abc"),
		},
	}

	return config, nil
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt 获取环境变量并转换为int，如果不存在或转换失败则返回默认值
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvAsBool 获取环境变量并转换为bool，如果不存在或转换失败则返回默认值
func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
