# Why

Существующий скелет был построен по абстрактным правилам из
`openspec/config.yaml`, но **не соответствует реальной архитектуре сервиса**
— в корне появился актуальный документ `ARCHITECTURE.md` (имеется локальная
копия реального репозитория `~/Documents/last-mile-task`). Цель этого
изменения — привести каркас в 1:1 соответствие дереву §4, доменным моделям
§5–§16 и потокам §23, оставив один работающий вертикальный срез
(`SetPackagesStatus`).

# What Changes

- Структура кода: контексты (`internal/shipping/{domain,application,queries,infrastructure}`,
  `support`, `documents`, `export`, ...), `internal/application/{public_driver,portal,tsd}`,
  `internal/platform/*`, `internal/clients`, `internal/config`, `internal/errors`,
  `internal/primitives`, `pkg/{gen,wms,outbox}`, `main.go`-bootstrap (без Cobra).
- Домен shipping: реальные сущности и стейт-машины (Shipment
  NEW→...→COMPLETED, Package READY_TO_LOAD→...→UNLOADED, RoutePoint, Reserve),
  события с `Type()`/`LogValue()`, политики.
- Вертикальный срез: `SetPackagesStatus` (пакеты+сканы в одной транзакции,
  outbox cargo+wms, WMS HTTP `/load`/`/unload`) + read-query
  `GetActiveShipmentByDriverID` (CQRS-чтение, тот же поток).
- Прото-контракты в `api/proto/...` по реальным именам RPC; генерация в `pkg/gen`.
- Миграции goose с таблицами `lmt.*`; конфиг через envconfig; ошибки через
  `internal/errors` (+ `errors.yaml`, TODO errgen).
- Всё остальное — каркасы-заглушки с `TODO(LASTMILE-NNN)`.

# Impact

- Переработка кода скелета: домен (старые PENDING/LOADED/UNLOADED уходят),
  инфраструктура, транспорт, bootstrap — меняются полностью.
- Удаляется: inline read-модель ($CQRS теперь логическое разделение на одних
  таблицах), старые `api/driver/shipment`, gRPC-WMS (заменён HTTP).
- Модель ошибок: `domerr` заменяется на `internal/errors` (code/protocode).
- Внешний контракт Driver (имена RPC, поля) приближается к реальному.
- Заглушки сохраняют зелёную сборку и базовый набор тестов (регрессия).