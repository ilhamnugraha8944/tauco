package api

import (
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gin-gonic/gin"
	"github.com/ilhamnugraha8944/tauco/backend/internal/contract/requestmeta"
)

const canonicalNonblankPattern = `^\S(?:[\s\S]*\S)?$`

func TestUnsafeGeneratedHandlerDefaultsAreNotUsed(t *testing.T) {
	t.Parallel()

	repositoryRoot := contractRepositoryRoot(t)
	backendRoot := filepath.Join(repositoryRoot, "backend")
	allowedFile := filepath.Clean(filepath.Join(
		backendRoot,
		"internal",
		"delivery",
		"api",
		"safe_handler.go",
	))
	forbidden := map[string]struct{}{
		"NewStrictHandler":            {},
		"NewStrictHandlerWithOptions": {},
		"RegisterHandlers":            {},
		"RegisterHandlersWithOptions": {},
	}

	err := filepath.WalkDir(
		backendRoot,
		func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() ||
				filepath.Ext(path) != ".go" ||
				strings.HasSuffix(path, "_test.go") ||
				filepath.Base(path) == "api.gen.go" ||
				filepath.Clean(path) == allowedFile {
				return nil
			}

			source, err := parser.ParseFile(
				token.NewFileSet(),
				path,
				nil,
				0,
			)
			if err != nil {
				return err
			}

			ast.Inspect(source, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}

				var calledName string
				switch function := call.Fun.(type) {
				case *ast.Ident:
					calledName = function.Name
				case *ast.SelectorExpr:
					calledName = function.Sel.Name
				default:
					return true
				}

				if _, unsafe := forbidden[calledName]; unsafe {
					t.Errorf(
						"%s calls unsafe generated default %s",
						path,
						calledName,
					)
				}
				return true
			})
			return nil
		},
	)
	if err != nil {
		t.Fatalf("scan generated-handler construction paths: %v", err)
	}
}

func TestEmbeddedOpenAPIContract(t *testing.T) {
	t.Parallel()

	spec := mustOpenAPISpec(t)
	if err := spec.Validate(
		context.Background(),
		openapi3.EnableExamplesValidation(),
	); err != nil {
		t.Fatalf("validate embedded OpenAPI document and examples: %v", err)
	}

	expectedOperations := operationSet(
		"GET /api/v1/about",
		"POST /api/v1/contact-messages",
		"GET /api/v1/home",
		"GET /api/v1/products",
		"GET /api/v1/products/{slug}",
		"GET /api/v1/tauco-guide",
		"GET /health/live",
		"GET /health/ready",
		"GET /internal/metrics",
		"POST /api/v1/admin/auth/login",
		"POST /api/v1/admin/auth/totp/setup",
		"POST /api/v1/admin/auth/totp/enable",
		"POST /api/v1/admin/auth/refresh",
		"POST /api/v1/admin/auth/logout",
		"GET /api/v1/admin/auth/me",
		"POST /api/v1/admin/auth/recovery-codes/regenerate",
		"GET /api/v1/admin/pages/{key}",
		"POST /api/v1/admin/pages/{key}/drafts",
		"GET /api/v1/admin/pages/{key}/revisions/{revisionId}",
		"POST /api/v1/admin/pages/{key}/revisions/{revisionId}/publish",
		"POST /api/v1/admin/pages/{key}/unpublish",
		"GET /api/v1/admin/products",
		"POST /api/v1/admin/products",
		"GET /api/v1/admin/products/{id}",
		"PATCH /api/v1/admin/products/{id}",
		"POST /api/v1/admin/products/{id}/drafts",
		"GET /api/v1/admin/products/{id}/revisions/{revisionId}",
		"POST /api/v1/admin/products/{id}/revisions/{revisionId}/publish",
		"POST /api/v1/admin/products/{id}/unpublish",
		"POST /api/v1/admin/products/{id}/archive",
		"POST /api/v1/admin/products/{id}/unarchive",
		"GET /api/v1/admin/media",
		"POST /api/v1/admin/media",
		"GET /api/v1/admin/media/{id}",
		"POST /api/v1/admin/media/{id}/retry",
		"GET /api/v1/media/{id}/display.webp",
		"GET /api/v1/media/{id}/variants/{width}.webp",
		"GET /api/v1/admin/contact-messages",
		"GET /api/v1/admin/contact-messages/{id}",
		"PATCH /api/v1/admin/contact-messages/{id}/status",
		"GET /api/v1/admin/activity-logs",
	)

	actualOperations := make(map[string]struct{})
	operationIDs := make(map[string]string)
	for route, pathItem := range spec.Paths.Map() {
		operations := pathItem.Operations()
		for method, operation := range operations {
			actualOperations[strings.ToUpper(method)+" "+route] = struct{}{}
			if operation.OperationID == "" {
				t.Errorf("%s %s has an empty operationId", method, route)
			} else if previous, exists := operationIDs[operation.OperationID]; exists {
				t.Errorf(
					"operationId %q is shared by %s and %s %s",
					operation.OperationID,
					previous,
					method,
					route,
				)
			} else {
				operationIDs[operation.OperationID] = method + " " + route
			}

			assertResponseContract(t, method, route, operation)
		}
	}

	if !reflect.DeepEqual(actualOperations, expectedOperations) {
		t.Fatalf(
			"operation inventory mismatch:\n got: %v\nwant: %v",
			sortedOperationSet(actualOperations),
			sortedOperationSet(expectedOperations),
		)
	}
}

