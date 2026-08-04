package api

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	catalogapp "github.com/ilhamnugraha8944/tauco/backend/internal/catalog/application"
	mediaapp "github.com/ilhamnugraha8944/tauco/backend/internal/media/application"
)

const mediaMultipartLimit = mediaapp.MaxUploadBytes + 64*1024

type AdminMediaServer struct {
	StrictServerInterface
	admin    *mediaapp.AdminService
	ingestor *mediaapp.Ingestor
	store    mediaapp.ObjectStore
}

func NewAdminMediaServer(admin *mediaapp.AdminService, ingestor *mediaapp.Ingestor, store mediaapp.ObjectStore) (*AdminMediaServer, error) {
	if admin == nil || ingestor == nil || store == nil {
		return nil, errors.New("admin media server requires service, ingestor, and storage")
	}
	return &AdminMediaServer{admin: admin, ingestor: ingestor, store: store}, nil
}

func RegisterSafeMediaHandlers(router gin.IRouter, server *AdminMediaServer, auth *AdminAuthHandler, publicLimit gin.HandlerFunc) {
	handler := NewSafeStrictHandler(server, nil)
	options := NewSafeGinServerOptions("")
	wrapper := ServerInterfaceWrapper{Handler: handler, ErrorHandler: options.ErrorHandler}

	read := []gin.HandlerFunc{auth.require(true, "media.read"), rejectUnknownQuery("cursor", "limit")}
	write := []gin.HandlerFunc{auth.browserMutation(), auth.require(true, "media.write"), auth.csrf()}
	router.GET("/api/v1/admin/media", appendRouteMiddleware(read, wrapper.AdminListMedia)...)
	router.POST("/api/v1/admin/media", appendRouteMiddleware(write, limitRequestBody(mediaMultipartLimit), wrapper.AdminUploadMedia)...)
	router.GET("/api/v1/admin/media/:id", appendRouteMiddleware([]gin.HandlerFunc{auth.require(true, "media.read"), rejectUnknownQuery()}, wrapper.AdminGetMedia)...)
	router.POST("/api/v1/admin/media/:id/retry", appendRouteMiddleware(write, wrapper.AdminRetryMedia)...)
	router.GET("/api/v1/media/:id/display.webp", publicLimit, rejectUnknownQuery(), wrapper.GetMediaDisplay)
	router.GET("/api/v1/media/:id/variants/:width.webp", publicLimit, rejectUnknownQuery(), wrapper.GetMediaVariant)
}

func limitRequestBody(limit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		c.Next()
	}
}

func (server *AdminMediaServer) AdminListMedia(ctx context.Context, request AdminListMediaRequestObject) (AdminListMediaResponseObject, error) {
	requestID := publicRequestID(request.Params.XRequestID)
	limit := optionalLimit(request.Params.Limit)
	result, err := server.admin.List(ctx, request.Params.Cursor, limit)
	if errors.Is(err, mediaapp.ErrInvalidCursor) || errors.Is(err, catalogapp.ErrInvalidPageLimit) {
		return AdminListMedia400ApplicationProblemPlusJSONResponse{adminBadRequest(requestID, "/api/v1/admin/media")}, nil
	}
	if err != nil {
		return AdminListMedia500ApplicationProblemPlusJSONResponse{adminInternal(requestID, "/api/v1/admin/media")}, nil
	}
	data := make([]AdminMedia, 0, len(result.Assets))
	for _, asset := range result.Assets {
		data = append(data, mediaDTO(asset))
	}
	meta, err := NewListResponseMeta(requestID, result.NextCursor, result.HasMore, result.Limit)
	if err != nil {
		return nil, err
	}
	return AdminListMedia200JSONResponse{AdminMediaListOKJSONResponse{
		Body:    AdminMediaListResponse{Data: data, Meta: meta},
		Headers: AdminMediaListOKResponseHeaders{CacheControl: "no-store", XRequestID: requestID},
	}}, nil
}

