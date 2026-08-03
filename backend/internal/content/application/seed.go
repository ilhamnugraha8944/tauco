package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/ilhamnugraha8944/tauco/backend/internal/content/domain"
)

const Phase1AManifestVersion uint32 = 1

var ErrSeedConflict = errors.New("deterministic seed conflicts with persisted state")

// RevisionSeed contains the immutable revision values prepared before a
// persistence transaction starts.
type RevisionSeed struct {
	EntityID       domain.UUIDv7
	RevisionID     domain.UUIDv7
	RevisionNumber uint32
	SchemaVersion  uint32
	Status         domain.RevisionStatus
	ContentJSON    json.RawMessage
	Checksum       domain.SHA256Checksum
	PublishedAt    time.Time
}

// PageSeed is one singleton page and its initial published revision.
type PageSeed struct {
	Key      domain.PageKey
	Revision RevisionSeed
}

// ProductSeed is one stable product identity and its initial published
// revision.
type ProductSeed struct {
	Slug      string
	SortOrder int
	Revision  RevisionSeed
}

// SeedPlan is a completely validated, deterministic Phase 1A import plan.
// Persistence adapters must treat it as immutable.
type SeedPlan struct {
	ManifestVersion uint32
	Pages           []PageSeed
	Products        []ProductSeed
}

// Validate checks persistence-independent invariants before the store opens a
// transaction.
func (plan SeedPlan) Validate() error {
	if plan.ManifestVersion != Phase1AManifestVersion {
		return fmt.Errorf(
			"manifest version %d is unsupported",
			plan.ManifestVersion,
		)
	}
	if len(plan.Pages) == 0 {
		return errors.New("seed plan must contain at least one page")
	}
	if len(plan.Products) == 0 {
		return errors.New("seed plan must contain at least one product")
	}

	pageKeys := make(map[domain.PageKey]struct{}, len(plan.Pages))
	productSlugs := make(map[string]struct{}, len(plan.Products))
	identities := make(map[domain.UUIDv7]string, 2*(len(plan.Pages)+len(plan.Products)))

	for index, page := range plan.Pages {
		if !page.Key.Valid() {
			return fmt.Errorf("pages[%d] has invalid key %q", index, page.Key)
		}
		if _, duplicate := pageKeys[page.Key]; duplicate {
			return fmt.Errorf("page key %q is duplicated", page.Key)
		}
		pageKeys[page.Key] = struct{}{}
		if err := validateRevisionSeed(page.Revision); err != nil {
			return fmt.Errorf("page %q: %w", page.Key, err)
		}
		if err := addRevisionIdentities(
			identities,
			"page:"+string(page.Key),
			page.Revision,
		); err != nil {
			return err
		}
	}
	requiredPageKeys := []domain.PageKey{
		domain.PageKeyHome,
		domain.PageKeyAbout,
		domain.PageKeyTaucoGuide,
		domain.PageKeyProducts,
	}
	if len(pageKeys) != len(requiredPageKeys) {
		return fmt.Errorf(
			"phase 1A seed must contain exactly %d singleton pages",
			len(requiredPageKeys),
		)
	}
	for _, key := range requiredPageKeys {
		if _, exists := pageKeys[key]; !exists {
			return fmt.Errorf("phase 1A seed is missing page %q", key)
		}
	}

	for index, product := range plan.Products {
		if err := domain.ValidateProductSlug(product.Slug); err != nil {
			return fmt.Errorf("products[%d]: %w", index, err)
		}
		if _, duplicate := productSlugs[product.Slug]; duplicate {
			return fmt.Errorf("product slug %q is duplicated", product.Slug)
		}
		productSlugs[product.Slug] = struct{}{}
		if product.SortOrder < 0 || int64(product.SortOrder) > math.MaxInt32 {
			return fmt.Errorf(
				"product %q sort order is outside PostgreSQL integer range",
				product.Slug,
			)
		}
		if err := validateRevisionSeed(product.Revision); err != nil {
			return fmt.Errorf("product %q: %w", product.Slug, err)
		}
		if err := addRevisionIdentities(
			identities,
			"product:"+product.Slug,
			product.Revision,
		); err != nil {
			return err
		}
	}

	return nil
}

