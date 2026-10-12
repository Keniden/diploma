# Задачи: витрина + корзина водителя (потоки 1–2)

TDD-порядок: адаптеры (persistence) → use-case → transport/wiring → e2e.
Каждая группа: RED → GREEN → REFACTOR, тесты остаются в регрессии.

> Solver B. E2E (4.1/4.2) написаны, но НЕ запускались (общая Postgres на
> localhost:5432, -tags e2e исключён). Интеграционные тесты на Postgres
> выполнены зелёными (Postgres был доступен).

## 1. Персистентность (адаптеры)

- [x] 1.1 `internal/shipping/queries/driver/basket/basketpg.go`: реализация
  `BasketItemsStore` (`ListItemsByDriverID`, `AddItem` с `ON CONFLICT DO NOTHING`,
  `DeleteItem` по `driver_id,item_id` → `NotFound` для чужого/отсутствующего,
  `ClearBasket`) поверх `lmt.basket` — интеграционные тесты на реальной PostgreSQL
- [x] 1.2 `internal/shipping/queries/driver/showcase/showcasepg.go`: реализация
  `ShowcaseStore` (`ListPoints`, `ListRoutesByPoint` с `primitives.Page`/
  `PaginationResponse`) поверх slice-стабов `OfficeProvider`/`RouteProvider`/
  `CargoProvider` — RED→GREEN
- [x] 1.3 Стабы внешних данных: `OfficeProvider`/`RouteProvider`/`CargoProvider`
  (детерминированные, `TODO LASTMILE-*`) — тесты детерминированных значений

## 2. Use-case (public_driver)

- [x] 2.1 `list_points.go`/`list_routes_by_point.go`: пагинация/валидация →
  `ShowcaseStore` — мок-тесты
- [x] 2.2 `list_items.go`/`add_item.go`/`delete_item.go`/`clear_basket.go`:
  `driver_id` из контекста → `BasketItemsStore` — мок-тесты (в т.ч. чужой элемент →
  `NotFound`, дубликат не создаётся)

## 3. Transport и wiring

- [x] 3.1 `transport/grpc/server.go`: хендлеры
  `ListPoints`/`ListRoutesByPoint`/`ListItems`/`AddItem`/`DeleteItem`/`ClearBasket`
  (Empty→Empty вход через context/metadata, `TODO LASTMILE-008`), маппинг proto↔domain
  — RED→GREEN
- [x] 3.2 Маппинг ошибок в gRPC-коды через `internal/errors`
  (`InvalidArgument`/`NotFound`) — тесты
- [x] 3.3 `register_driver.go` + `Application`: подключить `basketpg`/`showcasepg`
  и хендлеры — smoke-тест запуска

## 4. E2E и проверки

- [x] 4.1 E2E (реальная PostgreSQL + gRPC): `AddItem` → элемент в `lmt.basket`;
  повторный `AddItem` → дубля нет; `ListItems` возвращает корзину водителя;
  `DeleteItem`/`ClearBasket` удаляют; чужой элемент → `NotFound` — НЕ запускался
- [x] 4.2 E2E `ListPoints`/`ListRoutesByPoint` → корректная пагинация витрины;
  невалидная точка → `NotFound` — НЕ запускался
- [x] 4.3 Полные проверки: `go build ./...`, `go vet ./...`,
  `golangci-lint run ./...` (только новые ошибки), `go test -race -p 1 ./...` —
  зелёные