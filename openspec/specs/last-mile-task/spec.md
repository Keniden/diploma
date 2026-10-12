# Спецификация: каркас по реальной архитектуре

## Purpose

Определяет поведение и контракты каркаса сервиса last-mile-task: Driver-API,
вертикальный срез SetPackagesStatus с outbox-доставкой и WMS-экспортом,
CQRS-чтение активной перевозки и структуру по архитектуре §4.

## Requirements

### Requirement: Драйверский API (Driver, proto.v1.public.driver)

Система должна предоставлять gRPC-сервис `Driver` (пакет
`proto.v1.public.driver`) с RPC `SetPackagesStatus`,
`GetActiveShipmentByDriverID`, `CreateShipment`,
`CreateShipmentWithTransport`, `UpdateShipmentTransport` по контрактам
реального репозитория; прочие RPC реального сервиса объявляются в
`service.proto`, но возвращают `Unimplemented` (TODO(LASTMILE-008)).

#### Scenario: Контракт 1:1 с реальным репо

- `SetPackagesStatusRequest`: `shipment_id` (uuid), `packages[]` с полями
  `id` (uuid), `point_id` (uuid), `action` (`UNSPECIFIED`/`LOAD`/`UNLOAD`,
  как в реальном прото; доменное `RESERVE` сюда не входит), `status`,
  `timestamp`, `tare_sticker`.
- Ответ: `success`, `data.failed_packages[]{id, error_description}`,
  `message`.

#### Scenario: Невалидный запрос отклоняется

- Невалидный `shipment_id`/`id`/`point_id` или пустой `packages` →
  `InvalidArgument`; состояние не изменяется.

### Requirement: Перевод пакетов по действию (SetPackagesStatus)

Система должна обновлять статус пакетов и сканы точек в одной транзакции,
публиковать доменные события и писать outbox-сообщения cargo и wms;
перевод каждого пакета применяется независимо: пакет, который не удалось
обработать, попадает в `failed_packages`, остальные применяются.

#### Scenario: Разрешённый переход (LOAD на точке)

- `package.status` и сканы `package_route_point` обновляются в одной
  транзакции; после успеха outbox получает `cargo.packages.scanned` /
  `cargo.packages.status.updated` и `wms.tares.load`; ответ `success=true`
  с пустым `failed_packages`.

#### Scenario: Частичный успех

- Недопустимый переход статуса пакета или ошибка обработки скана не
  откатывают весь запрос: пакет, который не удалось отсканировать,
  попадает в `failed_packages[]{id, error_description}` с `success=true`,
  остальные пакеты применяются (семантика реального репозитория).
- Недопустимый запрос в целом (невалидный `shipment_id`/`id`/`point_id`
  или пустой `packages`) отклоняется с `InvalidArgument`; состояние не
  изменяется.

### Requirement: Outbox (at-least-once)

Система должна записывать события в `lmt.outbox` в той же транзакции, что
и изменения агрегата, и доставлять их с повторами до успеха. Схема таблицы —
реальная: `id UUID PK`, `type`, `payload BYTEA`, `metadata JSONB`, `attempt`,
`scheduled_at`, `status ('TODO')`, `err`; повторы — по `id`/`status`/`attempt`.

#### Scenario: Запись только вместе с агрегатом

- Событие появляется в `lmt.outbox` только если транзакция агрегата
  зафиксирована.

#### Scenario: Повторная доставка при ошибке

- Ошибка доставки в WMS оставляет событие в `status='TODO'` с учётом
  `attempt`; следующий цикл доставляет снова; идемпотентность по `id`
  (дублей нет).

### Requirement: WMS-экспорт (HTTP)

Система должна доставлять события в WMS HTTP-клиентом (`pkg/wms`):
`POST /load` для LOAD, `POST /unload` для UNLOAD; при `--mock` — через
встроенный мок (`pkg/wms/wmsmock`).

#### Scenario: Доставка POST /load, /unload

- Успешный ответ WMS → событие помечается доставленным; поля события
  корректны.

#### Scenario: Mock-режим автономен

- При `--mock` сервис работает без внешних систем (встроенный WMS-мок).

### Requirement: Чтение активной перевозки (CQRS)

Система должна отдавать активную перевозку водителя через отдельные
read-репозитории (`shipping/queries/driver/shipment`) поверх тех же
таблиц (логический CQRS без event-sourcing).

#### Scenario: GetActiveShipmentByDriverID

