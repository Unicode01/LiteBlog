package utils

import (
	"context"
	"sync"
	"time"
)

// TimeWheel 时间轮调度器
type TimeWheel struct {
	interval   time.Duration // 每个槽的时间间隔
	slots      int           // 槽数量
	currentPos int           // 当前位置
	tasks      [][]func()    // 每个槽的任务列表
	ticker     *time.Ticker  // 定时器
	mu         sync.RWMutex  // 互斥锁
	ctx        context.Context
	cancelFunc context.CancelFunc
}

// NewTimeWheel 创建新的时间轮
func NewTimeWheel(interval time.Duration, slots int, ctx context.Context) *TimeWheel {
	ctx, cancel := context.WithCancel(ctx)

	tw := &TimeWheel{
		interval:   interval,
		slots:      slots,
		currentPos: 0,
		tasks:      make([][]func(), slots),
		ctx:        ctx,
		cancelFunc: cancel,
	}

	// 初始化每个槽
	for i := 0; i < slots; i++ {
		tw.tasks[i] = make([]func(), 0)
	}

	return tw
}

// Start 启动时间轮
func (tw *TimeWheel) Start() {
	tw.ticker = time.NewTicker(tw.interval)

	go func() {
		for {
			select {
			case <-tw.ctx.Done():
				tw.ticker.Stop()
				return
			case <-tw.ticker.C:
				tw.tick()
			}
		}
	}()
}

// Stop 停止时间轮
func (tw *TimeWheel) Stop() {
	tw.cancelFunc()
}

// tick 时间轮滴答
func (tw *TimeWheel) tick() {
	tw.mu.Lock()

	// 获取当前槽的任务
	tasks := tw.tasks[tw.currentPos]
	tw.tasks[tw.currentPos] = make([]func(), 0) // 清空当前槽

	// 移动到下一个槽
	tw.currentPos = (tw.currentPos + 1) % tw.slots

	tw.mu.Unlock()

	// 执行任务（不持锁，避免阻塞）
	for _, task := range tasks {
		go task() // 异步执行任务
	}
}

// AddTask 添加任务
// delay: 延迟时间，必须是 interval 的整数倍
func (tw *TimeWheel) AddTask(delay time.Duration, task func()) {
	if delay < tw.interval {
		delay = tw.interval
	}

	// 计算应该放在哪个槽
	slots := int(delay / tw.interval)
	if slots >= tw.slots {
		slots = tw.slots - 1
	}

	tw.mu.Lock()
	defer tw.mu.Unlock()

	pos := (tw.currentPos + slots) % tw.slots
	tw.tasks[pos] = append(tw.tasks[pos], task)
}

// AddRecurringTask 添加循环任务
func (tw *TimeWheel) AddRecurringTask(interval time.Duration, task func()) {
	recurringTask := func() {
		task()
		tw.AddRecurringTask(interval, task) // 重新添加自己
	}
	tw.AddTask(interval, recurringTask)
}

