package run

import (
	"context"
	"time"
)

func (s *Server) hookWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case job := <-s.hookCh:
			start := time.Now()
			if err := s.hook.Run(ctx, job); err != nil {
				s.logger.Errorf("hook: %v", err)
				continue
			}
			s.metrics.lastHook.Store(time.Since(start).Milliseconds())
			s.metrics.incSent()
		}
	}
}
