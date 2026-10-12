# proto

Внешние proto-контракты генерируются через **easyp** (см. `easyp.yaml` в корне
репозитория). Этот каталог содержит конфиги `easyp.<service>.yaml` для каждого
внешнего сервиса:

- `easyp.cargo-core.yaml`, `easyp.cargo-documents.yaml`, `easyp.cargo-shipment.yaml`
- `easyp.credeber.yaml`, `easyp.internationalization.yaml`, `easyp.office.yaml`
- `easyp.package-content.yaml`, `easyp.payments.yaml`, `easyp.reserve.yaml`
- `easyp.route.yaml`, `easyp.tsp.yaml`, `easyp.users.yaml`

Сгенерированный код попадает в `internal/pb/` (внешние генерируемые пакеты),
собственные контракты — в `api/proto/` и `pkg/gen/`.

TODO(LASTMILE-079): внутренний easyp недоступен в рамках skeleton-работы —
конфиги будут добавлены/выверены отдельным изменением; генерация клиентов
и `internal/pb` пока не запускается.