func TestAdminOpenAPISecurityAndConcurrencyContract(t *testing.T) {
	t.Parallel()

	spec := mustOpenAPISpec(t)
	for route, pathItem := range spec.Paths.Map() {
		if !strings.HasPrefix(route, "/api/v1/admin/") {
			continue
		}
		for method, operation := range pathItem.Operations() {
			if route == "/api/v1/admin/auth/login" {
				if operation.Security != nil && len(*operation.Security) != 0 {
					t.Errorf("%s %s must be public before credential verification", method, route)
				}
				continue
			}
			if !hasSecurityScheme(operation, "AdminCookie") {
				t.Errorf("%s %s must require AdminCookie", method, route)
			}
			if method != http.MethodGet && !hasSecurityScheme(operation, "AdminCSRF") {
				t.Errorf("%s %s must require AdminCSRF", method, route)
			}
		}
	}

	for _, route := range []string{
		"/api/v1/admin/pages/{key}/drafts",
		"/api/v1/admin/pages/{key}/revisions/{revisionId}/publish",
		"/api/v1/admin/pages/{key}/unpublish",
		"/api/v1/admin/products/{id}",
		"/api/v1/admin/products/{id}/drafts",
		"/api/v1/admin/products/{id}/revisions/{revisionId}/publish",
		"/api/v1/admin/products/{id}/unpublish",
		"/api/v1/admin/products/{id}/archive",
		"/api/v1/admin/products/{id}/unarchive",
		"/api/v1/admin/contact-messages/{id}/status",
	} {
		operation := onlyMutationOperation(t, spec, route)
		if !hasParameter(operation, "If-Match", openapi3.ParameterInHeader) {
			t.Errorf("%s must require If-Match", route)
		}
	}

	if spec.Paths.Find("/api/v1/admin/tauco-guide") != nil {
		t.Error("tauco guide must remain read-only in Phase 1C")
	}
}

