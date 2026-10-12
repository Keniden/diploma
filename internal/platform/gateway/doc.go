// Package gateway will host the HTTP/gRPC-gateway of the public driver API.
//
// The skeleton serves gRPC directly (internal/platform/grpc) and does not
// port grpc-gateway: the mobile client speaks gRPC, and the OpenAPI
// generation pipeline (easyp/openapi) is out of scope without BSR access.
//
// TODO(LASTMILE-109): port the real gateway (grpc-gateway mux, swagger
// handlers, websocket transport) and wire it into the service bootstrap.
package gateway

// Doc placeholder: the package intentionally has no exported API yet.