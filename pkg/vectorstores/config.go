package vectorstores

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Provider-specific configuration structures.
// Each provider can define its own configuration struct that embeds BaseConfig.

type BaseConfig struct {
	// Common configuration fields shared across all providers
	Name        string   `mapstructure:"name" yaml:"name" json:"name" validate:"required"`
	Enabled     bool     `mapstructure:"enabled" yaml:"enabled" json:"enabled" default:"true"`
	Description string   `mapstructure:"description" yaml:"description" json:"description"`
	Tags        []string `mapstructure:"tags" yaml:"tags" json:"tags"`
}

// InMemoryConfig holds configuration for the in-memory vector store provider.
type InMemoryConfig struct {
	BaseConfig `mapstructure:",squash" yaml:",inline" json:",inline"`

	// In-memory specific configuration
	MaxDocuments int `mapstructure:"max_documents" yaml:"max_documents" json:"max_documents" default:"10000"`
}

// PgVectorConfig holds configuration for the PostgreSQL vector store provider.
type PgVectorConfig struct {
	BaseConfig `mapstructure:",squash" yaml:",inline" json:",inline"`

	// PostgreSQL connection configuration
	Host     string `mapstructure:"host" yaml:"host" json:"host" validate:"required" default:"localhost"`
	Port     int    `mapstructure:"port" yaml:"port" json:"port" default:"5432"`
	Database string `mapstructure:"database" yaml:"database" json:"database" validate:"required"`
	User     string `mapstructure:"user" yaml:"user" json:"user" validate:"required"`
	Password string `mapstructure:"password" yaml:"password" json:"password" validate:"required"`
	SSLMode  string `mapstructure:"ssl_mode" yaml:"ssl_mode" json:"ssl_mode" default:"disable"`

	// Table configuration
	TableName  string `mapstructure:"table_name" yaml:"table_name" json:"table_name" default:"beluga_documents"`
	SchemaName string `mapstructure:"schema_name" yaml:"schema_name" json:"schema_name" default:"public"`

	// Vector configuration
	EmbeddingDim int `mapstructure:"embedding_dim" yaml:"embedding_dim" json:"embedding_dim" validate:"required,min=1"`

	// Connection pool configuration
	MaxConnections int `mapstructure:"max_connections" yaml:"max_connections" json:"max_connections" default:"10"`
	MinConnections int `mapstructure:"min_connections" yaml:"min_connections" json:"min_connections" default:"1"`

	// Search configuration
	DefaultSearchK int `mapstructure:"default_search_k" yaml:"default_search_k" json:"default_search_k" default:"5"`
}

// PineconeConfig holds configuration for the Pinecone vector store provider.
type PineconeConfig struct {
	BaseConfig `mapstructure:",squash" yaml:",inline" json:",inline"`

	// Pinecone API configuration
	APIKey      string `mapstructure:"api_key" yaml:"api_key" json:"api_key" validate:"required"`
	Environment string `mapstructure:"environment" yaml:"environment" json:"environment" validate:"required"`
	ProjectID   string `mapstructure:"project_id" yaml:"project_id" json:"project_id" validate:"required"`

	// Index configuration
	IndexName string `mapstructure:"index_name" yaml:"index_name" json:"index_name" validate:"required"`
	IndexHost string `mapstructure:"index_host" yaml:"index_host" json:"index_host"`

	// Vector configuration
	EmbeddingDim int `mapstructure:"embedding_dim" yaml:"embedding_dim" json:"embedding_dim" validate:"required,min=1"`

	// Search configuration
	DefaultSearchK int `mapstructure:"default_search_k" yaml:"default_search_k" json:"default_search_k" default:"5"`
}

// Validate validates the configuration using struct tags and custom validation rules.
func (c *InMemoryConfig) Validate() error {
	validate := validator.New()
	return validate.Struct(c)
}

// Validate validates the configuration using struct tags and custom validation rules.
func (c *PgVectorConfig) Validate() error {
	validate := validator.New()
	if err := validate.Struct(c); err != nil {
		return err
	}

	// Custom validation for port range
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", c.Port)
	}

	return nil
}

// Validate validates the configuration using struct tags and custom validation rules.
func (c *PineconeConfig) Validate() error {
	validate := validator.New()
	return validate.Struct(c)
}

// GetConnectionString returns a PostgreSQL connection string from the configuration.
func (c *PgVectorConfig) GetConnectionString() string {
	return fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		c.Host, c.Port, c.Database, c.User, c.Password, c.SSLMode)
}

// GetFullTableName returns the fully qualified table name including schema.
func (c *PgVectorConfig) GetFullTableName() string {
	if c.SchemaName != "" {
		return fmt.Sprintf("%s.%s", c.SchemaName, c.TableName)
	}
	return c.TableName
}

