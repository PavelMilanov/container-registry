# Container Registry

Приватный контейнерный реестр на Go, реализующий используемую Docker-клиентом
часть Docker Registry HTTP API V2.

## Текущее состояние

Реализованы:

- загрузка Blob частями через `POST`, `PATCH`, `PUT` и отмена через `DELETE`;
- проверка offset и SHA-256 digest при завершении загрузки;
- получение Blob через `GET` и `HEAD`;
- сохранение и получение OCI manifest и OCI image index;
- Bearer-аутентификация Docker-клиента;
- JWT с алгоритмом HS256, issuer, audience и ограниченным временем действия;
- хранение паролей пользователей в формате Argon2id;
- пространства, репозитории и теги;
- сборка мусора Blob и удаление старых тегов;
- удаление незавершённых загрузок по расписанию;
- административный CLI `cr`;
- локальное файловое хранилище.

### Ограничение S3

В проекте присутствует реализация чтения Blob, manifest и административных
операций для S3. Multipart-загрузка Blob и очистка незавершённых uploads для S3
не реализованы.

Команда `cr serve` требует обе эти возможности и завершит запуск с ошибкой при
`storage.type: s3`. Для работающего HTTP registry необходимо использовать:

```yaml
storage:
  type: local
```

## Требования

Для локальной разработки:

- Go 1.27;
- C compiler и CGO для SQLite;
- каталог `src/conf.d` с файлом `config.yaml`.

Для контейнерного запуска достаточно Docker и файла конфигурации.

## Конфигурация

Сервер читает `config.yaml`:

- при локальном запуске — из `src/conf.d` относительно рабочего каталога
  `src`;
- в Docker-образе — из `/etc/conf.d`.

Рабочий пример для LocalStorage:

```yaml
server:
  realm: http://192.168.1.38:5050
  jwt: replace-with-random-secret-at-least-32-bytes
  token_ttl: 2h

storage:
  type: local

default_user:
  login: admin
  password: replace-with-strong-password
```

### Параметры server

| Параметр | Назначение |
|---|---|
| `realm` | Внешний URL сервера, используемый в Bearer challenge |
| `jwt` | Ключ подписи HS256; минимум 32 байта |
| `token_ttl` | Срок действия JWT; по умолчанию `2h` |

`realm` должен быть доступен Docker-клиенту и соответствовать адресу, по
которому клиент обращается к registry.

JWT issuer и Docker Registry service не настраиваются пользователем. Проект
использует внутреннее значение `container-registry` одновременно как `iss` и
`aud`, а также передаёт его в параметре `service` Bearer challenge.

### Bootstrap-пользователь

При запуске сервер ищет пользователя из секции `default_user`. Если пользователь
не найден, он создаётся, а пароль сохраняется в SQLite как Argon2id-хеш.

Изменение `default_user.password` после создания записи не меняет пароль
существующего пользователя автоматически. Существующая база находится в
каталоге данных в файле `registry.db`.

## Локальный запуск

Все команды выполняются из каталога Go-модуля:

```bash
cd src
go build -o cr .
./cr serve
```

Сервер слушает:

```text
0.0.0.0:5050
```

Проверка:

```bash
curl http://127.0.0.1:5050/check
```

Успешный ответ имеет HTTP-статус `200 OK` и пустое тело.

## Запуск в Docker

Создать каталог конфигурации и поместить в него `config.yaml`:

```text
./conf/config.yaml
```

Создать volume:

```bash
docker volume create registry-data
```

Запустить registry:

```bash
docker run -d \
  --name registry \
  --restart unless-stopped \
  -p 5050:5050 \
  -e TZ=Europe/Moscow \
  -v "$PWD/conf:/etc/conf.d:ro" \
  -v registry-data:/app/var/registry \
  rosomilanov/container-registry:latest
```

Контейнер запускается от непривилегированного пользователя с UID `10000`.
Данные registry сохраняются в volume `registry-data`.

## Первый push и pull

Перед первым push пространство должно существовать. В следующем примере
используется пространство `dev`.

Авторизовать административный CLI и создать пространство:

