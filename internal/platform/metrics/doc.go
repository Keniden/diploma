// Package metrics will host the Prometheus metrics registry and collectors.
//
// The skeleton exposes metrics neither on the observability server nor as
// gRPC stats handlers; the real repository registers request/DB/outbox
// metrics. No metrics deps are added without prior approval (see go.mod).
//
// TODO(LASTMILE-092): port the metrics server + collectors (prometheus
// client, grpc stats handlers, db pool stats) once the deps are approved.
package metrics

// Doc placeholder: the package intentionally has no exported API yet.