# Дизайн: каркас по реальной архитектуре (ARCHITECTURE.md)

> Источник истины для этого дизайна — `ARCHITECTURE.md` в корне репозитория
> (актуальный документ архитектуры last-mile-task, соответствует реальному
> репозиторию `~/Documents/last-mile-task`). Реальный репо — референс структуры
> и контрактов; расхождения помечаются `TODO(LASTMILE-NNN)`.

## Контекст и цель

- Цель первой итерации — **каркас, 1:1 повторяющий дерево ARCHITECTURE.md §4**,
  с одним работающим вертикальным срезом: `SetPackagesStatus`
  (драйвер; пакеты+сканы → outbox cargo+wms → WMS HTTP) и
  `GetActiveShipmentByDriverID` (CQRS-read).
- Остальные модули (documents, support, export, bondedwarehouse, maintenance,
  порталы, ТСД-канал, клиенты) — **каркасы-заглушки**: каталоги существуют,
  внутри `doc.go` + TODO, без функциональности.
- Сервис исполняется автономно в `--mock`-режиме: встроенный HTTP WMS-мок,
  внешние gRPC-клиенты не вызываются.

## D1. Структура кода (дерево §4 — 1:1)

```
main.go                          # Точка входа → internal/app
api/proto/                       # Собственные protobuf-контракты API
  common/                        # общие модели
  v1/public/driver/              # Driver API (WBDRIVE)
  v1/public/portal/              # Публичный Portal API
  v1/portal_internal/            # Внутренний Portal API
  v1/internals/tsd/              # TSD API (складские терминалы)
  v1/internals/support/          # Support API
  v1/private/maintenance/        # Maintenance API
cmd/                             # Утилиты (claimsgen, createuser, errorgen, encrypt, outboxtest) — заглушки
internal/
  app/                           # Bootstrap приложения (main → builder.New)
  application/
    public_driver/               # Сценарии водителя (SetPackagesStatus, GetActiveShipmentByDriverID)
    portal/ documents/ tsd/      # Заглушки (doc.go + TODO)
  shipping/                      # Основной домен (DDD)
    domain/                      # entities, values, events, policies
    application/                 # Команды и оркестрация (+ ports/)
    queries/                     # CQRS read-модели (driver/, portal/)
    infrastructure/              # adapters/postgres, events, ...
  documents/ support/ export/ bondedwarehouse/ maintenance/   # Заглушки
  platform/
    application/                 # wiring (builder), worker_wms_exporter, мок-режим
    grpc/                        # gRPC-серверы + interceptor'ы
    gateway/                     # HTTP/gRPC-gateway (заглушка)
    outbox/                      # cargo, wms, documents, payments, portal, cleanup
    observability/               # logging (zerolog), health/readiness
    auth/                        # Заглушка
  clients/                       # gRPC-клиенты внешних сервисов (заглушки)
  config/                        # Конфигурация (envconfig)
  errors/                        # Доменные ошибки (ручной код + errors.yaml, TODO errgen)
  pb/                            # Сгенерированные proto внешних сервисов (заглушка)
  primitives/                    # id (UUID v7), snowflake, page, time, coordinates, country_filter
  utils/                         # Утилиты
pkg/
  gen/                           # Сгенерированный код из api/proto
  wms/                           # HTTP-клиент WMS-экспортёра (+ wmsmock для dev/e2e)
  outbox/                        # Реализация transactional outbox (generic)
  encryption/ errorgen/ qr/      # Заглушки
proto/                           # easyp-конфиги внешних proto (заглушка, TODO)
migrations/                      # goose SQL-миграции (подмножество lmt.*)
deploy/ docs/ openspec/ tasks/   # как в реальном репо
```

Все каталоги из §4 существуют. Отклонения от дерева §4 (каждое — осознанное, с TODO):
- `pkg/wms/wmsmock/` — in-process HTTP WMS-мок для dev/e2e (нет в реальном репо);
- каталоги, которых нет в §4, но есть в реальном репо и в каркасе:
  `internal/platform/{auth,locale,i18n,metrics,requestid,task_manager,transport}`,
  `internal/clients/<service>` (12 клиентов), `cmd/*` (errorgen, claimsgen и др.);
- генерация proto — локальный buf вместо easyp (BSR недоступен), см. D8;
- e2e-тест вертикального среза лежит в `internal/platform/application/slice_e2e_test.go`
  (`//go:build e2e`): в реальном репо — корневой каталог `e2e/`, которого нет в §4 —
  TODO(LASTMILE-108);
- `internal/shipping/infrastructure/eventdeps/` — срезовые реализации узких
  интерфейсов хендлеров outbox: SQL-стейджер поверх `lmt.*` + детерминированные
  стабы office/user (внешние клиенты не портированы) — TODO(LASTMILE-104/105/106);
- `infrastructure.NewProvider`/`NewAddressResolver` — стабы вместо клиентов
  cargo-core/package-content и office — TODO(LASTMILE-102/103);
- `publicdriver.NewContextDriversProvider` — идентификация водителя из
  gRPC-метаданных (`x-driver-id` интерцептор) вместо users/client-proxy —
  TODO(LASTMILE-081/086).

