// Package public contains the gRPC servers for the public (external) API
// surface: the driver app and the external carrier portal.
//
// The driver server lives in internal/application/public_driver/transport/grpc
// (ported in parallel) and is registered from register_driver.go once that
// package is ready. TODO(LASTMILE-092): register drivergrpc.Server here.
package public