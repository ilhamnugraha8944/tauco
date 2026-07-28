package application

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/ilhamnugraha8944/tauco/backend/internal/content/domain"
)

func TestReconcileIsIdempotentAndConflictSafe(t *testing.T) {
	t.Parallel()

	plan := validSeedPlan(t)
	insertActions, err := Reconcile(plan, SeedSnapshot{})
	if err != nil {
		t.Fatalf("Reconcile(empty) error = %v", err)
	}
	if len(insertActions) != 5 {
		t.Fatalf("Reconcile(empty) action count = %d, want 5", len(insertActions))
	}
	for _, action := range insertActions {
		if action.Kind != ReconcileInsert {
			t.Errorf("Reconcile(empty) action = %q, want insert", action.Kind)
		}
	}

	snapshot := exactSeedSnapshot(plan)
	replayActions, err := Reconcile(plan, snapshot)
	if err != nil {
		t.Fatalf("Reconcile(exact replay) error = %v", err)
	}
	for _, action := range replayActions {
		if action.Kind != ReconcileNoop {
			t.Errorf("Reconcile(exact replay) action = %q, want noop", action.Kind)
		}
	}

	productKey := SeedRecordKey{
		Kind:       SeedEntityProduct,
		NaturalKey: "tauco-cap-badak",
	}
	conflicting := snapshot.Records[productKey]
	conflicting.Checksum = domain.SHA256Checksum(
		"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
	)
	snapshot.Records[productKey] = conflicting
	if _, err := Reconcile(plan, snapshot); !errors.Is(err, ErrSeedConflict) {
		t.Fatalf("Reconcile(checksum conflict) error = %v, want ErrSeedConflict", err)
	}

	snapshot = exactSeedSnapshot(plan)
	conflicting = snapshot.Records[productKey]
	otherRevision := mustUUIDv7(t, "019bfc80-0000-7000-8000-000000000099")
	conflicting.PublishedRevisionID = &otherRevision
	snapshot.Records[productKey] = conflicting
	if _, err := Reconcile(plan, snapshot); !errors.Is(err, ErrSeedConflict) {
		t.Fatalf("Reconcile(pointer conflict) error = %v, want ErrSeedConflict", err)
	}

	snapshot = exactSeedSnapshot(plan)
	conflicting = snapshot.Records[productKey]
	changedSortOrder := *conflicting.ProductSortOrder + 1
	conflicting.ProductSortOrder = &changedSortOrder
	snapshot.Records[productKey] = conflicting
	if _, err := Reconcile(plan, snapshot); !errors.Is(err, ErrSeedConflict) {
		t.Fatalf("Reconcile(sort-order conflict) error = %v, want ErrSeedConflict", err)
	}

	snapshot = exactSeedSnapshot(plan)
	pageKey := SeedRecordKey{
		Kind:       SeedEntityPage,
		NaturalKey: string(domain.PageKeyHome),
	}
	productRecord := snapshot.Records[productKey]
	snapshot.IdentityOwners[productRecord.EntityID] = pageKey
	if _, err := Reconcile(plan, snapshot); !errors.Is(err, ErrSeedConflict) {
		t.Fatalf("Reconcile(identity-owner conflict) error = %v, want ErrSeedConflict", err)
	}
}

func TestSeedPlanValidateRejectsPublishedSeedMutationHazards(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*SeedPlan)
	}{
		{
			name: "draft source",
			mutate: func(plan *SeedPlan) {
				plan.Products[0].Revision.Status = domain.RevisionStatusDraft
			},
		},
		{
			name: "duplicate identity",
			mutate: func(plan *SeedPlan) {
				plan.Products[0].Revision.EntityID = plan.Pages[0].Revision.EntityID
			},
		},
		{
			name: "non UTC publication",
			mutate: func(plan *SeedPlan) {
				plan.Products[0].Revision.PublishedAt =
					plan.Products[0].Revision.PublishedAt.In(
						time.FixedZone("WIB", 7*60*60),
					)
			},
		},
		{
			name: "invalid JSON",
			mutate: func(plan *SeedPlan) {
				plan.Products[0].Revision.ContentJSON = json.RawMessage(`{`)
			},
		},
		{
			name: "checksum mismatch",
			mutate: func(plan *SeedPlan) {
				plan.Products[0].Revision.Checksum = domain.SHA256Checksum(
					"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
				)
			},
		},
		{
			name: "incomplete singleton page set",
			mutate: func(plan *SeedPlan) {
				plan.Pages = plan.Pages[:3]
			},
		},
		{
			name: "revision number exceeds PostgreSQL integer",
			mutate: func(plan *SeedPlan) {
				plan.Pages[0].Revision.RevisionNumber = 1 << 31
			},
		},
		{
			name: "schema version exceeds PostgreSQL integer",
			mutate: func(plan *SeedPlan) {
				plan.Pages[0].Revision.SchemaVersion = 1 << 31
			},
		},
		{
			name: "sort order exceeds PostgreSQL integer",
			mutate: func(plan *SeedPlan) {
				plan.Products[0].SortOrder = 1 << 31
			},
		},
		{
			name: "publication exceeds PostgreSQL timestamp precision",
			mutate: func(plan *SeedPlan) {
				plan.Pages[0].Revision.PublishedAt =
					plan.Pages[0].Revision.PublishedAt.Add(time.Nanosecond)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan := validSeedPlan(t)
			test.mutate(&plan)
			if err := plan.Validate(); err == nil {
				t.Fatal("SeedPlan.Validate() unexpectedly succeeded")
			}
		})
	}
}

