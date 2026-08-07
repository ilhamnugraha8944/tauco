package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ilhamnugraha8944/tauco/backend/internal/contract/requestmeta"
)

const problemMediaType = "application/problem+json"

var errNilStrictHandlerResponse = errors.New(
	"strict handler returned no response and no error",
)

// RegisterSafeHandlers registers the generated router with safe transport and
// strict-handler error adapters. Callers must reserve generated-owned routes
// such as /health/live before invoking it.
func RegisterSafeHandlers(
	router gin.IRouter,
	server StrictServerInterface,
	middlewares []StrictMiddlewareFunc,
	baseURL string,
) {
	RegisterHandlersWithOptions(
		router,
		NewSafeStrictHandler(server, middlewares),
		NewSafeGinServerOptions(baseURL),
	)
}

// NewSafeStrictHandler is the only supported constructor for generated strict
// handlers. It replaces every generated default that would otherwise expose
// err.Error() in an application/json response.
func NewSafeStrictHandler(
	server StrictServerInterface,
	middlewares []StrictMiddlewareFunc,
) ServerInterface {
	safeMiddlewares := make(
		[]StrictMiddlewareFunc,
		0,
		len(middlewares)+1,
	)
	safeMiddlewares = append(safeMiddlewares, middlewares...)
	// oapi-codegen currently leaves a (nil, nil) result unwritten. Keep this
	// guard outermost so it also catches a middleware returning (nil, nil).
	safeMiddlewares = append(safeMiddlewares, rejectNilResponse)

	return NewStrictHandlerWithOptions(
		server,
		safeMiddlewares,
		StrictGinServerOptions{
			RequestErrorHandlerFunc: func(ctx *gin.Context, err error) {
				var tooLarge *http.MaxBytesError
				if errors.As(err, &tooLarge) {
					writeContractProblem(
						ctx,
						http.StatusRequestEntityTooLarge,
						"urn:tauco-cap-badak:problem:payload-too-large",
						"Payload terlalu besar",
						"Ukuran payload melewati batas endpoint.",
						"PAYLOAD_TOO_LARGE",
					)
					return
				}
				writeContractProblem(
					ctx,
					http.StatusBadRequest,
					"urn:tauco-cap-badak:problem:bad-request",
					"Permintaan tidak valid",
					"Format permintaan tidak dapat diproses.",
					"BAD_REQUEST",
				)
			},
			HandlerErrorFunc: func(ctx *gin.Context, _ error) {
				writeInternalProblem(ctx)
			},
			ResponseErrorHandlerFunc: func(ctx *gin.Context, _ error) {
				writeInternalProblem(ctx)
			},
		},
	)
}

func rejectNilResponse(
	next StrictHandlerFunc,
	_ string,
) StrictHandlerFunc {
	return func(
		ctx *gin.Context,
		request any,
	) (any, error) {
		response, err := next(ctx, request)
		if err == nil && response == nil {
			return nil, errNilStrictHandlerResponse
		}
		return response, err
	}
}

// NewSafeGinServerOptions returns transport options whose binder error handler
// also follows the public problem contract. The caller still owns route
// registration and BaseURL selection.
func NewSafeGinServerOptions(baseURL string) GinServerOptions {
	return GinServerOptions{
		BaseURL: baseURL,
		ErrorHandler: func(
			ctx *gin.Context,
			_ error,
			statusCode int,
		) {
			if statusCode < 400 || statusCode > 499 {
				writeInternalProblem(ctx)
				return
			}
			writeContractProblem(
				ctx,
				statusCode,
				"urn:tauco-cap-badak:problem:bad-request",
				"Permintaan tidak valid",
				"Format permintaan tidak dapat diproses.",
				"BAD_REQUEST",
			)
		},
	}
}

func writeInternalProblem(ctx *gin.Context) {
	writeContractProblem(
		ctx,
		http.StatusInternalServerError,
		"urn:tauco-cap-badak:problem:internal",
		"Terjadi kesalahan internal",
		"Permintaan tidak dapat diproses saat ini.",
		"INTERNAL_SERVER_ERROR",
	)
}

func writeContractProblem(
	ctx *gin.Context,
	status int,
	problemType string,
	title string,
	detail string,
	code string,
) {
	requestID := ctx.GetHeader(requestmeta.Header)
	if !requestmeta.Valid(requestID) {
		requestID = "request-id-unavailable"
	}

	instance := requestmeta.ProblemInstancePath(ctx.Request)

	ctx.Header(requestmeta.Header, requestID)
	ctx.Header("Cache-Control", "no-store")
	ctx.Header("Content-Type", problemMediaType)
	ctx.JSON(status, Problem{
		Code:      code,
		Detail:    detail,
		Instance:  instance,
		RequestId: requestID,
		Status:    int32(status),
		Title:     title,
		Type:      problemType,
	})
}
