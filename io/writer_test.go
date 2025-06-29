package io

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/dlshle/gommon/retry"
)

func TestBufferedWriter_BasicWriteAndFlush(t *testing.T) {
	ctx := context.Background()
	var receivedData []int
	writeFunc := func(data []int) error {
		receivedData = append(receivedData, data...)
		return nil
	}

	flushThreshold := 3
	bw := NewBufferedWriter(ctx, writeFunc, &retry.RetryOptions{MaxRetries: 0}, flushThreshold, 0)

	// Test basic writes that don't exceed threshold
	bw.Write([]int{1})
	bw.Write([]int{2})
	time.Sleep(100 * time.Millisecond) // Allow time for async operations
	if len(receivedData) != 0 {
		t.Error("Expected no data to be written")
	}

	// This should trigger a flush
	bw.Write([]int{3})
	time.Sleep(200 * time.Millisecond) // Allow time for flush to complete

	if len(receivedData) != 3 {
		t.Error("Expected 3 pieces of data to be written")
	}
	if !reflect.DeepEqual(receivedData, []int{1, 2, 3}) {
		t.Error("Expected data to be [1, 2, 3]")
	}
}

func TestBufferedWriter_TimeBasedFlush(t *testing.T) {
	ctx := context.Background()
	var receivedData [][]int
	writeFunc := func(data []int) error {
		receivedData = append(receivedData, data)
		return nil
	}

	flushThreshold := 3
	flushDuration := 200 * time.Millisecond
	bw := NewBufferedWriter(ctx, writeFunc, &retry.RetryOptions{MaxRetries: 0}, flushThreshold, flushDuration)

	// Write less than threshold and wait for time-based flush
	bw.Write([]int{1})
	time.Sleep(100 * time.Millisecond)
	if len(receivedData) != 0 {
		t.Error("Data should not be flushed before timeout")
	}

	time.Sleep(150 * time.Millisecond) // Should trigger time-based flush
	if len(receivedData) != 1 {
		t.Error("Data should be flushed after timeout")
	}
	if !reflect.DeepEqual(receivedData[0], []int{1}) {
		t.Error("Data should be correctly flushed by time")
	}
}

func TestBufferedWriter_WaitFunctionality(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	var receivedData [][]int
	writeFunc := func(data []int) error {
		receivedData = append(receivedData, data)
		return nil
	}

	flushThreshold := 3
	bw := NewBufferedWriter(ctx, writeFunc, &retry.RetryOptions{MaxRetries: 0}, flushThreshold, 0)

	// Write multiple batches
	for i := 0; i < 5; i++ {
		bw.Write([]int{i})
	}

	cancelFunc()
	// Wait for all writes to complete
	waitErr := bw.Wait(200 * time.Millisecond)
	if waitErr != nil {
		t.Errorf("Wait failed: %v", waitErr)
	}
	if len(receivedData) != 2 {
		t.Error("Expected two batches of data")
	}
	if !reflect.DeepEqual(receivedData[0], []int{0, 1, 2}) {
		t.Error("Expected first batch to be [0, 1, 2]")
	}
	if !reflect.DeepEqual(receivedData[1], []int{3, 4}) {
		t.Error("Expected second batch to be [3, 4]")
	}
}
