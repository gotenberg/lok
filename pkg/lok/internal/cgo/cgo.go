package cgo

/*
#cgo CFLAGS: -DLOK_USE_UNSTABLE_API
#cgo LDFLAGS: -ldl
#include "lokbridge.h"
*/
import "C"

import (
	"errors"
	"unsafe"
)

// Office wraps a C LoKit handle.
type Office struct {
	handle *C.LoKit
}

// Document wraps a C LoKitDocument handle with a back-reference to the
// originating [Office] for error retrieval.
type Document struct {
	handle *C.LoKitDocument
	office *Office
}

// Init loads LibreOffice from the given install path and returns an [Office].
func Init(installPath string) (*Office, error) {
	cPath := C.CString(installPath)
	defer C.free(unsafe.Pointer(cPath))

	handle := C.lok_bridge_init(cPath)
	if handle == nil {
		return nil, errors.New("lok_bridge_init returned nil")
	}

	return &Office{handle: handle}, nil
}

// InitWithUserProfile loads LibreOffice with a custom user profile directory.
func InitWithUserProfile(installPath, profilePath string) (*Office, error) {
	cInstall := C.CString(installPath)
	defer C.free(unsafe.Pointer(cInstall))

	cProfile := C.CString(profilePath)
	defer C.free(unsafe.Pointer(cProfile))

	handle := C.lok_bridge_init_2(cInstall, cProfile)
	if handle == nil {
		return nil, errors.New("lok_bridge_init_2 returned nil")
	}

	return &Office{handle: handle}, nil
}

// Destroy releases the LibreOfficeKit instance.
func (o *Office) Destroy() {
	C.lok_bridge_destroy(o.handle)
}

// GetError retrieves and returns the last error message from LibreOffice.
func (o *Office) GetError() string {
	cErr := C.lok_bridge_get_error(o.handle)
	if cErr == nil {
		return ""
	}

	msg := C.GoString(cErr)
	C.lok_bridge_free_error(o.handle, cErr)

	return msg
}

// GetVersionInfo returns LibreOffice version information as a JSON string.
func (o *Office) GetVersionInfo() string {
	cInfo := C.lok_bridge_get_version_info(o.handle)
	if cInfo == nil {
		return ""
	}

	info := C.GoString(cInfo)
	C.lok_bridge_free_error(o.handle, cInfo)

	return info
}

// GetFilterTypes returns the available document filter types as a JSON string.
func (o *Office) GetFilterTypes() string {
	cTypes := C.lok_bridge_get_filter_types(o.handle)
	if cTypes == nil {
		return ""
	}

	types := C.GoString(cTypes)
	C.lok_bridge_free_error(o.handle, cTypes)

	return types
}

// HasTrimMemory reports whether the loaded LibreOffice version supports
// the trimMemory API (available since LibreOffice 7.6).
func (o *Office) HasTrimMemory() bool {
	return C.lok_bridge_has_trim_memory(o.handle) != 0
}

// TrimMemory asks LibreOffice to release cached memory. The target parameter
// controls aggressiveness: 0 for gentle, 2000 for aggressive.
func (o *Office) TrimMemory(target int) {
	C.lok_bridge_trim_memory(o.handle, C.int(target))
}

// LoadDocument opens a document at the given file path.
func (o *Office) LoadDocument(path string) (*Document, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	handle := C.lok_bridge_document_load(o.handle, cPath)
	if handle == nil {
		return nil, errors.New(o.GetError())
	}

	return &Document{handle: handle, office: o}, nil
}

// LoadDocumentWithOptions opens a document with additional load options.
func (o *Office) LoadDocumentWithOptions(path, options string) (*Document, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	cOptions := C.CString(options)
	defer C.free(unsafe.Pointer(cOptions))

	handle := C.lok_bridge_document_load_with_options(o.handle, cPath, cOptions)
	if handle == nil {
		return nil, errors.New(o.GetError())
	}

	return &Document{handle: handle, office: o}, nil
}

// Destroy releases the document handle.
func (d *Document) Destroy() {
	C.lok_bridge_document_destroy(d.handle)
}

// SaveAs exports the document to the given path in the specified format.
// LOK saveAs returns 0 on failure, unlike typical C convention.
func (d *Document) SaveAs(path, format, filterOptions string) error {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	cFormat := C.CString(format)
	defer C.free(unsafe.Pointer(cFormat))

	cFilter := C.CString(filterOptions)
	defer C.free(unsafe.Pointer(cFilter))

	result := C.lok_bridge_document_save_as(d.handle, cPath, cFormat, cFilter)
	if result == 0 {
		return errors.New(d.office.GetError())
	}

	return nil
}

// GetType returns the document type as an integer matching LOK's
// LibreOfficeKitDocumentType enum.
func (d *Document) GetType() int {
	return int(C.lok_bridge_document_get_type(d.handle))
}

// PostUnoCommand sends a UNO command to the document.
func (d *Document) PostUnoCommand(command, arguments string, notify bool) {
	cCommand := C.CString(command)
	defer C.free(unsafe.Pointer(cCommand))

	cArgs := C.CString(arguments)
	defer C.free(unsafe.Pointer(cArgs))

	var cNotify C.int
	if notify {
		cNotify = 1
	}

	C.lok_bridge_document_post_uno_command(d.handle, cCommand, cArgs, cNotify)
}
