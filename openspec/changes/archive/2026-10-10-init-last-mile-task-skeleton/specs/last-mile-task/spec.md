# Спецификация: каркас по реальной архитектуре

> Capability: `last-mile-task`. Поведение соответствует ARCHITECTURE.md
> (источник истины), реализация — вертикальный срез `SetPackagesStatus`
> в структуре §4. Неохваченные потоки — заглушки с `TODO(LASTMILE-NNN)`.

## Purpose

Определяет поведение и контракты каркаса сервиса last-mile-task: Driver-API,
вертикальный срез SetPackagesStatus с outbox-доставкой и WMS-экспортом,
CQRS-чтение активной перевозки и структуру по архитектуре §4.

## ADDED Requirements

### Requirement: Драйверский API (Driver, proto.v1.public.driver)

Система должна предоставлять gRPC-сервис `Driver` (пакет
`proto.v1.public.driver`) с RPC `SetPackagesStatus` и
`GetActiveShipmentByDriverID` по контрактам реального репозитория; прочие
RPC реального сервиса объявляются в `service.proto`, но возвращают
`Unimplemented` (TODO(LASTMILE-008)).

#### Scenario: Контракт 1:1 с реальным репо

- `SetPackagesStatusRequest`: `shipment_id` (uuid), `packages[]` с полями
  `id` (uuid), `point_id` (uuid), `action` (`UNSPECIFIED`/`LOAD`/`UNLOAD`,
  как в реальном прото; доменное `RESERVE` сюда не входит), `status`,
  `timestamp`, `tare_sticker`.
- Ответ: `success`, `data.failed_packages[]{id, error_description}`,
  `message`.

#### Scenario: Невалидный запрос отклоняется

- Невалидный `shipment_id`/`id`/`point_id` или пустой `packages` →
  `InvalidArgument`; состояние не изменяется.

### Requirement: Перевод пакетов по действию (SetPackagesStatus)

Система должна обновлять статус пакетов и сканы точек в одной транзакции,
публиковать доменные события и писать outbox-сообщения cargo и wms;
недопустимые переходы откатывают транзакцию без изменений.

#### Scenario: Разрешённый переход (LOAD на точке)

- `package.status` и сканы `package_route_point` обновляются в одной
  транзакции; после успеха outbox получает `cargo.packages.scanned` /
  `cargo.packages.status.updated` и `wms.tares.load`; ответ `success=true`
  с пустым `failed_packages`.

#### Scenario: Недопустимый переход — FailedPrecondition

- Недопустимый переход статуса пакета/ошибка транзакционного слоя →
  `FailedPrecondition`; ни пакеты, ни сканы, ни outbox не изменяются; ошибка
  несёт стабильный код.
- `failed_packages` заполняется при **частичном успехе** (как в реальном репо):
  пакет, который не удалось отсканировать, попадает в
  `failed_packages[]{id, error_description}` с `success=true`, остальные
  применяются.

### Requirement: Outbox (at-least-once)

Система должна записывать события в `lmt.outbox` в той же транзакции, что
и изменения агрегата, и доставлять их с повторами до успеха. Схема таблицы —
реальная: `id UUID PK`, `type`, `payload BYTEA`, `metadata JSONB`, `attempt`,
`scheduled_at`, `status ('TODO')`, `err`; повторы — по `id`/`status`/`attempt`.

#### Scenario: Запись только вместе с агрегатом

- Событие появляется в `lmt.outbox` только если транзакция агрегата
  зафиксирована.

#### Scenario: Повторная доставка при ошибке

- Ошибка доставки в WMS оставляет событие в `status='TODO'` с учётом
  `attempt`; следующий цикл доставляет снова; идемпотентность по `id`
  (дублей нет).

### Requirement: WMS-экспорт (HTTP)

Система должна доставлять события в WMS HTTP-клиентом (`pkg/wms`):
`POST /load` для LOAD, `POST /unload` для UNLOAD; при `--mock` — через
встроенный мок (`pkg/wms/wmsmock`).

#### Scenario: Доставка POST /load, /unload

- Успешный ответ WMS → событие помечается доставленным; поля события
  корректны.

#### Scenario: Mock-режим автономен

- При `--mock` сервис работает без внешних систем (встроенный WMS-мок).

### Requirement: Чтение активной перевозки (CQRS)

Система должна отдавать активную перевозку водителя через отдельные
read-репозитории (`shipping/queries/driver/shipment`) поверх тех же
таблиц (логический CQRS без event-sourcing).

#### Scenario: GetActiveShipmentByDriverID

- Возвращает актуальное состояние перевозки после среза; при отсутствии
  активной перевозки — корректное пустое состояние без ошибки.

### Requirement: Обработка ошибок

Система должна маппить доменные ошибки в gRPC-коды через `internal/errors`.

#### Scenario: Маппинг кодов

- Валидация → `InvalidArgument`; отсутствие пакета/перевозки → `NotFound`;
  недопустимый переход → `FailedPrecondition`; прочее → `Internal`.

### Requirement: Структура и запуск по ARCHITECTURE.md

Система должна повторять дерево каталогов §4 (нереализованные модули —
заглушки), стартовать через `main.go` → `internal/app` → `builder.New` →
`application.New`, конфигурироваться через envconfig, gracefully
завершаться и логировать через zerolog; миграции — goose с таблицами
`lmt.*`.

#### Scenario: Дерево §4 и bootstrap

- Все каталоги §4 существуют; заглушки не ломают `go build ./...`;
  запуск/остановка сервиса корректны.

#### Scenario: Миграции

- `migrations/` (goose, версионные миграции): `lmt.shipment`,
  `lmt.route`, `lmt.route_point`, `lmt.package`, `lmt.package_route_point`,
  `lmt.reserve`, `lmt.outbox`; `task migrate` применяет их (goose up).