func (server *AdminMediaServer) AdminUploadMedia(ctx context.Context, request AdminUploadMediaRequestObject) (AdminUploadMediaResponseObject, error) {
	requestID := publicRequestID(request.Params.XRequestID)
	source, alt, decorative, err := readMediaMultipart(request.Body)
	if err != nil {
		if errors.Is(err, errMediaTooLarge) {
			return AdminUploadMedia413ApplicationProblemPlusJSONResponse{adminPayloadTooLarge(requestID, "/api/v1/admin/media")}, nil
		}
		return AdminUploadMedia422ApplicationProblemPlusJSONResponse{adminValidation(requestID, "/api/v1/admin/media", err.Error())}, nil
	}
	id, _, err := server.ingestor.Ingest(ctx, source, alt, decorative)
	if err != nil {
		if strings.Contains(err.Error(), "only JPEG") || strings.Contains(err.Error(), "animated WebP") {
			return AdminUploadMedia415ApplicationProblemPlusJSONResponse{adminUnsupported(requestID, "/api/v1/admin/media")}, nil
		}
		return AdminUploadMedia422ApplicationProblemPlusJSONResponse{adminValidation(requestID, "/api/v1/admin/media", "Gambar atau metadata tidak valid.")}, nil
	}
	asset, err := server.admin.Get(ctx, id)
	if err != nil {
		return AdminUploadMedia500ApplicationProblemPlusJSONResponse{adminInternal(requestID, "/api/v1/admin/media")}, nil
	}
	return AdminUploadMedia202JSONResponse{AdminMediaAcceptedJSONResponse{
		Body:    AdminMediaResponse{Data: mediaDTO(asset), Meta: mustResponseMeta(requestID)},
		Headers: AdminMediaAcceptedResponseHeaders{CacheControl: "no-store", XRequestID: requestID},
	}}, nil
}

func (server *AdminMediaServer) AdminGetMedia(ctx context.Context, request AdminGetMediaRequestObject) (AdminGetMediaResponseObject, error) {
	requestID, id := publicRequestID(request.Params.XRequestID), request.Id.String()
	asset, err := server.admin.Get(ctx, id)
	if errors.Is(err, mediaapp.ErrAssetNotFound) {
		return AdminGetMedia404ApplicationProblemPlusJSONResponse{adminNotFound(requestID, "/api/v1/admin/media/"+id)}, nil
	}
	if err != nil {
		return AdminGetMedia500ApplicationProblemPlusJSONResponse{adminInternal(requestID, "/api/v1/admin/media/"+id)}, nil
	}
	return AdminGetMedia200JSONResponse{AdminMediaOKJSONResponse{
		Body:    AdminMediaResponse{Data: mediaDTO(asset), Meta: mustResponseMeta(requestID)},
		Headers: AdminMediaOKResponseHeaders{CacheControl: "no-store", XRequestID: requestID},
	}}, nil
}

func (server *AdminMediaServer) AdminRetryMedia(ctx context.Context, request AdminRetryMediaRequestObject) (AdminRetryMediaResponseObject, error) {
	requestID, id := publicRequestID(request.Params.XRequestID), request.Id.String()
	actor := currentPrincipalFromContext(ctx)
	asset, err := server.admin.Retry(ctx, id, actor)
	if errors.Is(err, mediaapp.ErrAssetNotFound) {
		return AdminRetryMedia404ApplicationProblemPlusJSONResponse{adminNotFound(requestID, "/api/v1/admin/media/"+id+"/retry")}, nil
	}
	if errors.Is(err, mediaapp.ErrRetryConflict) {
		return AdminRetryMedia409ApplicationProblemPlusJSONResponse{adminConflict(requestID, "/api/v1/admin/media/"+id+"/retry")}, nil
	}
	if err != nil {
		return AdminRetryMedia500ApplicationProblemPlusJSONResponse{adminInternal(requestID, "/api/v1/admin/media/"+id+"/retry")}, nil
	}
	return AdminRetryMedia202JSONResponse{AdminMediaAcceptedJSONResponse{
		Body:    AdminMediaResponse{Data: mediaDTO(asset), Meta: mustResponseMeta(requestID)},
		Headers: AdminMediaAcceptedResponseHeaders{CacheControl: "no-store", XRequestID: requestID},
	}}, nil
}

func (server *AdminMediaServer) GetMediaDisplay(ctx context.Context, request GetMediaDisplayRequestObject) (GetMediaDisplayResponseObject, error) {
	return server.mediaBinary(ctx, request.Id.String(), nil, request.Params.XRequestID, request.Params.IfNoneMatch, "/display.webp")
}

