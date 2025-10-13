package relay

import (
	"context"
	"fmt"
	"io"
	"one-api/common"
	"time"
)

// TimeoutHandler 超时处理器
type TimeoutHandler struct {
	baseTimeout    time.Duration
	maxTimeout     time.Duration
	retryCount     int
	adaptiveFactor float64
}

// NewTimeoutHandler 创建新的超时处理器
func NewTimeoutHandler() *TimeoutHandler {
	return &TimeoutHandler{
		baseTimeout:    30 * time.Second, // 基础超时时间
		maxTimeout:     5 * time.Minute,  // 最大超时时间
		retryCount:     3,                // 重试次数
		adaptiveFactor: 1.5,              // 自适应因子
	}
}

// AdaptiveTimeout 自适应超时处理
func (th *TimeoutHandler) AdaptiveTimeout(reader io.Reader, expectedSize int64) ([]byte, error) {
	// 根据预期大小计算超时时间
	timeout := th.calculateTimeout(expectedSize)

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 使用管道进行读取
	pipeReader, pipeWriter := io.Pipe()

	// 启动写入协程
	go func() {
		defer pipeWriter.Close()
		_, err := io.Copy(pipeWriter, reader)
		if err != nil {
			common.SysLog(fmt.Sprintf("Error copying data: %v", err))
		}
	}()

	// 读取数据
	var result []byte
	done := make(chan error, 1)

	go func() {
		var err error
		result, err = io.ReadAll(pipeReader)
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}
		return result, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("read timeout after %v (expected size: %d bytes)", timeout, expectedSize)
	}
}

// calculateTimeout 根据预期大小计算超时时间
func (th *TimeoutHandler) calculateTimeout(expectedSize int64) time.Duration {
	// 基础超时时间
	timeout := th.baseTimeout

	// 根据大小调整超时时间
	if expectedSize > 0 {
		// 假设每秒传输 1MB，计算所需时间
		estimatedTime := time.Duration(expectedSize/1024/1024) * time.Second
		if estimatedTime > timeout {
			timeout = estimatedTime
		}
	}

	// 应用自适应因子
	timeout = time.Duration(float64(timeout) * th.adaptiveFactor)

	// 限制最大超时时间
	if timeout > th.maxTimeout {
		timeout = th.maxTimeout
	}

	return timeout
}

// RetryWithBackoff 带退避的重试机制
func (th *TimeoutHandler) RetryWithBackoff(operation func() ([]byte, error)) ([]byte, error) {
	var lastErr error

	for i := 0; i < th.retryCount; i++ {
		result, err := operation()
		if err == nil {
			return result, nil
		}

		lastErr = err

		// 如果不是超时错误，直接返回
		if !th.isTimeoutError(err) {
			return nil, err
		}

		// 计算退避时间
		backoffTime := time.Duration(i+1) * time.Second
		common.SysLog(fmt.Sprintf("Retry %d/%d after %v due to timeout", i+1, th.retryCount, backoffTime))

		time.Sleep(backoffTime)
	}

	return nil, fmt.Errorf("operation failed after %d retries: %w", th.retryCount, lastErr)
}

// isTimeoutError 检查是否为超时错误
func (th *TimeoutHandler) isTimeoutError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	return contains(errStr, "timeout") || contains(errStr, "deadline exceeded")
}

// contains 检查字符串是否包含子字符串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}

// SetBaseTimeout 设置基础超时时间
func (th *TimeoutHandler) SetBaseTimeout(timeout time.Duration) {
	th.baseTimeout = timeout
}

// SetMaxTimeout 设置最大超时时间
func (th *TimeoutHandler) SetMaxTimeout(timeout time.Duration) {
	th.maxTimeout = timeout
}

// SetRetryCount 设置重试次数
func (th *TimeoutHandler) SetRetryCount(count int) {
	th.retryCount = count
}

// SetAdaptiveFactor 设置自适应因子
func (th *TimeoutHandler) SetAdaptiveFactor(factor float64) {
	th.adaptiveFactor = factor
}
