package lok

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestDocument_Close_Idempotent(t *testing.T) {
	d := &Document{closed: true}

	// Calling Close on an already-closed document must not panic.
	d.Close()
	d.Close()
}

func TestDocument_SaveAs_AfterClose(t *testing.T) {
	d := &Document{closed: true}

	err := d.SaveAs("output.pdf", "pdf", "")
	if err == nil {
		t.Fatal("expected error when saving after close")
	}

	if !errors.Is(err, ErrDocumentDestroyed) {
		t.Fatalf("expected ErrDocumentDestroyed, got: %v", err)
	}
}

func TestDocument_PostUnoCommand_AfterClose(t *testing.T) {
	d := &Document{closed: true}

	err := d.PostUnoCommand(".uno:UpdateAll", "")
	if err == nil {
		t.Fatal("expected error when posting UNO command after close")
	}

	if !errors.Is(err, ErrDocumentDestroyed) {
		t.Fatalf("expected ErrDocumentDestroyed, got: %v", err)
	}
}

func TestDocument_SetLandscape_AfterClose(t *testing.T) {
	d := &Document{closed: true}

	err := d.SetLandscape(true)
	if err == nil {
		t.Fatal("expected error when setting landscape after close")
	}

	if !errors.Is(err, ErrDocumentDestroyed) {
		t.Fatalf("expected ErrDocumentDestroyed, got: %v", err)
	}
}

func TestDocument_ExportPDFViaUnoCommand_AfterClose(t *testing.T) {
	d := &Document{closed: true}

	err := d.ExportPDFViaUnoCommand("/tmp/output.pdf", "")
	if err == nil {
		t.Fatal("expected error when exporting after close")
	}

	if !errors.Is(err, ErrDocumentDestroyed) {
		t.Fatalf("expected ErrDocumentDestroyed, got: %v", err)
	}
}

func TestDocument_Type_AfterClose(t *testing.T) {
	d := &Document{closed: true}

	if d.Type().IsValid() {
		t.Fatal("Type() on a closed document must report an invalid type")
	}
}

func TestDocument_SaveAs_AfterOfficeClose(t *testing.T) {
	d := &Document{office: &Office{closed: true}}

	err := d.SaveAs("output.pdf", "pdf", "")
	if !errors.Is(err, ErrOfficeDestroyed) {
		t.Fatalf("expected ErrOfficeDestroyed, got: %v", err)
	}
}

func TestDocument_Type_AfterOfficeClose(t *testing.T) {
	d := &Document{office: &Office{closed: true}}

	if d.Type().IsValid() {
		t.Fatal("Type() must report an invalid type when the office is closed")
	}
}

func TestDocument_Close_AfterOfficeClose(t *testing.T) {
	d := &Document{office: &Office{closed: true}}

	// Closing the office invalidates its documents, so Close must not call the
	// C destroy here. With a nil internal handle this would panic if it did.
	d.Close()

	if !d.IsClosed() {
		t.Fatal("expected document to be marked closed")
	}
}

func TestBuildExportPDFArgs_FileURLAndFilterData(t *testing.T) {
	args, err := buildExportPDFArgs("/tmp/out put.pdf", `{"PageRange":{"type":"string","value":"1-3"}}`)
	if err != nil {
		t.Fatalf("buildExportPDFArgs failed: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(args), &parsed); err != nil {
		t.Fatalf("args is not valid JSON: %v\nraw: %s", err, args)
	}

	urlProp, ok := parsed["URL"].(map[string]any)
	if !ok {
		t.Fatal("missing URL property")
	}

	if want := "file:///tmp/out%20put.pdf"; urlProp["value"] != want {
		t.Errorf("URL value = %v, want %s", urlProp["value"], want)
	}

	fd, ok := parsed["FilterData"].(map[string]any)
	if !ok {
		t.Fatal("missing FilterData property")
	}

	if fd["type"] != "[]com.sun.star.beans.PropertyValue" {
		t.Errorf("FilterData type = %v, want []com.sun.star.beans.PropertyValue", fd["type"])
	}
}

func TestBuildExportPDFArgs_EscapesSpecialChars(t *testing.T) {
	args, err := buildExportPDFArgs(`/tmp/a"b\c.pdf`, "")
	if err != nil {
		t.Fatalf("buildExportPDFArgs failed: %v", err)
	}

	if !json.Valid([]byte(args)) {
		t.Fatalf("args is not valid JSON: %s", args)
	}
}

func TestBuildExportPDFArgs_InvalidFilterOptions(t *testing.T) {
	_, err := buildExportPDFArgs("/tmp/out.pdf", "{not json")
	if err == nil {
		t.Fatal("expected an error for invalid filter options JSON")
	}
}
