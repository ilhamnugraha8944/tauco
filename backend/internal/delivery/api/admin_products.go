package api

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	catalogapp "github.com/ilhamnugraha8944/tauco/backend/internal/catalog/application"
	contentapp "github.com/ilhamnugraha8944/tauco/backend/internal/content/application"
)

type AdminProductServer struct {
	StrictServerInterface
	products *catalogapp.AdminProductService
}

func NewAdminProductServer(products *catalogapp.AdminProductService) (*AdminProductServer, error) {
	if products == nil {
		return nil, errors.New("admin product server requires service")
	}
	return &AdminProductServer{products: products}, nil
}

func RegisterSafeAdminProductHandlers(router gin.IRouter, server *AdminProductServer, auth *AdminAuthHandler) {
	handler := NewSafeStrictHandler(server, nil)
	options := NewSafeGinServerOptions("")
	wrapper := ServerInterfaceWrapper{Handler: handler, ErrorHandler: options.ErrorHandler}
	read := []gin.HandlerFunc{auth.require(true, "product.read")}
	write := func(permission string) []gin.HandlerFunc {
		return []gin.HandlerFunc{auth.browserMutation(), auth.require(true, permission), auth.csrf()}
	}
	router.GET("/api/v1/admin/products", appendRouteMiddleware(append(read, rejectUnknownQuery("cursor", "limit")), wrapper.AdminListProducts)...)
	router.POST("/api/v1/admin/products", appendRouteMiddleware(write("product.write"), wrapper.AdminCreateProduct)...)
	router.GET("/api/v1/admin/products/:id", appendRouteMiddleware(append(read, rejectUnknownQuery()), wrapper.AdminGetProduct)...)
	router.PATCH("/api/v1/admin/products/:id", appendRouteMiddleware(write("product.write"), wrapper.AdminUpdateProduct)...)
	router.POST("/api/v1/admin/products/:id/drafts", appendRouteMiddleware(write("product.write"), wrapper.AdminCreateProductDraft)...)
	router.GET("/api/v1/admin/products/:id/revisions/:revisionId", appendRouteMiddleware(append(read, rejectUnknownQuery()), wrapper.AdminGetProductRevision)...)
	router.POST("/api/v1/admin/products/:id/revisions/:revisionId/publish", appendRouteMiddleware(write("product.publish"), wrapper.AdminPublishProductRevision)...)
	router.POST("/api/v1/admin/products/:id/unpublish", appendRouteMiddleware(write("product.publish"), wrapper.AdminUnpublishProduct)...)
	router.POST("/api/v1/admin/products/:id/archive", appendRouteMiddleware(write("product.write"), wrapper.AdminArchiveProduct)...)
	router.POST("/api/v1/admin/products/:id/unarchive", appendRouteMiddleware(write("product.write"), wrapper.AdminUnarchiveProduct)...)
}

func (server *AdminProductServer) AdminListProducts(ctx context.Context, request AdminListProductsRequestObject) (AdminListProductsResponseObject, error) {
	requestID := publicRequestID(request.Params.XRequestID)
	result, err := server.products.List(ctx, request.Params.Cursor, optionalLimit(request.Params.Limit))
	if errors.Is(err, catalogapp.ErrInvalidCursor) || errors.Is(err, catalogapp.ErrInvalidPageLimit) {
		return AdminListProducts400ApplicationProblemPlusJSONResponse{adminBadRequest(requestID, "/api/v1/admin/products")}, nil
	}
	if err != nil {
		return AdminListProducts500ApplicationProblemPlusJSONResponse{adminInternal(requestID, "/api/v1/admin/products")}, nil
	}
	data := make([]AdminProduct, 0, len(result.Products))
	for _, product := range result.Products {
		data = append(data, productDTO(product))
	}
	meta, err := NewListResponseMeta(requestID, result.NextCursor, result.HasMore, result.Limit)
	if err != nil {
		return nil, err
	}
	return AdminListProducts200JSONResponse{AdminProductListOKJSONResponse{Body: AdminProductListResponse{Data: data, Meta: meta}, Headers: AdminProductListOKResponseHeaders{CacheControl: "no-store", XRequestID: requestID}}}, nil
}