func TestApplyPhase1AValidatesBeforeStore(t *testing.T) {
	t.Parallel()

	store := &recordingSeedStore{}
	invalid := validSeedPlan(t)
	invalid.Products[0].Revision.Status = domain.RevisionStatusDraft
	if _, err := ApplyPhase1A(context.Background(), store, invalid); err == nil {
		t.Fatal("ApplyPhase1A(invalid) unexpectedly succeeded")
	}
	if store.calls != 0 {
		t.Fatalf("store calls after invalid plan = %d, want 0", store.calls)
	}

	result, err := ApplyPhase1A(
		context.Background(),
		store,
		validSeedPlan(t),
	)
	if err != nil {
		t.Fatalf("ApplyPhase1A(valid) error = %v", err)
	}
	if store.calls != 1 || result.Inserted != 5 {
		t.Fatalf("ApplyPhase1A(valid) calls/result = %d/%+v", store.calls, result)
	}
}

type recordingSeedStore struct {
	calls int
}

func (store *recordingSeedStore) ApplyPhase1A(
	_ context.Context,
	plan SeedPlan,
) (SeedApplyResult, error) {
	store.calls++
	return SeedApplyResult{
		Inserted: len(plan.Pages) + len(plan.Products),
	}, nil
}

func validSeedPlan(t *testing.T) SeedPlan {
	t.Helper()

	publishedAt := time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)
	content, checksum, err := domain.CanonicalJSONChecksum(
		json.RawMessage(`{"ok":true}`),
	)
	if err != nil {
		t.Fatalf("CanonicalJSONChecksum() error = %v", err)
	}
	revision := func(entityID, revisionID string) RevisionSeed {
		return RevisionSeed{
			EntityID:       mustUUIDv7(t, entityID),
			RevisionID:     mustUUIDv7(t, revisionID),
			RevisionNumber: 1,
			SchemaVersion:  1,
			Status:         domain.RevisionStatusPublished,
			ContentJSON:    append(json.RawMessage(nil), content...),
			Checksum:       checksum,
			PublishedAt:    publishedAt,
		}
	}
	return SeedPlan{
		ManifestVersion: Phase1AManifestVersion,
		Pages: []PageSeed{
			{
				Key: domain.PageKeyHome,
				Revision: revision(
					"019bfc80-0000-7000-8000-000000000001",
					"019bfc80-0000-7000-8000-000000000011",
				),
			},
			{
				Key: domain.PageKeyAbout,
				Revision: revision(
					"019bfc80-0000-7000-8000-000000000002",
					"019bfc80-0000-7000-8000-000000000012",
				),
			},
			{
				Key: domain.PageKeyTaucoGuide,
				Revision: revision(
					"019bfc80-0000-7000-8000-000000000003",
					"019bfc80-0000-7000-8000-000000000013",
				),
			},
			{
				Key: domain.PageKeyProducts,
				Revision: revision(
					"019bfc80-0000-7000-8000-000000000004",
					"019bfc80-0000-7000-8000-000000000014",
				),
			},
		},
		Products: []ProductSeed{
			{
				Slug:      "tauco-cap-badak",
				SortOrder: 0,
				Revision: revision(
					"019bfc80-0000-7000-8000-000000000101",
					"019bfc80-0000-7000-8000-000000000111",
				),
			},
		},
	}
}

func exactSeedSnapshot(plan SeedPlan) SeedSnapshot {
	records := expectedSeedRecords(plan)
	owners := make(map[domain.UUIDv7]SeedRecordKey, 2*len(records))
	for key, record := range records {
		owners[record.EntityID] = key
		owners[record.RevisionID] = key
	}
	return SeedSnapshot{
		Records:        records,
		IdentityOwners: owners,
	}
}

func mustUUIDv7(t *testing.T, value string) domain.UUIDv7 {
	t.Helper()
	identifier, err := domain.ParseUUIDv7(value)
	if err != nil {
		t.Fatalf("ParseUUIDv7(%q) error = %v", value, err)
	}
	return identifier
}
