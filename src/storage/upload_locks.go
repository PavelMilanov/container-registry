package storage

import (
	"context"
	"sync"
)

type uploadLock struct {
	mu      sync.Mutex
	refs    int
	changed chan struct{}
}

/*
uploadLockManager управляет mutex загрузок и уведомляет ожидающие goroutine.

Нулевое значение готово к использованию; неиспользуемые записи удаляются.
*/
type uploadLockManager struct {
	mu    sync.Mutex
	locks map[string]*uploadLock
}

/*
reference получает запись mutex и увеличивает число владельцев/ожидающих.
Вызывающий код обязан удерживать m.mu.
*/
func (m *uploadLockManager) reference(id string) *uploadLock {
	if m.locks == nil {
		m.locks = make(map[string]*uploadLock)
	}
	lock := m.locks[id]
	if lock == nil {
		lock = &uploadLock{changed: make(chan struct{})}
		m.locks[id] = lock
	}
	lock.refs++
	return lock
}

/*
release снимает mutex и будит ожидающих либо убирает отменённое ожидание.
*/
func (m *uploadLockManager) release(id string, lock *uploadLock, acquired bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if acquired {
		lock.mu.Unlock()
		close(lock.changed)
		lock.changed = make(chan struct{})
	}
	lock.refs--
	if lock.refs == 0 {
		delete(m.locks, id)
	}
}

/*
LockContext ожидает mutex без polling и прекращает ожидание при отмене context.
*/
func (m *uploadLockManager) LockContext(ctx context.Context, id string) (func(), error) {
	m.mu.Lock()
	lock := m.reference(id)
	m.mu.Unlock()
	for {
		m.mu.Lock()
		if err := ctx.Err(); err != nil {
			m.mu.Unlock()
			m.release(id, lock, false)
			return nil, err
		}
		if lock.mu.TryLock() {
			m.mu.Unlock()
			return func() { m.release(id, lock, true) }, nil
		}
		changed := lock.changed
		m.mu.Unlock()
		select {
		case <-ctx.Done():
			m.release(id, lock, false)
			return nil, ctx.Err()
		case <-changed:
		}
	}
}

/*
Lock ожидает mutex без timeout; используется там, где context не предоставлен.
*/
func (m *uploadLockManager) Lock(id string) func() {
	unlock, _ := m.LockContext(context.Background(), id)
	return unlock
}

/*
TryLock получает mutex без ожидания; cleanup пропускает активные сессии.
*/
func (m *uploadLockManager) TryLock(id string) (func(), bool) {
	m.mu.Lock()
	lock := m.reference(id)
	if !lock.mu.TryLock() {
		lock.refs--
		m.mu.Unlock()
		return nil, false
	}
	m.mu.Unlock()
	return func() { m.release(id, lock, true) }, true
}
