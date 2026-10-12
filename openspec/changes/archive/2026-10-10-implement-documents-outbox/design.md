# Дизайн: домен документов + outbox (documents.publish_after_start_shipment)

> Мотивация — см. `proposal.md`. Данное изменение развивает существующий каркас
> `last-mile-task` (ARCHITECTURE.md) по уже отработанному паттерну outbox,
> реализованному в срезе `SetPackagesStatus` (cargo/wms). Реальные таблицы
> `lmt.documents`/`lmt.document_package` уже в миграциях — новых не требуется.

## Контекст

- Существующие outbox-пары (wms, cargo) показывают устоявшийся шаблон:
  - `database.NewOutboxAdapter(repo)` → `outbox.Querier`;
  - `pkg/outbox.Writer` поверх `outbox.NewPostgresOutbox(querier, "lmt.outbox")`,
    сообщение — JSON-байты с `MessageWithID/Type`;
  - `pkg/outbox.Reader` + `ReadWithMessageType` + `MessageHandlerFunc`
    (at-least-once: complete на успехе, reschedule с backoff на ошибке);
  - доставка — воркер `worker_wms_exporter.go`, читает `outboxReader`
    (`Start/Stop/Errors`), ошибки инжектятся в лог.
- Событийный мост: синхронный in-memory bus (`inmem.NewSyncEventBus`)
  подписывает `events.PackageScanned`/`ShipmentCreated`/... на хендлеры,
  которые в той же транзакции пишут outbox-сообщения.
- `lmt.documents` (схема реальная): `id UUID PK`, `shipment_id`, `event_uuid`,
  `document_id BIGINT`, `data_print_task_uuid`, `generate_task_uuid`,
  `from_route_point_id`, `to_route_point_id`, `document_type SMALLINT DEFAULT 0`,
  `created_at`. Связь — `lmt.document_package(document_id, package_id)`.
- `internal/documents/*` — полностью заглушки (`doc.go` + `TODO(LASTMILE-072)`);
  `internal/clients/cargo-documents/v1` — заглушка `TODO(LASTMILE-075)`.

## Goals / Non-Goals

**Goals:**
- Домен `Document` (фабрика `NewDocument`), value-объекты и билдер
  (`DocumentBuilderService.Build`) с группировкой по направлениям load/unload.
- Персистентность `DocumentRepository` поверх `lmt.documents` +
  `lmt.document_package` (без новых миграций).
- Outbox `documents.publish_after_start_shipment`: Writer (в транзакции) +
  Reader (at-least-once в cargo-documents), воркер доставки.
- Триггеры `ShipmentStartedEvent` (старт) и `PointCompletedEvent` (убытие,
  условный `ensureReturnDocsScheduled`).
- Wiring в composition root без изменения внешнего Driver API.

**Non-Goals:**
- Полный порт реального SDK cargo-documents (внешний easyp-прото не копируем).
- portal-запросы документов (`portal_queries.go`), экспорт/печать документов
  (`data_print_task_uuid`), генерация в S3 — вне среза (TODO).
- Оптимизация маршрута, резервы, полноценный support/export — вне среза.

## Решения

### D1. Скоуп: только outbox `documents.publish_after_start_shipment`
Реальный репо использует один тип сообщения для двух триггеров — старт
перевозки и убытие с промежуточной точки (§22.1), покрывая и возвратный поток.
Выбираем **один** тип сообщения + условную публикацию, зеркаля реальный
`ensureReturnDocsScheduled` (только при `HasLoadedReturnPackagesAtRoutePoint`).
Альтернатива «два типа (старт/точка)» отклонена: расходится с реальным
контрактом outbox и усложняет архивацию/сверку с репо.

### D2. cargo-documents — узкий шов, без внешнего SDK
Не тянем easyp/внешний прото. Вводим интерфейс
`documents.DocumentsClient{ CreateEventByType(ctx, Event) error }` и
адаптер-заглушку (по аналогии с `infrastructure.CargoShipment` /
`outbox/cargo.CargoClient`). Это сохраняемо: при порте реального клиента
замена — только адаптер (`TODO(LASTMILE-099): switch to cargo_documents_client`).

### D3. Персистентность — адаптер `documentspg` поверх двух таблиц
`DocumentRepository` сохраняет агрегат целиком (правило конфига: whole aggregate):
вставка в `lmt.documents` + связи в `lmt.document_package` в одной транзакции.
Имя пакета — уникальное (`documentspg`), как требует конвенция имён адаптеров
(не `postgres`, чтобы не конфликтовать с другими контекстами).

### D4. Триггеры через существующий sync-bus
Не строим новый шин: хендлеры `documentsEventHandler` подписываются в том же
`CreateDriver` на `ShipmentStarted` и `PointCompleted`. Хендлер собирает данные
из read-источников (перевозка, маршрут, пакеты/точки) через узкие интерфейсы
(аналог `eventdeps.Store`) и пишет outbox через `documents_outbox.Writer` в той
же транзакции. Ошибка builder/доставки **не** откатывает доменный поток: смена
статуса агрегата и outbox-сообщение идут в одной транзакции, а неуспех внешнего
вызова обрабатывается Reader'ом (at-least-once), а не синхронно.

### D5. Воркер и wiring
По образцу `worker_wms_exporter.go`: `worker_documents_exporter.go` запускает
`documentsNewPublishAfterStartShipmentReader`, прокидывает `Errors()` в лог,
останавливается по shutdown. Регистрация — `register_documents.go`
(или расширение `CreateDriver`), чтение — через `builder.Outbox()`.

### D6. Ошибки
Доменные ошибки документов — через `internal/errors` (стабильный code +
`protocode`); Reader маппит ошибку доставки в `Internal`; транспортные хендлеры
не трогают (внешний API не меняется). `errors.yaml` — пул `namespace: lmt`
(коды `documents_*` добавляются вручную, `// TODO(LASTMILE-014): errgen`).

## Risks / Trade-offs

- **Собрано по фрагментам реального репо (без easyp/LMT-прото)** → фактический
  JSON-контракт `CreateEventByType` может отличаться от продакшена.
  → Mitigation: зафиксировать шов в отдельном адаптере и помечать
  `TODO(LASTMILE-099)`; e2e проверяет доставку в мок, не в реальный сервис.
- **Последовательность build/grouping load→unload** — модель направлений может
  не покрыть все кейсы (перенос груза между точками, возврат).
  → Mitigation: покрыть table-driven тестами основные направления; расширение —
  отдельной задачей.
- **Условная публикация (`ensureReturnDocsScheduled`)** — граничный случай
  «есть возвратные пакеты» зависит от корректности read-источника.
  → Mitigation: тест на пустой/непустой случай возвратных пакетов.
- **Отсутствие реальной cargo-documents** в e2e → проверить только at-least-once
  на уровне outbox (мок-приёмник), фактическая доставка — smoke.

## План миграции

- Без разрыва совместимости: внешний Driver/Portal API не меняется.
- Новые таблицы не требуются (схема уже есть). Обратная совместимость чтения
  `lmt.documents` — если какие-то данные уже заполнены другим процессом,
  репозиторий просто читает/пишет по той же схеме.
- Откат: удаление wiring `register_documents.go` + воркера; таблицы и outbox
  остаются, доставка прекращается без потери данных.

## Открытые вопросы

- Точный набор полей JSON-события `CreateEventByType` (имена полей/структура)
  — уточнить по реальному клиенту при порте; на срез — зафиксировать в
  адаптере-заглушке.
- Нужны ли document-запросы/read-model в этом срезе или только запись +
  outbox? На срез — только запись + outbox (portal-чтение — вне Non-Goals).