func (server *AdminProductServer) AdminCreateProduct(ctx context.Context, request AdminCreateProductRequestObject) (AdminCreateProductResponseObject, error) {
	requestID := publicRequestID(request.Params.XRequestID)
	instance := "/api/v1/admin/products"
	if request.Body == nil {
		return AdminCreateProduct422ApplicationProblemPlusJSONResponse{adminValidation(requestID, instance, "Identitas produk wajib diisi.")}, nil
	}
	product, err := server.products.Create(ctx, string(request.Body.Slug), request.Body.Sku, request.Body.SortOrder, currentPrincipalFromContext(ctx))
	if errors.Is(err, catalogapp.ErrInvalidProduct) {
		return AdminCreateProduct422ApplicationProblemPlusJSONResponse{adminValidation(requestID, instance, "Slug, SKU, atau urutan tidak valid.")}, nil
	}
	if errors.Is(err, catalogapp.ErrProductConflict) {
		return AdminCreateProduct409ApplicationProblemPlusJSONResponse{contentConflict(requestID, instance, "Slug atau SKU sudah digunakan.")}, nil
	}
	if err != nil {
		return AdminCreateProduct500ApplicationProblemPlusJSONResponse{adminInternal(requestID, instance)}, nil
	}
	return AdminCreateProduct201JSONResponse{AdminProductCreatedJSONResponse{Body: AdminProductResponse{Data: productDTO(product), Meta: mustResponseMeta(requestID)}, Headers: AdminProductCreatedResponseHeaders{CacheControl: "no-store", ETag: contentapp.RevisionETag(product.CurrentRevisionID()), XRequestID: requestID}}}, nil
}

func (server *AdminProductServer) AdminGetProduct(ctx context.Context, request AdminGetProductRequestObject) (AdminGetProductResponseObject, error) {
	requestID, id := publicRequestID(request.Params.XRequestID), request.Id.String()
	product, err := server.products.Get(ctx, id)
	if errors.Is(err, catalogapp.ErrAdminProductNotFound) {
		return AdminGetProduct404ApplicationProblemPlusJSONResponse{adminNotFound(requestID, "/api/v1/admin/products/"+id)}, nil
	}
	if err != nil {
		return AdminGetProduct500ApplicationProblemPlusJSONResponse{adminInternal(requestID, "/api/v1/admin/products/"+id)}, nil
	}
	return AdminGetProduct200JSONResponse{AdminProductOKJSONResponse{Body: AdminProductResponse{Data: productDTO(product), Meta: mustResponseMeta(requestID)}, Headers: AdminProductOKResponseHeaders{CacheControl: "no-store", ETag: contentapp.RevisionETag(product.CurrentRevisionID()), XRequestID: requestID}}}, nil
}

func (server *AdminProductServer) AdminUpdateProduct(ctx context.Context, request AdminUpdateProductRequestObject) (AdminUpdateProductResponseObject, error) {
	requestID, id := publicRequestID(request.Params.XRequestID), request.Id.String()
	instance := "/api/v1/admin/products/" + id
	if request.Body == nil {
		return AdminUpdateProduct422ApplicationProblemPlusJSONResponse{adminValidation(requestID, instance, "Perubahan identitas wajib diisi.")}, nil
	}
	var slug *string
	if request.Body.Slug != nil {
		value := string(*request.Body.Slug)
		slug = &value
	}
	product, err := server.products.Update(ctx, id, request.Params.IfMatch, slug, request.Body.Sku, request.Body.SortOrder, currentPrincipalFromContext(ctx))
	if errors.Is(err, catalogapp.ErrAdminProductNotFound) {
		return AdminUpdateProduct404ApplicationProblemPlusJSONResponse{adminNotFound(requestID, instance)}, nil
	}
	if errors.Is(err, catalogapp.ErrProductPrecondition) {
		return AdminUpdateProduct412ApplicationProblemPlusJSONResponse{contentPrecondition(requestID, instance)}, nil
	}
	if errors.Is(err, catalogapp.ErrProductConflict) {
		return AdminUpdateProduct409ApplicationProblemPlusJSONResponse{contentConflict(requestID, instance, "Slug yang pernah dipublikasikan tidak dapat diubah, atau identitas sudah digunakan.")}, nil
	}
	if errors.Is(err, catalogapp.ErrInvalidProduct) {
		return AdminUpdateProduct422ApplicationProblemPlusJSONResponse{adminValidation(requestID, instance, "Perubahan identitas tidak valid.")}, nil
	}
	if err != nil {
		return AdminUpdateProduct500ApplicationProblemPlusJSONResponse{adminInternal(requestID, instance)}, nil
	}
	return AdminUpdateProduct200JSONResponse{AdminProductOKJSONResponse{Body: AdminProductResponse{Data: productDTO(product), Meta: mustResponseMeta(requestID)}, Headers: AdminProductOKResponseHeaders{CacheControl: "no-store", ETag: contentapp.RevisionETag(product.CurrentRevisionID()), XRequestID: requestID}}}, nil
}