func TestOpenAPIOperationRequirements(t *testing.T) {
	t.Parallel()

	spec := mustOpenAPISpec(t)

	requiredStatuses := map[string][]string{
		"/api/v1/products": {
			"200", "304", "400", "429", "500", "503",
		},
		"/api/v1/products/{slug}": {
			"200", "304", "400", "404", "429", "500", "503",
		},
		"/api/v1/contact-messages": {
			"201", "400", "409", "413", "415", "422", "429", "500", "503",
		},
		"/health/ready":     {"200", "503"},
		"/internal/metrics": {"200", "401", "403", "500"},
	}
	for route, statuses := range requiredStatuses {
		operation := onlyOperation(t, spec, route)
		for _, status := range statuses {
			if operation.Responses.Value(status) == nil {
				t.Errorf("%s is missing required response %s", route, status)
			}
		}
	}

	for _, route := range []string{
		"/api/v1/home",
		"/api/v1/about",
		"/api/v1/tauco-guide",
		"/api/v1/products",
		"/api/v1/products/{slug}",
	} {
		operation := onlyOperation(t, spec, route)
		assertResponseHeaders(t, route+" 200", operation.Responses.Value("200"), "ETag", "Cache-Control")
		if operation.Responses.Value("304") == nil {
			t.Errorf("%s is missing ETag revalidation response 304", route)
		}
	}

	products := onlyOperation(t, spec, "/api/v1/products")
	if products.Extensions["x-reject-unknown-query-parameters"] != true {
		t.Error("product list must reject unknown query parameters")
	}
	pageSchema := spec.Components.Schemas["PageMeta"]
	if pageSchema == nil ||
		pageSchema.Value == nil ||
		pageSchema.Value.Extensions["x-invariant"] !=
			"nextCursor-non-null-iff-hasMore" {
		t.Error("PageMeta must declare its cursor/hasMore invariant")
	}
	homeSchema := spec.Components.Schemas["HomeContent"]
	featuredSlugs := schemaProperty(homeSchema, "featuredProductSlugs")
	if featuredSlugs == nil || !featuredSlugs.UniqueItems {
		t.Error("featured product slugs must be unique")
	}
	guideSchema := spec.Components.Schemas["TaucoGuideContent"]
	updatedAt := schemaProperty(guideSchema, "updatedAt")
	if updatedAt == nil ||
		updatedAt.Extensions["x-invariant"] !=
			"updatedAt-greater-than-or-equal-to-publishedAt" {
		t.Error("tauco guide must declare its date-order invariant")
	}
	imageSchema := spec.Components.Schemas["InformativeImageAsset"]
	imageAlt := schemaProperty(imageSchema, "alt")
	if imageAlt == nil ||
		imageAlt.Extensions["x-case-insensitive-exclusion"] !=
			"generic-image-label-v1" {
		t.Error("informative image alt exclusion must be case-insensitive")
	}

	contact := onlyOperation(t, spec, "/api/v1/contact-messages")
	if contact.RequestBody == nil ||
		contact.RequestBody.Value == nil ||
		!contact.RequestBody.Value.Required {
		t.Fatal("contact request body must be present and required")
	}
	if got := numericExtension(
		contact.RequestBody.Value.Extensions["x-max-body-bytes"],
	); got != 32*1024 {
		t.Fatalf("contact x-max-body-bytes = %d, want %d", got, 32*1024)
	}
	if len(contact.RequestBody.Value.Content) != 1 ||
		contact.RequestBody.Value.Content.Get("application/json") == nil {
		t.Error("contact request body must accept only application/json")
	}
	contactSchema := spec.Components.Schemas["ContactMessageRequest"]
	if contactSchema == nil ||
		contactSchema.Value == nil ||
		contactSchema.Value.Extensions["x-string-normalization"] !=
			"trim-before-transport" {
		t.Error("contact schema must declare canonical trim normalization")
	}
	if !hasParameter(contact, "Idempotency-Key", openapi3.ParameterInHeader) {
		t.Error("contact operation is missing required Idempotency-Key header")
	}
	assertResponseHeaders(
		t,
		"contact 201",
		contact.Responses.Value("201"),
		"Cache-Control",
		"Idempotency-Replayed",
	)

	metrics := onlyOperation(t, spec, "/internal/metrics")
	if metrics.Security == nil || len(*metrics.Security) != 1 {
		t.Fatal("internal metrics must have exactly one security requirement")
	}
	metricsRequirement := (*metrics.Security)[0]
	scopes, exists := metricsRequirement["InternalMetricsBearer"]
	if !exists || len(metricsRequirement) != 1 || len(scopes) != 0 {
		t.Error("internal metrics must require only InternalMetricsBearer")
	}
	metricsScheme := spec.Components.SecuritySchemes["InternalMetricsBearer"]
	if metricsScheme == nil ||
		metricsScheme.Value == nil ||
		metricsScheme.Value.Type != "http" ||
		metricsScheme.Value.Scheme != "bearer" {
		t.Error("InternalMetricsBearer must be an HTTP bearer scheme")
	}
}

