package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config represents the application configuration
type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Kafka     KafkaConfig
	Services  ServicesConfig
	Scraper   ScraperConfig
	LogLevel  string
	Environment string
}

// ServerConfig represents the HTTP server configuration
type ServerConfig struct {
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DatabaseConfig represents the database configuration
type DatabaseConfig struct {
	Host                   string
	Port                   int
	Username               string
	Password               string
	DBName                 string
	MaxIdleConns           int
	MaxOpenConns           int
	ConnMaxLifetimeMinutes int
}

// KafkaConfig represents the Kafka configuration
type KafkaConfig struct {
	Brokers          []string
	ConsumerGroup    string
	ProductTopic     string
	NotificationTopic string
}

// ServicesConfig represents the service configurations
type ServicesConfig struct {
	CrawlerServicePort     int
	AnalyzerServicePort    int
	NotificationServicePort int
}

// ScraperConfig represents the scraper configuration
type ScraperConfig struct {
	BaseURL            string
	UserAgent          string
	RequestTimeout     time.Duration
	ConcurrentRequests int
	RequestDelay       time.Duration
	RetryAttempts      int
	RetryDelay         time.Duration
}

// LoadConfig loads the application configuration from environment variables
func LoadConfig() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	config := &Config{
		Server: ServerConfig{
			Port:         getEnvAsInt("SERVER_PORT", 8080),
			ReadTimeout:  time.Duration(getEnvAsInt("SERVER_READ_TIMEOUT", 15)) * time.Second,
			WriteTimeout: time.Duration(getEnvAsInt("SERVER_WRITE_TIMEOUT", 15)) * time.Second,
			IdleTimeout:  time.Duration(getEnvAsInt("SERVER_IDLE_TIMEOUT", 60)) * time.Second,
		},
		Database: DatabaseConfig{
			Host:                   getEnv("DB_HOST", "localhost"),
			Port:                   getEnvAsInt("DB_PORT", 5432),
			Username:               getEnv("DB_USER", "postgres"),
			Password:               getEnv("DB_PASSWORD", "postgres"),
			DBName:                 getEnv("DB_NAME", "ecommerce"),
			MaxIdleConns:           getEnvAsInt("DB_MAX_IDLE_CONNS", 10),
			MaxOpenConns:           getEnvAsInt("DB_MAX_OPEN_CONNS", 100),
			ConnMaxLifetimeMinutes: getEnvAsInt("DB_CONN_MAX_LIFETIME", 30),
		},
		Kafka: KafkaConfig{
			Brokers:           getEnvAsSlice("KAFKA_BROKERS", []string{"localhost:9092"}),
			ConsumerGroup:     getEnv("KAFKA_CONSUMER_GROUP", "ecommerce-group"),
			ProductTopic:      getEnv("KAFKA_PRODUCT_TOPIC", "product-updates"),
			NotificationTopic: getEnv("KAFKA_NOTIFICATION_TOPIC", "user-notifications"),
		},
		Services: ServicesConfig{
			CrawlerServicePort:      getEnvAsInt("CRAWLER_SERVICE_PORT", 9001),
			AnalyzerServicePort:     getEnvAsInt("ANALYZER_SERVICE_PORT", 9002),
			NotificationServicePort: getEnvAsInt("NOTIFICATION_SERVICE_PORT", 9003),
		},
		Scraper: ScraperConfig{
			BaseURL:            getEnv("SCRAPER_BASE_URL", "https://www.trendyol.com"),
			UserAgent:          getEnv("SCRAPER_USER_AGENT", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
			RequestTimeout:     time.Duration(getEnvAsInt("SCRAPER_REQUEST_TIMEOUT", 30)) * time.Second,
			ConcurrentRequests: getEnvAsInt("SCRAPER_CONCURRENT_REQUESTS", 5),
			RequestDelay:       time.Duration(getEnvAsInt("SCRAPER_REQUEST_DELAY", 1000)) * time.Millisecond,
			RetryAttempts:      getEnvAsInt("SCRAPER_RETRY_ATTEMPTS", 3),
			RetryDelay:         time.Duration(getEnvAsInt("SCRAPER_RETRY_DELAY", 5)) * time.Second,
		},
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		Environment: getEnv("ENVIRONMENT", "development"),
	}

	return config, nil
}

// Helper functions to get environment variables
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, fmt.Sprintf("%d", defaultValue))
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvAsSlice(key string, defaultValue []string) []string {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	return split(valueStr, ",")
}

func split(s string, sep string) []string {
	if s == "" {
		return []string{}
	}
	return []string{s}
}