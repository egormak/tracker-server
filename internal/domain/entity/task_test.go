package entity

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

// TestTaskRecord_BSONOmitemptyBoundary ensures optional fields (ID, SourceDay, CreatedAt, SourceDevice)
// are strictly omitted from BSON payloads when unset, preventing document bloat and preserving index sparsity.
func TestTaskRecord_BSONOmitemptyBoundary(t *testing.T) {
	// 1. Regular task without rollover or device metadata
	regularRecord := TaskRecord{
		Name:         "coding",
		Role:         "work",
		TimeDuration: 45,
		Date:         "20 September 2026",
	}

	bsonBytes, err := bson.Marshal(regularRecord)
	if err != nil {
		t.Fatalf("failed to marshal regular record: %v", err)
	}

	var rawDoc bson.M
	if err := bson.Unmarshal(bsonBytes, &rawDoc); err != nil {
		t.Fatalf("failed to unmarshal regular record: %v", err)
	}

	// Mandatory fields must be present
	for _, reqKey := range []string{"name", "role", "time_duration", "date"} {
		if _, ok := rawDoc[reqKey]; !ok {
			t.Errorf("expected required field '%s' in BSON doc, but missing", reqKey)
		}
	}

	// Optional empty fields must be omitted
	for _, optKey := range []string{"_id", "source_day", "created_at", "source_device"} {
		if _, ok := rawDoc[optKey]; ok {
			t.Errorf("expected optional field '%s' to be omitted via omitempty, but found in BSON doc", optKey)
		}
	}

	// 2. Rollover task with full metadata
	now := time.Now().UTC().Truncate(time.Millisecond)
	rolloverRecord := TaskRecord{
		ID:           "obj_id_456",
		Name:         "reading",
		Role:         "learn",
		TimeDuration: 20,
		Date:         "20 September 2026",
		SourceDay:    "monday",
		CreatedAt:    now,
		SourceDevice: "desktop",
	}

	fullBytes, err := bson.Marshal(rolloverRecord)
	if err != nil {
		t.Fatalf("failed to marshal rollover record: %v", err)
	}

	var decodedRecord TaskRecord
	if err := bson.Unmarshal(fullBytes, &decodedRecord); err != nil {
		t.Fatalf("failed to decode rollover record: %v", err)
	}

	if decodedRecord.SourceDay != "monday" {
		t.Errorf("expected SourceDay 'monday', got '%s'", decodedRecord.SourceDay)
	}
	if !decodedRecord.CreatedAt.Equal(now) {
		t.Errorf("expected CreatedAt %v, got %v", now, decodedRecord.CreatedAt)
	}
	if decodedRecord.SourceDevice != "desktop" {
		t.Errorf("expected SourceDevice 'desktop', got '%s'", decodedRecord.SourceDevice)
	}
}
