# Дизайн: витрина + корзина водителя (ARCHITECTURE.md §23 потоки 1–2)

> Мотивация — см. `proposal.md`. Опора на уже существующие интерфейсы
> `ShowcaseStore`/`BasketItemsStore` в `shipping/queries/driver/{showcase,basket}`,
> таблицу `lmt.basket`, `primitives.Page`/`PaginationResponse` и стабы внешних
> клиентов (office/route/cargo/package-content).

## Контекст

- `showcase.ShowcaseStore`: `ListPoints(page)`, `ListRoutesByPoint(page, pointID)`;
  readmodels `ShowcasePoint`/`ShowcaseRoute`/`ShowcaseRoutePoint`.
- `basket.BasketItemsStore`: `ListItemsByDriverID`, `AddItem(driverID, routeID)`,
  `DeleteItem(driverID, itemID)`, `ClearBasket(driverID)`;
  readmodel `BasketItem`.
- `lmt.basket(driver_id, item_id, routecar_id, created_at, updated_at)`.
- Внешние клиенты office (ListOffices/ListPointsV2), route (ListRoutecars),
  cargo (ListPackagesToLoad), package-content (GetPackagesContent) — **не
  портированы** (стабы `TODO LASTMILE-075`). В срезе витрина отдаёт
  детерминированные данные.
- Driver RPC в `service.proto` — `Empty→Empty` (`TODO(LASTMILE-008)`): входные
  параметры (point_id, page, route_id, item_id) передаются через gRPC-metadata/
  context.
- `internal/application/public_driver/application.go` — `Application` +
  `Config`, куда добавляются новые зависимости.

## Goals / Non-Goals

**Goals:**
- Адаптеры `basketpg` (локальная `lmt.basket`) и `showcasepg` (+ slice-стабы
  внешних сервисов) поверх существующих интерфейсов.
- Use-case `ListPoints`/`ListRoutesByPoint`/`ListItems`/`AddItem`/`DeleteItem`/
  `ClearBasket` с пагинацией/валидацией.
- gRPC-хендлеры + wiring; маппинг ошибок.
- Идемпотентное добавление и «чужой элемент → NotFound» для удаления.

**Non-Goals:**
- Порт реальных office/route/cargo/package-content — срез-стабы (TODO).
- Полный easyp-контракт с реальными полями RPC (остаётся `Empty→Empty`,
  TODO LASTMILE-008).
- Портал/экспорт/оптимизация маршрута — вне среза.

## Решения

### D1. Переиспользуем объявленные интерфейсы, пишем адаптеры
Не меняем `ShowcaseStore`/`BasketItemsStore` — реализуем их адаптерами с
уникальными именами (`basketpg`, `showcasepg`). `basketpg` — реальный SQL по
`lmt.basket` (pgxpool + squirrel/scany, как `shipping/infrastructure`);
`showcasepg` — read поверх детерминированных стабов внешних сервисов
(аналог `eventdeps`).

### D2. Витрина — детерминированные данные в срезе
Поскольку office/route/cargo не портированы, `showcasepg` собирает
`ShowcasePoint`/`ShowcaseRoute` из slice-стабов `OfficeProvider`/
`RouteProvider`/`CargoProvider` (`TODO LASTMILE-*`), с пагинацией на уровне
`primitives.Page`. Это сохраняет реальные точки замены при порте клиентов.

### D3. Корзина — локальная `lmt.basket`
`basketpg` работает с `lmt.basket` напрямую: `ListItemsByDriverID` (join по
названию маршрута — через стаб `RouteProvider`), `AddItem` (INSERT ... ON
CONFLICT DO NOTHING — идемпотентно), `DeleteItem` (по `driver_id,item_id`,
чужая/отсутствующая → `NotFound`), `ClearBasket`. В одной транзакции не нуждается
(одна таблица).

### D4. Empty→Empty вход через context
RPC-хендлеры берут вход из gRPC-metadata/context (как в reserve/tsd):
`driver_id` из контекста (интерцептор), `point_id`/`route_id`/`item_id`/`page` —
из метаданных (`TODO LASTMILE-008`). Пагинация — defaults + валидация
(`InvalidArgument`).

### D5. Wiring и Application
Добавляем в `Application`/`Config` зависимости `showcaseStore`/`basketStore` и
регистрируем 6 хендлеров в `transport/grpc/server.go`; сборка в
`register_driver.go`.

### D6. Ошибки — `internal/errors`
Коды `showcase_*`/`basket_*`/`page_*` в `errors.yaml` (`namespace: lmt`),
маппинг `InvalidArgument`/`NotFound` через `MapToGRPC` (`// TODO(LASTMILE-014)`).

## Risks / Trade-offs

- **Витрина на стабах** → данные/пагинация могут отличаться от прода.
  → Mitigation: срез-стабы + `TODO`; e2e на детерминированных данных.
- **Empty→Empty** → реальные поля скрыты.
  → Mitigation: вход через context/metadata (`TODO LASTMILE-008`).
- **Идемпотентность `AddItem`** — `ON CONFLICT DO NOTHING` гарантирует отсутствие
  дублей, но не обновляет `updated_at`. → Mitigation: принимать как есть; при
  необходимости — `ON CONFLICT DO UPDATE`.
- **Удаление «чужого» элемента** — сравнение по `driver_id` в WHERE.

## План миграции

- Новых миграций нет. Откат: снятие хендлеров/адаптеров; `lmt.basket` остаётся.
- Внешний Driver API: реализуются 6 RPC (контракты `Empty→Empty` упрощены).

## Открытые вопросы

- Источник и формат «количества грузомест» для маршрута (`CountPackages`) —
  уточнить по cargo/package-content при порте; на срез — стаб.
- Пагинация (cursor vs page) — реальный API использует cursor;
  срез — `primitives.Page`/`PaginationResponse` (курсорная, уже в `primitives`).