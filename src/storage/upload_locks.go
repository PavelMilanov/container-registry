package storage

import "sync"

type uploadLock struct {
	mu   sync.Mutex
	refs int
}

/*
uploadLockManager управляет отдельными mutex для активных загрузок Blob.

Нулевое значение готово к использованию. Записи удаляются после завершения
последнего владельца или ожидающей goroutine.
*/
type uploadLockManager struct {
	mu    sync.Mutex
	locks map[string]*uploadLock
}

/*
Lock блокирует операции над указанной загрузкой Blob.

	uploadID - идентификатор загрузки.

Возвращает функцию освобождения блокировки. Вызывающий код должен вызвать её
ровно один раз, предпочтительно через defer.
*/
func (m *uploadLockManager) Lock(uploadID string) func() {
	m.mu.Lock()
	if m.locks == nil {
		m.locks = make(map[string]*uploadLock)
	}

	lock := m.locks[uploadID]
	if lock == nil {
		lock = new(uploadLock)
		m.locks[uploadID] = lock
	}
	lock.refs++
	m.mu.Unlock()

	lock.mu.Lock()

	return func() {
		lock.mu.Unlock()

		m.mu.Lock()
		lock.refs--
		if lock.refs == 0 && m.locks[uploadID] == lock {
			delete(m.locks, uploadID)
		}
		m.mu.Unlock()
	}
}
