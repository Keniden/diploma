# Задачи: создание перевозки (поток 3)

TDD-порядок: домен → application → use-case/transport → synchronization → e2e.
Каждая группа: RED → GREEN → REFACTOR, тесты остаются в регрессии.

## 1. Домен shipping (агрегаты и политика)

- [x] 1.1 `entities/shipment.go`: переходы `NEW→ASSIGNED→RESERVE_LOADING→LOADING→
  IN_PROGRESS→RESERVE_UNLOADING→UNLOADING→COMPLETED`, `Cancel`; методы
  `AssignDriver`, `SetLicensePLate` — table-driven тесты переходов (недопустимый →
  ошибка), без моков
- [x] 1.2 `entities/route.go`/`route_point.go`: `NewRouteFromTemplate` (валидация
  `>=2` точек), `RebuildSequence` — тесты фабрики/перестроения
- [x] 1.3 `entities/package.go`: посев `Package` + `PackageAction` по направлениям
  (LOAD/UNLOAD) — тесты хелперов `ReadyToLoadPackages`/`PackageIDs`
- [x] 1.4 `domain/policies/shipment_policy.go`: реализовать `CanCreateShipment`
  (пакеты на source) и `CanCreateShipmentByDriverID` (нет активной перевозки) +
  порт; оставшиеся политики — `NOT_IMPLEMENTED`; тесты на мок-репозитории

## 2. Application shipping (Assembler / Progress / Demand)

- [x] 2.1 `assembler.go`: `CreateShipmentFromRouteTemplate` (сборка
  Shipment/Route/RoutePoint из шаблона) + `DriverCanCreateShipment` — мок-тесты
- [x] 2.2 `progress.go`: `AssignDriverByShipmentID`, `AssignTransportByShipmentID`,
  `CancelShipmentByID` — мок-тесты (публикация `ShipmentCreated`,
  `ShipmentDriverAssign`, `ShipmentTransportAssign`, `ShipmentCanceled`)
- [x] 2.3 `demand/`: `PlanPackagesToShipmentByID` (посев пакетов +
  `package_route_point`, прямой и возвратный поток) — мок-тесты

## 3. Use-case и транспорт (public_driver)

- [x] 3.1 `create_shipment.go`/`create_shipment_with_transport.go`: довести до
  рабочего состояния через уже собранные сервисы; мок-тесты на успех/сбой шага
  (в т.ч. `cancelCreatedShipment` при частичном сбое)
- [x] 3.2 `assign_transport.go`: `parseLicensePlate` (нормализация/пустой → ошибка),
  `AssignTransport` → `AssignTransportByShipmentID` — мок-тесты
- [x] 3.3 `transport/grpc/`: маппинг proto↔domain для `CreateShipment`/
  `CreateShipmentWithTransport`/`UpdateShipmentTransport`, валидация входа → RED→GREEN

## 4. Синхронизация с cargo-shipment

- [x] 4.1 `infrastructure.CargoShipment`-адаптер (`handlers/shipment_event_handler.go`):
  довести реализации `CreateShipment`/`SetTransport` поверх узкого интерфейса —
  тесты вызова с корректными полями
- [x] 4.2 Проверить/wiring: подписки `ShipmentCreated`/`ShipmentDriverAssign`/
  `ShipmentTransportAssign` в `register_driver.go` — smoke-тест запуска

## 5. E2E и проверки

- [x] 5.1 E2E (реальная PostgreSQL + gRPC): `CreateShipment` → таблицы
  shipment/route/route_point/package + `package_route_point`, событие →
  cargo-shipment-стаб; невалидный шаблон → ошибка без изменений
- [x] 5.2 E2E `CreateShipmentWithTransport`: атомарный успех + отмена
  (`CancelShipmentByID`) при сбое шага — активная перевозка не остаётся
- [x] 5.3 Полные проверки: `go build ./...`, `go vet ./...`,
  `golangci-lint run ./...` (только новые ошибки), `go test -race ./...` — зелёные