func validateRevisionSeed(revision RevisionSeed) error {
	if _, err := domain.ParseUUIDv7(string(revision.EntityID)); err != nil {
		return fmt.Errorf("invalid entity ID: %w", err)
	}
	if _, err := domain.ParseUUIDv7(string(revision.RevisionID)); err != nil {
		return fmt.Errorf("invalid revision ID: %w", err)
	}
	if revision.EntityID == revision.RevisionID {
		return errors.New("entity and revision IDs must differ")
	}
	if revision.RevisionNumber == 0 ||
		uint64(revision.RevisionNumber) > math.MaxInt32 {
		return errors.New(
			"revision number must fit a positive PostgreSQL integer",
		)
	}
	if revision.SchemaVersion == 0 ||
		uint64(revision.SchemaVersion) > math.MaxInt32 {
		return errors.New(
			"schema version must fit a positive PostgreSQL integer",
		)
	}
	if revision.Status != domain.RevisionStatusPublished {
		return fmt.Errorf(
			"phase 1A seed revision must be published, got %q",
			revision.Status,
		)
	}
	if err := domain.ValidateCanonicalJSON(
		revision.ContentJSON,
		revision.Checksum,
	); err != nil {
		return fmt.Errorf("invalid canonical content: %w", err)
	}
	if revision.PublishedAt.IsZero() {
		return errors.New("published revision must have a publication timestamp")
	}
	if revision.PublishedAt.Location() != time.UTC {
		return errors.New("publication timestamp must use UTC")
	}
	if revision.PublishedAt.Nanosecond()%1_000 != 0 {
		return errors.New(
			"publication timestamp must not exceed PostgreSQL microsecond precision",
		)
	}
	return nil
}

func addRevisionIdentities(
	identities map[domain.UUIDv7]string,
	naturalKey string,
	revision RevisionSeed,
) error {
	for label, identity := range map[string]domain.UUIDv7{
		"entity":   revision.EntityID,
		"revision": revision.RevisionID,
	} {
		if owner, duplicate := identities[identity]; duplicate {
			return fmt.Errorf(
				"%s ID %q for %s is already assigned to %s",
				label,
				identity,
				naturalKey,
				owner,
			)
		}
		identities[identity] = naturalKey + ":" + label
	}
	return nil
}

// SeedEntityKind distinguishes singleton-page and product natural keys.
type SeedEntityKind string

const (
	SeedEntityPage    SeedEntityKind = "page"
	SeedEntityProduct SeedEntityKind = "product"
)

// SeedRecordKey is the natural key used to inspect current persistence state.
type SeedRecordKey struct {
	Kind       SeedEntityKind
	NaturalKey string
}

// StoredSeedRecord is the minimum state needed to make an idempotent decision.
// A repository must populate it from both the identity and revision tables.
type StoredSeedRecord struct {
	EntityID            domain.UUIDv7
	RevisionID          domain.UUIDv7
	RevisionNumber      uint32
	SchemaVersion       uint32
	Status              domain.RevisionStatus
	Checksum            domain.SHA256Checksum
	PublishedAt         time.Time
	PublishedRevisionID *domain.UUIDv7
	ProductSortOrder    *int
}

// SeedSnapshot contains records for every natural key in a requested plan and
// ownership for every preassigned UUID that is already occupied. Missing record
// entries mean that neither identity nor revision exists for that natural key.
type SeedSnapshot struct {
	Records        map[SeedRecordKey]StoredSeedRecord
	IdentityOwners map[domain.UUIDv7]SeedRecordKey
}

// ReconcileActionKind describes whether a deterministic seed is inserted or
// already present.
type ReconcileActionKind string

const (
	ReconcileInsert ReconcileActionKind = "insert"
	ReconcileNoop   ReconcileActionKind = "noop"
)

// ReconcileAction is an ordered persistence decision for one seed record.
type ReconcileAction struct {
	Key  SeedRecordKey
	Kind ReconcileActionKind
}

// Reconcile compares a validated plan with persisted state. Any mismatch in a
// stable identity, revision, checksum, status, publication timestamp, or
// published pointer is a conflict. It never proposes overwriting or archiving a
// published revision.
func Reconcile(plan SeedPlan, snapshot SeedSnapshot) ([]ReconcileAction, error) {
	if err := plan.Validate(); err != nil {
		return nil, fmt.Errorf("validate seed plan: %w", err)
	}

	expected := expectedSeedRecords(plan)
	keys := make([]SeedRecordKey, 0, len(expected))
	for key := range expected {
		keys = append(keys, key)
	}
	slices.SortFunc(keys, func(left, right SeedRecordKey) int {
		if left.Kind != right.Kind {
			if left.Kind < right.Kind {
				return -1
			}
			return 1
		}
		if left.NaturalKey < right.NaturalKey {
			return -1
		}
		if left.NaturalKey > right.NaturalKey {
			return 1
		}
		return 0
	})

	actions := make([]ReconcileAction, 0, len(keys))
	for _, key := range keys {
		want := expected[key]
		for _, identity := range []domain.UUIDv7{want.EntityID, want.RevisionID} {
			if owner, occupied := snapshot.IdentityOwners[identity]; occupied && owner != key {
				return nil, fmt.Errorf(
					"%w: UUID %q for %s %q is owned by %s %q",
					ErrSeedConflict,
					identity,
					key.Kind,
					key.NaturalKey,
					owner.Kind,
					owner.NaturalKey,
				)
			}
		}

		got, exists := snapshot.Records[key]
		if !exists {
			actions = append(actions, ReconcileAction{
				Key:  key,
				Kind: ReconcileInsert,
			})
			continue
		}
		for _, identity := range []domain.UUIDv7{want.EntityID, want.RevisionID} {
			owner, occupied := snapshot.IdentityOwners[identity]
			if !occupied {
				return nil, fmt.Errorf(
					"%w: snapshot omitted owner for existing UUID %q",
					ErrSeedConflict,
					identity,
				)
			}
			if owner != key {
				return nil, fmt.Errorf(
					"%w: existing UUID %q has a different owner",
					ErrSeedConflict,
					identity,
				)
			}
		}
		if !storedSeedRecordsEqual(got, want) {
			return nil, fmt.Errorf(
				"%w for %s %q",
				ErrSeedConflict,
				key.Kind,
				key.NaturalKey,
			)
		}
		actions = append(actions, ReconcileAction{
			Key:  key,
			Kind: ReconcileNoop,
		})
	}
	return actions, nil
}

