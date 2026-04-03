package lok

import (
	"errors"
	"testing"
)

func TestInit_EmptyPath(t *testing.T) {
	_, err := Init("")
	if err == nil {
		t.Fatal("expected error for empty path")
	}

	if !errors.Is(err, ErrInitFailed) {
		t.Fatalf("expected ErrInitFailed, got: %v", err)
	}
}

func TestClose_Idempotent(t *testing.T) {
	o := &Office{closed: true}

	// Calling Close on an already-closed office must not panic.
	o.Close()
	o.Close()
}

func TestIsClosed(t *testing.T) {
	o := &Office{}

	if o.IsClosed() {
		t.Fatal("expected IsClosed to return false for new office")
	}

	o.closed = true

	if !o.IsClosed() {
		t.Fatal("expected IsClosed to return true after marking closed")
	}
}

func TestLoadDocument_AfterClose(t *testing.T) {
	o := &Office{closed: true}

	_, err := o.LoadDocument("test.docx")
	if err == nil {
		t.Fatal("expected error when loading document on closed office")
	}

	if !errors.Is(err, ErrOfficeDestroyed) {
		t.Fatalf("expected ErrOfficeDestroyed, got: %v", err)
	}
}
