// Package cgo provides low-level CGO bindings to the LibreOfficeKit C API.
//
// This package wraps the LibreOfficeKit vtable calls through a C bridge layer,
// providing type-safe Go functions for office lifecycle management, document
// loading, conversion, and error handling.
//
// The trimMemory API exists only in LibreOffice 7.6+. The C bridge guards it
// behind the LOK_HAS_TRIM_MEMORY macro because older headers do not declare the
// struct member. Build with CGO_CFLAGS=-DLOK_HAS_TRIM_MEMORY against 7.6+
// headers to enable it; otherwise [Office.HasTrimMemory] reports false and
// [Office.TrimMemory] is a no-op.
//
// This package is internal and must not be imported outside of
// [github.com/gotenberg/lok/pkg/lok].
package cgo