func (server *AdminProductServer) AdminCreateProductDraft(ctx context.Context, request AdminCreateProductDraftRequestObject) (AdminCreateProductDraftResponseObject, error) {
	requestID, id := publicRequestID(request.Params.XRequestID), request.Id.String()
	instance := "/api/v1/admin/products/" + id + "/drafts"
	if request.Body == nil {
		return AdminCreateProductDraft422ApplicationProblemPlusJSONResponse{adminValidation(requestID, instance, "Konten produk wajib diisi.")}, nil
	}
	product, err := server.products.Get(ctx, id)
	if errors.Is(err, catalogapp.ErrAdminProductNotFound) {
		return AdminCreateProductDraft404ApplicationProblemPlusJSONResponse{adminNotFound(requestID, instance)}, nil
	}
	if err != nil {
		return AdminCreateProductDraft500ApplicationProblemPlusJSONResponse{adminInternal(requestID, instance)}, nil
	}
	raw, err := validateProductContent(product.Slug, request.Body.Content)
	if err != nil {
		return AdminCreateProductDraft422ApplicationProblemPlusJSONResponse{adminValidation(requestID, instance, "Konten produk tidak memenuhi kontrak atau canonical produk.")}, nil
	}
	revision, err := server.products.SaveDraft(ctx, id, request.Params.IfMatch, request.Body.BaseRevisionId.String(), currentPrincipalFromContext(ctx), raw)
	if errors.Is(err, catalogapp.ErrAdminProductNotFound) {
		return AdminCreateProductDraft404ApplicationProblemPlusJSONResponse{adminNotFound(requestID, instance)}, nil
	}
	if errors.Is(err, catalogapp.ErrProductPrecondition) {
		return AdminCreateProductDraft412ApplicationProblemPlusJSONResponse{contentPrecondition(requestID, instance)}, nil
	}
	if errors.Is(err, catalogapp.ErrInvalidProduct) || errors.Is(err, catalogapp.ErrProductConflict) {
		return AdminCreateProductDraft422ApplicationProblemPlusJSONResponse{adminValidation(requestID, instance, "Konten atau state produk tidak valid.")}, nil
	}
	if err != nil {
		return AdminCreateProductDraft500ApplicationProblemPlusJSONResponse{adminInternal(requestID, instance)}, nil
	}
	dto, err := productRevisionDTO(revision)
	if err != nil {
		return nil, err
	}
	return AdminCreateProductDraft201JSONResponse{AdminRevisionCreatedJSONResponse{Body: AdminRevisionResponse{Data: dto, Meta: mustResponseMeta(requestID)}, Headers: AdminRevisionCreatedResponseHeaders{CacheControl: "no-store", ETag: contentapp.RevisionETag(revision.ID), XRequestID: requestID}}}, nil
}

func (server *AdminProductServer) AdminGetProductRevision(ctx context.Context, request AdminGetProductRevisionRequestObject) (AdminGetProductRevisionResponseObject, error) {
	requestID, id, revisionID := publicRequestID(request.Params.XRequestID), request.Id.String(), request.RevisionId.String()
	instance := "/api/v1/admin/products/" + id + "/revisions/" + revisionID
	revision, err := server.products.Revision(ctx, id, revisionID)
	if errors.Is(err, catalogapp.ErrProductRevisionNotFound) {
		return AdminGetProductRevision404ApplicationProblemPlusJSONResponse{adminNotFound(requestID, instance)}, nil
	}
	if err != nil {
		return AdminGetProductRevision500ApplicationProblemPlusJSONResponse{adminInternal(requestID, instance)}, nil
	}
	dto, err := productRevisionDTO(revision)
	if err != nil {
		return nil, err
	}
	return AdminGetProductRevision200JSONResponse{AdminRevisionOKJSONResponse{Body: AdminRevisionResponse{Data: dto, Meta: mustResponseMeta(requestID)}, Headers: AdminRevisionOKResponseHeaders{CacheControl: "no-store", ETag: contentapp.RevisionETag(revision.ID), XRequestID: requestID}}}, nil
}

