# Дизайн: создание перевозки (ARCHITECTURE.md §23 поток 3)

> Мотивация — см. `proposal.md`. Поток опирается на уже существующий каркас:
> assembler/progress/demand-сервисы, политика `ShipmentPolicy`, репозитории
> агрегатов, sync-bus с обработчиком cargo-shipment — всё уже заведено в
> `register_driver.go` (см. `CreateDriver`).

## Контекст

- `internal/application/public_driver/{create_shipment,
  create_shipment_with_transport,assign_transport}.go` — есть каркасы use-case
  (вызовы assembler/progress/demand, `WithTx`, `cancelCreatedShipment`).
- `internal/shipping/application/{assembler,progress,demand}` — есть сборки,
  политика `CanStartShipmentByID`/`CanCompleteShipmentByID` реализованы,
  остальные политики — `NOT_IMPLEMENTED`.
- Домен: `Shipment` (переходы `NEW→ASSIGNED→RESERVE_LOADING→LOADING→IN_PROGRESS→...`,
  `Cancel`), `Route` (`NewRouteFromTemplate` ≥2 точек), `RoutePoint`, `Package`.
- Sync-bus подписывает `ShipmentCreated`/`ShipmentDriverAssign`/
  `ShipmentTransportAssign` на `CargoShipment`-адаптер (сейчас лог-заглушка
  `infrastructure.NewCargoShipment()`, `TODO(LASTMILE-112)`).
- Внешние клиенты (route, office, cargo-shipment) не портированы —
  заменены `RouteTemplatesProvider`/`DemandProvider`-заглушками (`TODO`).

## Goals / Non-Goals

**Goals:**
- Довести `CreateShipment`/`CreateShipmentWithTransport`/`AssignTransport` до
  рабочего состояния внутри одной транзакции (атомарность + отмена при сбое).
- Засев пакетов и `package_route_point` по спросу (прямой + возвратный поток).
- Синхронизация с cargo-shipment через доменные события.
- Валидация (активная перевозка, шаблон ≥2 точек, непустой госномер) + маппинг.

**Non-Goals:**
- Порт реальных внешних клиентов (route/office/cargo-shipment/vehicles-core) — TODO.
- Оптимизация маршрута, резерв-ворот, погрузка/разгрузка (уже в других срезах).
- Валидация carrier-scope и portal-сценарии.

## Решения

### D1. Переиспользуем существующие сервисы, не пишем новые
Создание уже разложено на `Assembler` (сборка из шаблона),
`ShipmentProgressService` (AssignDriver/AssignTransport/Cancel),
`DemandService` (PlanPackagesToShipmentByID) и `ShipmentPolicy`. Задача —
довести реализации до зелёных тестов и достроить недостающие политики
(`CanCreateShipment`, `CanCreateShipmentByDriverID`), а не вводить новую
оркестрацию.

### D2. Атомарность: одна транзакция + отмена при сбое
`CreateShipmentWithTransport` уже оборачивается в `a.tx.WithTx` и при ошибке
вызывает `cancelCreatedShipment` → `CancelShipmentByID`. Закрепляем это
поведение тестом (e2e): частичный сбой не оставляет активную перевозку.
`CreateShipment` (без транспорта) — тот же поток, но без назначения ТС.

### D3. Внешние сервисы — через заглушки с узкими интерфейсами
`RouteTemplatesProvider` (маршрутный шаблон), `DemandProvider` (спрос) и
`CargoShipment` (создание/SetTransport) остаются slice-скоповыми стабами
(детерминированные данные, как `eventdeps`), но с фиксированными портами —
при порте реального клиента замена только в адаптере(`TODO(LASTMILE-110/111/112)`).

### D4. События → cargo-shipment (уже настроено)
Не строим новый механизм: переходы `Shipment` публикуют
`ShipmentCreated`/`ShipmentDriverAssign`/`ShipmentTransportAssign`, sync-bus
доставляет их в `CargoShipment.CreateShipment`/`SetTransport`. Значит, внешняя
синхронизация «бесплатна» после доведения агрегата.

### D5. Политика и валидация
`CanCreateShipment` (есть пакеты на source) и `CanCreateShipmentByDriverID`
(нет активных перевозок) реализуются в `ShipmentPolicy` через репозиторий;
нарушение → `FailedPrecondition`. Невалидный шаблон → `NotFound`; `>=2` точек —
`InvalidArgument`. Пустой госномер — `InvalidArgument`.

### D6. Ошибки — `internal/errors`
Коды `shipment_*`/`license_plate_*` в `errors.yaml` (`namespace: lmt`),
маппинг в gRPC — через существующий `MapToGRPC` (`// TODO(LASTMILE-014): errgen`).

## Risks / Trade-offs

- **Заглушки вместо реальных клиентов** → фактическая маршрутизация/спрос могут
  отличаться от продакшена. → Mitigation: узкие порты + `TODO`; e2e на мок-данных.
- **Шаблон ≥2 точек** — граничные случаи (вырожденные шаблоны).
  → Mitigation: table-driven тесты фабрики `NewRouteFromTemplate`/`RebuildSequence`.
- **Отмена при сбое** может конфликтовать с уже опубликованными событиями в
  транзакции. → Mitigation: `cancelCreatedShipment` публикует `ShipmentCanceled`
  в той же транзакции; тест на идемпотентность отката.

## План миграции

- Без разрыва: внешний Driver API контракты не меняются (реализации добавляются).
- Новых миграций нет. Откат — удаление реализаций/подписок; домен и таблицы
  остаются, перевозки не создаются.

## Открытые вопросы

- Точная структура маршрутного шаблона (поля `RouteTemplate`) — уточнить по
  реальному route-service при порте; на срез — детерминированный стаб.
- Источник `supplier_id`/`carrier_id` при назначении водителя (пока из
  контекста водителя) — уточнить позже.