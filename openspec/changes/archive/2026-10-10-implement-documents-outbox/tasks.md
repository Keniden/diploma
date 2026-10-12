# Задачи: домен документов + outbox (documents.publish_after_start_shipment)

TDD-порядок: домен → персистентность → outbox → события/wiring → e2e.
Каждая группа: RED (падающий тест) → GREEN → REFACTOR, тесты остаются в регрессии.

## 1. Домен документов (`internal/documents/domain/`)

- [x] 1.1 `values/`: `DocumentType` (`WaybillForward=1`, `WaybillReturn=2`), `Meta`
  (Driver, RouteCarType), `Driver`, `OfficePoint`, `ContentItem`, `Good`, `Price`,
  `Quantity`, примитивы (`document.go`, `ids.go`, `primitives.go`) — table-driven
  тесты маппинга/значений; убрать `TODO(LASTMILE-072)` из `values/doc.go`
- [x] 1.2 `directions/`: `PackageAction` (LOAD/UNLOAD), `DirectionKey`
  (SrcPoint→DstPoint), `RoutePoint`, `Package` — группировка goods по парам точек;
  тесты группировки load/unload (прямой и возвратный потоки)
- [x] 1.3 `entities/document.go`: агрегат `Document` (ID, ShipmentID,
  FromRoutePointID, ToRoutePointID, DocumentType, EventUUID, ExternalID,
  DataPrintTaskID, GenerateTaskID, PackageIDs) + фабрика `NewDocument`; переходы
  бизнес-методами — RED→GREEN (без моков)
- [x] 1.4 `services/`: `DocumentBuilderService.Build(BuildInput) → WithShortGoods`
  — сборка строк товаров и метаданных по направлениям; тесты на корректные
  `ContentItem`/`Good` + корректный `DocumentType`

## 2. Персистентность (`internal/documents/infrastructure/adapters/postgres/`)

- [x] 2.1 `documentspg` (уникальное имя пакета): `DocumentRepository` поверх
  `lmt.documents` + `lmt.document_package`, сохранение агрегата целиком в одной
  транзакции (pgxpool + squirrel/scany, как `shipping`); интеграционные тесты на
  реальной PostgreSQL (Docker), RED→GREEN; убрать `TODO(LASTMILE-072)` из
  `adapters/postgres/doc.go`
- [x] 2.2 `queries/`: минимальный read-доступ (по `shipment_id`, по
  `event_uuid`) для e2e-проверки; тесты на пустых/заполненных данных

## 3. Outbox documents и клиент cargo-documents

- [x] 3.1 `internal/platform/outbox/documents/`: тип сообщения
  `documents.publish_after_start_shipment`, Writer (`outbox.NewWriter` +
  `NewPostgresOutbox(querier,"lmt.outbox")`, JSON-пейлоад) — RED→GREEN
- [x] 3.2 Читающая часть: Reader (`outbox.ReaderWithMessageType` +
  `MessageHandlerFunc`) с доставкой в `CreateEventByType` — тест at-least-once
  (success → complete; error → reschedule + attempt; идемпотентность по id)
- [x] 3.3 `internal/clients/cargo-documents/`: узкий интерфейс
  `DocumentsClient{ CreateEventByType }` + адаптер-заглушка
  (`// TODO(LASTMILE-099): switch to cargo_documents_client`) — мок-тесты

## 4. Событый мост и wiring

- [x] 4.1 Хендлер (`internal/shipping/infrastructure/events/handlers/`):
  `documentsEventHandler` на `ShipmentStartedEvent` и `PointCompletedEvent` —
  собирает данные (перевозка/маршрут/пакеты) через узкие read-интерфейсы
  (аналог `eventdeps.Store`) и пишет outbox в той же транзакции; тесты на моках
  (успех/пустые данные)
- [x] 4.2 Условная публикация `ensureReturnDocsScheduled` (только при
  `HasLoadedReturnPackagesAtRoutePoint`) — тесты на непустой/пустой возвратный
  поток, сообщение не публикуется при отсутствии возвратных пакетов
- [x] 4.3 `internal/platform/application/register_documents.go`: подписки
  `ShipmentStarted`/`PointCompleted` в `CreateDriver`, reader поверх
  `builder.Outbox()`; воркер `worker_documents_exporter.go` (Start/Stop/Errors,
  как `worker_wms_exporter.go`) — smoke-тест запуска/остановки
- [x] 4.4 `internal/errors`: коды `documents_*` в `errors.yaml` (`namespace: lmt`)
  + маппинг в gRPC-коды (`// TODO(LASTMILE-014): errgen`) — тесты маппинга

## 5. E2E и проверки

- [x] 5.1 E2E (реальная PostgreSQL + gRPC): старт перевозки/убытие точки →
  запись в `lmt.documents` + `lmt.document_package` + outbox-сообщение
  `documents.publish_after_start_shipment`; условная публикация только при
  возвратных пакетах
- [x] 5.2 E2E tobox-доставки: мок-приёмник cargo-documents принимает сообщение;
  `FailNext(1)` → повторная доставка (at-least-once), дублей нет
- [x] 5.3 Полные проверки: `go build ./...`, `go vet ./...`,
  `golangci-lint run ./...` (только новые ошибки), `go test -race ./...` —
  зелёные; `task migrate` не ломается (схема документов уже применена)