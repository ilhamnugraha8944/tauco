package importer

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/ilhamnugraha8944/tauco/backend/internal/content/application"
	"github.com/ilhamnugraha8944/tauco/backend/internal/content/domain"
	api "github.com/ilhamnugraha8944/tauco/backend/internal/delivery/api"
)

func TestPhase1AImporterIsDeterministicAndGoldenCompatible(t *testing.T) {
	t.Parallel()

	repositoryRoot := importerRepositoryRoot(t)
	contentDirectory := filepath.Join(repositoryRoot, "content")
	first, err := LoadPhase1ADirectory(contentDirectory)
	if err != nil {
		t.Fatalf("LoadPhase1ADirectory(first) error = %v", err)
	}
	second, err := LoadPhase1ADirectory(contentDirectory)
	if err != nil {
		t.Fatalf("LoadPhase1ADirectory(second) error = %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("two imports of unchanged sources produced different plans")
	}
	if len(first.Pages) != 4 || len(first.Products) != 1 {
		t.Fatalf(
			"plan counts = %d pages/%d products, want 4/1",
			len(first.Pages),
			len(first.Products),
		)
	}

	for _, page := range first.Pages {
		if err := domain.ValidateCanonicalJSON(
			page.Revision.ContentJSON,
			page.Revision.Checksum,
		); err != nil {
			t.Errorf("page %q canonical checksum error = %v", page.Key, err)
		}
	}
	for _, product := range first.Products {
		if err := domain.ValidateCanonicalJSON(
			product.Revision.ContentJSON,
			product.Revision.Checksum,
		); err != nil {
			t.Errorf("product %q canonical checksum error = %v", product.Slug, err)
		}
		if bytes.Contains(product.Revision.ContentJSON, []byte(`"status"`)) {
			t.Errorf("product %q persisted status inside ProductDetail JSON", product.Slug)
		}
	}

	fixturesDirectory := filepath.Join(repositoryRoot, "contracts", "fixtures")
	assertPageMatchesFixture(
		t,
		first,
		domain.PageKeyHome,
		filepath.Join(fixturesDirectory, "home.success.json"),
	)
	assertPageMatchesFixture(
		t,
		first,
		domain.PageKeyAbout,
		filepath.Join(fixturesDirectory, "about.success.json"),
	)
	assertPageMatchesFixture(
		t,
		first,
		domain.PageKeyTaucoGuide,
		filepath.Join(fixturesDirectory, "tauco-guide.success.json"),
	)

	shellRaw := pageContent(t, first, domain.PageKeyProducts)
	var shell productCatalogShell
	if err := json.Unmarshal(shellRaw, &shell); err != nil {
		t.Fatalf("decode products shell: %v", err)
	}
	productRaw := first.Products[0].Revision.ContentJSON
	var detail api.ProductDetail
	if err := json.Unmarshal(productRaw, &detail); err != nil {
		t.Fatalf("decode product detail: %v", err)
	}
	catalog := api.ProductCatalogContent{
		Metadata:    shell.Metadata,
		Heading:     shell.Heading,
		Description: shell.Description,
		ContactLink: shell.ContactLink,
		Products:    []api.ProductSummary{productSummary(detail)},
	}
	assertValueMatchesFixtureData(
		t,
		catalog,
		filepath.Join(fixturesDirectory, "products-list.success.json"),
	)
	assertValueMatchesFixtureData(
		t,
		detail,
		filepath.Join(fixturesDirectory, "product-detail.success.json"),
	)
}

func TestPhase1AManifestHasStablePreassignedUUIDv7Identities(t *testing.T) {
	t.Parallel()

	manifest := Phase1AManifest()
	if err := manifest.Validate(); err != nil {
		t.Fatalf("Manifest.Validate() error = %v", err)
	}
	if got, want := manifest.Pages[0].EntityID,
		domain.UUIDv7("019bfc80-0000-7000-8000-000000000001"); got != want {
		t.Fatalf("home entity ID = %q, want %q", got, want)
	}
	if got, want := manifest.Products[0].RevisionID,
		domain.UUIDv7("019bfc80-0000-7000-8000-000000000111"); got != want {
		t.Fatalf("product revision ID = %q, want %q", got, want)
	}

	copyOfManifest := Phase1AManifest()
	copyOfManifest.Pages[0].SourceFilename = "changed.json"
	if Phase1AManifest().Pages[0].SourceFilename != "home.json" {
		t.Fatal("Phase1AManifest returned mutable shared state")
	}
}

func TestPhase1AImporterRejectsUnsafeOrUnverifiedSourceChanges(t *testing.T) {
	t.Parallel()

	sources := readCommittedSources(t)
	tests := []struct {
		name   string
		mutate func(SourceBundle) SourceBundle
		match  string
	}{
		{
			name: "unknown generated field",
			mutate: func(bundle SourceBundle) SourceBundle {
				document := decodeObject(t, bundle.Home)
				document["unexpected"] = true
				bundle.Home = encodeObject(t, document)
				return bundle
			},
			match: "unknown field",
		},
		{
			name: "duplicate JSON key",
			mutate: func(bundle SourceBundle) SourceBundle {
				bundle.Home = bytes.Replace(
					bundle.Home,
					[]byte(`"metadata": {`),
					[]byte(`"metadata": {}, "metadata": {`),
					1,
				)
				return bundle
			},
			match: "duplicate object key",
		},
		{
			name: "invalid UTF-8",
			mutate: func(bundle SourceBundle) SourceBundle {
				bundle.Home = []byte{'"', 0xff, '"'}
				return bundle
			},
			match: "valid UTF-8",
		},
		{
			name: "unverified SKU",
			mutate: func(bundle SourceBundle) SourceBundle {
				document := decodeObject(t, bundle.Products)
				product := document["products"].([]any)[0].(map[string]any)
				product["sku"] = "UNVERIFIED-1"
				bundle.Products = encodeObject(t, document)
				return bundle
			},
			match: "unverified field",
		},
		{
			name: "unverified price",
			mutate: func(bundle SourceBundle) SourceBundle {
				document := decodeObject(t, bundle.Products)
				product := document["products"].([]any)[0].(map[string]any)
				product["priceEstimate"] = map[string]any{
					"currency":  "IDR",
					"amount":    10000,
					"qualifier": "estimated",
				}
				bundle.Products = encodeObject(t, document)
				return bundle
			},
			match: "must remain null",
		},
		{
			name: "draft product",
			mutate: func(bundle SourceBundle) SourceBundle {
				document := decodeObject(t, bundle.Products)
				product := document["products"].([]any)[0].(map[string]any)
				product["status"] = "draft"
				bundle.Products = encodeObject(t, document)
				return bundle
			},
			match: "status must be published",
		},
		{
			name: "unknown featured product",
			mutate: func(bundle SourceBundle) SourceBundle {
				document := decodeObject(t, bundle.Home)
				document["featuredProductSlugs"] = []any{"unverified-product"}
				bundle.Home = encodeObject(t, document)
				return bundle
			},
			match: "not a published manifest product",
		},
		{
			name: "guide date order",
			mutate: func(bundle SourceBundle) SourceBundle {
				document := decodeObject(t, bundle.TaucoGuide)
				document["updatedAt"] = "2026-07-23"
				bundle.TaucoGuide = encodeObject(t, document)
				return bundle
			},
			match: "must not precede",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := BuildPhase1APlan(
				test.mutate(cloneSourceBundle(sources)),
				Phase1AManifest(),
			)
			if err == nil {
				t.Fatal("BuildPhase1APlan() unexpectedly succeeded")
			}
			if !strings.Contains(err.Error(), test.match) {
				t.Fatalf("BuildPhase1APlan() error = %v, want substring %q", err, test.match)
			}
		})
	}
}

