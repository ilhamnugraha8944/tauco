package importer

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/ilhamnugraha8944/tauco/backend/internal/content/application"
	"github.com/ilhamnugraha8944/tauco/backend/internal/content/domain"
	api "github.com/ilhamnugraha8944/tauco/backend/internal/delivery/api"
)

const maxSourceDocumentBytes = 1024 * 1024

var forbiddenUnverifiedFields = map[string]struct{}{
	"address":           {},
	"availability":      {},
	"certification":     {},
	"certifications":    {},
	"email":             {},
	"establishedyear":   {},
	"healthclaim":       {},
	"healthclaims":      {},
	"inventory":         {},
	"legality":          {},
	"legalities":        {},
	"netweight":         {},
	"netweightgrams":    {},
	"offer":             {},
	"offers":            {},
	"officiallogo":      {},
	"owner":             {},
	"packagingsizes":    {},
	"phone":             {},
	"price":             {},
	"productionprocess": {},
	"rating":            {},
	"ratings":           {},
	"review":            {},
	"reviews":           {},
	"shelflife":         {},
	"sku":               {},
	"sociallinks":       {},
	"stock":             {},
	"weight":            {},
	"whatsapp":          {},
}

// SourceBundle contains the four committed Phase 1A JSON documents.
type SourceBundle struct {
	Home       []byte
	About      []byte
	TaucoGuide []byte
	Products   []byte
}

// LoadPhase1ADirectory reads and validates the committed Phase 1A content
// directory, then builds a deterministic import plan.
func LoadPhase1ADirectory(directory string) (application.SeedPlan, error) {
	manifest := Phase1AManifest()
	sources, err := readSourceBundle(directory, manifest)
	if err != nil {
		return application.SeedPlan{}, err
	}
	return BuildPhase1APlan(sources, manifest)
}

func readSourceBundle(directory string, manifest Manifest) (SourceBundle, error) {
	if err := manifest.Validate(); err != nil {
		return SourceBundle{}, fmt.Errorf("validate Phase 1A manifest: %w", err)
	}
	if directory == "" {
		return SourceBundle{}, errors.New("content directory is required")
	}

	var sources SourceBundle
	for _, page := range manifest.Pages {
		raw, err := readBoundedFile(filepath.Join(directory, page.SourceFilename))
		if err != nil {
			return SourceBundle{}, fmt.Errorf(
				"read %s source %q: %w",
				page.Key,
				page.SourceFilename,
				err,
			)
		}
		switch page.Key {
		case domain.PageKeyHome:
			sources.Home = raw
		case domain.PageKeyAbout:
			sources.About = raw
		case domain.PageKeyTaucoGuide:
			sources.TaucoGuide = raw
		case domain.PageKeyProducts:
			sources.Products = raw
		default:
			return SourceBundle{}, fmt.Errorf("unsupported page key %q", page.Key)
		}
	}
	return sources, nil
}

func readBoundedFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()

	raw, err := io.ReadAll(io.LimitReader(file, maxSourceDocumentBytes+1))
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, errors.New("source document is empty")
	}
	if len(raw) > maxSourceDocumentBytes {
		return nil, fmt.Errorf(
			"source document exceeds %d bytes",
			maxSourceDocumentBytes,
		)
	}
	return raw, nil
}

type productRecordWire struct {
	api.ProductDetail
	Status string `json:"status"`
}

type productCatalogDocumentWire struct {
	Metadata    api.SeoMetadata     `json:"metadata"`
	Heading     string              `json:"heading"`
	Description string              `json:"description"`
	ContactLink api.InternalLink    `json:"contactLink"`
	Products    []productRecordWire `json:"products"`
}

type productCatalogShell struct {
	Metadata    api.SeoMetadata  `json:"metadata"`
	Heading     string           `json:"heading"`
	Description string           `json:"description"`
	ContactLink api.InternalLink `json:"contactLink"`
}

