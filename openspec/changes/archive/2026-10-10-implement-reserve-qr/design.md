# Дизайн: резерв ворот и QR (ARCHITECTURE.md §23 потоки 4 и 7)

> Мотивация — см. `proposal.md`. Поток опирается на существующий домен: агрегат
> `Reserve` (статусы `QUEUED→WAITING_APPROVAL→ACTIVE→RELEASED/CANCELED`,
> `EXPIRED_*`) и порт `ReserveManager` в `shipping/application/ports/reserver.go`
> уже есть; `lmt.reserve` (включая `status`/`expires_at`) — в миграциях.

## Контекст

- `entities/reserve.go`: `NewQueuedReserve`, `NewReserveForApproval`,
  `NewFakeReserveForTargetApproval`, `IsFake`, `Approve`, `Release`, `Cancel`,
  `ExpireWaitingApproval`, `ExpireActive` — реализованы.
- `ports.ReserveManager`: `ReserveOrQueue`, `CancelReserve`, `ApproveReserveTask`
  — интерфейс есть, реализации-стаба нет.
- `lmt.reserve`: `reserve_id, shipment_id, route_point_id, task_id, scanned_at,
  gate_id, gate_name, status, expires_at, created_at, updated_at`.
- `pkg/qr` и `handshake.go` — заглушки (`TODO(LASTMILE-098)`).
- `ReserveService` в `shipping/application` отсутствует — нужно создать.
- Триггер подтверждения резерва — ТСД `ScanShipment` (DecodeQR → `ReserveDriverArrived`),
  реализуется в отдельном изменении TSD; здесь закладываем интерфейс
  подтверждения (`ApproveReserveByRoutePointID`), который вызывает TSD-флоу.

## Goals / Non-Goals

**Goals:**
- `ReserveService` (Queue/Approve/Release/Get/Cancel/Expire) + `ReserveRepository`.
- Driver use-case: `CreateReserve`, `GetReserveQRStatus`, `UpdateReserveQRStatus`,
  `CancelReserve`.
- Перевод `Shipment` в `RESERVE_LOADING`/`RESERVE_UNLOADING` и обратно.
- Интеграция `ReserveManager` (gate-reserve) + рукопожатие QR (`pkg/qr`).
- Воркер протухания резервов по расписанию.

**Non-Goals:**
- Реализация ТСД-канала `ScanShipment`/`ScanPackages` (отдельное изменение TSD);
  здесь — только интерфейс подтверждения.
- Порт реальных gate-reserve/handshake-validator/users — срез-стабы (TODO).
- Оптимизация маршрута, экспорт документов.

## Решения

### D1. ReserveService — один сервис, как в реальном репо
`internal/shipping/application/reserve.go` с методами
`QueueReserveByShipmentID`, `ApproveReserveByShipmentID`,
`ApproveReserveByRoutePointID`, `ReleaseReserveByShipmentID`,
`GetReserveByRoutePointID`, `CancelReserveByShipmentID`,
`ExpireDueWaitingApprovalReserves`, `ExpireDueActiveReserves`. Использует
порты `ReserveRepository`, `ReserveManager` и `Transactional` (одна транзакция
на резерв + перевод `Shipment`).

### D2. Создание резерва — через `ReserveManager.ReserveOrQueue`
`CreateReserve` не вызывает gate напрямую, а делегирует в `ReserveManager`
(внешний gate-reserve). В срезе — стаб реализации (`TODO(LASTMILE-099)`)
с детерминированным `ReserveTask`/`Gate`, рукопожатие QR — через `pkg/qr`
(`CreateSession`) в `handshake.go`. Так сохраняется реальная точка замены при
порте внешних сервисов.

### D3. Перевод `Shipment` в резервные стадии
`QueueReserveByShipmentID` в той же транзакции переводит `Shipment` в
`RESERVE_LOADING` (source) или `RESERVE_UNLOADING` (последняя точка) и сохраняет
резерв; отмена возвращает перевозку в предыдущую стадию. Проверка — политика
`CanReserveGate` (без активного резерва, назначенная перевозка, source/последняя
точка).

### D4. Статус QR и подтверждение ТСД
`GetReserveQRStatus` читает `lmt.reserve` через read-запрос (is_scanned =
`scanned_at != nil`). `UpdateReserveQRStatus`/подтверждение вызывают
`ApproveReserveByRoutePointID` (вызовет `ReserveManager.ApproveReserveTask` +
`Reserve.Approve` + перевод `Shipment` в `LOADING`/`UNLOADING`); ТСД-флоу
`ScanShipment` подключается к этому же методу (интерфейс, не реализация).

### D5. Воркер протухания
`worker_reserve_expirer.go` по расписанию вызывает
`ExpireDueWaitingApprovalReserves`/`ExpireDueActiveReserves` (источник —
`expires_at` в `lmt.reserve`); идемпотентно по статусу. Ошибки — в лог.

### D6. Ошибки — `internal/errors`
Коды `reserve_*`/`gate_*` в `errors.yaml` (`namespace: lmt`); маппинг
`FailedPrecondition`/`NotFound`/`InvalidArgument` через `MapToGRPC`
(`// TODO(LASTMILE-014): errgen`).

## Risks / Trade-offs

- **Стаб `ReserveManager`/`pkg/qr`** → не совпадает с реальным gate-reserve и
  handshake-validator. → Mitigation: узкие порты + `TODO(LASTMILE-099/098)`,
  e2e на мок-данных.
- **Переходы `RESERVE_LOADING`/`RESERVE_UNLOADING`** — граничные статусы
  `Shipment` уже в стейт-машине; риск рассинхрона с реальной схемой.
  → Mitigation: table-driven тесты переходов; перевод только бизнес-методами.
- **Протухание по `expires_at`** — зависит от корректности времени/настройки
  воркера. → Mitigation: идемпотентный воркер, тест на границе истечения.
- **Частичная сборка ТСД-потока** → подтверждение резерва из ТСД остаётся на
  интерфейсе. → Mitigation: чёткий порт `ApproveReserveByRoutePointID`, TSD
  реализует его в своём изменении.

## План миграции

- Новых миграций нет. Обратная совместимость: `ReserveRepository` читает/пишет
  по существующей схеме; уже заполненные резервы не затрагиваются.
- Откат: снятие воркера протухания и подписок; таблицы остаются.

## Открытые вопросы

- Точный контракт `ReserveOrQueue`/`ApproveReserveTask` (поля task/gate) —
  уточнить при порте gate-reserve; на срез — детерминированный стаб.
- Формат QR-сессии и эндпоинты handshake-validator — по реальному сервису (TODO).