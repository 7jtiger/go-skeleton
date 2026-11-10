package controller

import (
	"context"
	"runtime"
	"sync/atomic"
	"time"

	log "ms-gateway/common/logger"
)

// Logs local resource pressure without exposing an unauthenticated diagnostics API.
func (p *Controller) LogResourceUsage(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			var memory runtime.MemStats
			runtime.ReadMemStats(&memory)
			sig, dm, work, broadcast, waiting := int64(0), int64(0), 0, 0, 0
			if p.Signaling != nil {
				sig = atomic.LoadInt64(&p.Signaling.totalConn)
				work = len(p.Signaling.wrkQueue)
				broadcast = len(p.Signaling.brcQueue)
				waiting = len(p.Signaling.waitingEvents)
			}
			if p.ChatCtl != nil {
				dm = atomic.LoadInt64(&p.ChatCtl.totalConn)
			}
			log.Debug("resource_usage", " heap_bytes=", memory.HeapAlloc, " heap_sys_bytes=", memory.HeapSys, " goroutines=", runtime.NumGoroutine(), " gc_count=", memory.NumGC, " signaling_connections=", sig, " dm_connections=", dm, " work_queue=", work, " broadcast_queue=", broadcast, " waiting_queue=", waiting)
			if p.StoryCtl != nil && p.StoryCtl.sdb != nil {
				stats := p.StoryCtl.sdb.PoolStats()
				log.Debug("story_db_pool", " in_use=", stats.InUse, " idle=", stats.Idle, " wait_count=", stats.WaitCount, " wait_duration=", stats.WaitDuration)
			}
		}
	}
}