func TestOpenAPIImportantStringConstraints(t *testing.T) {
	t.Parallel()

	spec := mustOpenAPISpec(t)
	canonicalProperties := []struct {
		component string
		property  string
		arrayItem bool
	}{
		{"InternalLink", "label", false},
		{"SeoMetadata", "title", false},
		{"SeoMetadata", "description", false},
		{"TextSection", "heading", false},
		{"TextSection", "paragraphs", true},
		{"SourceReference", "label", false},
		{"SourceReference", "publisher", false},
		{"PageHero", "eyebrow", false},
		{"PageHero", "title", false},
		{"PageHero", "description", false},
		{"HomeHero", "eyebrow", false},
		{"HomeHero", "title", false},
		{"HomeHero", "description", false},
		{"ContentPreview", "heading", false},
		{"ContentPreview", "description", false},
		{"ProductFact", "label", false},
		{"ProductFact", "value", false},
		{"ProductResearchEvidence", "heading", false},
		{"ProductResearchEvidence", "summary", false},
		{"ProductResearchEvidence", "scopeNote", false},
		{"ProductSummary", "name", false},
		{"ProductSummary", "category", false},
		{"ProductSummary", "summary", false},
		{"ProductDetail", "name", false},
		{"ProductDetail", "category", false},
		{"ProductDetail", "summary", false},
		{"ProductDetail", "description", true},
		{"ProductDetail", "usageSuggestions", true},
		{"ProductDetail", "purchaseNote", false},
		{"ProductCatalogContent", "heading", false},
		{"ProductCatalogContent", "description", false},
		{"ValidationError", "field", false},
		{"ValidationError", "message", false},
		{"Problem", "title", false},
		{"Problem", "detail", false},
	}

	for _, contract := range canonicalProperties {
		property := schemaProperty(
			spec.Components.Schemas[contract.component],
			contract.property,
		)
		if contract.arrayItem && property != nil {
			if property.Items == nil {
				t.Errorf(
					"%s.%s has no item schema",
					contract.component,
					contract.property,
				)
				continue
			}
			property = property.Items.Value
		}
		if property == nil {
			t.Errorf(
				"%s.%s is missing",
				contract.component,
				contract.property,
			)
			continue
		}
		if property.Pattern != canonicalNonblankPattern {
			t.Errorf(
				"%s.%s pattern = %q, want canonical nonblank pattern",
				contract.component,
				contract.property,
				property.Pattern,
			)
		}
	}

	imageSchema := spec.Components.Schemas["InformativeImageAsset"]
	if imageSchema == nil ||
		imageSchema.Value == nil ||
		imageSchema.Value.Extensions["x-string-normalization"] !=
			"trim-on-parse-emit-canonical" {
		t.Fatal("informative image normalization contract is missing")
	}
	imageAlt := schemaProperty(imageSchema, "alt")
	if imageAlt == nil {
		t.Fatal("InformativeImageAsset.alt is missing")
	}
	if imageAlt.Pattern != canonicalNonblankPattern {
		t.Fatalf("image alt pattern = %q, want canonical nonblank", imageAlt.Pattern)
	}
	if imageAlt.Not == nil ||
		imageAlt.Not.Value == nil ||
		imageAlt.Not.Value.Pattern == "" {
		t.Fatal("image alt must define a case-insensitive generic-label exclusion")
	}
	for _, genericAlt := range []string{
		"foto tauco",
		"FOTO TAUCO",
		"Gambar Produk",
		"IMAGE PRODUCT",
		"photo product",
	} {
		if err := imageAlt.VisitJSON(genericAlt); err == nil {
			t.Errorf("OpenAPI accepted generic image alt %q", genericAlt)
		}
	}
	for _, invalidAlt := range []string{
		"        ",
		" Tauco semipadat dalam mangkuk",
		"Tauco semipadat dalam mangkuk ",
	} {
		if err := imageAlt.VisitJSON(invalidAlt); err == nil {
			t.Errorf("OpenAPI accepted non-canonical image alt %q", invalidAlt)
		}
	}
	if err := imageAlt.VisitJSON(
		"Tauco semipadat dalam mangkuk dengan kedelai",
	); err != nil {
		t.Errorf("OpenAPI rejected descriptive canonical image alt: %v", err)
	}
}