- Возвращает актуальное состояние перевозки после среза; при отсутствии
  активной перевозки — корректное пустое состояние без ошибки.

### Requirement: Обработка ошибок

Система должна маппить ошибки в gRPC-коды через `internal/errors`
(`GRPCStatus`/`FromProto`/`MapToGRPC`, пул кодов в `errors.yaml`).

#### Scenario: Маппинг кодов

- Валидация запроса → `InvalidArgument`; нераспознанная ошибка → `Internal`;
  доменные ошибки несут стабильный код и маппятся в соответствующие
  gRPC-коды (в т.ч. `NotFound`, `FailedPrecondition` для флоу вне среза
  SetPackagesStatus, где применяется семантика частичного успеха).

### Requirement: Структура и запуск по ARCHITECTURE.md

Система должна повторять дерево каталогов §4 (нереализованные модули —
заглушки), стартовать через `main.go` → `internal/app` → `builder.New` →
`application.New`, конфигурироваться через envconfig, gracefully
завершаться и логировать структурированно через `log/slog` (JSON);
миграции — goose с таблицами
`lmt.*`.

#### Scenario: Дерево §4 и bootstrap

- Все каталоги §4 существуют; заглушки не ломают `go build ./...`;
  запуск/остановка сервиса корректны.

#### Scenario: Миграции

- `migrations/` (goose, версионные миграции): `lmt.shipment`,
  `lmt.route`, `lmt.route_point`, `lmt.package`, `lmt.package_route_point`,
  `lmt.reserve`, `lmt.outbox`; `task migrate` применяет их (goose up).

### Requirement: Домен документа (Document)

Система SHALL строить агрегат `Document` для накладной перевозки (Waybill:
вперёд `Forward=1` / возврат `Return=2`) из данных перевозки и маршрута:
собирать грузоместа по парам точек `Load→Unload` (направления), формировать
строки товаров (`ContentItem/Good/Price/Quantity`), метаданные (`Meta`:
`Driver`, тип ТС, `OfficePoint`) и набор `PackageIDs`, связывающий документ с
грузоместами; документ собирается фабрикой `NewDocument` и хранится в
`lmt.documents` + `lmt.document_package`.

#### Scenario: Сборка документа по парам load/unload

- **WHEN** перевозка стартует или водитель убывает с промежуточной точки
- **THEN** система группирует грузоместа по направлению `SrcPoint→DstPoint`,
  строит строки товаров и создаёт `Document` с корректным `DocumentType`
  (`Forward` для прямого потока, `Return` для возвратного) и корректным
  набором `PackageIDs`.

#### Scenario: Документ сохраняется с грузоместами

- **WHEN** документ собран
- **THEN** запись о документе появляется в `lmt.documents`, а связи с
  грузоместами — в `lmt.document_package` (в той же транзакции).

### Requirement: Публикация outbox-сообщения документов

Система SHALL записывать в `lmt.outbox` сообщение типа
`documents.publish_after_start_shipment` в **той же транзакции**, что и смена
состояния перевозки/точки, и доставлять его в cargo-documents
(`CreateEventByType`) с семантикой **at-least-once**.

#### Scenario: Запись сообщения вместе с событием

- **WHEN** перевозка переходит в `IN_PROGRESS` (старт) или точка завершается
  (убытие водителя)
- **THEN** outbox-сообщение `documents.publish_after_start_shipment`
  появляется в `lmt.outbox` только если транзакция агрегата зафиксирована.

#### Scenario: Доставка в cargo-documents и повтор при ошибке

- **WHEN** воркер доставляет сообщение
- **THEN** успешный ответ cargo-documents помечает сообщение доставленным;
  ошибка оставляет его в `status='TODO'` с инкрементом `attempt`, и следующий
  цикл доставляет снова; идемпотентность по идентификатору события (дублей нет).

#### Scenario: Возвратный документ публикуется условно

- **WHEN** убытие с промежуточной точки происходит и на точке есть загруженные
  возвратные пакеты
- **THEN** система публикует `documents.publish_after_start_shipment`;
  **WHEN** возвратных пакетов на точке нет — сообщение не публикуется
  (`ensureReturnDocsScheduled`).

### Requirement: Триггеры генерации документов

Система SHALL инициировать генерацию/публикацию документа на события
`ShipmentStartedEvent` (старт перевозки) и `PointCompletedEvent` (убытие с
промежуточной точки), используя данные перевозки, маршрута и грузомест,
доступные в том же запросе.