func expectedSeedRecords(plan SeedPlan) map[SeedRecordKey]StoredSeedRecord {
	records := make(
		map[SeedRecordKey]StoredSeedRecord,
		len(plan.Pages)+len(plan.Products),
	)
	for _, page := range plan.Pages {
		key := SeedRecordKey{
			Kind:       SeedEntityPage,
			NaturalKey: string(page.Key),
		}
		records[key] = storedRecordFromRevision(page.Revision, nil)
	}
	for _, product := range plan.Products {
		key := SeedRecordKey{
			Kind:       SeedEntityProduct,
			NaturalKey: product.Slug,
		}
		sortOrder := product.SortOrder
		records[key] = storedRecordFromRevision(product.Revision, &sortOrder)
	}
	return records
}

func storedRecordFromRevision(
	revision RevisionSeed,
	productSortOrder *int,
) StoredSeedRecord {
	publishedRevisionID := revision.RevisionID
	var sortOrderCopy *int
	if productSortOrder != nil {
		value := *productSortOrder
		sortOrderCopy = &value
	}
	return StoredSeedRecord{
		EntityID:            revision.EntityID,
		RevisionID:          revision.RevisionID,
		RevisionNumber:      revision.RevisionNumber,
		SchemaVersion:       revision.SchemaVersion,
		Status:              revision.Status,
		Checksum:            revision.Checksum,
		PublishedAt:         revision.PublishedAt,
		PublishedRevisionID: &publishedRevisionID,
		ProductSortOrder:    sortOrderCopy,
	}
}

func storedSeedRecordsEqual(left, right StoredSeedRecord) bool {
	if left.EntityID != right.EntityID ||
		left.RevisionID != right.RevisionID ||
		left.RevisionNumber != right.RevisionNumber ||
		left.SchemaVersion != right.SchemaVersion ||
		left.Status != right.Status ||
		left.Checksum != right.Checksum ||
		!left.PublishedAt.Equal(right.PublishedAt) {
		return false
	}
	if left.PublishedRevisionID == nil || right.PublishedRevisionID == nil {
		if left.PublishedRevisionID != nil || right.PublishedRevisionID != nil {
			return false
		}
	} else if *left.PublishedRevisionID != *right.PublishedRevisionID {
		return false
	}
	if left.ProductSortOrder == nil || right.ProductSortOrder == nil {
		return left.ProductSortOrder == nil && right.ProductSortOrder == nil
	}
	return *left.ProductSortOrder == *right.ProductSortOrder
}

// SeedApplyResult summarizes one atomic importer execution.
type SeedApplyResult struct {
	Inserted  int
	Unchanged int
}

// SeedStore is the persistence port for the deterministic Phase 1A plan.
//
// Implementations must validate all current natural keys, acquire a
// transaction-scoped importer lock, reconcile and write within one database
// transaction, and roll back on any conflict. A replay with the same stable
// identities and checksums is a no-op. Implementations must never replace a
// different published pointer or mutate an immutable revision.
type SeedStore interface {
	ApplyPhase1A(context.Context, SeedPlan) (SeedApplyResult, error)
}

// ApplyPhase1A validates every input before invoking the single atomic store
// operation.
func ApplyPhase1A(
	ctx context.Context,
	store SeedStore,
	plan SeedPlan,
) (SeedApplyResult, error) {
	if store == nil {
		return SeedApplyResult{}, errors.New("seed store is required")
	}
	if err := plan.Validate(); err != nil {
		return SeedApplyResult{}, fmt.Errorf("validate Phase 1A seed: %w", err)
	}
	return store.ApplyPhase1A(ctx, plan)
}
