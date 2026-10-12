# Дизайн: завершение перевозки + payments (ARCHITECTURE.md §23 поток 8)

> Мотивация — см. `proposal.md`. Опора на уже готовое: стейт-машина
> `Shipment` (`Complete()` есть, `IsUnloading`/`canTransition`), событие
> `ShipmentCompletedEvent`, `RoutePointService.ArriveToRoutePoint` (паттерн
> транзакции/статусной модели точки), outbox-механизм (cargo/wms/documents,
> `pkg/outbox`, `database.NewOutboxAdapter`).

## Контекст

- `ShipmentProgressService` имеет `AssignDriver/AssignTransport/Cancel`, но
  **нет `CompleteShipmentByID`**.
- `RoutePointService` имеет `ArriveToRoutePoint`, но **нет
  `DepartFromRoutePoint`**. Статусная модель точки:
  `WAITING_ARRIVAL → DRIVER_ARRIVED → PACKAGES_NOT_UNLOADED/DRIVER_ARRIVED →
  COMPLETED`, `PACKAGES_NOT_LOADED → COMPLETED`.
- `ShipmentCompletedEvent` (ID) существует. `shipment_event_handler.go` сейчас
  знает только `CargoShipment{CreateShipment,SetTransport}` и
  `ShipmentCreated/DriverAssign/TransportAssign` — нужно добавить `DriveShipment`
  (trigger `FINISH`).
- `internal/platform/outbox/`: cargo, cleanup, documents, wms — **нет payments**.
- `internal/clients/payments/v1` — заглушка (`TODO(LASTMILE-075)`).
- Прибытие/убытие: `register_driver.go` уже подключает `progressService`,
  `RoutePointService`, `shipmentPolicy` — расширяем без новой корневой сборки.

## Goals / Non-Goals

**Goals:**
- `RoutePointService.DepartFromRoutePoint` (завершение точки, включая
  `PACKAGES_NOT_*` ветки).
- `ShipmentProgressService.CompleteShipmentByID` (→ `COMPLETED`,
  `ShipmentCompleted`).
- Use-case `UpdateRoutePointDriverDeparted` + `EndShipment`.
- Outbox `payments.last_mile_shipment_accrual` (Writer/Reader) + payments-клиент
  seam + воркер доставки.
- `ShipmentCompleted` → cargo-shipment (FINISH) + payments outbox через sync-bus.

**Non-Goals:**
- Порт реальных payments/office/cargo-shipment — slice-стабы (TODO).
- Экспорт/печать, документы возврата (уже в documents), оптимизация маршрута.
- `register_outbox_retention` (очистка outbox) — вне среза.

## Решения

### D1. Завершение точки — расширение `RoutePointService.DepartFromRoutePoint`
Паттерн как `ArriveToRoutePoint`: `tx.WithTx`, загрузка маршрута, поиск точки,
переход по `routePointStatusModel` (для завершаемой точки) с сохранением в
`RouteRepository` и публикацией `PointCompletedEvent` при необходимости
(документы возврата уже слушают этот же event). Завершение последней точки
дополнительно вызывает `completeShipment` (через `ShipmentProgressService`).

### D2. Завершение перевозки — `CompleteShipmentByID`
`tx.WithTx` → загрузка `Shipment` → проверка политики
`CanCompleteShipmentByID` (нет незавершённых unload-действий) → `Shipment.Complete()`
→ `SaveShipment` → публикация `ShipmentCompletedEvent`. Используется и
`EndShipment`, и завершением последней точки.

### D3. payments outbox — отдельный модуль
`internal/platform/outbox/payments` зеркалит `wms`/`documents`: тип
`payments.last_mile_shipment_accrual`, Writer (`outbox.NewWriter` +
`NewPostgresOutbox`), Reader (`ReaderWithMessageType` + `MessageHandlerFunc`)
доставки в `PaymentsClient.LastMileShipmentAccrual` (at-least-once).
`internal/clients/payments` — узкий интерфейс + slice-стаб
(`TODO(LASTMILE-099)` при порте реального клиента).

### D4. Событийный мост — `ShipmentCompleted`
В `shipment_event_handler.go` добавляем `HandleShipmentCompletedEvent`:
- sync: `CargoShipment.DriveShipment` (trigger `FINISH`) — расширить интерфейс
  стаба (create/SetTransport уже есть);
- async: publisher в `outbox/payments` (в той же транзакции, что и завершение).
Подписка в `register_driver.go` на `events.ShipmentCompleted`.

### D5. Воркер и wiring
`worker_payments_exporter.go` (по образцу `worker_wms_exporter.go`): запускает
Reader, `Errors()` в лог, остановка по shutdown. Сборка — в `register_driver.go`
(или `register_end_shipment.go`) поверх `builder.Outbox()`.

### D6. Ошибки — `internal/errors`
Коды `shipment_complete_*`/`route_point_*`/`payments_*` в `errors.yaml`
(`namespace: lmt`), маппинг `FailedPrecondition`/`NotFound`/`InvalidArgument`
через `MapToGRPC` (`// TODO(LASTMILE-014): errgen`).

## Risks / Trade-offs

- **Заглушка payments/cargo-shipment** → актуальный контракт начисления может
  отличаться. → Mitigation: узкий шов + `TODO`, e2e на мок-данных.
- **`PACKAGES_NOT_*` ветки** — граничные статусы точки; риск рассинхрона с
  реальной схемой. → Mitigation: table-driven тесты `routePointStatusModel`.
- **At-least-once для payments** — дубли начислений при ретраях.
  → Mitigation: идемпотентность по id события (уже в `pkg/outbox`).
- **Завершение последней точки vs EndShipment** — два пути к `Complete`.
  → Mitigation: единый `CompleteShipmentByID`, оба флоу используют одну проверку.

## План миграции

- Новых миграций нет. Откат: снятие воркера payments и подписок; таблицы остаются.
- Внешний Driver API: добавляются реализации `EndShipment`,
  `UpdateRoutePointDriverDeparted` — обратной несовместимости нет.

## Открытые вопросы

- Состав payload `payments.last_mile_shipment_accrual` (сумма/валютa/детали) —
  уточнить по реальному payments-клиенту; на срез — зафиксировать в
  адаптере-заглушке.
- Нужен ли отдельный `update_route_point_driver_departed` RPC или он уже объявлен
  в `service.proto` (проверить при имплементации) — на срез — использовать
  существующий контракт.