# Почему

Путь водителя пройден до погрузки/разгрузки и документов (SetPackagesStatus,
create-shipment, reserve-qr, tsd, documents), но **финальный шаг** —
завершение маршрута (ARCHITECTURE.md §23 поток 8) — отсутствует. Нет
`UpdateRoutePointDriverDeparted` (завершение точки + перевод перевозки в
`COMPLETED`), нет `EndShipment`, а событие `ShipmentCompleted` не синхронизирует
cargo-shipment (trigger `FINISH`) и не порождает outbox-сообщение
`payments.last_mile_shipment_accrual` (начисление за перевозку). Без этого
перевозка «зависает» в `IN_PROGRESS`/`UNLOADING`, и начисления в payments не
уходят.

# What Changes

- **Домен/application shipping**: `ShipmentProgressService.CompleteShipmentByID`
  (перевод в `COMPLETED`, публикация `ShipmentCompleted`),
  `RoutePointService.DepartFromRoutePoint` (завершение точки по статусной
  модели: `COMPLETED`/`PACKAGES_NOT_UNLOADED`/`PACKAGES_NOT_LOADED`,
  завершение последней точки → complete перевозки).
- **Use-case** (`internal/application/public_driver/`): `EndShipment` +
  `UpdateRoutePointDriverDeparted` — маппинг, валидация (все пакеты обработаны),
  вызов сервисов.
- **Outbox payments** (`internal/platform/outbox/payments/`): тип
  `payments.last_mile_shipment_accrual`, Writer (в транзакции) + Reader
  (at-least-once) с доставкой в payments-клиент; воркер доставки
  (`worker_payments_exporter.go`).
- **Событийный мост**: хендлер `ShipmentCompletedEvent` → cargo-shipment
  (`DriveShipment` trigger `FINISH`) + publisher в `outbox/payments`.
- **Клиент payments** (`internal/clients/payments/`): узкий интерфейс-шов
  `LastMileShipmentAccrual` + адаптер-заглушка (`TODO LASTMILE-075`).
- **Wiring**: `register_end_shipment.go` (или расширение `register_driver.go`) +
  воркер; `register_outbox_retention` не трогаем.
- **Ошибки** в `internal/errors`: `FailedPrecondition` (не все пакеты
  разгружены/точка не завершена), `InvalidArgument` (невалидный вход).

# Capabilities

### New Capabilities

- (нет: относится к существующему `last-mile-task`)

### Modified Capabilities

- `last-mile-task`: добавляются требования завершения перевозки —
  `UpdateRoutePointDriverDeparted`, `EndShipment` (перевод в `COMPLETED`),
  синхронизация с cargo-shipment (FINISH), outbox `payments.last_mile_shipment_accrual`
  (at-least-once) и обработка ошибок.

# Impact

- Новые файлы: `internal/platform/outbox/payments/{payments,writer,reader}.go`,
  `internal/application/public_driver/{end_shipment,update_route_point_driver_departed}.go`,
  `internal/platform/application/worker_payments_exporter.go`,
  `internal/clients/payments/*` (частично), хендлер `ShipmentCompleted` →
  payments/cargo, e2e-тест.
- Изменяются: `internal/shipping/application/{progress,routepoint}.go`,
  `internal/shipping/domain/policies/shipment.go` (проверка завершения),
  `register_driver.go`, `internal/shipping/infrastructure/events/handlers/shipment_event_handler.go`.
- Новых миграций нет. Внешний Driver API: добавляются реализации
  `EndShipment`, `UpdateRoutePointDriverDeparted` (контракты не меняются).
- Заглушки payments/office/cargo-shipment сохраняются (внешние сервисы не
  портируем — TODO).