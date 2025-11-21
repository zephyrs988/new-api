package relay

import (
	"fmt"
	"github.com/QuantumNous/new-api/common"
	"sync"
	"time"
)

// TimeoutDiagnostics 超时诊断工具
type TimeoutDiagnostics struct {
	timeoutEvents []TimeoutEvent
	mutex         sync.RWMutex
}

// TimeoutEvent 超时事件记录
type TimeoutEvent struct {
	Timestamp    time.Time     `json:"timestamp"`
	Duration     time.Duration `json:"duration"`
	BytesRead    int64         `json:"bytes_read"`
	ExpectedSize int64         `json:"expected_size"`
	ErrorType    string        `json:"error_type"`
	Source       string        `json:"source"`
}

// GlobalTimeoutDiagnostics 全局超时诊断器
var GlobalTimeoutDiagnostics = &TimeoutDiagnostics{
	timeoutEvents: make([]TimeoutEvent, 0),
}

// RecordTimeoutEvent 记录超时事件
func (td *TimeoutDiagnostics) RecordTimeoutEvent(event TimeoutEvent) {
	td.mutex.Lock()
	defer td.mutex.Unlock()

	td.timeoutEvents = append(td.timeoutEvents, event)

	// 保持最近100个事件
	if len(td.timeoutEvents) > 100 {
		td.timeoutEvents = td.timeoutEvents[1:]
	}
}

// AnalyzeTimeoutPatterns 分析超时模式
func (td *TimeoutDiagnostics) AnalyzeTimeoutPatterns() map[string]interface{} {
	td.mutex.RLock()
	defer td.mutex.RUnlock()

	if len(td.timeoutEvents) == 0 {
		return map[string]interface{}{
			"message": "No timeout events recorded",
		}
	}

	// 统计信息
	totalEvents := len(td.timeoutEvents)
	avgDuration := time.Duration(0)
	avgBytesRead := int64(0)
	errorTypes := make(map[string]int)

	for _, event := range td.timeoutEvents {
		avgDuration += event.Duration
		avgBytesRead += event.BytesRead
		errorTypes[event.ErrorType]++
	}

	avgDuration /= time.Duration(totalEvents)
	avgBytesRead /= int64(totalEvents)

	// 分析最近的事件
	recentEvents := td.timeoutEvents
	if len(recentEvents) > 10 {
		recentEvents = recentEvents[len(recentEvents)-10:]
	}

	// 计算趋势
	trend := td.calculateTrend(recentEvents)

	return map[string]interface{}{
		"total_events":   totalEvents,
		"avg_duration":   avgDuration.String(),
		"avg_bytes_read": avgBytesRead,
		"error_types":    errorTypes,
		"trend":          trend,
		"recent_events":  len(recentEvents),
	}
}

// calculateTrend 计算超时趋势
func (td *TimeoutDiagnostics) calculateTrend(events []TimeoutEvent) string {
	if len(events) < 2 {
		return "insufficient_data"
	}

	// 比较前半部分和后半部分的平均持续时间
	mid := len(events) / 2
	firstHalf := events[:mid]
	secondHalf := events[mid:]

	firstAvg := td.calculateAverageDuration(firstHalf)
	secondAvg := td.calculateAverageDuration(secondHalf)

	if secondAvg > time.Duration(float64(firstAvg)*1.2) {
		return "increasing"
	} else if secondAvg < time.Duration(float64(firstAvg)*0.8) {
		return "decreasing"
	} else {
		return "stable"
	}
}

// calculateAverageDuration 计算平均持续时间
func (td *TimeoutDiagnostics) calculateAverageDuration(events []TimeoutEvent) time.Duration {
	if len(events) == 0 {
		return 0
	}

	total := time.Duration(0)
	for _, event := range events {
		total += event.Duration
	}

	return total / time.Duration(len(events))
}

// GetTimeoutRecommendations 获取超时优化建议
func (td *TimeoutDiagnostics) GetTimeoutRecommendations() []string {
	analysis := td.AnalyzeTimeoutPatterns()
	recommendations := make([]string, 0)

	// 基于分析结果提供建议
	if avgBytes, ok := analysis["avg_bytes_read"].(int64); ok && avgBytes > 0 {
		if avgBytes > 10*1024*1024 { // 大于10MB
			recommendations = append(recommendations, "考虑增加超时时间，响应体较大")
		}
	}

	if trend, ok := analysis["trend"].(string); ok {
		if trend == "increasing" {
			recommendations = append(recommendations, "超时频率增加，检查网络连接和服务器性能")
		}
	}

	if errorTypes, ok := analysis["error_types"].(map[string]int); ok {
		if errorTypes["network"] > errorTypes["timeout"] {
			recommendations = append(recommendations, "网络问题较多，检查网络配置")
		}
	}

	// 默认建议
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "当前超时配置正常，继续监控")
	}

	return recommendations
}

// LogTimeoutDiagnostics 记录超时诊断信息
func (td *TimeoutDiagnostics) LogTimeoutDiagnostics() {
	analysis := td.AnalyzeTimeoutPatterns()
	recommendations := td.GetTimeoutRecommendations()

	common.SysLog(fmt.Sprintf("Timeout Diagnostics - Analysis: %+v", analysis))
	common.SysLog(fmt.Sprintf("Timeout Recommendations: %v", recommendations))
}

// ResetTimeoutEvents 重置超时事件记录
func (td *TimeoutDiagnostics) ResetTimeoutEvents() {
	td.mutex.Lock()
	defer td.mutex.Unlock()

	td.timeoutEvents = make([]TimeoutEvent, 0)
}