func TestGeneratedProblemWriterEmitsOpenAPIValidEscapedInstances(t *testing.T) {
	spec := mustOpenAPISpec(t)
	problemSchema := spec.Components.Schemas["Problem"]
	if problemSchema == nil || problemSchema.Value == nil {
		t.Fatal("Problem schema is missing")
	}

	for _, target := range []string{
		"/API/V1/Missing",
		"/api/v1/missing/",
		"/API/V1/Missing%20Product/",
	} {
		t.Run(target, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, target, nil)
			expectedInstance := requestmeta.ProblemInstancePath(request)
			recorder := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(recorder)
			request.Header.Set(requestmeta.Header, "problem-schema-test")
			context.Request = request
			writeContractProblem(
				context,
				http.StatusBadRequest,
				"urn:tauco-cap-badak:problem:bad-request",
				"Permintaan tidak valid",
				"Format permintaan tidak dapat diproses.",
				"BAD_REQUEST",
			)
			if recorder.Code < 400 {
				t.Fatalf("status = %d, want problem response", recorder.Code)
			}

			var document any
			if err := json.Unmarshal(recorder.Body.Bytes(), &document); err != nil {
				t.Fatalf("decode problem response: %v", err)
			}
			if err := problemSchema.Value.VisitJSON(document); err != nil {
				t.Fatalf("problem response violates OpenAPI: %v", err)
			}

			body := document.(map[string]any)
			if body["instance"] != expectedInstance {
				t.Fatalf(
					"instance = %q, want %q",
					body["instance"],
					expectedInstance,
				)
			}
		})
	}
}

func TestCommittedFixturesMatchOpenAPIAndGeneratedTypes(t *testing.T) {
	t.Parallel()

	spec := mustOpenAPISpec(t)
	fixtures := []struct {
		name       string
		filename   string
		schemaName string
		target     func() any
	}{
		{
			name:       "home success",
			filename:   "home.success.json",
			schemaName: "HomeResponse",
			target:     func() any { return &HomeResponse{} },
		},
		{
			name:       "about success",
			filename:   "about.success.json",
			schemaName: "AboutResponse",
			target:     func() any { return &AboutResponse{} },
		},
		{
			name:       "tauco guide success",
			filename:   "tauco-guide.success.json",
			schemaName: "TaucoGuideResponse",
			target:     func() any { return &TaucoGuideResponse{} },
		},
		{
			name:       "product list success",
			filename:   "products-list.success.json",
			schemaName: "ProductListResponse",
			target:     func() any { return &ProductListResponse{} },
		},
		{
			name:       "product detail success",
			filename:   "product-detail.success.json",
			schemaName: "ProductDetailResponse",
			target:     func() any { return &ProductDetailResponse{} },
		},
		{
			name:       "validation problem",
			filename:   "validation.problem.json",
			schemaName: "Problem",
			target:     func() any { return &Problem{} },
		},
		{
			name:       "contact message request",
			filename:   "contact-message.request.json",
			schemaName: "ContactMessageRequest",
			target:     func() any { return &ContactMessageRequest{} },
		},
		{
			name:       "contact message success",
			filename:   "contact-message.success.json",
			schemaName: "ContactMessageResponse",
			target:     func() any { return &ContactMessageResponse{} },
		},
	}

	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			raw := readContractFixture(t, fixture.filename)

			var sourceDocument any
			if err := json.Unmarshal(raw, &sourceDocument); err != nil {
				t.Fatalf("decode fixture JSON: %v", err)
			}

			schemaRef := spec.Components.Schemas[fixture.schemaName]
			if schemaRef == nil || schemaRef.Value == nil {
				t.Fatalf("OpenAPI schema %q is missing", fixture.schemaName)
			}
			if err := schemaRef.Value.VisitJSON(sourceDocument); err != nil {
				t.Fatalf("fixture violates OpenAPI schema %s: %v", fixture.schemaName, err)
			}

			generatedModel := fixture.target()
			if err := json.Unmarshal(raw, generatedModel); err != nil {
				t.Fatalf("decode fixture into generated type: %v", err)
			}
			roundTripRaw, err := json.Marshal(generatedModel)
			if err != nil {
				t.Fatalf("encode generated type: %v", err)
			}

			var roundTripDocument any
			if err := json.Unmarshal(roundTripRaw, &roundTripDocument); err != nil {
				t.Fatalf("decode generated round trip: %v", err)
			}
			if !reflect.DeepEqual(roundTripDocument, sourceDocument) {
				t.Fatalf(
					"generated type changed fixture shape:\n got: %s\nwant: %s",
					roundTripRaw,
					raw,
				)
			}
		})
	}
}

func mustOpenAPISpec(t *testing.T) *openapi3.T {
	t.Helper()

	spec, err := GetSpec()
	if err != nil {
		t.Fatalf("load embedded OpenAPI document: %v", err)
	}
	return spec
}

