package cube

import (
	"fmt"
	"time"
)

var DefaultConfig = Config{
	Enable:           	false,
	BaseURL:          	"",
	APIKey:          	"",
	DefaultBucketKey: 	"",
	DefaultBucketName: 	"",

	Timeout: 10 * time.Second,
}

type Config struct {
	Enable           	bool   `json:"enable" yaml:"enable" mapstructure:"enable"`                     // 是否启用
	BaseURL          	string `json:"baseUrl" yaml:"baseUrl" mapstructure:"base_url"`                // 基础 URL
	APIKey           	string `json:"apiKey" yaml:"apiKey" mapstructure:"api_key"`                   // 应用密钥
	DefaultBucketKey 	string `json:"bucketKey" yaml:"bucketKey" mapstructure:"default_bucket_key"`  // 默认存储桶 key（业务标识）
	DefaultBucketName 	string `json:"bucketName" yaml:"bucketName" mapstructure:"default_bucket_name"` // 默认存储桶名称

	Timeout time.Duration `json:"timeout" yaml:"timeout" mapstructure:"timeout"` // HTTP 请求超时时间
}

type CubeError struct {
	Code int    
	Msg  string 
}

// Error 实现 error 接口
func (e *CubeError) Error() string {
	return fmt.Sprintf("Cube API 业务错误 [Code: %d]: %s", e.Code, e.Msg)
}

// NewCubeError 是 CubeError 的工厂方法
func NewCubeError(code int, message string) *CubeError {
	return &CubeError{
		Code: code,
		Msg:  message,
	}
}