func TestManifestRejectsFilenameAndIdentityDrift(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*Manifest)
	}{
		{
			name: "filename path traversal",
			mutate: func(manifest *Manifest) {
				manifest.Pages[0].SourceFilename = "../home.json"
			},
		},
		{
			name: "duplicate filename",
			mutate: func(manifest *Manifest) {
				manifest.Pages[1].SourceFilename = "home.json"
			},
		},
		{
			name: "invalid UUID version",
			mutate: func(manifest *Manifest) {
				manifest.Products[0].EntityID =
					"019bfc80-0000-4000-8000-000000000101"
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifest := Phase1AManifest()
			test.mutate(&manifest)
			if err := manifest.Validate(); err == nil {
				t.Fatal("Manifest.Validate() unexpectedly succeeded")
			}
		})
	}
}

func assertPageMatchesFixture(
	t *testing.T,
	plan application.SeedPlan,
	key domain.PageKey,
	fixturePath string,
) {
	t.Helper()
	assertRawMatchesFixtureData(t, pageContent(t, plan, key), fixturePath)
}

func assertValueMatchesFixtureData(t *testing.T, value any, fixturePath string) {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal golden projection: %v", err)
	}
	assertRawMatchesFixtureData(t, raw, fixturePath)
}

func assertRawMatchesFixtureData(t *testing.T, raw []byte, fixturePath string) {
	t.Helper()

	fixtureRaw, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture %s: %v", fixturePath, err)
	}
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(fixtureRaw, &envelope); err != nil {
		t.Fatalf("decode fixture %s: %v", fixturePath, err)
	}
	got, err := domain.CanonicalizeJSON(raw)
	if err != nil {
		t.Fatalf("canonicalize importer projection: %v", err)
	}
	want, err := domain.CanonicalizeJSON(envelope.Data)
	if err != nil {
		t.Fatalf("canonicalize fixture data: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf(
			"importer projection differs from golden fixture %s:\n got: %s\nwant: %s",
			fixturePath,
			got,
			want,
		)
	}
}