## D2. Домен shipping (DDD, `internal/shipping/`)

- **values/** — `id.go` (`ShipmentID`, `PackageID`, `RouteID`, `RoutePointID`,
  `ReserveID = uuid.UUID`, `ShipmentShortID` snowflake, `DriverID`, `SupplierID`,
  `PointID`, `GateID`, `TareID`, `TareType`), `shipment_status.go`,
  `package_status.go` (`UNSPECIFIED → READY_TO_LOAD → LOADED → IN_TRANSIT →
  UNLOADING → UNLOADED`, + `CANCELED`), `route_point_action_type.go`
  (`PackageActionType`: LOAD/UNLOAD/RESERVE), `route_point_package_status.go`
  (`PackageActionStatus`: TODO/DONE/CANCELED), `route_point_status.go`,
  `reserve.go`, `time.go` (`Now()` UTC), `route.go` (шаблоны), `tare.go`
  (`SplitTareSticker`), `driver.go`, `vehicle.go`, `package_demand.go`,
  `scan/` (объект скана), `tsd/`, `optimal/`. Перечень values — минимальный
  набор для среза; остальное по §5.2 вне рамок (TODO).
- **entities/** — `Shipment` (`ID`, `ShortID`, `Status`, `DriverID`, `SupplierID`,
  `LicensePlate`; переходы только бизнес-методами: `AssignDriver`,
  `SetLicensePLate`, `ReserveLoading`, `Loading`, `Start`, `ReserveUnloading`,
  `Unloading`, `Complete`, `Cancel`; машина состояний — table-driven,
  однонаправленная: `NEW → ASSIGNED → RESERVE_LOADING → LOADING → IN_PROGRESS →
  RESERVE_UNLOADING → UNLOADING → COMPLETED`, `→ CANCELED`),
  `Package` (`ID`, `ShipmentID`, `Status`, `IsReturn`, `TareID`, `TareType`,
  `Actions []*PackageAction`; хелперы `ReadyToLoadPackages`, `LoadedPackages`,
  `PackageIDs`), `Route`/`RoutePoint` (`NewRouteFromTemplate` ≥ 2 точек,
  `RebuildSequence`, `routePointStatusModel`), `Reserve`.
- **events/** — интерфейс события: `Type() T` + `LogValue() slog.Value`;
  файлы `shipment.go`, `package.go`, `route.go`, `reserve.go`,
  `shipment_created.go`; минимальный набор: `PackageScannedEvent`,
  `ShipmentLoadingEvent`, `ShipmentStartedEvent`, `ShipmentCompletedEvent`,
  `RouteOptimizedEvent`, `PointCompletedEvent`, `ReserveQueuedEvent`,
  `ReserveApprovedEvent`, `ShipmentCreatedEvent`.
- **policies/** — `ShipmentPolicy` + порт `ShipmentPolicyRepository`
  (`CanStartShipmentByID`, `CanCompleteShipmentByID` — реализуются;
  остальные — `NOT_IMPLEMENTED` + TODO).

## D3. Application-слой shipping

- `internal/shipping/application/`: скан-логика среза живёт в
  `PackageSyncService.SyncPackagesAtRoutePoint` (и `scanner/`), как в реальном
  репо; `PackageService` (Delete/Cancel/Transition-методы) — минимально
  необходимый набор:
  - Семантика (§23.6): в **одной транзакции** — обновление `package.status`
    и запись сканов `package_route_point`, публикация доменных событий
    (`PackageScannedEvent`) в in-memory bus; события транслируются в outbox
    (cargo + wms). Недопустимый переход → доменная ошибка, транзакция
    откатывается, изменений нет.
  - `ReserveService`, `Assembler`, `Progress`, `optimizer/`, `demand/` —
    заглушки (TODO).
- **ports/** — `ShipmentRepository`, `PackageRepository`, `RouteRepository`,
  `ReserveRepository`, `Transactional` (`WithTx`), `PackageEventPublisher`
  (+ `ShipmentEventPublisher`, `RouteEventPublisher`, `ReserveEventPublisher`),
  `ShipmentPolicyRepository`, ошибки портов.
- Правило: proto-типы не протекают в domain/application/queries (маппинг в
  transport-хендлерах).

## D4. CQRS (без event-sourcing)

- Физические таблицы **одни**; разделение логическое:
  - write: `shipping/infrastructure/adapters/postgres/{shipments,packages,routes,reserves}` —
    агрегатные репозитории;
  - read: `shipping/queries/driver/shipment` (readmodels + `ShipmentStore`/`Queries`) +
    `shipping/infrastructure/driver/postgres` (`Store`).
- Никакой inline read-проекции в Save (отличие от первого скелета).
- События — только для интеграции.

## D5. Outbox

- `pkg/outbox` — generic: `Writer`/`Reader` на pgx, таблица `lmt.outbox`
  (реальная схема: `id UUID PK`, `type`, `payload BYTEA`, `metadata JSONB`,
  `attempt`, `scheduled_at`, `status TEXT 'TODO'`, `err`), at-least-once.
- `internal/platform/outbox/` — паблишеры по типам: `cargo`
  (`cargo.packages.scanned`, `cargo.packages.status.updated`), `wms`
  (`wms.tares.load`, `wms.tares.unload`), `documents`, `payments`, `portal`,
  `cleanup`. Сообщения посредник — типизированные record'ы с JSON-телом.
- Доставка: `worker_wms_exporter.go` (цикл чтения→отправки→пометки,
  at-least-once; в mock-режиме WMS-мок принимает POST).
- cargo — внешний gRPC; в скелете — только publisher + TODO (easyp-прото
  недоступны; контракт cargo в репо не копируется без необходимости).

## D6. WMS

- `pkg/wms` — HTTP-клиент: `POST /load`, `POST /unload` (заголовки, таймауты,
  коды ошибок); `pkg/wms/tares` — парсер (TODO).
- `pkg/wms/wmsmock` — in-process HTTP-сервер для dev/e2e (`--mock`), хранит
  принятые события, умеет отказывать по расписанию (`FailNext`) — для
  проверки at-least-once.

## D7. Ошибки

- `internal/errors`: `Error{message, code, details, requestID,
  correlationID, attrs, protocode (gRPC), httpcode (HTTP), wrapped}`,
  `Reason()` — производное от code; `GRPCStatus()`/`FromProto`;
  `errors.yaml` — пул кодов (`namespace: lmt`).
- Скелет: код генерируется **вручную** (пакеты `internal/errors`),
  `// TODO(LASTMILE-014): в проде — errgen (cmd/errorgen), не редактировать`.
- Маппинг в gRPC-статусы — `internal/errors` (`GRPCStatus`/`FromProto`,
  §14); transport-хендлеры только вызывают маппер.

## D8. Proto и генерация

- `api/proto/` по дереву §4; реальные имена пакетов
  (`proto.v1.public.driver`, `proto.v1.internals.tsd`, ...) и сервисов
  (`Driver`, `TSD`, ...); `go_package` — в `pkg/gen/...;driverv1` и т.п.
- Реализуемые RPC: `Driver.SetPackagesStatus`, `Driver.GetActiveShipmentByDriverID`.
  Контракты 1:1 с реальным `service.proto` (полевые имена `shipment_id`,
  `packages[]{id, point_id, action, status, timestamp, tare_sticker}`,
  ответ `success`/`data.failed_packages`/`message`); значения `action` —
  только `UNSPECIFIED/LOAD/UNLOAD` (как в реальном прото; `RESERVE` — только
  доменное значение `values.PackageActionType`).
- Упрощения (TODO(LASTMILE-008)): нет `scopes`-аннотаций, openapi,
  grpc-gateway (генерация только стабов: protoc-gen-go + grpc);
  валидация — PGV `(validate.rules)` вместо `buf.validate` (BSR недоступен);
  `go_package` — полный путь `last-mile-task/pkg/gen/...;driverv1` вместо
  module-маппинга реального репо.
- Генерация — только через `task proto`. Отклонение от §2: локальный buf
  вместо easyp (BSR/внутренние proto недоступны), см. TODO(LASTMILE-011).

## D9. Конфигурация и запуск

- `main.go` → `internal/app.App.Run` → `builder.New` → `application.New` →
  `Run`/`Shutdown`; graceful shutdown по сигналу.
- `internal/config` — `envconfig`; блоки Config: `GRPC`, `Postgres`, `WMS`,
  `Outbox`, `Lifecycle`, `Log`, `Mock` (остальные блоки — TODO).
- `--mock`-флаг: WMS-мок поднимается in-process, внешние вызовы отключены.

## D10. Миграции

- `pressly/goose`; `migrations/` — подмножество реальных таблиц:
  `lmt.shipment`, `lmt.route`, `lmt.route_point`, `lmt.package`,
  `lmt.package_route_point`, `lmt.reserve`, `lmt.outbox`
  (+ goose-инициализация); остальные 30+ — TODO(LASTMILE-015).
- `task migrate` — goose up.

## D11. Заглушки-каркасы

- `internal/application/{portal,documents,tsd}`, `internal/{documents,support,
  export,bondedwarehouse,maintenance}`, `internal/clients/*`,
  `internal/pb/`, `internal/platform/{gateway,auth,locale,i18n,metrics,
  requestid,task_manager,transport}` (частично), `cmd/*`, `pkg/{encryption,
  errorgen,qr}`, `proto/` — `doc.go` + `TODO(LASTMILE-NNN)`; без импорта
  реальных внешних SDK.

## D12. TDD и проверки

- Порядок: домен → application → queries → infrastructure → transport/wiring.
- RED → GREEN → REFACTOR обязателен на каждом слое; тесты остаются регрессионными.
- Проверки: `go build ./...`, `go vet ./...`, `golangci-lint run ./...`,
  `go test ./...` (+ e2e на реальной PostgreSQL: `--mock`).
- Домен — без моков (in-memory entities/values, table-driven);
  application — моки портов; инфраструктура — реальная БД (Docker), e2e.