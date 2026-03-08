package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	bolt "go.etcd.io/bbolt"
)

func importLegacyBolt(path string, s *Store) error {
	legacy, err := bolt.Open(path, 0o600, &bolt.Options{
		ReadOnly: true,
		Timeout:  time.Second,
	})
	if err != nil {
		return fmt.Errorf("open legacy workspace: %w", err)
	}
	defer legacy.Close()

	var dump struct {
		Templates []Template
		Audiences []AudienceMember
		Imports   []ImportedMessage
		Campaigns []CampaignSnapshot
		Events    []Event
	}

	if err := legacy.View(func(tx *bolt.Tx) error {
		var readErr error
		if dump.Templates, readErr = readLegacyBucket[Template](tx, "templates"); readErr != nil {
			return readErr
		}
		if dump.Audiences, readErr = readLegacyBucket[AudienceMember](tx, "audiences"); readErr != nil {
			return readErr
		}
		if dump.Imports, readErr = readLegacyBucket[ImportedMessage](tx, "imports"); readErr != nil {
			return readErr
		}
		if dump.Campaigns, readErr = readLegacyBucket[CampaignSnapshot](tx, "campaigns"); readErr != nil {
			return readErr
		}
		if dump.Events, readErr = readLegacyBucket[Event](tx, "events"); readErr != nil {
			return readErr
		}

		return nil
	}); err != nil {
		return fmt.Errorf("read legacy workspace: %w", err)
	}

	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin legacy import: %w", err)
	}
	defer tx.Rollback()

	for i := range dump.Templates {
		if _, err := upsertTemplateTx(tx, dump.Templates[i].Plugin, &dump.Templates[i]); err != nil {
			return fmt.Errorf("import template %q: %w", dump.Templates[i].Slug, err)
		}
	}
	for i := range dump.Audiences {
		if _, err := upsertAudienceMemberTx(tx, dump.Audiences[i].Segment, dump.Audiences[i].Source, &dump.Audiences[i]); err != nil {
			return fmt.Errorf("import audience member %q: %w", dump.Audiences[i].Email, err)
		}
	}
	for i := range dump.Imports {
		if _, err := upsertImportedMessageTx(tx, &dump.Imports[i]); err != nil {
			return fmt.Errorf("import message %q: %w", dump.Imports[i].SourcePath, err)
		}
	}
	for i := range dump.Campaigns {
		if _, err := upsertCampaignSnapshotTx(tx, &dump.Campaigns[i]); err != nil {
			return fmt.Errorf("import campaign snapshot %q: %w", dump.Campaigns[i].SourcePath, err)
		}
	}
	for i := range dump.Events {
		if err := insertEventTx(tx, &dump.Events[i]); err != nil {
			return fmt.Errorf("import event %d: %w", dump.Events[i].ID, err)
		}
	}

	if _, err := tx.Exec(
		`INSERT INTO meta (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		"legacy_imported_from",
		path,
	); err != nil {
		return fmt.Errorf("record legacy import source: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit legacy import: %w", err)
	}

	return nil
}

func readLegacyBucket[T any](tx *bolt.Tx, bucketName string) ([]T, error) {
	bucket := tx.Bucket([]byte(bucketName))
	if bucket == nil {
		return nil, nil
	}

	items := []T{}
	if err := bucket.ForEach(func(_, value []byte) error {
		var item T
		if err := json.Unmarshal(value, &item); err != nil {
			return err
		}
		items = append(items, item)
		return nil
	}); err != nil {
		return nil, fmt.Errorf("decode legacy bucket %q: %w", bucketName, err)
	}

	return items, nil
}
