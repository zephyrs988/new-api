package relay

import (
	"time"
)

// OptimizationConfig 优化配置
type OptimizationConfig struct {
	// 响应体大小限制（字节）
	MaxResponseSize int64 `json:"max_response_size"`

	// 读取超时时间
	ReadTimeout time.Duration `json:"read_timeout"`

	// 缓冲区大小（字节）
	BufferSize int `json:"buffer_size"`

	// 并发处理数
	Concurrency int `json:"concurrency"`

	// 是否启用流式处理
	EnableStreaming bool `json:"enable_streaming"`

	// 是否启用并发处理
	EnableConcurrency bool `json:"enable_concurrency"`
}

// DefaultOptimizationConfig 默认优化配置
func DefaultOptimizationConfig() *OptimizationConfig {
	return &OptimizationConfig{
		MaxResponseSize:   20 * 1024 * 1024, // 20MB
		ReadTimeout:       30 * time.Second, // 30秒
		BufferSize:        64 * 1024,        // 64KB
		Concurrency:       2,                // 2个并发
		EnableStreaming:   true,             // 启用流式处理
		EnableConcurrency: true,             // 启用并发处理
	}
}

// GetOptimizationConfig 获取优化配置
func GetOptimizationConfig() *OptimizationConfig {
	// 从环境变量或配置文件读取配置
	// 这里使用默认配置，实际项目中可以从配置文件读取
	config := DefaultOptimizationConfig()

	// 可以根据环境变量调整配置
	// 这里可以根据实际需求调整配置
	// 例如：根据服务器配置或环境变量来设置不同的参数

	return config
}

// ValidateConfig 验证配置
func (config *OptimizationConfig) ValidateConfig() error {
	if config.MaxResponseSize <= 0 {
		config.MaxResponseSize = 20 * 1024 * 1024
	}

	if config.ReadTimeout <= 0 {
		config.ReadTimeout = 30 * time.Second
	}

	if config.BufferSize <= 0 {
		config.BufferSize = 64 * 1024
	}

	if config.Concurrency <= 0 {
		config.Concurrency = 2
	}

	return nil
}

// GetOptimizedResponseOptimizer 获取优化的响应优化器
func GetOptimizedResponseOptimizer() *ResponseOptimizer {
	config := GetOptimizationConfig()
	config.ValidateConfig()

	optimizer := NewResponseOptimizer()
	optimizer.SetMaxSize(config.MaxResponseSize)
	optimizer.SetTimeout(config.ReadTimeout)
	optimizer.SetBufferSize(config.BufferSize)
	optimizer.SetConcurrency(config.Concurrency)

	return optimizer
}
