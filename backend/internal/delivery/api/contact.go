package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	contactapp "github.com/ilhamnugraha8944/tauco/backend/internal/contact/application"
	contactdomain "github.com/ilhamnugraha8944/tauco/backend/internal/contact/domain"
)

const maxContactBodyBytes int64 = 32 * 1024

// RegisterSafeContactHandler adds only the B5 contact route.
func RegisterSafeContactHandler(
	router gin.IRouter,
	server StrictServerInterface,
	middlewares []StrictMiddlewareFunc,
	baseURL string,
	routeMiddleware ...gin.HandlerFunc,
) {
	handler := NewSafeStrictHandler(server, middlewares)
	options := NewSafeGinServerOptions(baseURL)
	wrapper := ServerInterfaceWrapper{Handler: handler, ErrorHandler: options.ErrorHandler}
	router.POST(
		baseURL+"/api/v1/contact-messages",
		appendRouteMiddleware(
			routeMiddleware,
			rejectUnknownQuery(),
			strictContactBody(),
			wrapper.CreateContactMessage,
		)...,
	)
}

func (server *PublicReadServer) CreateContactMessage(
	ctx context.Context,
	request CreateContactMessageRequestObject,
) (CreateContactMessageResponseObject, error) {
	requestID := publicRequestID(request.Params.XRequestID)
	if server.contacts == nil {
		return CreateContactMessage503ApplicationProblemPlusJSONResponse{
			serviceUnavailableProblem(requestID, "/api/v1/contact-messages"),
		}, nil
	}
	if request.Body == nil {
		return contactBadRequest(requestID), nil
	}

	result, err := server.contacts.Submit(ctx, contactapp.Submission{
		Message: contactdomain.Message{
			Name:           request.Body.Name,
			Email:          string(request.Body.Email),
			Phone:          request.Body.Phone,
			Subject:        contactdomain.Subject(request.Body.Subject),
			Body:           request.Body.Message,
			PrivacyConsent: bool(request.Body.PrivacyConsent),
			BotField:       request.Body.BotField,
		},
		IdempotencyKey: request.Params.IdempotencyKey,
		RequestID:      requestID,
	})
	switch {
	case errors.Is(err, contactapp.ErrInvalidSubmission):
		return contactValidationFailed(requestID), nil
	case errors.Is(err, contactapp.ErrInvalidIdempotency):
		return contactBadRequest(requestID), nil
	case errors.Is(err, contactapp.ErrIdempotencyConflict):
		return contactIdempotencyConflict(requestID), nil
	case err != nil:
		return CreateContactMessage503ApplicationProblemPlusJSONResponse{
			serviceUnavailableProblem(requestID, "/api/v1/contact-messages"),
		}, nil
	}

	meta, err := NewResponseMeta(requestID)
	if err != nil {
		return nil, err
	}
	var replayed *bool
	if result.Replayed {
		value := true
		replayed = &value
	}
	return CreateContactMessage201JSONResponse{
		Body: ContactMessageResponse{
			Data: ContactMessageResult{Status: Received},
			Meta: meta,
		},
		Headers: CreateContactMessage201ResponseHeaders{
			CacheControl:        "no-store",
			IdempotencyReplayed: replayed,
			XRequestID:          requestID,
		},
	}, nil
}

func contactBadRequest(requestID string) CreateContactMessage400ApplicationProblemPlusJSONResponse {
	return CreateContactMessage400ApplicationProblemPlusJSONResponse{
		BadRequestApplicationProblemPlusJSONResponse{
			Body: problem(http.StatusBadRequest, "urn:tauco-cap-badak:problem:bad-request",
				"Permintaan tidak valid", "Format permintaan tidak dapat diproses.",
				"BAD_REQUEST", requestID, "/api/v1/contact-messages"),
			Headers: BadRequestResponseHeaders{CacheControl: "no-store", XRequestID: requestID},
		},
	}
}

func contactValidationFailed(requestID string) CreateContactMessage422ApplicationProblemPlusJSONResponse {
	return CreateContactMessage422ApplicationProblemPlusJSONResponse{
		ValidationFailedApplicationProblemPlusJSONResponse{
			Body: problem(http.StatusUnprocessableEntity,
				"urn:tauco-cap-badak:problem:validation-failed", "Data tidak valid",
				"Periksa kembali data yang dikirim.", "VALIDATION_FAILED", requestID,
				"/api/v1/contact-messages"),
			Headers: ValidationFailedResponseHeaders{CacheControl: "no-store", XRequestID: requestID},
		},
	}
}

func contactIdempotencyConflict(requestID string) CreateContactMessage409ApplicationProblemPlusJSONResponse {
	return CreateContactMessage409ApplicationProblemPlusJSONResponse{
		IdempotencyConflictApplicationProblemPlusJSONResponse{
			Body: problem(http.StatusConflict,
				"urn:tauco-cap-badak:problem:idempotency-conflict", "Idempotency key konflik",
				"Gunakan idempotency key baru untuk payload berbeda.",
				"IDEMPOTENCY_CONFLICT", requestID, "/api/v1/contact-messages"),
			Headers: IdempotencyConflictResponseHeaders{CacheControl: "no-store", XRequestID: requestID},
		},
	}
}

func strictContactBody() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		mediaType, _, err := mime.ParseMediaType(ctx.GetHeader("Content-Type"))
		if err != nil || mediaType != "application/json" {
			writeContractProblem(ctx, http.StatusUnsupportedMediaType,
				"urn:tauco-cap-badak:problem:unsupported-media-type", "Media type tidak didukung",
				"Gunakan Content-Type application/json.", "UNSUPPORTED_MEDIA_TYPE")
			ctx.Abort()
			return
		}
		body, err := io.ReadAll(io.LimitReader(ctx.Request.Body, maxContactBodyBytes+1))
		if err != nil {
			writeContractProblem(ctx, http.StatusBadRequest,
				"urn:tauco-cap-badak:problem:bad-request", "Permintaan tidak valid",
				"Format permintaan tidak dapat diproses.", "BAD_REQUEST")
			ctx.Abort()
			return
		}
		if int64(len(body)) > maxContactBodyBytes {
			writeContractProblem(ctx, http.StatusRequestEntityTooLarge,
				"urn:tauco-cap-badak:problem:payload-too-large", "Payload terlalu besar",
				"Ukuran request melebihi batas.", "PAYLOAD_TOO_LARGE")
			ctx.Abort()
			return
		}
		decoder := json.NewDecoder(bytes.NewReader(body))
		decoder.DisallowUnknownFields()
		var payload ContactMessageRequest
		if err := decoder.Decode(&payload); err != nil {
			writeContractProblem(ctx, http.StatusBadRequest,
				"urn:tauco-cap-badak:problem:bad-request", "Permintaan tidak valid",
				"Format permintaan tidak dapat diproses.", "BAD_REQUEST")
			ctx.Abort()
			return
		}
		var trailing json.RawMessage
		if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
			writeContractProblem(ctx, http.StatusBadRequest,
				"urn:tauco-cap-badak:problem:bad-request", "Permintaan tidak valid",
				"Format permintaan tidak dapat diproses.", "BAD_REQUEST")
			ctx.Abort()
			return
		}
		ctx.Request.Body = io.NopCloser(strings.NewReader(string(body)))
		ctx.Request.ContentLength = int64(len(body))
		ctx.Next()
	}
}
