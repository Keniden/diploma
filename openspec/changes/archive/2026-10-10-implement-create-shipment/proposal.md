# Почему

Сервис уже умеет срез `SetPackagesStatus` (погрузка/разгрузка) и генерацию
документов, но **начало пути водителя** — создание перевозки (ARCHITECTURE.md
§23 поток 3) — остаётся заглушкой. Без `CreateShipment` /
`CreateShipmentWithTransport` / `UpdateShipmentTransport` нельзя собрать
перевозку из маршрутного шаблона, назначить водителя и ТС, спланировать спрос и
засеять пакеты, а также синхронизировать перевозку с cargo-shipment.

Помимо этого, нужно корректно вписать **новые стадии фабрикации перевозки** в
домен: только после создания перевозка (пакеты/маршрут/точки) становится
«готова» к погрузке, с которой уже работает существующий срез.

# What Changes

- **Use-case** (`internal/application/public_driver/`): реализация
  `CreateShipment` (сборка из шаблона + назначение водителя + планирование
  спроса), `CreateShipmentWithTransport` (те же шаги + назначение ТС, атомарно,
  с отменой созданной перевозки при ошибке), `AssignTransport` /
  `UpdateShipmentTransport` (простановка госномера). Параллельно доводятся
  маппинги в `transport/grpc` и валидация входа.
- **Домен shipping**: доработка агрегата `Shipment` (переходы
  `NEW→ASSIGNED→...`, `AssignDriver`, `SetLicensePLate`), `Route`/`RoutePoint`
  (сборка из шаблона `>=2` точек), посев `Package`/`PackageAction` по спросу.
- **Application shipping**: `Assembler` (сборка из маршрутного шаблона),
  `Progress` (AssignDriverByShipmentID, AssignTransportByShipmentID,
  CancelShipmentByID), `demand` (PlanPackagesToShipmentByID) + политика
  `CanCreateShipment`/`CanCreateShipmentByDriverID`.
- **Синхронизация с cargo-shipment** через доменные события
  (`ShipmentCreated`, `ShipmentDriverAssign`, `ShipmentTransportAssign`) уже
  подписанные на sync-bus — довести реализацию `CargoShipment`-адаптера.
- **Wiring**: `register_driver.go` (уже подключает assembler/progress/demand) —
  проверка и досконструирование зависимостей (`RouteTemplatesProvider`,
  `DemandProvider`).
- **Ошибки** в `internal/errors` (`InvalidArgument` для невалидного шаблона,
  `FailedPrecondition` для «у водителя уже есть активная перевозка»).

# Capabilities

### New Capabilities

- (нет: относится к существующему `last-mile-task`)

### Modified Capabilities

- `last-mile-task`: добавляются требования создания перевозки — сборка из
  маршрутного шаблона, назначение водителя/ТС, планирование спроса и посев
  пакетов, синхронизация с cargo-shipment, отмена при сбое.

# Impact

- Новые файлы: `internal/shipping/application/assembler.go`,
  `internal/shipping/application/progress.go`, `internal/shipping/application/demand/`
  (частично), политика в `domain/policies`, транспорт-хендлеры
  `public_driver/transport/grpc/*`.
- Изменяются: `internal/application/public_driver/{create_shipment,
  create_shipment_with_transport,assign_transport}.go`, `register_driver.go`,
  `internal/shipping/domain/entities/{shipment,route,route_point,package}.go`.
- Новых миграций нет (таблицы `lmt.shipment/route/route_point/package/
  package_route_point` уже есть).
- Внешний Driver API: добавляются реализации `CreateShipment`,
  `CreateShipmentWithTransport`, `UpdateShipmentTransport` (контракты не меняются).
- Заглушки `RouteTemplatesProvider`/`DemandProvider`/`CargoShipment` сохраняются
  (внешние сервисы не портируем — TODO).