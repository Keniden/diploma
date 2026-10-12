## ADDED Requirements

### Requirement: Завершение точки и перевод перевозки (UpdateRoutePointDriverDeparted)

Система SHALL по `UpdateRoutePointDriverDeparted` завершать точку маршрута по
статусной модели (`COMPLETED`, либо `PACKAGES_NOT_UNLOADED`/
`PACKAGES_NOT_LOADED` при необработанных грузоместах) и, если точка последняя в
маршруте, переводить `Shipment` в `COMPLETED` (`Complete()`), публикуя
`ShipmentCompleted`.

#### Scenario: Завершение промежуточной точки

- **WHEN** водитель убывает с промежуточной точки и все пакеты обработаны
- **THEN** точка получает статус `COMPLETED`; перевозка остаётся активной.

#### Scenario: Завершение последней точки

- **WHEN** водитель убывает с последней точки и все грузоместа обработаны
- **THEN** точка завершается, `Shipment` переходит в `COMPLETED`, публикуется
  `ShipmentCompleted`.

#### Scenario: Необработанные грузоместа на точке

- **WHEN** на завершаемой точке остались не разгруженные/не загруженные пакеты
- **THEN** точка получает `PACKAGES_NOT_UNLOADED`/`PACKAGES_NOT_LOADED`, ошибка
  `FailedPrecondition` с указанием необработанных точек; перевозка не завершается.

### Requirement: Завершение перевозки (EndShipment)

Система SHALL по `EndShipment` завершать перевозку: проверить, что маршрут
пройден и грузоместа обработаны, перевести перевозку в `COMPLETED`, опубликовать
`ShipmentCompleted` и синхронизировать cargo-shipment (trigger `FINISH`) в одной
транзакции.

#### Scenario: Успешное завершение

- **WHEN** водитель вызывает `EndShipment` для пройденной перевозки
- **THEN** `Shipment` в `COMPLETED`, событие `ShipmentCompleted` опубликовано,
  грузоместа без незавершённых действий; cargo-shipment уведомлён (FINISH).

#### Scenario: Перевозка не готова к завершению

- **WHEN** есть незавершённые unload-действия или маршрут не пройден
- **THEN** система отвечает `FailedPrecondition` и не изменяет перевозку.

### Requirement: Начисление за перевозку (outbox payments)

Система SHALL писать в `lmt.outbox` сообщение `payments.last_mile_shipment_accrual`
в **той же транзакции**, что и завершение перевозки, и доставлять его в payments
(`LastMileShipmentAccrual`) с семантикой **at-least-once**.

#### Scenario: Запись вместе с завершением

- **WHEN** перевозка завершается (`ShipmentCompleted`)
- **THEN** outbox-сообщение `payments.last_mile_shipment_accrual` появляется в
  `lmt.outbox` только если транзакция агрегата зафиксирована.

#### Scenario: Доставка и повтор при ошибке

- **WHEN** воркер доставляет сообщение
- **THEN** успешный ответ payments помечает сообщение доставленным; ошибка
  оставляет его в `status='TODO'` с инкрементом `attempt`, следующий цикл
  доставляет снова; идемпотентность по идентификатору события (дублей нет).

### Requirement: Обработка ошибок завершения

Система SHALL маппить ошибки завершения в gRPC-коды через `internal/errors` и не
подвергать откату уже завершённое состояние при сбое асинхронной доставки.

#### Scenario: Маппинг

- **WHEN** перевозка не готова к завершению
- **THEN** код `FailedPrecondition`; **WHEN** перевозка/точка не найдены —
  `NotFound`; невалидный вход — `InvalidArgument`; прочее — `Internal`.