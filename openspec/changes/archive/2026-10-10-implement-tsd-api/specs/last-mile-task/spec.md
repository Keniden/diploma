## ADDED Requirements

### Requirement: Сканирование перевозки (ScanShipment)

Система SHALL по `ScanShipment` подтверждать резерв ворот: декодировать QR,
вызывать `ApproveReserveByRoutePointID` (резерв `ACTIVE`, `CalcScanAt`,
`Shipment` в `LOADING`/`UNLOADING`) и завершать подтверждение.

#### Scenario: Успешное сканирование перевозки

- **WHEN** ТСД сканирует QR перевозки на точке
- **THEN** резерв подтверждается (`ACTIVE`, `scanned_at` установлен), `Shipment`
  переходит в `LOADING`/`UNLOADING`, точка — в соответствующий статус; ошибок нет.

#### Scenario: Невалидный/непроверяемый QR

- **WHEN** QR не распознан или резерв не найден/недопустим
- **THEN** система отвечает `InvalidArgument`/`FailedPrecondition` и не меняет
  состояние.

### Requirement: Сканирование пакетов (ScanPackages)

Система SHALL по `ScanPackages` сканировать и проверять пакеты на точке через
`PackageScanner`: при допустимом действии обновлять статус пакета и записи
`package_route_point`, публиковать `PackageScannedEvent` (→ outbox cargo+wms);
недопустимые пакеты попадают в `failed_packages`, остальные применяются.

#### Scenario: Успешное сканирование пакетов

- **WHEN** ТСД сканирует допустимые пакеты на точке
- **THEN** статус пакетов и сканы обновляются, outbox получает cargo+wms
  (`PackageScannedEvent`).

#### Scenario: Частичный успех скна

- **WHEN** часть пакетов не удалось обработать
- **THEN** применимые пакеты применяются, ошибки попадают в `failed_packages`
  (семантика частичного успеха, как в SetPackagesStatus).

### Requirement: Список пакетов к сканированию (ListPackagesToScan)

Система SHALL по `ListPackagesToScan` возвращать пакеты, ожидающие сканирования
на указанной точке.

#### Scenario: Пустой список на точке без ожидающих

- **WHEN** на точке нет пакетов к сканированию
- **THEN** система возвращает корректный пустой список без ошибки.

#### Scenario: Список с ожидающими пакетами

- **WHEN** на точке есть пакеты `READY_TO_LOAD`/`IN_TRANSIT`, требующие скана
- **THEN** система возвращает их список (без протечки proto-типов в домен).

### Requirement: Обработка ошибок ТСД

Система SHALL маппить ошибки ТСД в gRPC-коды через `internal/errors`.

#### Scenario: Маппинг

- **WHEN** невалидный QR/скан
- **THEN** код `InvalidArgument`; **WHEN** недопустимое действие точки/резерва —
  `FailedPrecondition`; **WHEN** нет данных — `NotFound`; прочее — `Internal`.