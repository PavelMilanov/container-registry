package storage

import (
	"context"
	"sync"
)

// maintenanceGate разделяет GC и мутации постоянного хранилища внутри процесса.
type maintenanceGate struct {
	mu      sync.Mutex
	readers int
	writer  bool
	changed chan struct{}
}

/*
lock допускает параллельные мутации либо один эксклюзивный GC; ожидание отменяемо.
*/
func (g *maintenanceGate) lock(ctx context.Context, exclusive bool) (func(), error) {
	for {
		g.mu.Lock()
		if g.changed == nil {
			g.changed = make(chan struct{})
		}
		if err := ctx.Err(); err != nil {
			g.mu.Unlock()
			return nil, err
		}
		if !g.writer && (!exclusive || g.readers == 0) {
			if exclusive {
				g.writer = true
			} else {
				g.readers++
			}
			g.mu.Unlock()
			return func() {
				g.mu.Lock()
				defer g.mu.Unlock()
				if exclusive {
					g.writer = false
				} else {
					g.readers--
				}
				close(g.changed)
				g.changed = make(chan struct{})
			}, nil
		}
		changed := g.changed
		g.mu.Unlock()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-changed:
		}
	}
}
