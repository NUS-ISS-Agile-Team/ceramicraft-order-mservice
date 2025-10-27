package utils

import (
    "context"
    "time"

    "github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/log"
)

type MyTimer interface {
	Start(ctx context.Context, task func())
	Stop()
}

// MyTimerImpl demonstrates how to use time.Ticker for periodic tasks
type MyTimerImpl struct {
    interval time.Duration
    stopChan chan struct{}
}

// NewMyTimer creates a new ticker demo instance
func NewMyTimer(interval time.Duration) *MyTimerImpl {
    return &MyTimerImpl{
        interval: interval,
        stopChan: make(chan struct{}),
    }
}

// Start begins the periodic task execution
func (t *MyTimerImpl) Start(ctx context.Context, task func()) {
    ticker := time.NewTicker(t.interval)
    defer ticker.Stop()

    log.Logger.Infof("Ticker started with interval: %v", t.interval)

    for {
        select {
        case <-ticker.C:
            // Execute task on each tick
			log.Logger.Infof("Task run at %v", time.Now())
            task()
        case <-t.stopChan:
            log.Logger.Info("Ticker stopped")
            return
        case <-ctx.Done():
            log.Logger.Info("Ticker stopped due to context cancellation")
            return
        }
    }
}

// Stop stops the ticker
func (t *MyTimerImpl) Stop() {
    close(t.stopChan)
}