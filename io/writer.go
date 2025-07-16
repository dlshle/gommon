package io

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/dlshle/gommon/logging"
	"github.com/dlshle/gommon/retry"
)

type BufferedWriter[T any] interface {
	Write(p []T)
	Wait(timeout time.Duration) error
}

type bufferedWriter[T any] struct {
	ctx            context.Context
	writeFunc      func([]T) error
	itemChan       chan []T
	flushThreshold int
	retryCfg       *retry.RetryOptions
	flushDuration  time.Duration
	wg             *sync.WaitGroup
}

func (b *bufferedWriter[T]) Write(p []T) {
	b.itemChan <- p
}

func (b *bufferedWriter[T]) Wait(timeout time.Duration) error {
	waitCompleteChan := make(chan struct{})
	go func() {
		b.wg.Wait()
		waitCompleteChan <- struct{}{}
	}()
	select {
	case <-time.After(timeout):
		close(waitCompleteChan)
		return errors.New("timeout")
	case <-waitCompleteChan:
		return nil
	}
}

func (b *bufferedWriter[T]) init() {
	b.wg.Add(1)
	buffer := make([]T, 0)

	go func() {
		var tickerChan <-chan time.Time
		if b.flushDuration > 0 {
			ticker := time.NewTicker(b.flushDuration)
			defer ticker.Stop()
			tickerChan = ticker.C
		} else {
			tickerChan = make(<-chan time.Time)
		}

		flushFunc := func() {
			if len(buffer) == 0 {
				return
			}
			err := retry.RetryWithBackoff(func() error {
				return b.flush(buffer)
			}, retry.WithRetryOptions(b.retryCfg))
			if err != nil {
				logging.GlobalLogger.Errorf(b.ctx, "flush error: %v", err)
			}
		}
		for {
			select {
			case <-b.ctx.Done():
				if len(b.itemChan) > 0 {
					for items := range b.itemChan {
						buffer = append(buffer, items...)
					}
				}
				close(b.itemChan)
				flushFunc()
				buffer = nil
				b.wg.Done()
				return
			case items := <-b.itemChan:
				buffer = append(buffer, items...)
				if len(buffer) >= b.flushThreshold {
					flushFunc()
					buffer = nil
					buffer = make([]T, 0)
				}
			case <-tickerChan:
				if len(buffer) > 0 {
					flushFunc()
					buffer = nil
					buffer = make([]T, 0)
				}
			}
		}
	}()
}

func (b *bufferedWriter[T]) flush(buffer []T) error {
	return retry.Retry(func() error {
		return b.writeFunc(buffer)
	}, retry.WithRetryOptions(b.retryCfg))
}

func NewBufferedWriter[T any](ctx context.Context, writeFunc func([]T) error, retryCfg *retry.RetryOptions, flushThreshold int, flushDuration time.Duration) BufferedWriter[T] {
	bw := &bufferedWriter[T]{
		ctx:            ctx,
		writeFunc:      writeFunc,
		itemChan:       make(chan []T, flushThreshold),
		flushThreshold: flushThreshold,
		flushDuration:  flushDuration,
		retryCfg:       retryCfg,
		wg:             &sync.WaitGroup{},
	}
	bw.init()
	return bw
}