// BuildPhase1APlan validates every source in memory before returning the one
// plan that a persistence adapter may apply atomically.
func BuildPhase1APlan(
	sources SourceBundle,
	manifest Manifest,
) (application.SeedPlan, error) {
	if err := manifest.Validate(); err != nil {
		return application.SeedPlan{}, fmt.Errorf("validate Phase 1A manifest: %w", err)
	}
	if err := validateSourceBundleEnvelope(sources); err != nil {
		return application.SeedPlan{}, err
	}

	spec, err := api.GetSwagger()
	if err != nil {
		return application.SeedPlan{}, fmt.Errorf("load embedded OpenAPI contract: %w", err)
	}

	home, err := decodeAndValidate[api.HomeContent](
		spec,
		"home.json",
		"HomeContent",
		sources.Home,
	)
	if err != nil {
		return application.SeedPlan{}, err
	}
	about, err := decodeAndValidate[api.AboutContent](
		spec,
		"about.json",
		"AboutContent",
		sources.About,
	)
	if err != nil {
		return application.SeedPlan{}, err
	}
	guide, err := decodeAndValidate[api.TaucoGuideContent](
		spec,
		"tauco-guide.json",
		"TaucoGuideContent",
		sources.TaucoGuide,
	)
	if err != nil {
		return application.SeedPlan{}, err
	}
	productsDocument, err := decodeStrict[productCatalogDocumentWire](
		"products.json",
		sources.Products,
	)
	if err != nil {
		return application.SeedPlan{}, err
	}

	if guide.UpdatedAt.Time.Before(guide.PublishedAt.Time) {
		return application.SeedPlan{}, errors.New(
			"tauco-guide.json updatedAt must not precede publishedAt",
		)
	}

	productDetails, productSummaries, sourceOrder, err := validateProducts(
		spec,
		productsDocument,
		manifest,
	)
	if err != nil {
		return application.SeedPlan{}, err
	}

	catalog := api.ProductCatalogContent{
		Metadata:    productsDocument.Metadata,
		Heading:     productsDocument.Heading,
		Description: productsDocument.Description,
		ContactLink: productsDocument.ContactLink,
		Products:    productSummaries,
	}
	if err := validateGeneratedAgainstSchema(
		spec,
		"ProductCatalogContent",
		catalog,
	); err != nil {
		return application.SeedPlan{}, fmt.Errorf(
			"products.json API catalog projection violates OpenAPI: %w",
			err,
		)
	}

	shell := productCatalogShell{
		Metadata:    productsDocument.Metadata,
		Heading:     productsDocument.Heading,
		Description: productsDocument.Description,
		ContactLink: productsDocument.ContactLink,
	}

	if err := validateBundleInvariants(home, about, guide, catalog, productDetails); err != nil {
		return application.SeedPlan{}, err
	}

	pageContent := make(map[domain.PageKey][]byte, len(manifest.Pages))
	for key, value := range map[domain.PageKey]any{
		domain.PageKeyHome:       home,
		domain.PageKeyAbout:      about,
		domain.PageKeyTaucoGuide: guide,
		domain.PageKeyProducts:   shell,
	} {
		canonical, err := canonicalizeValue(value)
		if err != nil {
			return application.SeedPlan{}, fmt.Errorf(
				"canonicalize %s content: %w",
				key,
				err,
			)
		}
		pageContent[key] = canonical
	}

	plan := application.SeedPlan{
		ManifestVersion: manifest.Version,
		Pages:           make([]application.PageSeed, 0, len(manifest.Pages)),
		Products:        make([]application.ProductSeed, 0, len(manifest.Products)),
	}
	for _, entry := range manifest.Pages {
		content, exists := pageContent[entry.Key]
		if !exists {
			return application.SeedPlan{}, fmt.Errorf(
				"no validated content projection for page %q",
				entry.Key,
			)
		}
		revision, err := newRevisionSeed(
			entry.EntityID,
			entry.RevisionID,
			entry.RevisionNumber,
			entry.SchemaVersion,
			entry.PublishedAt,
			content,
		)
		if err != nil {
			return application.SeedPlan{}, fmt.Errorf(
				"build page %q revision: %w",
				entry.Key,
				err,
			)
		}
		plan.Pages = append(plan.Pages, application.PageSeed{
			Key:      entry.Key,
			Revision: revision,
		})
	}

	productManifestBySlug := make(
		map[string]ProductManifestEntry,
		len(manifest.Products),
	)
	for _, entry := range manifest.Products {
		productManifestBySlug[entry.Slug] = entry
	}
	for slug, index := range sourceOrder {
		entry := productManifestBySlug[slug]
		canonical, err := canonicalizeValue(productDetails[slug])
		if err != nil {
			return application.SeedPlan{}, fmt.Errorf(
				"canonicalize product %q: %w",
				slug,
				err,
			)
		}
		revision, err := newRevisionSeed(
			entry.EntityID,
			entry.RevisionID,
			entry.RevisionNumber,
			entry.SchemaVersion,
			entry.PublishedAt,
			canonical,
		)
		if err != nil {
			return application.SeedPlan{}, fmt.Errorf(
				"build product %q revision: %w",
				slug,
				err,
			)
		}
		plan.Products = append(plan.Products, application.ProductSeed{
			Slug:      slug,
			SortOrder: index,
			Revision:  revision,
		})
	}
	sort.Slice(plan.Products, func(left, right int) bool {
		return plan.Products[left].SortOrder < plan.Products[right].SortOrder
	})

	if err := plan.Validate(); err != nil {
		return application.SeedPlan{}, fmt.Errorf(
			"validate deterministic Phase 1A plan: %w",
			err,
		)
	}
	return plan, nil
}

