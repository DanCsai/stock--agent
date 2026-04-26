package config

import (
	"fmt"
	"os"
)

type Config struct {
	Port           string
	MySQL          MySQLConfig
	SessionMaxAge  int
	DefaultAvatar  string
	LegacyChatFile string
	AvatarUploadDir string
}

type MySQLConfig struct {
	Host      string
	Port      string
	User      string
	Password  string
	Database  string
	Charset   string
	ParseTime string
	Location  string
}

func (c MySQLConfig) DSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?parseTime=%s&charset=%s&loc=%s",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.Database,
		c.ParseTime,
		c.Charset,
		c.Location,
	)
}

func Load() Config {
	return Config{
		Port: getenv("PORT", "8888"),
		MySQL: MySQLConfig{
			Host:      getenv("MYSQL_HOST", "127.0.0.1"),
			Port:      getenv("MYSQL_PORT", "3306"),
			User:      getenv("MYSQL_USER", "root"),
			Password:  getenv("MYSQL_PASSWORD", "root"),
			Database:  getenv("MYSQL_DATABASE", "stock_agent"),
			Charset:   getenv("MYSQL_CHARSET", "utf8mb4"),
			ParseTime: getenv("MYSQL_PARSE_TIME", "true"),
			Location:  getenv("MYSQL_LOCATION", "UTC"),
		},
		SessionMaxAge:  60 * 60 * 24 * 7,
		DefaultAvatar:  "fern",
		LegacyChatFile: getenv("LEGACY_CHAT_FILE", "data/chat.json"),
		AvatarUploadDir: getenv("AVATAR_UPLOAD_DIR", "uploads"),
	}
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
