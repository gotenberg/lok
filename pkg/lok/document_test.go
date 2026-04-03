package lok

import (
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
