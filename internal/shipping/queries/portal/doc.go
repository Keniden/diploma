// Package portal holds the portal-facing shipment queries of the real
// repository (bypackage, card, filters, list, load, packages, queryparse,
// route_points, sublists, tare, users, ...).
//
// TODO(LASTMILE-034): the portal query packages of the real repository are
// not ported yet. They depend on internal/platform/database,
// internal/{app,config,primitives,...} and generated proto clients that are
// not available in the skeleton. The directory exists only to keep the
// porting layout aligned with the reference repository; add packages here
// as their dependencies land.
package portal
