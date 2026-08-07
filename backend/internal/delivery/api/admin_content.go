package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	contentapp "github.com/ilhamnugraha8944/tauco/backend/internal/content/application"
)

type AdminContentServer struct {
	StrictServerInterface
	content *contentapp.AdminService
}

func NewAdminContentServer(content *contentapp.AdminService) (*AdminContentServer, error) {
	if content == nil {
		return nil, errors.New("admin content server requires service")
	}
	return &AdminContentServer{content: content}, nil
}

func RegisterSafeAdminContentHandlers(router gin.IRouter, server *AdminContentServer, auth *AdminAuthHandler) {
	handler := NewSafeStrictHandler(server, nil)
	options := NewSafeGinServerOptions("")
	wrapper := ServerInterfaceWrapper{Handler: handler, ErrorHandler: options.ErrorHandler}
	read := []gin.HandlerFunc{auth.bff(), auth.require(true, "content.read"), rejectUnknownQuery()}
	write := func(permission string) []gin.HandlerFunc {
		return []gin.HandlerFunc{auth.bff(), auth.browserMutation(), auth.require(true, permission), auth.csrf()}
	}
	router.GET("/api/v1/admin/pages/:key", appendRouteMiddleware(read, wrapper.AdminGetPage)...)
	router.POST("/api/v1/admin/pages/:key/drafts", appendRouteMiddleware(write("content.write"), wrapper.AdminCreatePageDraft)...)
	router.GET("/api/v1/admin/pages/:key/revisions/:revisionId", appendRouteMiddleware(read, wrapper.AdminGetPageRevision)...)
	router.POST("/api/v1/admin/pages/:key/revisions/:revisionId/publish", appendRouteMiddleware(write("content.publish"), wrapper.AdminPublishPageRevision)...)
	router.POST("/api/v1/admin/pages/:key/unpublish", appendRouteMiddleware(write("content.publish"), wrapper.AdminUnpublishPage)...)
}

func (server *AdminContentServer) AdminGetPage(ctx context.Context, request AdminGetPageRequestObject) (AdminGetPageResponseObject, error) {
	requestID, key := publicRequestID(request.Params.XRequestID), string(request.Key)
	page, err := server.content.Get(ctx, key)
	if errors.Is(err, contentapp.ErrAdminPageNotFound) {
		return AdminGetPage404ApplicationProblemPlusJSONResponse{contentNotFound(requestID, "/api/v1/admin/pages/"+key)}, nil
	}
	if err != nil {
		return AdminGetPage500ApplicationProblemPlusJSONResponse{adminInternal(requestID, "/api/v1/admin/pages/"+key)}, nil
	}
	dto, err := pageDTO(page)
	if err != nil {
		return nil, err
	}
	return AdminGetPage200JSONResponse{AdminPageOKJSONResponse{Body: AdminPageResponse{Data: dto, Meta: mustResponseMeta(requestID)}, Headers: AdminPageOKResponseHeaders{CacheControl: "no-store", ETag: contentapp.RevisionETag(page.Latest.ID), XRequestID: requestID}}}, nil
}

func (server *AdminContentServer) AdminCreatePageDraft(ctx context.Context, request AdminCreatePageDraftRequestObject) (AdminCreatePageDraftResponseObject, error) {
	requestID, key := publicRequestID(request.Params.XRequestID), string(request.Key)
	if request.Body == nil {
		return AdminCreatePageDraft422ApplicationProblemPlusJSONResponse{adminValidation(requestID, "/api/v1/admin/pages/"+key+"/drafts", "Konten wajib diisi.")}, nil
	}
	raw, err := validateEditableContent(key, request.Body.Content)
	if err != nil {
		return AdminCreatePageDraft422ApplicationProblemPlusJSONResponse{adminValidation(requestID, "/api/v1/admin/pages/"+key+"/drafts", "Konten tidak memenuhi kontrak halaman.")}, nil
	}
	revision, err := server.content.SaveDraft(ctx, key, request.Params.IfMatch, request.Body.BaseRevisionId.String(), currentPrincipalFromContext(ctx), raw)
	if errors.Is(err, contentapp.ErrAdminPageNotFound) {
		return AdminCreatePageDraft404ApplicationProblemPlusJSONResponse{contentNotFound(requestID, "/api/v1/admin/pages/"+key)}, nil
	}
	if errors.Is(err, contentapp.ErrPrecondition) {
		return AdminCreatePageDraft412ApplicationProblemPlusJSONResponse{contentPrecondition(requestID, "/api/v1/admin/pages/"+key+"/drafts")}, nil
	}
	if errors.Is(err, contentapp.ErrInvalidPage) {
		return AdminCreatePageDraft422ApplicationProblemPlusJSONResponse{adminValidation(requestID, "/api/v1/admin/pages/"+key+"/drafts", "Konten tidak valid.")}, nil
	}
	if err != nil {
		return AdminCreatePageDraft500ApplicationProblemPlusJSONResponse{adminInternal(requestID, "/api/v1/admin/pages/"+key+"/drafts")}, nil
	}
	dto, err := revisionDTO(key, revision)
	if err != nil {
		return nil, err
	}
	return AdminCreatePageDraft201JSONResponse{AdminRevisionCreatedJSONResponse{Body: AdminRevisionResponse{Data: dto, Meta: mustResponseMeta(requestID)}, Headers: AdminRevisionCreatedResponseHeaders{CacheControl: "no-store", ETag: contentapp.RevisionETag(revision.ID), XRequestID: requestID}}}, nil
}

