# Задачи: каркас по реальной архитектуре

TDD-порядок: домен → application → queries → infrastructure → transport/wiring.
Каждая группа: RED (падающий тест) → GREEN → REFACTOR, тесты остаются в регрессии.

## 1. Bootstrap и дерево (§4)

- [x] 1.1 Создать полное дерево каталогов по ARCHITECTURE.md §4 (все каталоги, в т.ч. заглушки с `doc.go` + `TODO(LASTMILE-NNN)`), `main.go` → `internal/app`, Taskfile (`task test/lint/proto/migrate/run`), go.mod с реальными зависимостями (pgx, squirrel, scany, goose, envconfig, uuid, lo, zerolog, otel, protovalidate/validate, grpc)
- [x] 1.2 `internal/primitives`: `id.go` (`NewID()` UUID v7, типы ID), `snowflake.go`, `page.go`, `time.go`, `coordinates.go` — table-driven тесты
- [x] 1.3 `internal/config`: envconfig-конфиг (`GRPC`, `Postgres`, `WMS`, `Outbox`, `Lifecycle`, `Log`, `Mock`) + тест дефолтов
- [x] 1.4 `internal/errors`: тип `Error` (code/protocode/httpcode/details/attrs/requestID/correlationID/wrapped), `errors.yaml` (пул `namespace: lmt`), `GRPCStatus()`/`FromProto`/`Reason()`, маппинг-тесты; TODO про errgen
- [x] 1.5 Миграции goose: `migrations/0001_goose_initial` + `lmt.shipment`, `lmt.route`, `lmt.route_point`, `lmt.package`, `lmt.package_route_point`, `lmt.reserve`, `lmt.outbox`; `task migrate` применяет, идемпотентно

## 2. Proto-контракты

- [x] 2.1 `api/proto/v1/public/driver/service.proto`: пакет `proto.v1.public.driver`, сервис `Driver` с реальными именами RPC (SetPackagesStatus, GetActiveShipmentByDriverID реализуются; остальные объявлены), контракты 1:1 с реальным репо (поля, ответы), валидация PGV; TODO(LASTMILE-008) про упрощения (без scopes/openapi/gateway)
- [x] 2.2 Остальные API-поверхности: `api/proto/common`, `v1/internals/tsd` (+support), `v1/portal_internal`, `v1/public/portal`, `v1/private/maintenance` — файлы-заглушки с сервисами/пакетами (TODO)
- [x] 2.3 Генерация в `pkg/gen` через `task proto` (buf, локальные плагины); проверка, что сгенерированный код компилируется; proto не импортируется из domain/application/queries

## 3. Домен shipping (DDD)

- [x] 3.1 `values/`: статусы (ShipmentStatus, PackageStatus, RoutePointStatus, ReserveStatus, PointAction), ID-типы, голосованные enum — table-driven тесты маппинга/валидности
- [x] 3.2 `entities/shipment.go`: агрегат Shipment, машина состояний (NEW→ASSIGNED→RESERVE_LOADING→LOADING→IN_PROGRESS→RESERVE_UNLOADING→UNLOADING→COMPLETED, CANCELED) бизнес-методами; недопустимый переход → ошибка — RED→GREEN
- [x] 3.3 `entities/package.go`: Package (статусы READY_TO_LOAD→LOADED→IN_TRANSIT→UNLOADING→UNLOADED+CANCELED, Actions, хелперы) + PackageAction — RED→GREEN
- [x] 3.4 `entities/route.go` + `route_point.go`: Route/RoutePoint (NewRouteFromTemplate ≥2, RebuildSequence, routePointStatusModel) — RED→GREEN
- [x] 3.5 `entities/reserve.go`: Reserve (QUEUED→WAITING_APPROVAL→ACTIVE→RELEASED/CANCELED + Expired-ветки EXPIRED_WAITING_APPROVAL/EXPIRED_ACTIVE) — RED→GREEN
- [x] 3.6 `events/`: интерфейс `{Type() T; LogValue() slog.Value}` + события (PackageScanned, ShipmentLoading/Started/Completed, RouteOptimized, PointCompleted, ReserveQueued/Approved, ShipmentCreated) — тесты Type/LogValue
- [x] 3.7 `policies/`: ShipmentPolicy + порт репозитория; реализовать CanStartShipmentByID, CanCompleteShipmentByID (остальные — TODO)

## 4. Application shipping