func (server *AdminProductServer) AdminPublishProductRevision(ctx context.Context, request AdminPublishProductRevisionRequestObject) (AdminPublishProductRevisionResponseObject, error) {
	requestID, id, revisionID := publicRequestID(request.Params.XRequestID), request.Id.String(), request.RevisionId.String()
	instance := "/api/v1/admin/products/" + id + "/revisions/" + revisionID + "/publish"
	revision, err := server.products.Publish(ctx, id, revisionID, request.Params.IfMatch, currentPrincipalFromContext(ctx))
	if errors.Is(err, catalogapp.ErrAdminProductNotFound) || errors.Is(err, catalogapp.ErrProductRevisionNotFound) {
		return AdminPublishProductRevision404ApplicationProblemPlusJSONResponse{adminNotFound(requestID, instance)}, nil
	}
	if errors.Is(err, catalogapp.ErrProductPrecondition) {
		return AdminPublishProductRevision412ApplicationProblemPlusJSONResponse{contentPrecondition(requestID, instance)}, nil
	}
	if errors.Is(err, catalogapp.ErrProductConflict) || errors.Is(err, catalogapp.ErrProductMediaNotReady) {
		return AdminPublishProductRevision409ApplicationProblemPlusJSONResponse{contentConflict(requestID, instance, "Produk diarsipkan atau media belum ready.")}, nil
	}
	if err != nil {
		return AdminPublishProductRevision500ApplicationProblemPlusJSONResponse{adminInternal(requestID, instance)}, nil
	}
	dto, err := productRevisionDTO(revision)
	if err != nil {
		return nil, err
	}
	return AdminPublishProductRevision200JSONResponse{AdminRevisionOKJSONResponse{Body: AdminRevisionResponse{Data: dto, Meta: mustResponseMeta(requestID)}, Headers: AdminRevisionOKResponseHeaders{CacheControl: "no-store", ETag: contentapp.RevisionETag(revision.ID), XRequestID: requestID}}}, nil
}

func (server *AdminProductServer) AdminUnpublishProduct(ctx context.Context, request AdminUnpublishProductRequestObject) (AdminUnpublishProductResponseObject, error) {
	requestID, id := publicRequestID(request.Params.XRequestID), request.Id.String()
	instance := "/api/v1/admin/products/" + id + "/unpublish"
	err := server.products.Unpublish(ctx, id, request.Params.IfMatch, currentPrincipalFromContext(ctx))
	if errors.Is(err, catalogapp.ErrAdminProductNotFound) {
		return AdminUnpublishProduct404ApplicationProblemPlusJSONResponse{adminNotFound(requestID, instance)}, nil
	}
	if errors.Is(err, catalogapp.ErrProductPrecondition) {
		return AdminUnpublishProduct412ApplicationProblemPlusJSONResponse{contentPrecondition(requestID, instance)}, nil
	}
	if err != nil {
		return AdminUnpublishProduct500ApplicationProblemPlusJSONResponse{adminInternal(requestID, instance)}, nil
	}
	return AdminUnpublishProduct204Response{Headers: AdminNoContentResponseHeaders{CacheControl: "no-store", XRequestID: requestID}}, nil
}

func (server *AdminProductServer) AdminArchiveProduct(ctx context.Context, request AdminArchiveProductRequestObject) (AdminArchiveProductResponseObject, error) {
	requestID, id := publicRequestID(request.Params.XRequestID), request.Id.String()
	instance := "/api/v1/admin/products/" + id + "/archive"
	err := server.products.Archive(ctx, id, request.Params.IfMatch, currentPrincipalFromContext(ctx))
	if errors.Is(err, catalogapp.ErrAdminProductNotFound) {
		return AdminArchiveProduct404ApplicationProblemPlusJSONResponse{adminNotFound(requestID, instance)}, nil
	}
	if errors.Is(err, catalogapp.ErrProductPrecondition) {
		return AdminArchiveProduct412ApplicationProblemPlusJSONResponse{contentPrecondition(requestID, instance)}, nil
	}
	if errors.Is(err, catalogapp.ErrProductConflict) {
		return AdminArchiveProduct409ApplicationProblemPlusJSONResponse{contentConflict(requestID, instance, "Unpublish produk sebelum archive.")}, nil
	}
	if err != nil {
		return AdminArchiveProduct500ApplicationProblemPlusJSONResponse{adminInternal(requestID, instance)}, nil
	}
	return AdminArchiveProduct204Response{Headers: AdminNoContentResponseHeaders{CacheControl: "no-store", XRequestID: requestID}}, nil
}

