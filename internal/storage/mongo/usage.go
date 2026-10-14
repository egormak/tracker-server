package mongo

import (
	"errors"
	"tracker-server/internal/domain/entity"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	usageCountersCollection  = "usage_counters"
	digestSnapshotCollection = "digest_snapshot"
	digestSnapshotDocID      = "last"
)

// IncUsageCounter adds one to the (week, event, client) counter, creating it on first use.
func (s *Storage) IncUsageCounter(week, event, client string) error {
	coll := s.Client.Database(dbName).Collection(usageCountersCollection)
	_, err := coll.UpdateOne(
		s.Context,
		bson.M{"week": week, "event": event, "client": client},
		bson.M{"$inc": bson.M{"count": 1}},
		options.Update().SetUpsert(true),
	)
	return err
}

func (s *Storage) GetUsageCounters(week string) ([]entity.UsageCounter, error) {
	coll := s.Client.Database(dbName).Collection(usageCountersCollection)
	cursor, err := coll.Find(s.Context, bson.M{"week": week})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(s.Context)

	counters := []entity.UsageCounter{}
	if err := cursor.All(s.Context, &counters); err != nil {
		return nil, err
	}
	return counters, nil
}

// SaveDigestSnapshot replaces the single stored digest snapshot.
func (s *Storage) SaveDigestSnapshot(snapshot entity.DigestSnapshot) error {
	coll := s.Client.Database(dbName).Collection(digestSnapshotCollection)
	_, err := coll.ReplaceOne(
		s.Context,
		bson.M{"_id": digestSnapshotDocID},
		snapshot,
		options.Replace().SetUpsert(true),
	)
	return err
}

// GetDigestSnapshot returns the stored snapshot; found is false when there is none.
func (s *Storage) GetDigestSnapshot() (snapshot entity.DigestSnapshot, found bool, err error) {
	coll := s.Client.Database(dbName).Collection(digestSnapshotCollection)
	err = coll.FindOne(s.Context, bson.M{"_id": digestSnapshotDocID}).Decode(&snapshot)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return entity.DigestSnapshot{}, false, nil
	}
	if err != nil {
		return entity.DigestSnapshot{}, false, err
	}
	return snapshot, true, nil
}
