package relay

import (
	"fmt"
	"one-api/common"
	"sync"
	"time"
)

// PerformanceMetrics 性能指标
type PerformanceMetrics struct {
	TotalRequests      int64         `json:"total_requests"`
	TotalBytesRead     int64         `json:"total_bytes_read"`
	AverageReadTime    time.Duration `json:"average_read_time"`
	MaxReadTime        time.Duration `json:"max_read_time"`
	MinReadTime        time.Duration `json:"min_read_time"`
	TruncatedResponses int64         `json:"truncated_responses"`
	ErrorCount         int64         `json:"error_count"`
	LastResetTime      time.Time     `json:"last_reset_time"`
}

// PerformanceMonitor 性能监控器
type PerformanceMonitor struct {
	metrics PerformanceMetrics
	mutex   sync.RWMutex
}

// GlobalPerformanceMonitor 全局性能监控器
var GlobalPerformanceMonitor = &PerformanceMonitor{
	metrics: PerformanceMetrics{
		LastResetTime: time.Now(),
	},
}

// RecordReadOperation 记录读取操作
func (pm *PerformanceMonitor) RecordReadOperation(bytesRead int64, readTime time.Duration, truncated bool, hasError bool) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.metrics.TotalRequests++
	pm.metrics.TotalBytesRead += bytesRead

	// 更新平均读取时间
	if pm.metrics.TotalRequests == 1 {
		pm.metrics.AverageReadTime = readTime
		pm.metrics.MaxReadTime = readTime
		pm.metrics.MinReadTime = readTime
	} else {
		// 计算新的平均时间
		totalTime := pm.metrics.AverageReadTime * time.Duration(pm.metrics.TotalRequests-1)
		pm.metrics.AverageReadTime = (totalTime + readTime) / time.Duration(pm.metrics.TotalRequests)

		// 更新最大和最小时间
		if readTime > pm.metrics.MaxReadTime {
			pm.metrics.MaxReadTime = readTime
		}
		if readTime < pm.metrics.MinReadTime {
			pm.metrics.MinReadTime = readTime
		}
	}

	if truncated {
		pm.metrics.TruncatedResponses++
	}

	if hasError {
		pm.metrics.ErrorCount++
	}
}

// GetMetrics 获取性能指标
func (pm *PerformanceMonitor) GetMetrics() PerformanceMetrics {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	// 返回副本以避免竞态条件
	return PerformanceMetrics{
		TotalRequests:      pm.metrics.TotalRequests,
		TotalBytesRead:     pm.metrics.TotalBytesRead,
		AverageReadTime:    pm.metrics.AverageReadTime,
		MaxReadTime:        pm.metrics.MaxReadTime,
		MinReadTime:        pm.metrics.MinReadTime,
		TruncatedResponses: pm.metrics.TruncatedResponses,
		ErrorCount:         pm.metrics.ErrorCount,
		LastResetTime:      pm.metrics.LastResetTime,
	}
}

// ResetMetrics 重置性能指标
func (pm *PerformanceMonitor) ResetMetrics() {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.metrics = PerformanceMetrics{
		LastResetTime: time.Now(),
	}
}

// LogMetrics 记录性能指标到日志
func (pm *PerformanceMonitor) LogMetrics() {
	metrics := pm.GetMetrics()

	common.SysLog(fmt.Sprintf("Performance Metrics - Requests: %d, Bytes: %d, Avg Time: %v, Max Time: %v, Min Time: %v, Truncated: %d, Errors: %d",
		metrics.TotalRequests,
		metrics.TotalBytesRead,
		metrics.AverageReadTime,
		metrics.MaxReadTime,
		metrics.MinReadTime,
		metrics.TruncatedResponses,
		metrics.ErrorCount,
	))
}

// GetOptimizationEffectiveness 获取优化效果
func (pm *PerformanceMonitor) GetOptimizationEffectiveness() map[string]interface{} {
	metrics := pm.GetMetrics()

	if metrics.TotalRequests == 0 {
		return map[string]interface{}{
			"message": "No data available",
		}
	}

	avgBytesPerRequest := float64(metrics.TotalBytesRead) / float64(metrics.TotalRequests)
	truncationRate := float64(metrics.TruncatedResponses) / float64(metrics.TotalRequests) * 100
	errorRate := float64(metrics.ErrorCount) / float64(metrics.TotalRequests) * 100

	return map[string]interface{}{
		"total_requests":        metrics.TotalRequests,
		"avg_bytes_per_request": avgBytesPerRequest,
		"avg_read_time":         metrics.AverageReadTime.String(),
		"truncation_rate":       fmt.Sprintf("%.2f%%", truncationRate),
		"error_rate":            fmt.Sprintf("%.2f%%", errorRate),
		"uptime":                time.Since(metrics.LastResetTime).String(),
	}
}