func onlyOperation(t *testing.T, spec *openapi3.T, route string) *openapi3.Operation {
	t.Helper()

	pathItem := spec.Paths.Find(route)
	if pathItem == nil {
		t.Fatalf("OpenAPI route %q is missing", route)
	}
	operations := pathItem.Operations()
	if len(operations) != 1 {
		t.Fatalf("%s has %d operations, want exactly 1", route, len(operations))
	}
	for _, operation := range operations {
		return operation
	}
	panic("unreachable")
}

func onlyMutationOperation(t *testing.T, spec *openapi3.T, route string) *openapi3.Operation {
	t.Helper()

	pathItem := spec.Paths.Find(route)
	if pathItem == nil {
		t.Fatalf("OpenAPI route %q is missing", route)
	}
	for method, operation := range pathItem.Operations() {
		if method != http.MethodGet {
			return operation
		}
	}
	t.Fatalf("OpenAPI route %q has no mutation operation", route)
	return nil
}

func assertResponseContract(
	t *testing.T,
	method string,
	route string,
	operation *openapi3.Operation,
) {
	t.Helper()

	if operation.Responses == nil || operation.Responses.Len() == 0 {
		t.Errorf("%s %s has no responses", method, route)
		return
	}
	for status, responseRef := range operation.Responses.Map() {
		label := fmt.Sprintf("%s %s response %s", method, route, status)
		assertResponseHeaders(t, label, responseRef, "X-Request-ID")

		if strings.HasPrefix(status, "4") || strings.HasPrefix(status, "5") {
			if responseRef == nil ||
				responseRef.Value == nil ||
				responseRef.Value.Content.Get("application/problem+json") == nil {
				t.Errorf("%s must use application/problem+json", label)
			}
		}
		if status == "429" {
			assertResponseHeaders(t, label, responseRef, "Retry-After")
		}
	}
}

func assertResponseHeaders(
	t *testing.T,
	label string,
	responseRef *openapi3.ResponseRef,
	headers ...string,
) {
	t.Helper()

	if responseRef == nil || responseRef.Value == nil {
		t.Errorf("%s is unresolved", label)
		return
	}
	for _, header := range headers {
		if responseRef.Value.Headers[header] == nil {
			t.Errorf("%s is missing %s", label, header)
		}
	}
}

func hasParameter(
	operation *openapi3.Operation,
	name string,
	location string,
) bool {
	for _, parameterRef := range operation.Parameters {
		if parameterRef != nil &&
			parameterRef.Value != nil &&
			parameterRef.Value.Name == name &&
			parameterRef.Value.In == location &&
			parameterRef.Value.Required {
			return true
		}
	}
	return false
}

func hasSecurityScheme(operation *openapi3.Operation, name string) bool {
	if operation.Security == nil {
		return false
	}
	for _, requirement := range *operation.Security {
		if _, exists := requirement[name]; exists {
			return true
		}
	}
	return false
}

func numericExtension(value any) int {
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case int:
		return typed
	case json.Number:
		number, _ := strconv.Atoi(typed.String())
		return number
	default:
		return 0
	}
}

func schemaProperty(
	schemaRef *openapi3.SchemaRef,
	property string,
) *openapi3.Schema {
	if schemaRef == nil || schemaRef.Value == nil {
		return nil
	}
	propertyRef := schemaRef.Value.Properties[property]
	if propertyRef == nil {
		return nil
	}
	return propertyRef.Value
}

func sortedOperationSet(operations map[string]struct{}) []string {
	inventory := make([]string, 0, len(operations))
	for operation := range operations {
		inventory = append(inventory, operation)
	}
	sort.Strings(inventory)
	return inventory
}

func operationSet(operations ...string) map[string]struct{} {
	result := make(map[string]struct{}, len(operations))
	for _, operation := range operations {
		result[operation] = struct{}{}
	}
	return result
}

func readContractFixture(t *testing.T, filename string) []byte {
	t.Helper()

	repositoryRoot := contractRepositoryRoot(t)
	fixturePath := filepath.Join(repositoryRoot, "contracts", "fixtures", filename)
	raw, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture %s: %v", fixturePath, err)
	}
	return raw
}

func contractRepositoryRoot(t *testing.T) string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve contract test source location")
	}
	return filepath.Clean(filepath.Join(
		filepath.Dir(currentFile),
		"..",
		"..",
		"..",
		"..",
	))
}
