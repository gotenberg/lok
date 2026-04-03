// Package cgo provides low-level CGO bindings to the LibreOfficeKit C API.
//
// This package wraps the LibreOfficeKit vtable calls through a C bridge layer,
// providing type-safe Go functions for office lifecycle management, document
// loading, conversion, and error handling.
//
// This package is internal and must not be imported outside of
// [github.com/gotenberg/lok/pkg/lok].
package cgo