#### Scenario: Документ на старте перевозки

- **WHEN** перевозка стартует (`ShipmentStartedEvent`)
- **THEN** система собирает и публикует документ по данным стартовавшей
  перевозки.

#### Scenario: Документ на убытии с промежуточной точки

- **WHEN** точка завершается и путь водителя продолжается
  (`PointCompletedEvent`)
- **THEN** система публикует документ по возвратным грузоместам точки, если они
  есть.

### Requirement: Обработка ошибок документов

Система SHALL маппить ошибки слоя документов в gRPC-коды через `internal/errors`
и не ломать основной поток перевозки при неверном сборе данных документа.

#### Scenario: Маппинг и изоляция ошибок

- **WHEN** данные документа некорректны или cargo-documents недоступен
- **THEN** ошибка несёт стабильный код и маппится в соответствующий gRPC-код
  (`Internal`/`FailedPrecondition`), а основной доменный поток перевозки не
  откатывается из-за ошибки асинхронной доставки.

### Requirement: Создание перевозки из маршрутного шаблона

Система SHALL создавать перевозку по `CreateShipment`: проверять, что у водителя
нет активной перевозки, собирать `Shipment`/`Route`/`RoutePoint` из маршрутного
шаблона (минимум 2 точки), назначать водителя, планировать спрос и засевать
пакеты/действия в той же транзакции; при сбое любого шага перевозка не остаётся
в частично созданном состоянии.

#### Scenario: Успешное создание перевозки

- **WHEN** водитель вызывает `CreateShipment` с валидным `route_template_id`
- **THEN** создаются `lmt.shipment` (статус `NEW`→`ASSIGNED`), `lmt.route`,
  `lmt.route_point`, пакеты и `lmt.package_route_point`; водитель назначен;
  перевозка синхронизируется с cargo-shipment (`ShipmentCreated`,
  `ShipmentDriverAssign`).

#### Scenario: У водителя уже есть активная перевозка

- **WHEN** водитель создаёт перевозку при наличии активной
- **THEN** система отвечает `FailedPrecondition` и не создаёт новую перевозку.

#### Scenario: Невалидный шаблон/недостаточно точек

- **WHEN** шаблон не найден или содержит менее 2 точек
- **THEN** система отвечает `InvalidArgument`/`NotFound` и не изменяет состояние.

### Requirement: Создание перевозки с транспортом (атомарно)

Система SHALL по `CreateShipmentWithTransport` выполнять создание, назначение
водителя, планирование спроса и назначение транспорта (госномера) внутри одной
транзакции; при ошибке любого шага созданная перевозка отменяется
(`CancelShipmentByID`), и наружу не возвращается «висящий» идентификатор
перевозки.

#### Scenario: Атомарный успех

- **WHEN** `CreateShipmentWithTransport` проходит все шаги
- **THEN** перевозка создана, водитель и ТС назначены, госномер сохранён,
  nil-ошибок нет; идентификатор возвращён.

#### Scenario: Отмена при сбое шага

- **WHEN** назначение транспорта/спроса падает
- **THEN** созданная перевозка отменяется (статус `CANCELED`), ошибка возвращается
  с причиной сбоя.

### Requirement: Простановка/обновление транспортного средства

Система SHALL по `UpdateShipmentTransport`/`AssignTransport` проставлять госномер
(license plate) перевозке, нормализуя ввод и отклоняя пустые значения.

#### Scenario: Установка госномера

- **WHEN** водитель вызывает `UpdateShipmentTransport` с непустым номером
- **THEN** `license_plate` перевозки обновляется и синхронизируется с
  cargo-shipment (`ShipmentTransportAssign`).

#### Scenario: Пустой госномер

- **WHEN** передан пустой/невалидный номер
- **THEN** система отвечает `InvalidArgument` и не изменяет перевозку.

### Requirement: Валидация и обработка ошибок создания

Система SHALL маппить ошибки создания перевозки в gRPC-коды и не портить
ранее созданные данные при частичном сбое.

#### Scenario: Маппинг кодов

- **WHEN** ошибка валидации/невалидного шаблона
- **THEN** код `InvalidArgument`/`NotFound`; **WHEN** есть активная перевозка —
  `FailedPrecondition`; прочее — `Internal`.

### Requirement: Создание резерва ворот (CreateReserve)