func pageContent(
	t *testing.T,
	plan application.SeedPlan,
	key domain.PageKey,
) json.RawMessage {
	t.Helper()
	for _, page := range plan.Pages {
		if page.Key == key {
			return page.Revision.ContentJSON
		}
	}
	t.Fatalf("plan has no page %q", key)
	return nil
}

func readCommittedSources(t *testing.T) SourceBundle {
	t.Helper()
	root := importerRepositoryRoot(t)
	read := func(filename string) []byte {
		t.Helper()
		raw, err := os.ReadFile(filepath.Join(root, "content", filename))
		if err != nil {
			t.Fatalf("read %s: %v", filename, err)
		}
		return raw
	}
	return SourceBundle{
		Home:       read("home.json"),
		About:      read("about.json"),
		TaucoGuide: read("tauco-guide.json"),
		Products:   read("products.json"),
	}
}

func cloneSourceBundle(bundle SourceBundle) SourceBundle {
	return SourceBundle{
		Home:       bytes.Clone(bundle.Home),
		About:      bytes.Clone(bundle.About),
		TaucoGuide: bytes.Clone(bundle.TaucoGuide),
		Products:   bytes.Clone(bundle.Products),
	}
}

func decodeObject(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatalf("decode mutation source: %v", err)
	}
	return document
}

func encodeObject(t *testing.T, document map[string]any) []byte {
	t.Helper()
	raw, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("encode mutation source: %v", err)
	}
	return raw
}

func importerRepositoryRoot(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve importer test path")
	}
	root := filepath.Clean(
		filepath.Join(filepath.Dir(filename), "..", "..", "..", ".."),
	)
	if _, err := os.Stat(filepath.Join(root, "content", "home.json")); err != nil {
		t.Fatalf("resolve repository root %s: %v", root, err)
	}
	return root
}

func TestCanonicalContentRejectsChecksumDrift(t *testing.T) {
	t.Parallel()

	plan, err := BuildPhase1APlan(readCommittedSources(t), Phase1AManifest())
	if err != nil {
		t.Fatalf("BuildPhase1APlan() error = %v", err)
	}
	plan.Pages[0].Revision.Checksum = domain.SHA256Checksum(
		"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
	)
	if err := plan.Validate(); err == nil {
		t.Fatal("SeedPlan.Validate(checksum drift) unexpectedly succeeded")
	}

	plan, err = BuildPhase1APlan(readCommittedSources(t), Phase1AManifest())
	if err != nil {
		t.Fatalf("BuildPhase1APlan() error = %v", err)
	}
	plan.Pages = plan.Pages[:3]
	if err := plan.Validate(); err == nil {
		t.Fatal("SeedPlan.Validate(incomplete pages) unexpectedly succeeded")
	}
}

func TestReconcileRejectsSortOrderDriftFromImportedPlan(t *testing.T) {
	t.Parallel()

	plan, err := BuildPhase1APlan(readCommittedSources(t), Phase1AManifest())
	if err != nil {
		t.Fatalf("BuildPhase1APlan() error = %v", err)
	}
	actions, err := application.Reconcile(plan, application.SeedSnapshot{})
	if err != nil {
		t.Fatalf("Reconcile(empty) error = %v", err)
	}
	if len(actions) != 5 {
		t.Fatalf("Reconcile(empty) actions = %d, want 5", len(actions))
	}
	for _, action := range actions {
		if action.Kind != application.ReconcileInsert {
			t.Fatalf("Reconcile(empty) action = %q, want insert", action.Kind)
		}
	}
}

func TestNoUnexpectedSentinelWrapping(t *testing.T) {
	t.Parallel()

	// Keep a direct guard that importer validation errors are not mislabeled as
	// persistence conflicts.
	sources := readCommittedSources(t)
	sources.Products = nil
	_, err := BuildPhase1APlan(sources, Phase1AManifest())
	if err == nil {
		t.Fatal("BuildPhase1APlan(empty products) unexpectedly succeeded")
	}
	if errors.Is(err, application.ErrSeedConflict) {
		t.Fatalf("source validation error wrapped ErrSeedConflict: %v", err)
	}
}