- [x] 4.1 `application/ports/`: ShipmentRepository, PackageRepository, RouteRepository, ReserveRepository, Transactional, PackageEventPublisher (+ Shipment/Route/ReserveEventPublisher), ShipmentPolicyRepository — контракты как в реальном репо
- [x] 4.2 Падающие тесты `PackageSyncService.SyncPackagesAtRoutePoint` (+ `scanner/`) на мок-портах: успех (пакеты+сканы в одной транзакции, события в bus/outbox), недопустимый переход (откат, контекст), пустой batches; RED
- [x] 4.3 Реализация `PackageSyncService` (inmem bus, транзакционная обёртка, оркестрация сканов по source/mid/target); GREEN; PackageService (Delete/Cancel/Transition) — минимально необходимый набор
- [x] 4.4 Заглушки сервисов: ReserveService, Assembler, Progress, optimizer/, demand/ — doc.go + TODO (не ломают сборку)

## 5. Queries (CQRS read)

- [x] 5.1 `queries/driver/shipment`: readmodels (Shipment, Route, RoutePoint, Package, PackageAction) + `ShipmentStore` — мок-тесты интерфейса, RED→GREEN
- [x] 5.2 `queries/driver/reserve|basket|showcase`, `queries/portal/*` — заглушки-интерфейсы + TODO

## 6. Infrastructure shipping

- [x] 6.1 `infrastructure/adapters/postgres/shipments`: ShipmentRepository (агрегат целиком, транзакция) — интеграционные тесты на реальной PostgreSQL (Docker), RED→GREEN
- [x] 6.2 `infrastructure/adapters/postgres/packages`: PackageRepository (пакеты + сканы package_route_point) — RED→GREEN
- [x] 6.3 `infrastructure/driver/postgres`: pgdriver.Store (read-сторона, readmodels) — RED→GREEN
- [x] 6.4 `infrastructure/events/bus/inmem` + `bus/outbox` + `handlers/{cargo,warehouse}`: маршрутизация событий в outbox-паблишеры — RED→GREEN

## 7. Outbox и WMS

- [x] 7.1 `pkg/outbox`: Writer/Reader (pgx, `lmt.outbox`, at-least-once, идемпотентность по event_id) — RED→GREEN
- [x] 7.2 `pkg/wms`: HTTP-клиент (POST /load, /unload, заголовки, таймауты) — тесты с httptest-сервером, RED→GREEN
- [x] 7.3 `pkg/wms/wmsmock`: in-process HTTP-мок (приём событий, FailNext, идемпотентность) — RED→GREEN
- [x] 7.4 `platform/outbox/{cargo,wms,cleanup}`: паблишеры (типы сообщений cargo.packages.*, wms.tares.*) — RED→GREEN
- [x] 7.5 `platform/application/worker_wms_exporter.go`: цикл доставки (чтение→отправка→пометка), at-least-once — RED→GREEN

## 8. Transport и bootstrap

- [x] 8.1 `platform/grpc`: сервер + interceptor'ы (logging, recovery, requestID) — тесты
- [x] 8.2 `application/public_driver/transport/grpc`: маппинг proto↔domain, валидация (InvalidArgument), маппинг ошибок (NotFound/FailedPrecondition/Internal) — RED→GREEN
- [x] 8.3 `application/public_driver/dtos` + use-case `SetPackagesStatus`, `GetActiveShipmentByDriverID` — мок-тесты, RED→GREEN
- [x] 8.4 `platform/application` wire (builder, register_*), `internal/app`, `main.go`, `--mock`, graceful shutdown — smoke-тест запуска

## 9. E2E и проверки

- [x] 9.1 E2E (реальная PostgreSQL + mock WMS + gRPC): SetPackagesStatus → таблицы+outbox → доставка в wmsmock; недопустимый переход → FailedPrecondition, без изменений; FailNext(1) → повторная доставка (at-least-once)
- [x] 9.2 E2E GetActiveShipmentByDriverID: read-модель видит изменения после среза
- [x] 9.3 Полные проверки: `go build ./...`, `go vet ./...`, `golangci-lint run ./...`, `go test ./...`, `go test -race ./...`, `govulncheck ./...` (если доступен), `task proto`, `task migrate` — зелёные (govulncheck: вызовов уязвимого кода нет; оставшиеся 2 находки — импортируемые пакеты без вызова, grpc поднят до v1.82.1 как в реальном репо)

## 10. Заглушки-каркасы (финальный штрих)

- [x] 10.1 `internal/application/{portal,documents,tsd}`, `internal/{documents,support,export,bondedwarehouse,maintenance}`, `internal/clients/*`, `internal/pb/`, `internal/platform/{gateway,auth,locale,i18n,metrics,requestid,task_manager,transport}`, `cmd/*`, `pkg/{encryption,errorgen,qr}`, `proto/`, `deploy/`, `docs/`, `tasks/` — doc.go + TODO, сборка зелёная