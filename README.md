# Container Registry

___

Этот проект представляет собой реализацию контейнерного реестра, построенного на основе официальной спецификации Docker Registry, которая описана в документации [distribution](https://distribution.github.io/distribution/). Основная цель данного проекта — предоставить функциональность для хранения и управления контейнерными образами с поддержкой стандартных операций, таких как загрузка, выгрузка, и поиск образов, аналогично Docker Hub, но с возможностью локальной настройки и развертывания.

## Особенности

- **Совместимость с Docker Registry API**: Полная поддержка [Docker Registry HTTP API v2](https://distribution.github.io/distribution/spec/api/), что позволяет использовать реестр с различными инструментами, поддерживающими стандарт Docker.
- **Безопасность**: Реализованы основные механизмы аутентификации и авторизации для безопасного доступа к реестру.
- **Масштабируемость**: Реализована возможность создавать несколько логических реестров для каждого проекта\стека.

## Стек технологий

- Реализация основывается на Docker Distribution, что гарантирует совместимость с существующими инструментами экосистемы Docker.
- Использование Go для реализации серверной логики.
- Использован Solid.js для реализации клиентского UI.

## Важные замечания

На текущем этапе проект находится в альфа-версии (v0.1-alpha), и могут быть внесены изменения в API или архитектуру.
Пожалуйста, сообщайте об ошибках и предложениях по улучшению через Issues.
