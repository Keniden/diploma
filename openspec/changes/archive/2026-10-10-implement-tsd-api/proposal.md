# Почему

Сервис уже запускает gRPC-сервер TSD, но он возвращает `Unimplemented` на все
методы (`internal/platform/grpc/tsd/server.go`, `TODO(LASTMILE-095)`). Между тем
складской канал — ТСД (терминал сбора данных) — это вторая половина полного
флоу: сотрудник склада сканирует QR и тары, разгружает/загружает и подтверждает
точки (потоки 4–6 §23). `PackageScanner` уже реализован и подключён к sync-bus,
а резерв-ворот и `Shipment`-переходы закладываются в изменении Reserve/QR —
осталось собрать ТСД-use-case и транспорт поверх них.

# What Changes

- **Use-case** (`internal/application/tsd/`): `ScanShipment` (декодирование QR
  рукопожатия → `ApproveReserveByRoutePointID` → перевод `Shipment` в
  `LOADING`/`UNLOADING`), `ScanPackages` (сканирование/проверка пакетов через
  `PackageScanner` → outbox cargo+wms), `ListPackagesToScan` (список пакетов,
  ожидающих сканирования на точке).
- **Transport** (`internal/platform/grpc/tsd/`): замена стаба на реальные
  хендлеры поверх `internal/application/tsd` (контракт — упрощённый
  Empty→Empty, `TODO(LASTMILE-008)`).
- **Домен shipping**: `ScanShipment` опирается на декодирование QR
  (`pkg/qr.DecodeQR`) и подтверждение резерва (`ReserveService`), перевод точки
  `route_point` по модели `routePointStatusModel`.
- **Wiring**: `register_tsd.go` — собрать TSD-сервер из `scannerService`,
  `ReserveService`, query-источников.
- **Ошибки** в `internal/errors`: `InvalidArgument` (невалидный QR/скана),
  `FailedPrecondition` (недопустимое действие точки).

# Capabilities

### New Capabilities

- (нет: относится к существующему `last-mile-task`)

### Modified Capabilities

- `last-mile-task`: добавляются требования ТСД-канала — сканирование перевозки
  (подтверждение резерва/перевод `Shipment`), сканирование пакетов (перевод в
  outbox cargo+wms), список пакетов к сканированию; убирается заглушечный
  `Unimplemented`.

# Impact

- Новые файлы: `internal/application/tsd/{application.go,scan_shipment.go,
  scan_packages.go,list_packages_to_scan.go}`, правки
  `internal/platform/grpc/tsd/server.go`, `internal/platform/application/register_tsd.go`.
- Изменяются: `internal/shipping/infrastructure/events/handlers/*` (нет — уже
  есть), `internal/application/public_driver` (нет), подключение
  `scannerService`/`ReserveService` в `register_tsd.go`.
- Новых миграций нет.
- Внешний/внутренний API: TSD-сервис перестаёт быть `Unimplemented` (контракт
  Empty→Empty упрощён, `TODO(LASTMILE-008)`).
- Заглушки `pkg/qr`/`handshake`/`office`/`cargo` сохраняются (внешние сервисы не
  портируем — TODO).