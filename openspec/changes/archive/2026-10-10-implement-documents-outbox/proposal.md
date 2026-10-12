# Почему

Скелет сервиса построен и содержит один работающий вертикальный срез
(`SetPackagesStatus` → outbox cargo+wms → WMS HTTP). Следующая интеграционная
поверхность по ARCHITECTURE.md §13/§22.1, которая имеет **реальные таблицы**
(`lmt.documents`, `lmt.document_package` — миграции уже в репозитории) и **реальный
тип outbox** (`documents.publish_after_start_shipment`), но полностью заглушена —
это модуль документов. Без него сервис не регистрирует накладные в cargo-documents
после старта перевозки и после убытия с промежуточной точки (возвратный поток),
то есть выпадает из полного потока §23 (сценарии 5–8).

# What Changes

- **Домен documents** (`internal/documents/domain/`): агрегат `Document`
  (фабрика `NewDocument`), value-объекты (`Meta`, `Driver`, `OfficePoint`,
  `ContentItem`, `Good`, `Price`, `Quantity`, примитивы, `DocumentType`
  Waybill Forward=1 / Return=2), сервис `DocumentBuilderService.Build`, модели
  направлений `directions/` (`PackageAction` LOAD/UNLOAD, `DirectionKey`
  SrcPoint→DstPoint, `RoutePoint`, `Package` — группировка goods по парам точек).
- **Персистентность** (`internal/documents/infrastructure/adapters/postgres/`):
  `DocumentRepository` поверх `lmt.documents` + `lmt.document_package`
  (схема уже в миграциях, новых не требуется).
- **Outbox documents** (`internal/platform/outbox/documents/`): Writer
  (кладёт `documents.publish_after_start_shipment` в `lmt.outbox`) и Reader
  (доставляет в cargo-documents `CreateEventByType`, at-least-once).
- **Клиент cargo-documents** (`internal/clients/cargo-documents/`): узкий
  интерфейс-шов + адаптер-заглушка (внешний SDK не подтягиваем).
- **Событийный мост**: handler на `ShipmentStartedEvent` и `PointCompletedEvent`
  → builder `Document` → outbox documents; публикация по `PointCompletedEvent`
  условна (`ensureReturnDocsScheduled` — только при возвратных пакетах).
- **Wiring** (`internal/platform/application/`): `register_documents.go` +
  воркер доставки документов (по аналогии с `worker_wms_exporter.go`).

# Capabilities

### New Capabilities

- (нет: функциональность относится к существующему capability `last-mile-task`)

### Modified Capabilities

- `last-mile-task`: добавляются требования генерации и интеграции документов —
  домен `Document`, билдер, направления, персистентность и outbox-поток
  `documents.publish_after_start_shipment` (старт перевозки + убытие с
  промежуточной точки).

# Impact

- Новые файлы: `internal/documents/*` (домен, инфраструктура, queries),
  `internal/platform/outbox/documents/*`, `internal/clients/cargo-documents/*`.
- Wiring: `internal/platform/application/register_documents.go` +
  документный воркер; подписки в `CreateDriver`/боотстрапе.
- Новых миграций нет: `lmt.documents` и `lmt.document_package` уже созданы.
- Внешний Driver API не меняется (внутренний поток, без изменения контрактов).
- Существующие срезы (SetPackagesStatus, CQRS-read, WMS) не затрагиваются.