func validateSourceBundleEnvelope(sources SourceBundle) error {
	documents := []struct {
		name string
		raw  []byte
	}{
		{name: "home.json", raw: sources.Home},
		{name: "about.json", raw: sources.About},
		{name: "tauco-guide.json", raw: sources.TaucoGuide},
		{name: "products.json", raw: sources.Products},
	}
	for _, document := range documents {
		if len(document.raw) == 0 {
			return fmt.Errorf("%s is empty", document.name)
		}
		if len(document.raw) > maxSourceDocumentBytes {
			return fmt.Errorf(
				"%s exceeds %d bytes",
				document.name,
				maxSourceDocumentBytes,
			)
		}
		if !utf8.Valid(document.raw) {
			return fmt.Errorf("%s must contain valid UTF-8", document.name)
		}
		if err := rejectDuplicateJSONKeys(document.raw); err != nil {
			return fmt.Errorf("%s has unsafe JSON shape: %w", document.name, err)
		}
		if err := rejectUnverifiedFields(document.raw); err != nil {
			return fmt.Errorf("%s: %w", document.name, err)
		}
	}
	return nil
}

func decodeAndValidate[T any](
	spec *openapi3.T,
	filename string,
	schemaName string,
	raw []byte,
) (T, error) {
	value, err := decodeStrict[T](filename, raw)
	if err != nil {
		var zero T
		return zero, err
	}
	if err := validateRawAgainstSchema(spec, schemaName, raw); err != nil {
		var zero T
		return zero, fmt.Errorf(
			"%s violates OpenAPI schema %s: %w",
			filename,
			schemaName,
			err,
		)
	}
	return value, nil
}

func decodeStrict[T any](filename string, raw []byte) (T, error) {
	var value T
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return value, fmt.Errorf("%s strict generated decoding failed: %w", filename, err)
	}
	if err := expectJSONEOF(decoder); err != nil {
		return value, fmt.Errorf("%s strict generated decoding failed: %w", filename, err)
	}
	return value, nil
}

func validateRawAgainstSchema(
	spec *openapi3.T,
	schemaName string,
	raw []byte,
) error {
	var document any
	if err := json.Unmarshal(raw, &document); err != nil {
		return err
	}
	schema := spec.Components.Schemas[schemaName]
	if schema == nil || schema.Value == nil {
		return fmt.Errorf("OpenAPI schema %q is missing", schemaName)
	}
	return schema.Value.VisitJSON(document)
}

func validateGeneratedAgainstSchema(
	spec *openapi3.T,
	schemaName string,
	value any,
) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return validateRawAgainstSchema(spec, schemaName, raw)
}