Система SHALL по `CreateReserve` создавать резерв ворот для точки маршрута:
проверять политику `CanReserveGate` (резерв только на source-точке назначенной
перевозки, без активного резерва), вызывать `ReserveManager.ReserveOrQueue`,
создавать сессию/QR рукопожатия, сохранять `lmt.reserve` и переводить
`Shipment` в `RESERVE_LOADING` (source) или `RESERVE_UNLOADING` (последняя точка)
в одной транзакции.

#### Scenario: Успешный резерв на source

- **WHEN** водитель вызывает `CreateReserve` для назначенной перевозки на
  source-точке
- **THEN** создаётся `lmt.reserve` (статус `QUEUED`/`WAITING_APPROVAL`),
  `Shipment` переходит в `RESERVE_LOADING`, сгенерирован QR-селф `session`
  (создание/декодирование рукопожатия), возвращается идентификатор резерва.

#### Scenario: Резерв на последней точке

- **WHEN** резерв выполняется на последней точке маршрута
- **THEN** `Shipment` переходит в `RESERVE_UNLOADING`, резерв сохраняется.

#### Scenario: Конфликт — активный резерв или чужая точка

- **WHEN** у перевозки/точки уже есть активный резерв, либо точка не source/не
  назначенная
- **THEN** система отвечает `FailedPrecondition` и не изменяет состояние.

### Requirement: Статус и обновление QR (Get/UpdateReserveQRStatus)

Система SHALL отдавать `is_scanned` (отсканирован ли QR ТСД) по
`GetReserveQRStatus` и обновлять статус по `UpdateReserveQRStatus`.

#### Scenario: GetReserveQRStatus

- **WHEN** водитель запрашивает статус QR резерва
- **THEN** система возвращает, отсканирован ли QR (`is_scanned`), по актуальному
  `lmt.reserve`; при отсутствии резерва — корректное пустое состояние без ошибки.

#### Scenario: UpdateReserveQRStatus

- **WHEN** ТСД или водитель подтверждает/обновляет статус QR
- **THEN** состояние `lmt.reserve` обновляется (при необходимости перевод
  `Shipment` в `LOADING`/`UNLOADING`), статус согласован.

### Requirement: Отмена резерва (CancelReserve)

Система SHALL по `CancelReserve` отменять резерв: `CancelReserveByShipmentID`
с переводом `Shipment` обратно (из `RESERVE_LOADING`/`RESERVE_UNLOADING`), при
допустимом статусе резерва (`QUEUED`/`WAITING_APPROVAL`).

#### Scenario: Успешная отмена

- **WHEN** резерв ещё не подтверждён
- **THEN** резерв получает статус `CANCELED`, перевозка возвращается в
  предыдущее состояние.

#### Scenario: Отмена подтверждённого резерва

- **WHEN** резерв уже активен (`ACTIVE`)
- **THEN** система отвечает ошибкой недопустимого перехода (без изменений).

### Requirement: Протухание резервов по расписанию

Система SHALL по расписанию переводить просроченные резервы
(`ExpireDueWaitingApprovalReserves`, `ExpireDueActiveReserves`) в
`EXPIRED_WAITING_APPROVAL`/`EXPIRED_ACTIVE`.

#### Scenario: Протухание ожидающих/активных

- **WHEN** у резерва истёк `expires_at`
- **THEN** воркер переводит его в соответствующий `EXPIRED_*` статус; уже
  обработанные не трогаются.

### Requirement: Обработка ошибок резервов

Система SHALL маппить ошибки резерва в gRPC-коды через `internal/errors`.

#### Scenario: Маппинг

- **WHEN** недопустимый переход/конфликт резерва
- **THEN** код `FailedPrecondition`; **WHEN** нет перевозки/точки — `NotFound`;
  невалидный вход — `InvalidArgument`; прочее — `Internal`.

### Requirement: Сканирование перевозки (ScanShipment)

Система SHALL по `ScanShipment` подтверждать резерв ворот: декодировать QR,
вызывать `ApproveReserveByRoutePointID` (резерв `ACTIVE`, `CalcScanAt`,
`Shipment` в `LOADING`/`UNLOADING`) и завершать подтверждение.

#### Scenario: Успешное сканирование перевозки

- **WHEN** ТСД сканирует QR перевозки на точке
- **THEN** резерв подтверждается (`ACTIVE`, `scanned_at` установлен), `Shipment`
  переходит в `LOADING`/`UNLOADING`, точка — в соответствующий статус; ошибок нет.

#### Scenario: Невалидный/непроверяемый QR