func (server *AdminProductServer) AdminUnarchiveProduct(ctx context.Context, request AdminUnarchiveProductRequestObject) (AdminUnarchiveProductResponseObject, error) {
	requestID, id := publicRequestID(request.Params.XRequestID), request.Id.String()
	instance := "/api/v1/admin/products/" + id + "/unarchive"
	err := server.products.Unarchive(ctx, id, request.Params.IfMatch, currentPrincipalFromContext(ctx))
	if errors.Is(err, catalogapp.ErrAdminProductNotFound) {
		return AdminUnarchiveProduct404ApplicationProblemPlusJSONResponse{adminNotFound(requestID, instance)}, nil
	}
	if errors.Is(err, catalogapp.ErrProductPrecondition) {
		return AdminUnarchiveProduct412ApplicationProblemPlusJSONResponse{contentPrecondition(requestID, instance)}, nil
	}
	if err != nil {
		return AdminUnarchiveProduct500ApplicationProblemPlusJSONResponse{adminInternal(requestID, instance)}, nil
	}
	return AdminUnarchiveProduct204Response{Headers: AdminNoContentResponseHeaders{CacheControl: "no-store", XRequestID: requestID}}, nil
}

func validateProductContent(slug string, content ProductDetail) (json.RawMessage, error) {
	raw, err := json.Marshal(content)
	if err != nil {
		return nil, err
	}
	var document any
	if json.Unmarshal(raw, &document) != nil {
		return nil, errors.New("invalid product JSON")
	}
	spec, err := GetSpec()
	if err != nil {
		return nil, err
	}
	schema := spec.Components.Schemas["ProductDetail"]
	if schema == nil || schema.Value == nil {
		return nil, errors.New("missing ProductDetail schema")
	}
	if err := schema.Value.VisitJSON(document); err != nil {
		return nil, err
	}
	if string(content.Slug) != slug || content.Metadata.CanonicalPath != "/produk/"+slug {
		return nil, errors.New("product identity mismatch")
	}
	return raw, nil
}

func productDTO(product catalogapp.AdminProduct) AdminProduct {
	summaries := make([]AdminRevisionSummary, 0, len(product.Revisions))
	for _, item := range product.Revisions {
		var actor *uuid.UUID
		if item.CreatedBy != nil {
			value := mustUUID(*item.CreatedBy)
			actor = &value
		}
		summaries = append(summaries, AdminRevisionSummary{Id: mustUUID(item.ID), RevisionNumber: item.Number, Status: AdminRevisionSummaryStatus(item.Status), CreatedBy: actor, CreatedAt: item.CreatedAt, PublishedAt: item.PublishedAt})
	}
	var published *uuid.UUID
	if product.PublishedRevisionID != nil {
		value := mustUUID(*product.PublishedRevisionID)
		published = &value
	}
	return AdminProduct{Id: mustUUID(product.ID), Slug: Slug(product.Slug), Sku: product.SKU, SortOrder: product.SortOrder, PublishedRevisionId: published, ArchivedAt: product.ArchivedAt, UpdatedAt: product.UpdatedAt, Revisions: summaries}
}
func productRevisionDTO(revision contentapp.AdminRevision) (AdminRevision, error) {
	value, err := decodeStrictJSON[ProductDetail](revision.Content)
	if err != nil {
		return AdminRevision{}, err
	}
	union := AdminRevisionContent{}
	if err := union.FromProductDetail(value); err != nil {
		return AdminRevision{}, err
	}
	var actor *uuid.UUID
	if revision.CreatedBy != nil {
		parsed := mustUUID(*revision.CreatedBy)
		actor = &parsed
	}
	return AdminRevision{Id: mustUUID(revision.ID), OwnerId: mustUUID(revision.OwnerID), RevisionNumber: revision.Number, Status: AdminRevisionStatus(revision.Status), SchemaVersion: revision.SchemaVersion, Content: union, CreatedBy: actor, CreatedAt: revision.CreatedAt, PublishedAt: revision.PublishedAt}, nil
}

var _ StrictServerInterface = (*AdminProductServer)(nil)
