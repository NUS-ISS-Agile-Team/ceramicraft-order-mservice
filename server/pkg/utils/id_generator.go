package utils

import (
	"fmt"
	"sync"
	"time"
)

var (
	orderIdMutex sync.Mutex
	orderIdCount = make(map[string]int)
)

// GenerateOrderID 生成唯一订单号，格式 No-20251004-163102-001
func GenerateOrderID() string {
	now := time.Now()
	prefix := "No-"
	timeStr := now.Format("20060102-150405") // 年月日-时分秒
	key := now.Format("20060102150405")      // 用于计数的秒级key

	orderIdMutex.Lock()
	defer orderIdMutex.Unlock()
	orderIdCount[key]++
	count := orderIdCount[key]
	// 只保留最近的key，防止map无限增长
	if len(orderIdCount) > 10000 {
		for k := range orderIdCount {
			if k != key {
				delete(orderIdCount, k)
			}
		}
	}
	return fmt.Sprintf("%s%s-%03d", prefix, timeStr, count)
}