func (server *AdminContentServer) AdminGetPageRevision(ctx context.Context, request AdminGetPageRevisionRequestObject) (AdminGetPageRevisionResponseObject, error) {
	requestID, key, id := publicRequestID(request.Params.XRequestID), string(request.Key), request.RevisionId.String()
	revision, err := server.content.Revision(ctx, key, id)
	if errors.Is(err, contentapp.ErrAdminPageNotFound) || errors.Is(err, contentapp.ErrRevisionNotFound) {
		return AdminGetPageRevision404ApplicationProblemPlusJSONResponse{contentNotFound(requestID, "/api/v1/admin/pages/"+key+"/revisions/"+id)}, nil
	}
	if err != nil {
		return AdminGetPageRevision500ApplicationProblemPlusJSONResponse{adminInternal(requestID, "/api/v1/admin/pages/"+key+"/revisions/"+id)}, nil
	}
	dto, err := revisionDTO(key, revision)
	if err != nil {
		return nil, err
	}
	return AdminGetPageRevision200JSONResponse{AdminRevisionOKJSONResponse{Body: AdminRevisionResponse{Data: dto, Meta: mustResponseMeta(requestID)}, Headers: AdminRevisionOKResponseHeaders{CacheControl: "no-store", ETag: contentapp.RevisionETag(revision.ID), XRequestID: requestID}}}, nil
}

func (server *AdminContentServer) AdminPublishPageRevision(ctx context.Context, request AdminPublishPageRevisionRequestObject) (AdminPublishPageRevisionResponseObject, error) {
	requestID, key, id := publicRequestID(request.Params.XRequestID), string(request.Key), request.RevisionId.String()
	instance := "/api/v1/admin/pages/" + key + "/revisions/" + id + "/publish"
	revision, err := server.content.Publish(ctx, key, id, request.Params.IfMatch, currentPrincipalFromContext(ctx))
	if errors.Is(err, contentapp.ErrAdminPageNotFound) || errors.Is(err, contentapp.ErrRevisionNotFound) {
		return AdminPublishPageRevision404ApplicationProblemPlusJSONResponse{contentNotFound(requestID, instance)}, nil
	}
	if errors.Is(err, contentapp.ErrPrecondition) {
		return AdminPublishPageRevision412ApplicationProblemPlusJSONResponse{contentPrecondition(requestID, instance)}, nil
	}
	if errors.Is(err, contentapp.ErrMediaNotReady) {
		return AdminPublishPageRevision409ApplicationProblemPlusJSONResponse{contentConflict(requestID, instance, "Media yang dirujuk belum berstatus ready.")}, nil
	}
	if err != nil {
		return AdminPublishPageRevision500ApplicationProblemPlusJSONResponse{adminInternal(requestID, instance)}, nil
	}
	dto, err := revisionDTO(key, revision)
	if err != nil {
		return nil, err
	}
	return AdminPublishPageRevision200JSONResponse{AdminRevisionOKJSONResponse{Body: AdminRevisionResponse{Data: dto, Meta: mustResponseMeta(requestID)}, Headers: AdminRevisionOKResponseHeaders{CacheControl: "no-store", ETag: contentapp.RevisionETag(revision.ID), XRequestID: requestID}}}, nil
}

func (server *AdminContentServer) AdminUnpublishPage(ctx context.Context, request AdminUnpublishPageRequestObject) (AdminUnpublishPageResponseObject, error) {
	requestID, key := publicRequestID(request.Params.XRequestID), string(request.Key)
	instance := "/api/v1/admin/pages/" + key + "/unpublish"
	err := server.content.Unpublish(ctx, key, request.Params.IfMatch, currentPrincipalFromContext(ctx))
	if errors.Is(err, contentapp.ErrAdminPageNotFound) {
		return AdminUnpublishPage404ApplicationProblemPlusJSONResponse{contentNotFound(requestID, instance)}, nil
	}
	if errors.Is(err, contentapp.ErrPrecondition) {
		return AdminUnpublishPage412ApplicationProblemPlusJSONResponse{contentPrecondition(requestID, instance)}, nil
	}
	if err != nil {
		return AdminUnpublishPage500ApplicationProblemPlusJSONResponse{adminInternal(requestID, instance)}, nil
	}
	return AdminUnpublishPage204Response{Headers: AdminNoContentResponseHeaders{CacheControl: "no-store", XRequestID: requestID}}, nil
}

