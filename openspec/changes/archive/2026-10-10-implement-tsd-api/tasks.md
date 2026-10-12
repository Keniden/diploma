# Задачи: ТСД-канал (11.2, потоки 4–6)

TDD-порядок: use-case → transport → wiring → e2e.
Каждая группа: RED → GREEN → REFACTOR, тесты остаются в регрессии.

## 1. Use-case (internal/application/tsd/)

- [x] 1.1 `scan_shipment.go`: `ScanShipment` — `pkg/qr.DecodeQR` →
  `ApproveReserveByRoutePointID` → `Shipment` в `LOADING`/`UNLOADING`, точка по
  `routePointStatusModel` — мок-тесты (успех/невалидный QR/недопустимый резерв)
- [x] 1.2 `scan_packages.go`: `ScanPackages` — делегирование в
  `PackageScanner` (частичный успех → `failed_packages`, событие `PackageScannedEvent`)
  — мок-тесты
- [x] 1.3 `list_packages_to_scan.go`: `ListPackagesToScan` (пакеты
  `READY_TO_LOAD`/`IN_TRANSIT` на точке) — мок-тесты (пустой список без ошибки)
- [x] 1.4 `application.go`/`doc.go`: собрать `Application` с портами
  (Scanner, ReserveService, read-источник), убрать `TODO(LASTMILE-071)`

## 2. Transport (platform/grpc/tsd)

- [x] 2.1 `server.go`: замена стаба на реальные хендлеры
  `ScanShipment`/`ScanPackages`/`ListPackagesToScan` поверх
  `internal/application/tsd`; маппинг контекста/метаданных (Empty→Empty),
  `RegisterGRPC` — RED→GREEN
- [x] 2.2 Маппинг ошибок ТСД в gRPC-коды через `internal/errors`
  (`InvalidArgument`/`FailedPrecondition`/`NotFound`) — тесты маппинга

## 3. Wiring и QR-рукопожатие

- [x] 3.1 `register_tsd.go`: сборка TSD-сервера из `scannerService`,
  `ReserveService`, read-источников — smoke-тест запуска
- [x] 3.2 `pkg/qr.DecodeQR` + `internal/platform/application/handshake.go`
  (`TODO(LASTMILE-098)`) — тесты декодирования

## 4. E2E и проверки

- [x] 4.1 E2E (реальная PostgreSQL + gRPC): `ScanShipment` → резерв `ACTIVE`,
  `Shipment` в `LOADING`/`UNLOADING`; невалидный QR → ошибка без изменений
- [x] 4.2 E2E `ScanPackages` → пакеты+сканы обновлены, outbox cargo+wms получил
  `PackageScannedEvent`; частичный успех → `failed_packages`
- [x] 4.3 E2E `ListPackagesToScan` → корректный список, пусто на точке без
  ожидающих
- [x] 4.4 Полные проверки: `go build ./...`, `go vet ./...`,
  `golangci-lint run ./...` (только новые ошибки), `go test -race ./...` — зелёные