- **WHEN** QR не распознан или резерв не найден/недопустим
- **THEN** система отвечает `InvalidArgument`/`FailedPrecondition` и не меняет
  состояние.

### Requirement: Сканирование пакетов (ScanPackages)

Система SHALL по `ScanPackages` сканировать и проверять пакеты на точке через
`PackageScanner`: при допустимом действии обновлять статус пакета и записи
`package_route_point`, публиковать `PackageScannedEvent` (→ outbox cargo+wms);
недопустимые пакеты попадают в `failed_packages`, остальные применяются.

#### Scenario: Успешное сканирование пакетов

- **WHEN** ТСД сканирует допустимые пакеты на точке
- **THEN** статус пакетов и сканы обновляются, outbox получает cargo+wms
  (`PackageScannedEvent`).

#### Scenario: Частичный успех скна

- **WHEN** часть пакетов не удалось обработать
- **THEN** применимые пакеты применяются, ошибки попадают в `failed_packages`
  (семантика частичного успеха, как в SetPackagesStatus).

### Requirement: Список пакетов к сканированию (ListPackagesToScan)

Система SHALL по `ListPackagesToScan` возвращать пакеты, ожидающие сканирования
на указанной точке.

#### Scenario: Пустой список на точке без ожидающих

- **WHEN** на точке нет пакетов к сканированию
- **THEN** система возвращает корректный пустой список без ошибки.

#### Scenario: Список с ожидающими пакетами

- **WHEN** на точке есть пакеты `READY_TO_LOAD`/`IN_TRANSIT`, требующие скана
- **THEN** система возвращает их список (без протечки proto-типов в домен).

### Requirement: Обработка ошибок ТСД

Система SHALL маппить ошибки ТСД в gRPC-коды через `internal/errors`.

#### Scenario: Маппинг

- **WHEN** невалидный QR/скан
- **THEN** код `InvalidArgument`; **WHEN** недопустимое действие точки/резерва —
  `FailedPrecondition`; **WHEN** нет данных — `NotFound`; прочее — `Internal`.

### Requirement: Витрина точек (ListPoints)

Система SHALL по `ListPoints` возвращать список точек/складов для витрины
(`ShowcasePoint`: id, название, адрес, геолокация) с курсорной пагинацией.

#### Scenario: Список точек с пагинацией

- **WHEN** водитель вызывает `ListPoints` с корректной страницей
- **THEN** система возвращает страницу точек и метаданные пагинации
  (`PaginationResponse`), без протечки proto-типов в домен.

#### Scenario: Пустая витрина

- **WHEN** точек нет или страница пустая
- **THEN** система возвращает корректный пустой список без ошибки.

### Requirement: Витрина маршрутов по точке (ListRoutesByPoint)

Система SHALL по `ListRoutesByPoint` возвращать маршруты для точки
(`ShowcaseRoute`: id, название, точки с порядком и количеством грузомест) с
пагинацией.

#### Scenario: Маршруты для точки

- **WHEN** водитель вызывает `ListRoutesByPoint` с корректной точкой и страницей
- **THEN** система возвращает страницу маршрутов с точками и количеством
  грузомест; невалидная точка → `NotFound`.

### Requirement: Корзина водителя (ListItems)

Система SHALL по `ListItems` возвращать элементы корзины текущего водителя
(`BasketItem`: item_id, route_id, название маршрута, office_id, время создания),
читая `lmt.basket` по `driver_id`.

#### Scenario: Список элементов корзины

- **WHEN** водитель вызывает `ListItems`
- **THEN** система возвращает его элементы корзины (или пусто, если корзина
  пуста — без ошибки).

### Requirement: Добавление элемента в корзину (AddItem)

Система SHALL по `AddItem` добавлять элемент `route_id` в корзину текущего
водителя (`lmt.basket`), идемпотентно по `(driver_id, item_id)`.

#### Scenario: Успешное добавление

- **WHEN** водитель вызывает `AddItem` с корректным `route_id`
- **THEN** элемент появляется в `lmt.basket` (или остаётся, если уже был);
  дубликат не создаётся.

#### Scenario: Невалидный элемент

- **WHEN** `route_id` некорректен/элемент недоступен
- **THEN** система отвечает `InvalidArgument`/`NotFound` и не изменяет корзину.

### Requirement: Удаление и очистка корзины (DeleteItem / ClearBasket)