// ConfigLoader handles loading and validating configuration from various sources.
type ConfigLoader struct {
	validator *validator.Validate
}

// NewConfigLoader creates a new configuration loader with custom validation rules.
func NewConfigLoader() *ConfigLoader {
	v := validator.New()

	// Register custom validation functions
	v.RegisterValidation("port_range", validatePortRange)

	return &ConfigLoader{
		validator: v,
	}
}

// LoadInMemoryConfig loads and validates InMemoryConfig from a map.
func (cl *ConfigLoader) LoadInMemoryConfig(data map[string]interface{}) (*InMemoryConfig, error) {
	config := &InMemoryConfig{}
	if err := cl.loadFromMap(data, config); err != nil {
		return nil, err
	}
	return config, config.Validate()
}

// LoadPgVectorConfig loads and validates PgVectorConfig from a map.
func (cl *ConfigLoader) LoadPgVectorConfig(data map[string]interface{}) (*PgVectorConfig, error) {
	config := &PgVectorConfig{}
	if err := cl.loadFromMap(data, config); err != nil {
		return nil, err
	}
	return config, config.Validate()
}

// LoadPineconeConfig loads and validates PineconeConfig from a map.
func (cl *ConfigLoader) LoadPineconeConfig(data map[string]interface{}) (*PineconeConfig, error) {
	config := &PineconeConfig{}
	if err := cl.loadFromMap(data, config); err != nil {
		return nil, err
	}
	return config, config.Validate()
}

// loadFromMap populates a struct from a map using reflection and struct tags.
func (cl *ConfigLoader) loadFromMap(data map[string]interface{}, target interface{}) error {
	v := reflect.ValueOf(target).Elem()
	t := reflect.TypeOf(target).Elem()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// Get the mapstructure tag
		tag := field.Tag.Get("mapstructure")
		if tag == "" {
			tag = strings.ToLower(field.Name)
		}

		// Handle embedded structs
		if tag == ",squash" || field.Anonymous {
			continue
		}

		if value, exists := data[tag]; exists {
			if err := cl.setFieldValue(fieldValue, value, field.Tag.Get("default")); err != nil {
				return fmt.Errorf("failed to set field %s: %w", field.Name, err)
			}
		} else if defaultValue := field.Tag.Get("default"); defaultValue != "" {
			if err := cl.setFieldValue(fieldValue, defaultValue, ""); err != nil {
				return fmt.Errorf("failed to set default value for field %s: %w", field.Name, err)
			}
		}
	}

	return nil
}

// setFieldValue sets a field value with type conversion.
func (cl *ConfigLoader) setFieldValue(field reflect.Value, value interface{}, defaultValue string) error {
	if !field.CanSet() {
		return nil
	}

	// Handle nil values
	if value == nil {
		if defaultValue != "" {
			value = defaultValue
		} else {
			return nil
		}
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(toString(value))
	case reflect.Int, reflect.Int64:
		intVal, err := toInt(value)
		if err != nil {
			return err
		}
		field.SetInt(intVal)
	case reflect.Bool:
		boolVal, err := toBool(value)
		if err != nil {
			return err
		}
		field.SetBool(boolVal)
	case reflect.Slice:
		if field.Type().Elem().Kind() == reflect.String {
			sliceVal, err := toStringSlice(value)
			if err != nil {
				return err
			}
			field.Set(reflect.ValueOf(sliceVal))
		}
	}

	return nil
}

// Helper functions for type conversion
func toString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	default:
		return fmt.Sprintf("%v", v)
	}
}

func toInt(v interface{}) (int64, error) {
	switch val := v.(type) {
	case int:
		return int64(val), nil
	case int64:
		return val, nil
	case float64:
		return int64(val), nil
	case string:
		return strconv.ParseInt(val, 10, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to int", v)
	}
}

func toBool(v interface{}) (bool, error) {
	switch val := v.(type) {
	case bool:
		return val, nil
	case string:
		return strconv.ParseBool(val)
	default:
		return false, fmt.Errorf("cannot convert %T to bool", v)
	}
}

func toStringSlice(v interface{}) ([]string, error) {
	switch val := v.(type) {
	case []string:
		return val, nil
	case []interface{}:
		result := make([]string, len(val))
		for i, item := range val {
			result[i] = toString(item)
		}
		return result, nil
	case string:
		return strings.Split(val, ","), nil
	default:
		return nil, fmt.Errorf("cannot convert %T to []string", v)
	}
}

// Custom validation functions
func validatePortRange(fl validator.FieldLevel) bool {
	port := fl.Field().Int()
	return port >= 1 && port <= 65535
}
