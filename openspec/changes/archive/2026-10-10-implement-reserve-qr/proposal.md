# Почему

Создание перевозки (поток 3) и SetPackagesStatus (погрузка) есть, но **резерв
ворот и QR-код** (ARCHITECTURE.md §23 потоки 4 и 7) — промежуточный шаг между
созданием и погрузкой — полностью отсутствуют. Без него водитель не может
зарезервировать ворота на пункте (source/последней точке), система не создаёт
`lmt.reserve` и не переводит `Shipment` в `RESERVE_LOADING`/`RESERVE_UNLOADING`,
а ТСД-скан не подтверждает резерв. Нужно довести резервный flow на стороне
водителя: `CreateReserve`, `GetReserveQRStatus`, `UpdateReserveQRStatus`,
`CancelReserve` + сервис `ReserveService` и интеграцию с gate-reserve через
`ReserveManager`.

# What Changes

- **Домен/application shipping**: `ReserveService` (`QueueReserveByShipmentID`,
  `ApproveReserveByShipmentID`, `ApproveReserveByRoutePointID`,
  `ReleaseReserveByShipmentID`, `GetReserveByRoutePointID`,
  `CancelReserveByShipmentID`, `ExpireDueWaitingApprovalReserves`,
  `ExpireDueActiveReserves`) + порт `ReserveManager` (уже есть в
  `ports/reserver.go`), доработка агрегата `Shipment` (переходы
  `RESERVE_LOADING`/`RESERVE_UNLOADING`).
- **Use-case** (`internal/application/public_driver/`): `CreateReserve` (резерв
  ворот: `ReserveManager.ReserveOrQueue` + рукопожатие QR, UPSERT `lmt.reserve`,
  перевод `Shipment` в `RESERVE_LOADING`/`RESERVE_UNLOADING`),
  `GetReserveQRStatus` (`is_scanned`), `UpdateReserveQRStatus`,
  `CancelReserve`.
- **Persistense**: `ReserveRepository` (агрегат целиком) поверх `lmt.reserve`
  (статус/expires_at уже в миграциях).
- **Рукопожатие/QR**: `pkg/qr` (`CreateSession`/`DecodeQR`) + подключение в
  `internal/platform/application/handshake.go` (срез-реализация),
  взаимодействие с ТСД `ScanShipment` (триггеродружественный интерфейс).
- **Воркер** протухших резервов (`ExpireDue*` повторяется по расписанию).
- **Ошибки** в `internal/errors`: `FailedPrecondition` для недопустимого
  перехода/конфликта резерва, `InvalidArgument` для невалидного входа.

# Capabilities

### New Capabilities

- (нет: относится к существующему `last-mile-task`)

### Modified Capabilities

- `last-mile-task`: добавляются требования резерва ворот и QR-кода — создание
  резерва (`CreateReserve`), перевод перевозки в `RESERVE_LOADING`/
  `RESERVE_UNLOADING`, статус/обновление QR (`GetReserveQRStatus`/
  `UpdateReserveQRStatus`), отмена (`CancelReserve`), протухание по расписанию,
  интеграция с gate-reserve и рукопожатием QR; ТСД-скан (подтверждение резерва)
  опирается на этот флоу (реализуется в отдельном изменении TSD).

# Impact

- Новые файлы: `internal/shipping/application/reserve.go`,
  `internal/shipping/application/ports/{reserve_repository.go,reserve_expirer.go}`,
  `internal/shipping/infrastructure/adapters/postgres/reserves/`,
  `internal/application/public_driver/{create_reserve,get_reserve_qr_status,
  update_reserve_qr_status,cancel_reserve}.go`, `pkg/qr/*` (частично),
  `internal/platform/application/worker_reserve_expirer.go`.
- Изменяются: `internal/shipping/domain/entities/{shipment,reserve}.go`,
  `register_driver.go` (+ подписки/воркер), `internal/platform/application/handshake.go`.
- Новых миграций нет (`lmt.reserve` + `status`/`expires_at` уже в миграциях).
- Внешний Driver API: добавляются реализации `CreateReserve`,
  `GetReserveQRStatus`, `UpdateReserveQRStatus`, `CancelReserve`.
- Заглушки `ReserveManager`/`pkg/qr`/`handshake` сохраняются (внешний
  gate-reserve и handshake-validator не портируем — TODO).