# Почему

Путь водителя реализован от создания до завершения перевозки, но **входной
флоу** (ARCHITECTURE.md §23 потоки 1–2) отсутствует: водитель не видит
витрину точек/маршрутов (`ListPoints`, `ListRoutesByPoint`) и не может собрать
корзину (`AddItem`/`DeleteItem`/`ClearBasket`/`ListItems`). Интерфейсы
`ShowcaseStore`/`BasketItemsStore` уже объявлены в
`shipping/queries/driver/{showcase,basket}`, но нет их адаптеров и use-case, а
RPC объявлены в `service.proto` как `Empty→Empty` и возвращают `Unimplemented`.
Без этого водитель не может начать сборку корзины для создания перевозки.

# What Changes

- **Персистентность**: адаптер `basketpg` поверх `lmt.basket`
  (`List/Add/Delete/Clear` по `driver_id`), реализующий
  `BasketItemsStore`; адаптер `showcasepg` (+ slice-стабы внешних office/route/
  cargo/package-content) реалующий `ShowcaseStore` (`ListPoints`,
  `ListRoutesByPoint` с пагинацией `primitives.Page`).
- **Use-case** (`internal/application/public_driver/`): `ListPoints`,
  `ListRoutesByPoint`, `ListItems`, `AddItem`, `DeleteItem`, `ClearBasket` —
  валидация/пагинация, передача `driver_id` из контекста.
- **Transport**: гRPC-хендлеры для этих RPC (Empty→Empty через
  gRPC-metadata/context, `TODO(LASTMILE-008)`), маппинг proto↔domain.
- **Wiring**: `register_driver.go` — подключение `showcasepg`/`basketpg`
  адаптеров и хендлеров.
- **Ошибки** в `internal/errors` (`InvalidArgument`/`NotFound` для пустых
  витрин/ошибок добавления).
- **Заглушки внешних сервисов** (office/route/cargo/package-content) остаются —
  витрина в срезе отдаёт детерминированные данные (`TODO LASTMILE-*`).

# Capabilities

### New Capabilities

- (нет: относится к существующему `last-mile-task`)

### Modified Capabilities

- `last-mile-task`: добавляются требования витрины и корзины водителя —
  `ListPoints`/`ListRoutesByPoint` (showcase) и
  `AddItem`/`DeleteItem`/`ClearBasket`/`ListItems` (basket) с пагинацией,
  валидацией и маппингом ошибок.

# Impact

- Новые файлы: `internal/shipping/queries/driver/showcase/{showcasepg.go}` +
  инфраструктурный адаптер, `internal/shipping/queries/driver/basket/{basketpg.go}`,
  `internal/application/public_driver/{list_points,list_routes_by_point,
  list_items,add_item,delete_item,clear_basket}.go`, `transport/grpc/*` (частично),
  e2e-тест.
- Изменяются: `register_driver.go`, `Application` (новые сервисы/зависимости),
  `internal/platform/application/application.go`.
- Новых миграций нет (`lmt.basket` уже есть). Внешний Driver API: добавляются
  реализации 6 RPC (контракты `Empty→Empty` упрощены, `TODO(LASTMILE-008)`).
- Заглушки office/route/cargo/package-content сохраняются (внешние сервисы не
  портируем — TODO).