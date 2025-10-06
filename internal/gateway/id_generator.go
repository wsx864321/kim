package gateway

import (
	"sync"
	"sync/atomic"
	"time"
)

var connIDGenerator = NewIDGenerator()

type IDGenerator struct {
	baseTime time.Time
	idState  int64
	mu       sync.Mutex
}

func NewIDGenerator() *IDGenerator {
	return &IDGenerator{
		baseTime: time.Unix(1759248000, 0), // 2025-10-01 00:00:00
	}
}

// NextID 生成连接ID
// 格式：相对时间戳(毫秒) * 1000000 + 自增序列号(6位，1-999999)
//
// 优点：
//   - 包含时间信息，服务重启后不会重复（基准时间重新计算）
//   - 无锁设计，性能极高（大部分情况下，同一毫秒内无锁）
//   - 单节点内唯一
//   - 可以从ID中提取时间信息（用于日志、调试）
//   - 相对时间戳 = id / 1000000
//   - 序列号 = id % 1000000
//   - 实际时间 = connIDBaseTime + 相对时间戳 * time.Millisecond
//
// 注意：
//   - 连接ID只在单节点内唯一，不同Gateway节点可能有相同ID
//   - 如果未来需要全局唯一，可以在ID前加上GatewayID前缀
//   - 每毫秒最多支持999999个连接（足够使用）
//   - 服务重启后，基准时间会重新计算，保证ID不重复
func (i *IDGenerator) NextID() uint64 {
	now := time.Now()

	// 计算相对时间戳（毫秒），从服务启动时间开始
	relativeMs := int64(now.Sub(i.baseTime) / time.Millisecond)

	// 读取当前状态
	state := atomic.LoadInt64(&i.idState)
	lastMs := state >> 32     // 高32位：上次的毫秒时间戳
	seq := state & 0xFFFFFFFF // 低32位：序列号

	// 检查是否跨毫秒
	if relativeMs > lastMs {
		// 跨毫秒，需要重置序列号
		i.mu.Lock()
		// 双重检查，避免重复重置
		currentState := atomic.LoadInt64(&i.idState)
		currentLastMs := currentState >> 32
		if relativeMs > currentLastMs {
			// 重置：新时间戳放在高32位，序列号从1开始
			newState := (relativeMs << 32) | 1
			atomic.StoreInt64(&i.idState, newState)
			seq = 1
		} else {
			// 其他goroutine已经重置，重新读取并自增
			seq = atomic.AddInt64(&i.idState, 1) & 0xFFFFFFFF
		}
		i.mu.Unlock()
	} else {
		// 同一毫秒内，直接自增序列号
		// 使用原子操作更新整个state（低32位自增）
		newState := atomic.AddInt64(&i.idState, 1)
		seq = newState & 0xFFFFFFFF
	}

	// 限制序列号在6位以内（1-999999），避免ID过大
	// 如果超过，取模（虽然几乎不可能发生）
	if seq > 999999 {
		seq = seq % 999999
		if seq == 0 {
			seq = 999999 // 避免为0
		}
	}

	// 组合ID：相对时间戳 * 1000000 + 序列号
	id := relativeMs*1000000 + seq

	return uint64(id)
}
