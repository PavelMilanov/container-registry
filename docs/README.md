# Container Registry

Проект реализует приватный контейнерный реестр, совместимый с Docker Registry API v2.  
Основные задачи: хранение образов, управление пространствами/репозиториями, очистка устаревших тегов и сборка мусора слоёв.

## Особенности

- Совместимость с Docker Registry HTTP API v2.
- Аутентификация и авторизация для операций API.
- Несколько пространств (`cloud`) для разделения окружений.
- Поддержка хранилищ: `local` и `s3`.
- CLI `cr` для администрирования сервера и данных.

## Установка

### Docker

```bash
docker volume create registry-data
docker run -d --restart unless-stopped -p 5050:5050 \
  -v ./conf:/etc/conf.d:ro \
  -v registry-data:/app/var/registry \
  --name registry rosomilanov/container-registry:latest
```

### Docker Compose

```yaml
services:
  registry:
    image: rosomilanov/container-registry:latest
    container_name: registry
    restart: unless-stopped
    ports:
      - "5050:5050"
    volumes:
      - ./conf:/etc/conf.d:ro
      - registry-data:/app/var/registry

volumes:
  registry-data:
```

## Конфигурация

Конфигурация хранится в `config.yaml` (каталог монтируется в `/etc/conf.d`).

Пример для локального хранилища:

```yaml
server:
  realm: http://192.168.1.38:5050
  service: 192.168.1.38:5050
  issuer: local-registry
  jwt: qwerty

storage:
  type: local

user:
  login: test
  password: test
```

Пример для S3:

```yaml
server:
  realm: http://192.168.1.38:5050
  service: 192.168.1.38:5050
  issuer: local-registry
  jwt: qwerty

storage:
  type: s3
  credentials:
    endpoint: https://storage.network.net
    access_key: your_access_key
    secret_key: your_secret_key
    ssl: true

user:
  login: test
  password: test
```

## CLI команды

Базовая команда: `cr`.

- `cr serve` — запуск API-сервера.
- `cr login -u <user> -p <password>` — авторизация.
- `cr healthcheck` — проверка доступности сервиса.
- `cr cloud list|add|show|del` — управление пространствами, репозиториями и тегами.
- `cr settings show` — показать текущий `tag_count`.
- `cr settings set --count <N>` — задать количество сохраняемых тегов.
- `cr garbage storage` — сборка мусора (удаление неиспользуемых blob-слоёв).
- `cr garbage tags` — удаление старых тегов по `tag_count`.

## Пример работы с образами

```bash
docker login -u test -p test 192.168.1.38:5050
docker push 192.168.1.38:5050/dev/postgres:17
docker pull 192.168.1.38:5050/dev/postgres:17
```
