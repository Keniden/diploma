// Package documentspg is the documents persistence adapter on PostgreSQL
// (pgxpool + squirrel/scany), persisting whole Document aggregates into
// lmt.documents and lmt.document_package. The package name is unique
// (documentspg) per the infrastructure adapter naming convention.
package documentspg