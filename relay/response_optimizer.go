package relay

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"one-api/common"
	relaycommon "one-api/relay/common"
	"sync"
	"time"
)

// ResponseOptimizer 响应优化器，用于高效处理大响应体
type ResponseOptimizer struct {
	maxSize     int64
	timeout     time.Duration
	bufferSize  int
	concurrency int
}

// NewResponseOptimizer 创建新的响应优化器
func NewResponseOptimizer() *ResponseOptimizer {
	return &ResponseOptimizer{
		maxSize:     10 * 1024 * 1024, // 10MB 默认限制
		timeout:     30 * time.Second, // 30秒超时
		bufferSize:  64 * 1024,        // 64KB 缓冲区
		concurrency: 2,                // 并发处理数
	}
}

// OptimizedReadAll 优化的读取方法，使用缓冲区和并发处理
func (ro *ResponseOptimizer) OptimizedReadAll(reader io.Reader) ([]byte, error) {
	startTime := time.Now()

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), ro.timeout)
	defer cancel()

	// 使用限制读取器
	limitedReader := io.LimitReader(reader, ro.maxSize)

	// 使用缓冲读取器提高性能
	bufferedReader := bufio.NewReaderSize(limitedReader, ro.bufferSize)

	// 使用管道进行并发处理
	pipeReader, pipeWriter := io.Pipe()

	// 启动写入协程
	go func() {
		defer pipeWriter.Close()
		_, err := io.Copy(pipeWriter, bufferedReader)
		if err != nil {
			common.SysLog(fmt.Sprintf("Error copying data: %v", err))
		}
	}()

	// 读取数据
	var result bytes.Buffer
	done := make(chan error, 1)

	go func() {
		_, err := io.Copy(&result, pipeReader)
		done <- err
	}()

	var err error
	select {
	case err = <-done:
		if err != nil {
			// 记录错误
			GlobalPerformanceMonitor.RecordReadOperation(0, time.Since(startTime), false, true)
			return nil, fmt.Errorf("failed to read response: %w", err)
		}
	case <-ctx.Done():
		// 记录超时错误
		GlobalPerformanceMonitor.RecordReadOperation(0, time.Since(startTime), false, true)
		return nil, fmt.Errorf("read timeout after %v", ro.timeout)
	}

	// 记录性能指标
	readTime := time.Since(startTime)
	bytesRead := int64(result.Len())
	truncated := bytesRead == ro.maxSize

	GlobalPerformanceMonitor.RecordReadOperation(bytesRead, readTime, truncated, false)

	return result.Bytes(), nil
}

// ParseTaskResultOptimized 优化的任务结果解析
func (ro *ResponseOptimizer) ParseTaskResultOptimized(reader io.Reader, adaptor TaskAdaptorInterface) (*relaycommon.TaskInfo, error) {
	body, err := ro.OptimizedReadAll(reader)
	if err != nil {
		return nil, err
	}

	// 检查是否达到大小限制
	if len(body) == int(ro.maxSize) {
		common.SysLog("Response body too large, truncated")
		return nil, fmt.Errorf("response body too large, exceeds %d bytes", ro.maxSize)
	}

	// 解析任务结果
	return adaptor.ParseTaskResult(body)
}

// ParseJSONOptimized 优化的 JSON 解析，使用流式解析
func (ro *ResponseOptimizer) ParseJSONOptimized(reader io.Reader, target interface{}) error {
	body, err := ro.OptimizedReadAll(reader)
	if err != nil {
		return err
	}

	// 使用 JSON 解码器
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber() // 使用数字类型而不是 float64

	return decoder.Decode(target)
}

// ConcurrentParse 并发解析多个响应
func (ro *ResponseOptimizer) ConcurrentParse(readers []io.Reader, adaptor TaskAdaptorInterface) ([]*relaycommon.TaskInfo, error) {
	if len(readers) == 0 {
		return nil, nil
	}

	// 限制并发数
	if ro.concurrency > len(readers) {
		ro.concurrency = len(readers)
	}

	results := make([]*relaycommon.TaskInfo, len(readers))
	errors := make([]error, len(readers))

	// 使用信号量控制并发
	semaphore := make(chan struct{}, ro.concurrency)
	var wg sync.WaitGroup

	for i, reader := range readers {
		wg.Add(1)
		go func(index int, r io.Reader) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 解析任务结果
			ti, err := ro.ParseTaskResultOptimized(r, adaptor)
			results[index] = ti
			errors[index] = err
		}(i, reader)
	}

	wg.Wait()

	// 检查是否有错误
	for i, err := range errors {
		if err != nil {
			return nil, fmt.Errorf("error parsing response %d: %w", i, err)
		}
	}

	return results, nil
}

// 设置最大响应体大小
func (ro *ResponseOptimizer) SetMaxSize(size int64) {
	ro.maxSize = size
}

// 设置解析超时时间
func (ro *ResponseOptimizer) SetTimeout(timeout time.Duration) {
	ro.timeout = timeout
}

// 设置缓冲区大小
func (ro *ResponseOptimizer) SetBufferSize(size int) {
	ro.bufferSize = size
}

// 设置并发数
func (ro *ResponseOptimizer) SetConcurrency(concurrency int) {
	ro.concurrency = concurrency
}