func (server *AdminMediaServer) GetMediaVariant(ctx context.Context, request GetMediaVariantRequestObject) (GetMediaVariantResponseObject, error) {
	width := int(request.Width)
	response, err := server.mediaBinary(ctx, request.Id.String(), &width, request.Params.XRequestID, request.Params.IfNoneMatch, "/variants/"+strconv.Itoa(width)+".webp")
	if err != nil {
		return nil, err
	}
	switch value := response.(type) {
	case GetMediaDisplay200ImagewebpResponse:
		return GetMediaVariant200ImagewebpResponse{value.MediaBinaryOKImagewebpResponse}, nil
	case GetMediaDisplay304Response:
		return GetMediaVariant304Response(value), nil
	case GetMediaDisplay404ApplicationProblemPlusJSONResponse:
		return GetMediaVariant404ApplicationProblemPlusJSONResponse{value.NotFoundApplicationProblemPlusJSONResponse}, nil
	case GetMediaDisplay500ApplicationProblemPlusJSONResponse:
		return GetMediaVariant500ApplicationProblemPlusJSONResponse{value.InternalServerErrorApplicationProblemPlusJSONResponse}, nil
	default:
		return nil, fmt.Errorf("unexpected media response %T", response)
	}
}

func (server *AdminMediaServer) mediaBinary(ctx context.Context, id string, width *int, requestHeader *XRequestID, ifNoneMatch *IfNoneMatch, suffix string) (GetMediaDisplayResponseObject, error) {
	requestID := publicRequestID(requestHeader)
	variant, err := server.admin.ReadyVariant(ctx, id, width)
	instance := "/api/v1/media/" + id + suffix
	if errors.Is(err, mediaapp.ErrAssetNotFound) {
		return GetMediaDisplay404ApplicationProblemPlusJSONResponse{adminNotFound(requestID, instance)}, nil
	}
	if err != nil {
		return GetMediaDisplay500ApplicationProblemPlusJSONResponse{adminInternal(requestID, instance)}, nil
	}
	etag := `"sha256-` + variant.SHA256 + `"`
	if matchesIfNoneMatch(ifNoneMatch, etag) {
		return GetMediaDisplay304Response{Headers: NotModifiedResponseHeaders{CacheControl: PublicContentCacheControl, ETag: etag, XRequestID: requestID}}, nil
	}
	data, err := server.store.Get(ctx, variant.ObjectKey)
	if err != nil {
		return GetMediaDisplay500ApplicationProblemPlusJSONResponse{adminInternal(requestID, instance)}, nil
	}
	return GetMediaDisplay200ImagewebpResponse{MediaBinaryOKImagewebpResponse{
		Body: bytes.NewReader(data), ContentLength: int64(len(data)),
		Headers: MediaBinaryOKResponseHeaders{CacheControl: PublicContentCacheControl, ETag: etag, XRequestID: requestID},
	}}, nil
}

var errMediaTooLarge = errors.New("media upload exceeds 10 MiB")

func readMediaMultipart(reader *multipart.Reader) ([]byte, string, bool, error) {
	if reader == nil {
		return nil, "", false, errors.New("multipart body is required")
	}
	var source []byte
	var alt string
	var decorative *bool
	seen := map[string]bool{}
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, "", false, err
		}
		name := part.FormName()
		if seen[name] || (name != "file" && name != "altText" && name != "decorative") {
			return nil, "", false, errors.New("field multipart tidak valid")
		}
		seen[name] = true
		if name == "file" {
			data, readErr := io.ReadAll(io.LimitReader(part, mediaapp.MaxUploadBytes+1))
			if readErr != nil {
				return nil, "", false, readErr
			}
			if len(data) > mediaapp.MaxUploadBytes {
				return nil, "", false, errMediaTooLarge
			}
			source = data
			continue
		}
		value, readErr := io.ReadAll(io.LimitReader(part, 1025))
		if readErr != nil || len(value) > 1024 {
			return nil, "", false, errors.New("metadata media terlalu panjang")
		}
		if name == "altText" {
			alt = string(value)
		} else {
			parsed, parseErr := strconv.ParseBool(string(value))
			if parseErr != nil {
				return nil, "", false, errors.New("decorative harus boolean")
			}
			decorative = &parsed
		}
	}
	if len(source) == 0 || decorative == nil {
		return nil, "", false, errors.New("file dan decorative wajib diisi")
	}
	return source, alt, *decorative, nil
}