func validateEditableContent(key string, union AdminPageContent) (json.RawMessage, error) {
	raw, err := json.Marshal(union)
	if err != nil {
		return nil, err
	}
	var document any
	if err := json.Unmarshal(raw, &document); err != nil {
		return nil, err
	}
	schemaName, exists := map[string]string{"home": "HomeContent", "about": "AboutContent"}[key]
	if !exists {
		return nil, contentapp.ErrInvalidPage
	}
	spec, err := GetSpec()
	if err != nil {
		return nil, err
	}
	schema := spec.Components.Schemas[schemaName]
	if schema == nil || schema.Value == nil {
		return nil, errors.New("missing editable schema")
	}
	if err := schema.Value.VisitJSON(document); err != nil {
		return nil, err
	}

	var value any
	switch key {
	case "home":
		typed, err := union.AsHomeContent()
		if err != nil {
			return nil, err
		}
		if typed.Metadata.CanonicalPath != "/" {
			return nil, errors.New("invalid home canonical")
		}
		value = typed
	case "about":
		typed, err := union.AsAboutContent()
		if err != nil {
			return nil, err
		}
		if typed.Metadata.CanonicalPath != "/tentang-kami" {
			return nil, errors.New("invalid about canonical")
		}
		seen := map[string]bool{}
		for _, section := range typed.Sections {
			if seen[section.Id] {
				return nil, errors.New("duplicate section id")
			}
			seen[section.Id] = true
		}
		value = typed
	default:
		return nil, contentapp.ErrInvalidPage
	}
	_ = value
	return raw, nil
}

func pageDTO(page contentapp.AdminPage) (AdminPage, error) {
	latest, err := revisionDTO(page.Key, page.Latest)
	if err != nil {
		return AdminPage{}, err
	}
	summaries := make([]AdminRevisionSummary, 0, len(page.Revisions))
	for _, item := range page.Revisions {
		var createdBy *uuid.UUID
		if item.CreatedBy != nil {
			value := mustUUID(*item.CreatedBy)
			createdBy = &value
		}
		summaries = append(summaries, AdminRevisionSummary{Id: mustUUID(item.ID), Status: AdminRevisionSummaryStatus(item.Status), RevisionNumber: item.Number, CreatedBy: createdBy, CreatedAt: item.CreatedAt, PublishedAt: item.PublishedAt})
	}
	var published *uuid.UUID
	if page.PublishedRevisionID != nil {
		value := mustUUID(*page.PublishedRevisionID)
		published = &value
	}
	return AdminPage{Id: mustUUID(page.ID), Key: AdminPageKey(page.Key), LatestRevision: latest, PublishedRevisionId: published, Revisions: summaries, UpdatedAt: page.UpdatedAt}, nil
}

func revisionDTO(key string, revision contentapp.AdminRevision) (AdminRevision, error) {
	content := AdminRevisionContent{}
	switch key {
	case "home":
		value, err := decodeStrictJSON[HomeContent](revision.Content)
		if err != nil {
			return AdminRevision{}, err
		}
		if err := content.FromHomeContent(value); err != nil {
			return AdminRevision{}, err
		}
	case "about":
		value, err := decodeStrictJSON[AboutContent](revision.Content)
		if err != nil {
			return AdminRevision{}, err
		}
		if err := content.FromAboutContent(value); err != nil {
			return AdminRevision{}, err
		}
	default:
		return AdminRevision{}, fmt.Errorf("unsupported page %s", key)
	}
	var createdBy *uuid.UUID
	if revision.CreatedBy != nil {
		value := mustUUID(*revision.CreatedBy)
		createdBy = &value
	}
	return AdminRevision{Id: mustUUID(revision.ID), OwnerId: mustUUID(revision.OwnerID), RevisionNumber: revision.Number, Status: AdminRevisionStatus(revision.Status), SchemaVersion: revision.SchemaVersion, Content: content, CreatedBy: createdBy, CreatedAt: revision.CreatedAt, PublishedAt: revision.PublishedAt}, nil
}

func contentNotFound(requestID, instance string) NotFoundApplicationProblemPlusJSONResponse {
	return NotFoundApplicationProblemPlusJSONResponse{Body: adminProblemResponse(404, requestID, instance, "CONTENT_NOT_FOUND", "Halaman atau revision tidak ditemukan."), Headers: NotFoundResponseHeaders{CacheControl: "no-store", XRequestID: requestID}}
}
func contentPrecondition(requestID, instance string) PreconditionFailedApplicationProblemPlusJSONResponse {
	return PreconditionFailedApplicationProblemPlusJSONResponse{Body: adminProblemResponse(412, requestID, instance, "PRECONDITION_FAILED", "Konten sudah berubah. Muat ulang sebelum melanjutkan."), Headers: PreconditionFailedResponseHeaders{CacheControl: "no-store", XRequestID: requestID}}
}
func contentConflict(requestID, instance, detail string) ConflictApplicationProblemPlusJSONResponse {
	return ConflictApplicationProblemPlusJSONResponse{Body: adminProblemResponse(409, requestID, instance, "CONTENT_STATE_CONFLICT", detail), Headers: ConflictResponseHeaders{CacheControl: "no-store", XRequestID: requestID}}
}

var _ StrictServerInterface = (*AdminContentServer)(nil)
