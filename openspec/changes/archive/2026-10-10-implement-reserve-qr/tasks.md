# Задачи: резерв ворот и QR (потоки 4 и 7)

TDD-порядок: домен → application → use-case/transport → воркер → e2e.
Каждая группа: RED → GREEN → REFACTOR, тесты остаются в регрессии.

## 1. Домен shipping (Shipment/Reserve переходы)

- [x] 1.1 `entities/shipment.go`: переходы через `RESERVE_LOADING`/
  `RESERVE_UNLOADING` (методы `ReserveLoading`/`ReserveUnloading`,
  возврат из резерва при отмене) — table-driven тесты, без моков
- [x] 1.2 `entities/reserve.go`: проверить/дополнить статусы
  (`QUEUED→WAITING_APPROVAL→ACTIVE→RELEASED/CANCELED`, `EXPIRED_*`) и методы
  `Approve/Release/Cancel/Expire*` — table-driven тесты
- [x] 1.3 `domain/policies/shipment_policy.go`: реализовать `CanReserveGate`
  (только source/последняя точка назначенной перевозки, без активного резерва) +
  порт репозитория; тесты на мок-репозитории

## 2. Application shipping (ReserveService)

- [x] 2.1 `shipping/application/reserve.go`: `QueueReserveByShipmentID`
  (ReserveManager.ReserveOrQueue → UPSERT lmt.reserve → перевод Shipment),
  `GetReserveByRoutePointID`, `CancelReserveByShipmentID` — мок-тесты (транзакция)
- [x] 2.2 `ApproveReserveByShipmentID`/`ApproveReserveByRoutePointID` (вызов
  `ApproveReserveTask`, `Reserve.Approve`, перевод Shipment в `LOADING`/
  `UNLOADING`) — мок-тесты
- [x] 2.3 `ReleaseReserveByShipmentID`, `ExpireDueWaitingApprovalReserves`,
  `ExpireDueActiveReserves` (по `expires_at`) — мок-тесты идемпотентности
- [x] 2.4 `ports/`: `ReserveRepository` (whole aggregate),
  реализация `ReserveManager`-стаба в `infrastructure` + `TODO` — мок-тесты

## 3. Use-case и транспорт (public_driver)

- [x] 3.1 `create_reserve.go`: драйверский `CreateReserve` →
  `QueueReserveByShipmentID` (политика, рукопожатие QR через `pkg/qr`) — мок-тесты
- [x] 3.2 `get_reserve_qr_status.go`/`update_reserve_qr_status.go`:
  `is_scanned` по `scanned_at`, обновление статуса → `ApproveReserveByRoutePointID`
  — мок-тесты (пустой резерв → пустое состояние без ошибки)
- [x] 3.3 `cancel_reserve.go`: `CancelReserveByShipmentID` + возврат `Shipment` —
  мок-тесты
- [x] 3.4 `transport/grpc/`: маппинг proto↔domain для
  `CreateReserve`/`GetReserveQRStatus`/`UpdateReserveQRStatus`/`CancelReserve`,
  валидация входа — RED→GREEN

## 4. QR-рукопожатие и воркер

- [x] 4.1 `pkg/qr` (`CreateSession`/`DecodeQR`) + подключение в
  `handshake.go` (`TODO(LASTMILE-098)`) — тесты сессии/декодирования
- [x] 4.2 `internal/platform/application/worker_reserve_expirer.go`: цикл
  протухания (Start/Stop/Errors, как `worker_wms_exporter.go`) — smoke-тест
- [x] 4.3 Wiring в `register_driver.go` (+ воркер протухания, подписки) —
  smoke-тест запуска

## 5. E2E и проверки

- [x] 5.1 E2E (реальная PostgreSQL + gRPC): `CreateReserve` → `lmt.reserve` +
  `Shipment` в `RESERVE_LOADING`/`RESERVE_UNLOADING`, QR-сессия; конфликт
  (активный резерв) → `FailedPrecondition` без изменений
- [x] 5.2 E2E `GetReserveQRStatus`/`CancelReserve`: `is_scanned`, отмена
  неактивного резерва возвращает `Shipment` в предыдущий статус
- [x] 5.3 E2E протухание: `expires_at` в прошлом → `EXPIRED_*`; повторный прогон
  воркера не меняет уже протухшие резервы
- [x] 5.4 Полные проверки: `go build ./...`, `go vet ./...`,
  `golangci-lint run ./...` (только новые ошибки), `go test -race ./...` — зелёные