```bash
./cr login --username admin --password 'your-password'
./cr cloud add dev
```

Авторизовать Docker-клиент без передачи пароля в аргументах процесса:

```bash
printf '%s' 'your-password' | \
  docker login 192.168.1.38:5050 \
    --username admin \
    --password-stdin
```

Отметить и отправить образ:

```bash
docker tag postgres:17 192.168.1.38:5050/dev/postgres:17
docker push 192.168.1.38:5050/dev/postgres:17
```

Получить образ:

```bash
docker pull 192.168.1.38:5050/dev/postgres:17
```

Для registry без TLS Docker daemon должен быть отдельно настроен на работу с
insecure registry. Эта настройка выполняется на стороне Docker, а не в
container-registry.

## Multi-platform images

Одна платформа:

```bash
docker buildx build . \
  --builder insecure-builder \
  --platform linux/amd64 \
  --tag 192.168.1.38:5050/dev/application:latest \
  --push
```

Несколько платформ:

```bash
docker buildx build . \
  --builder insecure-builder \
  --platform linux/amd64,linux/arm64 \
  --tag 192.168.1.38:5050/dev/application:latest \
  --push
```

Во втором случае BuildKit отправляет OCI image index и отдельный manifest для
каждой платформы. При обычном `docker pull` Docker выбирает подходящую
платформу автоматически.

Проверка опубликованного index:

```bash
docker buildx imagetools inspect \
  192.168.1.38:5050/dev/application:latest
```

## CLI

Базовая команда: `cr`.

| Команда | Назначение |
|---|---|
| `cr serve` | Запустить HTTP-сервер |
| `cr healthcheck` | Проверить `/check` |
| `cr login -u USER -p PASSWORD` | Получить административный JWT |
| `cr cloud list` | Показать пространства |
| `cr cloud add NAME` | Создать пространство |
| `cr cloud show NAME` | Показать репозитории и теги пространства |
| `cr cloud del NAME` | Удалить пространство |
| `cr cloud del NAME -r REPO` | Удалить репозиторий |
| `cr cloud del NAME -r REPO -t TAG` | Удалить тег |
| `cr settings show` | Показать количество сохраняемых тегов |
| `cr settings set --count N` | Установить количество сохраняемых тегов |
| `cr garbage tags` | Запустить удаление старых тегов |
| `cr garbage storage` | Запустить сборку мусора Blob |

Текущий CLI обращается к `http://0.0.0.0:5050` и хранит полученный
административный токен в `/tmp/.auth`. Использовать CLI для удалённого сервера
без изменения адреса клиента пока нельзя. Хранение токена в пользовательском
защищённом хранилище остаётся незавершённым рефакторингом.

## Фоновое обслуживание

Расписание рассчитывается в часовом поясе из переменной окружения `TZ`.

| Задача | Расписание | Поведение |
|---|---|---|
| Удаление старых тегов | воскресенье, `00:00` | Использует значение `tag_count` из SQLite |
| Garbage collection | воскресенье, `01:00` | Удаляет неиспользуемые Blob |
| Очистка uploads | ежедневно, `02:00` | Удаляет uploads старше 24 часов; timeout 5 минут |

Одновременный повторный запуск одной cron-задачи пропускается. Panic внутри
задачи перехватывается cron scheduler и записывается в лог.

## Данные LocalStorage

При локальной разработке базовый каталог — `src/var`. В Docker используется
`/app/var/registry`.

```text
registry data
├── blobs/       готовые Blob по SHA-256 digest
├── manifests/   manifest, OCI index и ссылки тегов
├── tmp/         незавершённые multipart uploads
└── registry.db  пользователи и настройки
```

`tmp` и `blobs` должны находиться на одной файловой системе: завершение upload
публикует Blob через `os.Rename`. Перенос между разными файловыми системами
завершается ошибкой.

## Проверка изменений

Из каталога `src`:

```bash
go test ./...
go vet ./...
go test -race ./...
```

Интеграционные S3-тесты запускаются отдельно и требуют настроенного S3 endpoint.