func mediaDTO(asset mediaapp.AdminAsset) AdminMedia {
	variants := make([]AdminMediaVariant, 0, len(asset.Variants))
	for _, variant := range asset.Variants {
		variants = append(variants, AdminMediaVariant{Width: variant.Width, Height: variant.Height, Bytes: variant.Bytes,
			Url: "/api/v1/media/" + asset.ID + "/variants/" + strconv.Itoa(variant.Width) + ".webp"})
	}
	return AdminMedia{Id: mustUUID(asset.ID), Status: AdminMediaStatus(asset.Status), MimeType: asset.MIME,
		Width: asset.Width, Height: asset.Height, Bytes: asset.Bytes, AltText: asset.AltText,
		Decorative: asset.Decorative, LastErrorCode: asset.LastErrorCode, Variants: variants,
		CreatedAt: asset.CreatedAt, UpdatedAt: asset.UpdatedAt}
}

func currentPrincipalFromContext(ctx context.Context) string {
	if ginContext, ok := ctx.(*gin.Context); ok {
		return currentPrincipal(ginContext).ID.String()
	}
	return ""
}

func mustUUID(raw string) uuid.UUID {
	value, _ := uuid.Parse(raw)
	return value
}

func mustResponseMeta(requestID string) ResponseMeta {
	value, _ := NewResponseMeta(requestID)
	return value
}

func adminProblemResponse(status int32, requestID, instance, code, detail string) Problem {
	return problem(status, "urn:tauco-cap-badak:problem:"+strings.ToLower(strings.ReplaceAll(code, "_", "-")),
		http.StatusText(int(status)), detail, code, requestID, instance)
}

func adminBadRequest(requestID, instance string) BadRequestApplicationProblemPlusJSONResponse {
	return BadRequestApplicationProblemPlusJSONResponse{Body: adminProblemResponse(400, requestID, instance, "BAD_REQUEST", "Format permintaan tidak valid."), Headers: BadRequestResponseHeaders{CacheControl: "no-store", XRequestID: requestID}}
}
func adminInternal(requestID, instance string) InternalServerErrorApplicationProblemPlusJSONResponse {
	return InternalServerErrorApplicationProblemPlusJSONResponse{Body: adminProblemResponse(500, requestID, instance, "INTERNAL_ERROR", "Permintaan tidak dapat diproses."), Headers: InternalServerErrorResponseHeaders{CacheControl: "no-store", XRequestID: requestID}}
}
func adminNotFound(requestID, instance string) NotFoundApplicationProblemPlusJSONResponse {
	return NotFoundApplicationProblemPlusJSONResponse{Body: adminProblemResponse(404, requestID, instance, "MEDIA_NOT_FOUND", "Media tidak ditemukan."), Headers: NotFoundResponseHeaders{CacheControl: "no-store", XRequestID: requestID}}
}
func adminConflict(requestID, instance string) ConflictApplicationProblemPlusJSONResponse {
	return ConflictApplicationProblemPlusJSONResponse{Body: adminProblemResponse(409, requestID, instance, "MEDIA_STATE_CONFLICT", "Hanya media gagal yang dapat diproses ulang."), Headers: ConflictResponseHeaders{CacheControl: "no-store", XRequestID: requestID}}
}
func adminPayloadTooLarge(requestID, instance string) PayloadTooLargeApplicationProblemPlusJSONResponse {
	return PayloadTooLargeApplicationProblemPlusJSONResponse{Body: adminProblemResponse(413, requestID, instance, "PAYLOAD_TOO_LARGE", "Ukuran gambar melebihi 10 MiB."), Headers: PayloadTooLargeResponseHeaders{CacheControl: "no-store", XRequestID: requestID}}
}
func adminUnsupported(requestID, instance string) UnsupportedMediaTypeApplicationProblemPlusJSONResponse {
	return UnsupportedMediaTypeApplicationProblemPlusJSONResponse{Body: adminProblemResponse(415, requestID, instance, "UNSUPPORTED_MEDIA_TYPE", "Gunakan JPEG, PNG, atau WebP statis."), Headers: UnsupportedMediaTypeResponseHeaders{CacheControl: "no-store", XRequestID: requestID}}
}
func adminValidation(requestID, instance, detail string) ValidationFailedApplicationProblemPlusJSONResponse {
	return ValidationFailedApplicationProblemPlusJSONResponse{Body: adminProblemResponse(422, requestID, instance, "VALIDATION_FAILED", detail), Headers: ValidationFailedResponseHeaders{CacheControl: "no-store", XRequestID: requestID}}
}
