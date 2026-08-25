# Технический долг и тема для изучения

Основной рефакторинг blob upload завершён. В этом документе остаётся отдельная
учебная тема по конкурентному доступу и связанный с ней технический долг.

## TD-01. Синхронизация операций над upload

### Причина переноса

Методы `AppendBlobUpload`, `CompleteBlobUpload`, `AbortBlobUpload` и
`CleanupUploads` могут одновременно обратиться к одному файлу
`TMP_PATH/<uploadID>`.

Без синхронизации возможны следующие ситуации:

- два PATCH одновременно проверят одинаковый offset и начнут запись;
- PUT начнёт вычислять digest во время выполнения PATCH;
- abort удалит файл во время записи;
- cleanup удалит старый файл, который клиент одновременно продолжает загружать;
- два PUT одновременно попытаются опубликовать один blob.

Добавление блокировок отложено, потому что перед реализацией необходимо понять
модель конкурентного доступа и границы действия `sync.Mutex`.

### Что необходимо изучить

1. Чем goroutine отличается от системного потока.
2. Что такое data race и логическая гонка операций.
3. Как работают `sync.Mutex.Lock` и `sync.Mutex.Unlock`.
4. Почему `Mutex` нельзя копировать после первого использования.
5. Почему Go mutex не является reentrant: одна goroutine не может повторно
   захватить тот же mutex без deadlock.
6. Как выбирать границы критической секции.
7. Почему `defer lock.Unlock()` обычно размещается сразу после `Lock()`.
8. Как проверять код командой `go test -race ./...`.
9. Чем блокировка map менеджера отличается от блокировки конкретного upload.
10. Почему `sync.Mutex` защищает только goroutine одного процесса и не
    синхронизирует несколько экземпляров registry.

После изучения нужно отдельно принять архитектурное решение:

```text
LocalStorage используется одним процессом
  -> допустим uploadLockManager на sync.Mutex

несколько процессов работают с общим TMP_PATH
  -> sync.Mutex недостаточен; нужна межпроцессная блокировка
     или запрет такой конфигурации
```

### Базовая реализация для изучения

Первый вариант manager хранит отдельный mutex для каждого upload ID:

```go
type uploadLockManager struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

func newUploadLockManager() *uploadLockManager {
	return &uploadLockManager{
		locks: make(map[string]*sync.Mutex),
	}
}

func (m *uploadLockManager) Get(uploadID string) *sync.Mutex {
	m.mu.Lock()
	defer m.mu.Unlock()

	lock := m.locks[uploadID]
	if lock == nil {
		lock = &sync.Mutex{}
		m.locks[uploadID] = lock
	}

	return lock
}
```

Здесь `m.mu` защищает только map `locks`. Mutex, возвращённый методом `Get`,
защищает операции конкретного upload. Поэтому разные upload ID могут
обрабатываться параллельно, а операции одного upload ID выполняются
последовательно.

Ограничение этой версии: записи никогда не удаляются из `locks`. При большом
числе загрузок map будет постоянно расти. Перед production-внедрением нужно
либо реализовать безопасное удаление записей с подсчётом пользователей, либо
обосновать допустимость ограниченного жизненного цикла manager.

### Техническая реализация после изучения

Добавить manager в `LocalStorage`:

```go
type LocalStorage struct {
	uploadLocks *uploadLockManager
}
```

Инициализировать его в конструкторе:

```go
func newLocalStorage() *LocalStorage {
	return &LocalStorage{
		uploadLocks: newUploadLockManager(),
	}
}
```

Для одного upload ID блокировка должна охватывать всю логическую операцию:

```text
Append:
  Lock -> проверить offset -> записать body -> при ошибке сделать truncate -> Unlock

Complete:
  Lock -> дописать final body -> вычислить digest -> rename -> Unlock

Abort:
  Lock -> удалить staging-файл -> Unlock

Cleanup:
  Lock -> повторно проверить время изменения -> удалить старый файл -> Unlock
```

Для `CompleteBlobUpload` нельзя захватывать один mutex дважды. Если public
`CompleteBlobUpload` уже держит блокировку, он должен вызвать внутренний метод
записи, который не пытается повторно выполнить `Lock`:

```go
func (s *LocalStorage) AppendBlobUpload(...) (...) {
	lock := s.uploadLocks.Get(uploadID)
	lock.Lock()
	defer lock.Unlock()

	return s.appendBlobUploadLocked(...)
}
```

```go
func (s *LocalStorage) CompleteBlobUpload(...) (...) {
	lock := s.uploadLocks.Get(uploadID)
	lock.Lock()
	defer lock.Unlock()

	// Блокировка уже удерживается.
	newOffset, err := s.appendBlobUploadLocked(...)
	// Проверка digest и rename выполняются до Unlock.
}
```

Название `appendBlobUploadLocked` фиксирует обязательное условие: вызывающий
код уже должен удерживать mutex соответствующего upload ID.

Одной upload-блокировки недостаточно для двух разных uploads с одинаковым
итоговым digest. Защиту участка `stat final blob -> rename` необходимо
рассмотреть отдельно: блокировка по digest либо корректная обработка результата
атомарной файловой операции.

### Шаги реализации техдолга

1. Пройти темы из раздела «Что необходимо изучить».
2. Зафиксировать поддерживаемую модель запуска LocalStorage: один процесс или
   несколько процессов с общим каталогом.
3. Написать небольшой отдельный тест с несколькими goroutine и запустить его с
   `-race`.
4. Добавить `uploadLockManager` и его unit-тесты.
5. Решить проблему удаления неиспользуемых записей из map.
6. Разделить append на public locking-метод и внутренний locked-метод.
7. Добавить блокировку в complete, abort и cleanup.
8. Отдельно решить синхронизацию публикации по digest.
9. Добавить конкурентные тесты LocalStorage.
10. Запустить полный набор тестов с race detector.

### Обязательные тесты

- Одновременные PATCH одного upload не перемешивают данные.
- PATCH и PUT одного upload не выполняют запись и digest-проверку одновременно.
- Abort не удаляет файл посреди PATCH.
- Cleanup повторно проверяет возраст файла после получения блокировки.
- Операции разных upload ID могут выполняться параллельно.
- Manager не создаёт разные mutex для одного upload ID.
- Завершённые uploads не приводят к неограниченному росту map.
- `go test -race ./...` не обнаруживает data race.

### Критерии закрытия техдолга

- документирована поддерживаемая модель запуска LocalStorage;
- понятны границы каждой критической секции;
- отсутствует повторный захват одного mutex;
- определён жизненный цикл записей в manager;
- конкурентные операции одного upload сериализованы;
- параллельность разных uploads не потеряна;
- тесты проходят с race detector.