func validateProducts(
	spec *openapi3.T,
	document productCatalogDocumentWire,
	manifest Manifest,
) (
	map[string]api.ProductDetail,
	[]api.ProductSummary,
	map[string]int,
	error,
) {
	manifestBySlug := make(map[string]struct{}, len(manifest.Products))
	for _, entry := range manifest.Products {
		manifestBySlug[entry.Slug] = struct{}{}
	}
	if len(document.Products) != len(manifest.Products) {
		return nil, nil, nil, fmt.Errorf(
			"products.json contains %d products but manifest contains %d",
			len(document.Products),
			len(manifest.Products),
		)
	}

	details := make(map[string]api.ProductDetail, len(document.Products))
	summaries := make([]api.ProductSummary, 0, len(document.Products))
	sourceOrder := make(map[string]int, len(document.Products))
	for index, record := range document.Products {
		slug := record.Slug
		if record.Status != string(domain.RevisionStatusPublished) {
			return nil, nil, nil, fmt.Errorf(
				"products.json product %q status must be published",
				slug,
			)
		}
		if _, exists := manifestBySlug[slug]; !exists {
			return nil, nil, nil, fmt.Errorf(
				"products.json product %q has no preassigned manifest identity",
				slug,
			)
		}
		if _, duplicate := details[slug]; duplicate {
			return nil, nil, nil, fmt.Errorf(
				"products.json contains duplicate slug %q",
				slug,
			)
		}
		if record.PriceEstimate != nil {
			return nil, nil, nil, fmt.Errorf(
				"products.json product %q priceEstimate is unverified and must remain null",
				slug,
			)
		}
		expectedCanonical := "/produk/" + slug
		if record.Metadata.CanonicalPath != expectedCanonical {
			return nil, nil, nil, fmt.Errorf(
				"products.json product %q canonicalPath must be %q",
				slug,
				expectedCanonical,
			)
		}
		if err := validateGeneratedAgainstSchema(spec, "ProductDetail", record.ProductDetail); err != nil {
			return nil, nil, nil, fmt.Errorf(
				"products.json product %q violates OpenAPI ProductDetail: %w",
				slug,
				err,
			)
		}

		details[slug] = record.ProductDetail
		sourceOrder[slug] = index
		summaries = append(summaries, productSummary(record.ProductDetail))
	}

	for slug := range manifestBySlug {
		if _, exists := details[slug]; !exists {
			return nil, nil, nil, fmt.Errorf(
				"manifest product %q is missing from products.json",
				slug,
			)
		}
	}
	return details, summaries, sourceOrder, nil
}

func productSummary(product api.ProductDetail) api.ProductSummary {
	return api.ProductSummary{
		Slug:     product.Slug,
		Name:     product.Name,
		Category: product.Category,
		Summary:  product.Summary,
		Image:    product.Image,
		Facts:    product.Facts,
	}
}

func validateBundleInvariants(
	home api.HomeContent,
	about api.AboutContent,
	guide api.TaucoGuideContent,
	catalog api.ProductCatalogContent,
	products map[string]api.ProductDetail,
) error {
	canonicals := []struct {
		label string
		got   string
		want  string
	}{
		{label: "home", got: home.Metadata.CanonicalPath, want: "/"},
		{label: "about", got: about.Metadata.CanonicalPath, want: "/tentang-kami"},
		{label: "tauco guide", got: guide.Metadata.CanonicalPath, want: "/tauco"},
		{label: "product catalog", got: catalog.Metadata.CanonicalPath, want: "/produk"},
	}
	for _, canonical := range canonicals {
		if canonical.got != canonical.want {
			return fmt.Errorf(
				"%s canonicalPath must be %q",
				canonical.label,
				canonical.want,
			)
		}
	}

	for _, slug := range home.FeaturedProductSlugs {
		if _, exists := products[slug]; !exists {
			return fmt.Errorf(
				"home featured product %q is not a published manifest product",
				slug,
			)
		}
	}

	if err := validateUniqueSectionIDs("about", about.Sections); err != nil {
		return err
	}
	if err := validateUniqueSectionIDs("tauco guide", guide.Sections); err != nil {
		return err
	}

	validPaths := map[string]struct{}{
		"/":                  {},
		"/kontak":            {},
		"/kebijakan-privasi": {},
		"/produk":            {},
		"/tauco":             {},
		"/tentang-kami":      {},
	}
	for slug := range products {
		validPaths["/produk/"+slug] = struct{}{}
	}

	links := make([]api.InternalLink, 0)
	links = append(links, home.Hero.Actions...)
	links = append(links, home.GuidePreview.Link, home.AboutPreview.Link)
	links = append(links, about.RelatedLinks...)
	links = append(links, guide.RelatedLinks...)
	links = append(links, catalog.ContactLink)
	for _, product := range products {
		links = append(links, product.ContactLink)
	}
	for _, link := range links {
		path := strings.SplitN(link.Href, "?", 2)[0]
		if _, exists := validPaths[path]; !exists {
			return fmt.Errorf("internal link %q points to an unknown route", link.Href)
		}
	}
	return nil
}

