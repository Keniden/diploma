# Дизайн: ТСД-канал (ARCHITECTURE.md 11.2, потоки 4–6)

> Мотивация — см. `proposal.md`. Опора на уже готовое: `PackageScanner`
> (сканирование/проверка пакетов, подключён к sync-bus → outbox cargo+wms),
> `ReserveService`/`Reserve` (закладывается в изменении Reserve/QR), модель
> `routePointStatusModel`. Протокол TSD — упрощённый Empty→Empty
> (`TODO(LASTMILE-008)`), easyp-контракт реального репо недоступен.

## Контекст

- `internal/platform/grpc/tsd/server.go` — стаб: `UnimplementedTSDServer`,
  `RegisterGRPC`. `register_tsd.go` возвращает `grpctsd.New()`.
- `internal/application/tsd/` — только `doc.go` (`TODO(LASTMILE-071)`).
- `shipping/application/scanner.PackageScanner` — реализован и протестирован
  (сканирование пакетов, событие `PackageScannedEvent`).
- `pkg/qr` — заглушка (`TODO(LASTMILE-098)`), `handshake.go` — `createHandshakeService`
  возвращает nil.
- `domain/values/scan` — модели скана; `routePointStatusModel` — переходы точек.

## Goals / Non-Goals

**Goals:**
- `ScanShipment`: декодирование QR → подтверждение резерва → перевод
  `Shipment` в `LOADING`/`UNLOADING`, завершение точки по статусной модели.
- `ScanPackages`: сканирование пакетов через `PackageScanner` (частичный успех).
- `ListPackagesToScan`: список ожидающих пакетов на точке.
- Убрать `Unimplemented`, собрать реальные хендлеры в `platform/grpc/tsd`.

**Non-Goals:**
- Реализация создания резерва (это изменение Reserve/QR) — здесь только
  подтверждение/approve.
- Полный easyp-контракт TSD (остаётся упрощённым Empty→Empty, TODO(LASTMILE-008)).
- Порт handshake-validator/office/cargo — срез-стабы.

## Решения

### D1. ТСД-use-case поверх существующих сервисов
`internal/application/tsd/` реализует три сценария, делегируя в
`PackageScanner` (ScanPackages), `ReserveService.ApproveReserveByRoutePointID`
(+ `EntityRepository`) и read-источники (ListPackagesToScan). Никакой новой
оркестрации: узлы из реального флоу уже разнесены по сервисам.

### D2. ScanShipment = подтверждение резерва
Резерв создаёт водитель (Reserve/QR); ТСД подтверждает его сканом.
`ScanShipment` декодирует QR (`pkg/qr.DecodeQR`) → находит резерв →
`ApproveReserveByRoutePointID` → `Shipment` в `LOADING`/`UNLOADING`, точка — по
`routePointStatusModel`. Ошибка QR/резерва — `InvalidArgument`/`FailedPrecondition`.

### D3. ScanPackages через PackageScanner (частичный успех)
`PackageScanner` уже обрабатывает пакеты и публикует `PackageScannedEvent`.
Переиспользуем его как есть; семантика частичного успеха (`failed_packages`)
сохраняется. Контракт Empty→Empty — данные скана передаются через контекст
(детали уточнить при порте реального прото, `TODO(LASTMILE-008)`).

### D4. ListPackagesToScan через read-источники
Возвращает пакеты `READY_TO_LOAD`/`IN_TRANSIT` на точке через query/adapter;
proto-типы не протекают в домен (маппинг в transport-хендлере).

### D5. Transport в `platform/grpc/tsd`
Замена стаба: `Server` держит `internal/application/tsd` (порты),
`RegisterGRPC` как раньше. `register_tsd.go` собирает зависимости
(`scannerService`, `ReserveService`, read-источники) — аналогично `CreateDriver`.

### D6. Ошибки — `internal/errors`
Коды `tsd_*`/`qr_*` в `errors.yaml` (`namespace: lmt`), маппинг
`InvalidArgument`/`FailedPrecondition`/`NotFound` через `MapToGRPC`.

## Risks / Trade-offs

- **Empty→Empty контракт** → реальные входные параметры скрыты.
  → Mitigation: передача через контекст/маппинг в transport-хендлере
  (`TODO(LASTMILE-008)`), поведение покрывается e2e на мок-данных.
- **Зависимость от Reserve/QR изменения** — `ApproveReserveByRoutePointID`
  появится там; риск взаимной блокировки. → Mitigation: чёткий порт, TSD
  зависит только от интерфейса.
- **Заглушка `pkg/qr`/handshake** → QR-декодирование не реальное.
  → Mitigation: узкий порт + `TODO(LASTMILE-098)`.
- **Протечка proto-типов** при Empty-контракте — держать маппинг в transport.

## План миграции

- Новых миграций нет. TSD-сервис перестаёт быть `Unimplemented` — это расширение,
  обратной несовместимости нет (свойство интерфейса сохраняется).
- Откат: вернуть стаб `grpctsd.New()` в `register_tsd.go`.

## Открытые вопросы

- Реальный прото-контракт TSD (методы/поля) — уточнить при порте easyp;
  на срез — Empty→Empty.
- Откуда брать идентификатор точки/резерва при Empty-запросе (контекст/метаданные)
  — решить в transport-хендлере.