# Задачи: завершение перевозки + payments (поток 8)

TDD-порядок: домен → application → outbox → события/wiring → e2e.
Каждая группа: RED → GREEN → REFACTOR, тесты остаются в регрессии.

## 1. Домен shipping (завершение точки/перевозки)

- [x] 1.1 `entities/route_point.go`: статусная модель завершения
  (`COMPLETED`/`PACKAGES_NOT_UNLOADED`/`PACKAGES_NOT_LOADED`) + методы
  (`Complete`, `CompleteAfterPackageCancellation`-подобные) — table-driven тесты
- [x] 1.2 `entities/shipment.go`: `Complete()` уже есть — проверить переход из
  `UNLOADING`/`IN_PROGRESS` в `COMPLETED`, тест недопустимых переходов
- [x] 1.3 `domain/policies/shipment.go`: `CanCompleteShipmentByID` (нет
  незавершённых unload-действий) + порт репозитория — тесты на мок-репозитории

## 2. Application shipping

- [x] 2.1 `routepoint.go`: `DepartFromRoutePoint` (завершение точки +
  `PACKAGES_NOT_*` ветки + публикация `PointCompletedEvent`) — мок-тесты
- [x] 2.2 `progress.go`: `CompleteShipmentByID` (загрузка → проверка политики →
  `Complete()` → Save → `ShipmentCompleted`) — мок-тесты
- [x] 2.3 Порт/событие: при необходимости добавить `PointCompletedEvent` поля и
  `RouteEventPublisher`-методы — тесты

## 3. Use-case и транспорт (public_driver)

- [x] 3.1 `update_route_point_driver_departed.go`: маппинг → `DepartFromRoutePoint`
  (завершение точки/перевозки) — мок-тесты
- [x] 3.2 `end_shipment.go`: `EndShipment` → `CompleteShipmentByID` — мок-тесты
  (успех/не готова к завершению → `FailedPrecondition`)
- [x] 3.3 `transport/grpc/`: маппинг proto↔domain + валидация для
  `UpdateRoutePointDriverDeparted`/`EndShipment` — RED→GREEN

## 4. Outbox payments и клиент

- [x] 4.1 `internal/platform/outbox/payments/`: тип
  `payments.last_mile_shipment_accrual`, Writer (`outbox.NewWriter`) — RED→GREEN
- [x] 4.2 Reader (`ReaderWithMessageType` + `MessageHandlerFunc`) с доставкой в
  `PaymentsClient.LastMileShipmentAccrual` — тест at-least-once (success/error
  reschedule/идемпотентность)
- [x] 4.3 `internal/clients/payments/`: узкий интерфейс + адаптер-заглушка
  (`// TODO(LASTMILE-099)`) — мок-тесты

## 5. Событийный мост и wiring

- [x] 5.1 `shipment_event_handler.go`: `HandleShipmentCompletedEvent` →
  `CargoShipment.DriveShipment` (FINISH) + publisher в `outbox/payments` —
  мок-тесты
- [x] 5.2 `register_driver.go`: подписки `ShipmentCompleted` (cargo-finish +
  payments), включение воркера payments — smoke-тест запуска
- [x] 5.3 `internal/platform/application/worker_payments_exporter.go`: цикл
  доставки (Start/Stop/Errors, как `worker_wms_exporter.go`) — smoke-тест

## 6. E2E и проверки

- [x] 6.1 E2E (реальная PostgreSQL + gRPC): убытие с последней точки →
  точка `COMPLETED`, `Shipment` в `COMPLETED`, `ShipmentCompleted` →
  cargo-stub FINISH + outbox `payments.last_mile_shipment_accrual`; необработанные
  пакеты → `FailedPrecondition` без изменений
- [x] 6.2 E2E доставки payments: мок-приёмник принимает сообщение; `FailNext(1)`
  → повторная доставка (at-least-once), дублей нет
- [x] 6.3 Полные проверки: `go build ./...`, `go vet ./...`,
  `golangci-lint run ./...` (только новые ошибки), `go test -race -p 1 ./...` —
  зелёные