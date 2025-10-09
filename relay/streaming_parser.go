package relay

import (
	"encoding/json"
	"fmt"
	"io"
	"one-api/common"
	relaycommon "one-api/relay/common"
	"time"
)

// StreamingParser 流式解析器，用于处理大响应体
type StreamingParser struct {
	maxSize int64
	timeout time.Duration
}

// NewStreamingParser 创建新的流式解析器
func NewStreamingParser() *StreamingParser {
	return &StreamingParser{
		maxSize: 20 * 1024 * 1024, // 10MB 默认限制
		timeout: 30 * time.Second, // 30秒超时
	}
}

// ParseTaskResultStreaming 流式解析任务结果
func (sp *StreamingParser) ParseTaskResultStreaming(reader io.Reader, adaptor TaskAdaptorInterface) (*relaycommon.TaskInfo, error) {
	// 使用限制读取器防止内存溢出
	limitedReader := io.LimitReader(reader, sp.maxSize)

	// 读取响应体
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// 检查是否达到大小限制
	if len(body) == int(sp.maxSize) {
		common.SysLog("Response body too large, truncated")
		return nil, fmt.Errorf("response body too large, exceeds %d bytes", sp.maxSize)
	}

	// 解析任务结果
	return adaptor.ParseTaskResult(body)
}

// ParseJSONStreaming 流式解析 JSON，避免一次性解析大 JSON
func (sp *StreamingParser) ParseJSONStreaming(reader io.Reader, target interface{}) error {
	// 使用限制读取器
	limitedReader := io.LimitReader(reader, sp.maxSize)

	// 创建 JSON 解码器
	decoder := json.NewDecoder(limitedReader)

	// 设置解析超时
	done := make(chan error, 1)
	go func() {
		done <- decoder.Decode(target)
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(sp.timeout):
		return fmt.Errorf("JSON parsing timeout after %v", sp.timeout)
	}
}

// TaskAdaptorInterface 任务适配器接口
type TaskAdaptorInterface interface {
	ParseTaskResult(body []byte) (*relaycommon.TaskInfo, error)
}

// 设置最大响应体大小
func (sp *StreamingParser) SetMaxSize(size int64) {
	sp.maxSize = size
}

// 设置解析超时时间
func (sp *StreamingParser) SetTimeout(timeout time.Duration) {
	sp.timeout = timeout
}