Система SHALL по `DeleteItem` удалять элемент из корзины водителя, по
`ClearBasket` — очищать всю корзину водителя.

#### Scenario: Удаление элемента

- **WHEN** водитель вызывает `DeleteItem` для элемента своей корзины
- **THEN** элемент удалён из `lmt.basket`; чужой/несуществующий элемент —
  `NotFound` без изменений.

#### Scenario: Очистка корзины

- **WHEN** водитель вызывает `ClearBasket`
- **THEN** все элементы его корзины удалены; пустая корзина — без ошибки.

### Requirement: Обработка ошибок витрины/корзины

Система SHALL маппить ошибки витрины/корзины в gRPC-коды через `internal/errors`.

#### Scenario: Маппинг

- **WHEN** невалидный вход/пагинация
- **THEN** код `InvalidArgument`; **WHEN** данных нет/элемент не найден —
  `NotFound`; прочее — `Internal`.

### Requirement: Завершение точки и перевод перевозки (UpdateRoutePointDriverDeparted)

Система SHALL по `UpdateRoutePointDriverDeparted` завершать точку маршрута по
статусной модели (`COMPLETED`, либо `PACKAGES_NOT_UNLOADED`/
`PACKAGES_NOT_LOADED` при необработанных грузоместах) и, если точка последняя в
маршруте, переводить `Shipment` в `COMPLETED` (`Complete()`), публикуя
`ShipmentCompleted`.

#### Scenario: Завершение промежуточной точки

- **WHEN** водитель убывает с промежуточной точки и все пакеты обработаны
- **THEN** точка получает статус `COMPLETED`; перевозка остаётся активной.

#### Scenario: Завершение последней точки

- **WHEN** водитель убывает с последней точки и все грузоместа обработаны
- **THEN** точка завершается, `Shipment` переходит в `COMPLETED`, публикуется
  `ShipmentCompleted`.

#### Scenario: Необработанные грузоместа на точке

- **WHEN** на завершаемой точке остались не разгруженные/не загруженные пакеты
- **THEN** точка получает `PACKAGES_NOT_UNLOADED`/`PACKAGES_NOT_LOADED`, ошибка
  `FailedPrecondition` с указанием необработанных точек; перевозка не завершается.

### Requirement: Завершение перевозки (EndShipment)

Система SHALL по `EndShipment` завершать перевозку: проверить, что маршрут
пройден и грузоместа обработаны, перевести перевозку в `COMPLETED`, опубликовать
`ShipmentCompleted` и синхронизировать cargo-shipment (trigger `FINISH`) в одной
транзакции.

#### Scenario: Успешное завершение

- **WHEN** водитель вызывает `EndShipment` для пройденной перевозки
- **THEN** `Shipment` в `COMPLETED`, событие `ShipmentCompleted` опубликовано,
  грузоместа без незавершённых действий; cargo-shipment уведомлён (FINISH).

#### Scenario: Перевозка не готова к завершению

- **WHEN** есть незавершённые unload-действия или маршрут не пройден
- **THEN** система отвечает `FailedPrecondition` и не изменяет перевозку.

### Requirement: Начисление за перевозку (outbox payments)

Система SHALL писать в `lmt.outbox` сообщение `payments.last_mile_shipment_accrual`
в **той же транзакции**, что и завершение перевозки, и доставлять его в payments
(`LastMileShipmentAccrual`) с семантикой **at-least-once**.

#### Scenario: Запись вместе с завершением

- **WHEN** перевозка завершается (`ShipmentCompleted`)
- **THEN** outbox-сообщение `payments.last_mile_shipment_accrual` появляется в
  `lmt.outbox` только если транзакция агрегата зафиксирована.

#### Scenario: Доставка и повтор при ошибке

- **WHEN** воркер доставляет сообщение
- **THEN** успешный ответ payments помечает сообщение доставленным; ошибка
  оставляет его в `status='TODO'` с инкрементом `attempt`, следующий цикл
  доставляет снова; идемпотентность по идентификатору события (дублей нет).

### Requirement: Обработка ошибок завершения

Система SHALL маппить ошибки завершения в gRPC-коды через `internal/errors` и не
подвергать откату уже завершённое состояние при сбое асинхронной доставки.

#### Scenario: Маппинг

- **WHEN** перевозка не готова к завершению
- **THEN** код `FailedPrecondition`; **WHEN** перевозка/точка не найдены —
  `NotFound`; невалидный вход — `InvalidArgument`; прочее — `Internal`.