func validateUniqueSectionIDs(label string, sections []api.TextSection) error {
	seen := make(map[string]struct{}, len(sections))
	for _, section := range sections {
		if _, duplicate := seen[section.Id]; duplicate {
			return fmt.Errorf("%s contains duplicate section ID %q", label, section.Id)
		}
		seen[section.Id] = struct{}{}
	}
	return nil
}

func newRevisionSeed(
	entityID domain.UUIDv7,
	revisionID domain.UUIDv7,
	revisionNumber uint32,
	schemaVersion uint32,
	publishedAt time.Time,
	content []byte,
) (application.RevisionSeed, error) {
	canonical, checksum, err := domain.CanonicalJSONChecksum(content)
	if err != nil {
		return application.RevisionSeed{}, err
	}
	if !bytes.Equal(content, canonical) {
		return application.RevisionSeed{}, errors.New("content is not canonical-json-v1")
	}
	return application.RevisionSeed{
		EntityID:       entityID,
		RevisionID:     revisionID,
		RevisionNumber: revisionNumber,
		SchemaVersion:  schemaVersion,
		Status:         domain.RevisionStatusPublished,
		ContentJSON:    append(json.RawMessage(nil), canonical...),
		Checksum:       checksum,
		PublishedAt:    publishedAt.UTC(),
	}, nil
}

func rejectUnverifiedFields(raw []byte) error {
	var document any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&document); err != nil {
		return err
	}
	if err := expectJSONEOF(decoder); err != nil {
		return err
	}

	var walk func(any, string) error
	walk = func(value any, path string) error {
		switch typed := value.(type) {
		case map[string]any:
			for key, child := range typed {
				normalized := normalizeFieldName(key)
				if _, forbidden := forbiddenUnverifiedFields[normalized]; forbidden {
					return fmt.Errorf(
						"unverified field %q is forbidden at %s",
						key,
						path,
					)
				}
				if normalized == "priceestimate" && child != nil {
					return fmt.Errorf(
						"unverified field %q must remain null at %s",
						key,
						path,
					)
				}
				if err := walk(child, path+"."+key); err != nil {
					return err
				}
			}
		case []any:
			for index, child := range typed {
				if err := walk(child, path+"["+strconv.Itoa(index)+"]"); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return walk(document, "$")
}

func normalizeFieldName(value string) string {
	replacer := strings.NewReplacer("-", "", "_", "", " ", "")
	return strings.ToLower(replacer.Replace(value))
}

func rejectDuplicateJSONKeys(raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()

	var walk func(string) error
	walk = func(path string) error {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		delimiter, isDelimiter := token.(json.Delim)
		if !isDelimiter {
			return nil
		}

		switch delimiter {
		case '{':
			keys := make(map[string]struct{})
			for decoder.More() {
				keyToken, err := decoder.Token()
				if err != nil {
					return err
				}
				key, ok := keyToken.(string)
				if !ok {
					return errors.New("object key is not a string")
				}
				if _, duplicate := keys[key]; duplicate {
					return fmt.Errorf("duplicate object key %q at %s", key, path)
				}
				keys[key] = struct{}{}
				if err := walk(path + "." + key); err != nil {
					return err
				}
			}
			end, err := decoder.Token()
			if err != nil {
				return err
			}
			if end != json.Delim('}') {
				return fmt.Errorf("object at %s is not closed", path)
			}
		case '[':
			index := 0
			for decoder.More() {
				if err := walk(path + "[" + strconv.Itoa(index) + "]"); err != nil {
					return err
				}
				index++
			}
			end, err := decoder.Token()
			if err != nil {
				return err
			}
			if end != json.Delim(']') {
				return fmt.Errorf("array at %s is not closed", path)
			}
		default:
			return fmt.Errorf("unexpected delimiter %q at %s", delimiter, path)
		}
		return nil
	}

	if err := walk("$"); err != nil {
		return err
	}
	return expectJSONEOF(decoder)
}

func canonicalizeValue(value any) ([]byte, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return domain.CanonicalizeJSON(raw)
}

func expectJSONEOF(decoder *json.Decoder) error {
	var trailing any
	err := decoder.Decode(&trailing)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return errors.New("JSON document contains more than one value")
	}
	return err
}
