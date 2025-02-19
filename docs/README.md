# Container Registry

___

Проект представляет собой реализацию контейнерного реестра, построенного на основе официальной спецификации Docker Registry, которая описана в документации [distribution](https://github.com/distribution/distribution). Основная цель данного проекта — предоставить функциональность для хранения и управления контейнерными образами с поддержкой стандартных операций, таких как загрузка, выгрузка, и поиск образов, аналогично Docker Hub, но с возможностью локальной настройки и развертывания.

## Особенности

- **Совместимость с Docker Registry API**: Полная поддержка [Docker Registry HTTP API v2](https://distribution.github.io/distribution/spec/api/), что позволяет использовать реестр с различными инструментами, поддерживающими стандарт Docker.
- **Безопасность**: Реализованы основные механизмы аутентификации и авторизации для безопасного доступа к реестру.
- **Масштабируемость**: Реализована возможность создавать несколько логических реестров для каждого проекта\стека.
- **Гибкость**: Реализована подключение различных видов хранилищ (локальное или S3).

## Стек технологий

- Реализация основывается на Docker Distribution, что гарантирует совместимость с существующими инструментами экосистемы Docker.
- Использован Go для реализации серверной логики.
- Использован Solid.js для реализации клиентского UI.

## Установка

### Docker

```bash
docker run -d --restart unless-stopped -p 5050:5050 \
-v ./registry:/registry/var \
--name registry rosomilanov/container-registry 
```

### Docker compose

```bash
services:
  registry:
    image: rosomilanov/container-registry:latest
    container_name: registry
    restart: unless-stopped
    ports:
      - 5050:5050
    volumes:
      - ./registry:/registry/var
```

Сервис будет доступен по адресу http://localhost:5050/login

![вход](./images/start.png)

## Конфигурация

- Конфигурация приложения осуществляется в декларативном стиле с помощью файла конфигурации `config.yaml`
- Пример конфигурации для локального хранилища:
```bash config.yaml
server:
  url: "http://127.0.0.1:5050"
  jwt: "qwerty"

storage:
  type: "local"

```
- Пример конфигурации для S3 хранилища:
```bash config.yaml
server:
  url: "http://127.0.0.1:5050"
  jwt: "qwerty"

storage:
  type: "s3"
  endpoint: "http://127.0.0.10:9000"
  access_key: "your_access_key"
  secret_key: "your_secret_key"

```
- папку с файлом `config.yaml` необходимо смонтировать в контейнер по пути `/registry/var`
## Использование

- создать реестр через веб-интерфейс:

![репозиторий](./images/repo.png)

- авторизоваться в container registry:

```bash
docker login -u test -p test localhost:5050
```

- собрать образ по правилу <адрес docker registry>/<название реестра>:<тег> - localhost:5050/dev/alpine

- загрузить образ:

```bash
docker push localhost:5050/test/alpine
```

- скачать образ:

```bash
docker pull localhost:5050/test/alpine
```
