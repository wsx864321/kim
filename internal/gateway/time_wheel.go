package gateway

import (
	"context"
	"sync"
	"time"

	"github.com/wsx864321/kim/pkg/log"
)

// timeWheel 时间轮定时器，用于定期刷新Session TTL
type timeWheel struct {
	interval    time.Duration // 时间轮转动间隔
	slots       int           // 时间轮槽数
	currentSlot int           // 当前槽位
	wheel       []*slot       // 时间轮槽数组
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	stopped     bool
}

// slot 时间轮的一个槽位
type slot struct {
	conns map[uint64]*connection // 该槽位中的连接
	mu    sync.RWMutex
}

// newTimeWheel 创建时间轮
// interval: 时间轮转动间隔（即每次刷新的间隔，如60s）
// slots: 时间轮槽数（建议为interval的倍数，如60个槽，每1s转动一次）
// 开源的时间不符合要求，重新实现一个简单的时间轮，只支持添加、删除连接和定时回调
func newTimeWheel(interval time.Duration, slots int) *timeWheel {
	if slots <= 0 {
		slots = int(interval.Seconds()) // 默认槽数等于间隔秒数
	}
	if slots > 3600 {
		slots = 3600 // 最大3600个槽
	}

	wheel := make([]*slot, slots)
	for i := 0; i < slots; i++ {
		wheel[i] = &slot{
			conns: make(map[uint64]*connection),
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &timeWheel{
		interval:    interval,
		slots:       slots,
		currentSlot: 0,
		wheel:       wheel,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// add 添加连接到时间轮
// 连接会被添加到当前槽位之后的一个完整间隔位置，确保在refreshTTLInterval时间后被处理
// 时间轮工作原理：
//   - 有N个槽位，每转动一次（slotInterval）处理当前槽位，然后移动到下一个槽位
//   - 如果slots=60，interval=60s，则slotInterval=1s
//   - 连接应该被添加到(currentSlot + slots - 1) % slots，这样会在slots-1次转动后被处理
//   - 例如：当前槽位0，添加到槽位59，会在59次转动（59秒）后被处理，接近完整的60秒间隔
func (tw *timeWheel) add(conn *connection) {
	if tw == nil {
		return
	}

	tw.mu.RLock()
	if tw.stopped {
		tw.mu.RUnlock()
		return
	}
	// 添加到当前槽位之后的一个完整间隔位置
	// 这样确保连接会在接近refreshTTLInterval时间后被处理
	slotIndex := (tw.currentSlot + tw.slots - 1) % tw.slots
	tw.mu.RUnlock()

	tw.wheel[slotIndex].mu.Lock()
	tw.wheel[slotIndex].conns[conn.id] = conn
	tw.wheel[slotIndex].mu.Unlock()
}

// remove 从时间轮移除连接
func (tw *timeWheel) remove(connID uint64) {
	if tw == nil {
		return
	}

	// 遍历所有槽位，移除该连接
	for i := 0; i < tw.slots; i++ {
		tw.wheel[i].mu.Lock()
		delete(tw.wheel[i].conns, connID)
		tw.wheel[i].mu.Unlock()
	}
}

// start 启动时间轮
func (tw *timeWheel) start(callback func([]*connection)) {
	if tw == nil {
		return
	}

	tw.mu.Lock()
	if tw.stopped {
		tw.mu.Unlock()
		return
	}
	tw.mu.Unlock()

	tw.wg.Add(1)
	go tw.run(callback)
}

// run 时间轮运行循环
func (tw *timeWheel) run(callback func([]*connection)) {
	defer tw.wg.Done()

	// 计算每个槽位的时间间隔
	slotInterval := tw.interval / time.Duration(tw.slots)
	ticker := time.NewTicker(slotInterval)
	defer ticker.Stop()

	for {
		select {
		case <-tw.ctx.Done():
			return
		case <-ticker.C:
			tw.mu.Lock()
			currentSlot := tw.currentSlot
			tw.currentSlot = (tw.currentSlot + 1) % tw.slots
			tw.mu.Unlock()

			// 获取当前槽位的所有连接
			tw.wheel[currentSlot].mu.RLock()
			conns := make([]*connection, 0, len(tw.wheel[currentSlot].conns))
			for _, conn := range tw.wheel[currentSlot].conns {
				conns = append(conns, conn)
			}
			tw.wheel[currentSlot].mu.RUnlock()

			// 执行回调
			if len(conns) > 0 && callback != nil {
				callback(conns)
			}

			// 清空当前槽位，连接会被重新添加到下一个槽位（如果需要继续刷新）
			tw.wheel[currentSlot].mu.Lock()
			tw.wheel[currentSlot].conns = make(map[uint64]*connection)
			tw.wheel[currentSlot].mu.Unlock()
		}
	}
}

// stop 停止时间轮
func (tw *timeWheel) stop() {
	if tw == nil {
		return
	}

	tw.mu.Lock()
	if tw.stopped {
		tw.mu.Unlock()
		return
	}
	tw.stopped = true
	tw.mu.Unlock()

	tw.cancel()

	log.Info(context.Background(), "time wheel stopped")
}
