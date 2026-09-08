package earchiva

import (
	"errors"
	"testing"
)

func TestRegistraturaArchiveOutboxIdempotencyKeyIsStableAndScoped(t *testing.T) {
	const documentID = "e3324845-b7a2-4f5b-a0fa-83e504ce4bd0"
	first := registraturaArchiveIdempotencyKey(documentID)
	second := registraturaArchiveIdempotencyKey(documentID)
	if first != "registratura-finalized:"+documentID || first != second {
		t.Fatalf("unexpected idempotency key: %q / %q", first, second)
	}
}

func TestRegistraturaArchiveOutboxPermanentErrorsNeverRetry(t *testing.T) {
	err := permanentRegistraturaArchiveError("no clean PDF")
	var permanent registraturaArchivePermanentError
	if !errors.As(err, &permanent) {
		t.Fatalf("permanent delivery error must preserve its classification: %v", err)
	}
	if permanent.Error() != "no clean PDF" {
		t.Fatalf("permanent delivery error = %q", permanent.Error())
	}
}

func TestRegistraturaArchiveOutboxDeterministicArchiveID(t *testing.T) {
	// The worker's UUIDv5 input includes institution ID, avoiding collisions if
	// two tenants use the same document UUID in independent databases/imports.
	first := uuidForRegistraturaArchive("inst-a", "document-a")
	if first != uuidForRegistraturaArchive("inst-a", "document-a") {
		t.Fatal("same tenant/document must derive the same archive identifier")
	}
	if first == uuidForRegistraturaArchive("inst-b", "document-a") {
		t.Fatal("different tenants must not share archive object identity")
	}
}
