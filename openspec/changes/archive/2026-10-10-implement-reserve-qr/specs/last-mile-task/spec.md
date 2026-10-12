## ADDED Requirements

### Requirement: Создание резерва ворот (CreateReserve)

Система SHALL по `CreateReserve` создавать резерв ворот для точки маршрута:
проверять политику `CanReserveGate` (резерв только на source-точке назначенной
перевозки, без активного резерва), вызывать `ReserveManager.ReserveOrQueue`,
создавать сессию/QR рукопожатия, сохранять `lmt.reserve` и переводить
`Shipment` в `RESERVE_LOADING` (source) или `RESERVE_UNLOADING` (последняя точка)
в одной транзакции.

#### Scenario: Успешный резерв на source

- **WHEN** водитель вызывает `CreateReserve` для назначенной перевозки на
  source-точке
- **THEN** создаётся `lmt.reserve` (статус `QUEUED`/`WAITING_APPROVAL`),
  `Shipment` переходит в `RESERVE_LOADING`, сгенерирован QR-селф `session`
  (создание/декодирование рукопожатия), возвращается идентификатор резерва.

#### Scenario: Резерв на последней точке

- **WHEN** резерв выполняется на последней точке маршрута
- **THEN** `Shipment` переходит в `RESERVE_UNLOADING`, резерв сохраняется.

#### Scenario: Конфликт — активный резерв или чужая точка

- **WHEN** у перевозки/точки уже есть активный резерв, либо точка не source/не
  назначенная
- **THEN** система отвечает `FailedPrecondition` и не изменяет состояние.

### Requirement: Статус и обновление QR (Get/UpdateReserveQRStatus)

Система SHALL отдавать `is_scanned` (отсканирован ли QR ТСД) по
`GetReserveQRStatus` и обновлять статус по `UpdateReserveQRStatus`.

#### Scenario: GetReserveQRStatus

- **WHEN** водитель запрашивает статус QR резерва
- **THEN** система возвращает, отсканирован ли QR (`is_scanned`), по актуальному
  `lmt.reserve`; при отсутствии резерва — корректное пустое состояние без ошибки.

#### Scenario: UpdateReserveQRStatus

- **WHEN** ТСД или водитель подтверждает/обновляет статус QR
- **THEN** состояние `lmt.reserve` обновляется (при необходимости перевод
  `Shipment` в `LOADING`/`UNLOADING`), статус согласован.

### Requirement: Отмена резерва (CancelReserve)

Система SHALL по `CancelReserve` отменять резерв: `CancelReserveByShipmentID`
с переводом `Shipment` обратно (из `RESERVE_LOADING`/`RESERVE_UNLOADING`), при
допустимом статусе резерва (`QUEUED`/`WAITING_APPROVAL`).

#### Scenario: Успешная отмена

- **WHEN** резерв ещё не подтверждён
- **THEN** резерв получает статус `CANCELED`, перевозка возвращается в
  предыдущее состояние.

#### Scenario: Отмена подтверждённого резерва

- **WHEN** резерв уже активен (`ACTIVE`)
- **THEN** система отвечает ошибкой недопустимого перехода (без изменений).

### Requirement: Протухание резервов по расписанию

Система SHALL по расписанию переводить просроченные резервы
(`ExpireDueWaitingApprovalReserves`, `ExpireDueActiveReserves`) в
`EXPIRED_WAITING_APPROVAL`/`EXPIRED_ACTIVE`.

#### Scenario: Протухание ожидающих/активных

- **WHEN** у резерва истёк `expires_at`
- **THEN** воркер переводит его в соответствующий `EXPIRED_*` статус; уже
  обработанные не трогаются.

### Requirement: Обработка ошибок резервов

Система SHALL маппить ошибки резерва в gRPC-коды через `internal/errors`.

#### Scenario: Маппинг

- **WHEN** недопустимый переход/конфликт резерва
- **THEN** код `FailedPrecondition`; **WHEN** нет перевозки/точки — `NotFound`;
  невалидный вход — `InvalidArgument`; прочее — `Internal`.