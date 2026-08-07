package api

import (
	"context"
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	auditapp "github.com/ilhamnugraha8944/tauco/backend/internal/audit/application"
	catalogapp "github.com/ilhamnugraha8944/tauco/backend/internal/catalog/application"
	contactapp "github.com/ilhamnugraha8944/tauco/backend/internal/contact/application"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

type AdminInboxActivityServer struct {
	StrictServerInterface
	inbox    *contactapp.AdminMessageService
	activity *auditapp.AdminService
}

func NewAdminInboxActivityServer(inbox *contactapp.AdminMessageService, activity *auditapp.AdminService) (*AdminInboxActivityServer, error) {
	if inbox == nil || activity == nil {
		return nil, errors.New("admin inbox/activity server requires services")
	}
	return &AdminInboxActivityServer{inbox: inbox, activity: activity}, nil
}

func RegisterSafeAdminInboxActivityHandlers(router gin.IRouter, server *AdminInboxActivityServer, auth *AdminAuthHandler) {
	handler := NewSafeStrictHandler(server, nil)
	options := NewSafeGinServerOptions("")
	wrapper := ServerInterfaceWrapper{Handler: handler, ErrorHandler: options.ErrorHandler}
	readInbox := []gin.HandlerFunc{auth.bff(), auth.require(true, "inbox.read")}
	router.GET("/api/v1/admin/contact-messages", appendRouteMiddleware(append(readInbox, rejectUnknownQuery("cursor", "limit", "status")), wrapper.AdminListContactMessages)...)
	router.GET("/api/v1/admin/contact-messages/:id", appendRouteMiddleware(append(readInbox, rejectUnknownQuery()), wrapper.AdminGetContactMessage)...)
	router.PATCH("/api/v1/admin/contact-messages/:id/status", appendRouteMiddleware([]gin.HandlerFunc{auth.bff(), auth.browserMutation(), auth.require(true, "inbox.write"), auth.csrf()}, wrapper.AdminUpdateContactMessageStatus)...)
	router.GET("/api/v1/admin/activity-logs", appendRouteMiddleware([]gin.HandlerFunc{auth.bff(), auth.require(true, "activity.read"), rejectUnknownQuery("cursor", "limit", "eventType", "entityType")}, wrapper.AdminListActivityLogs)...)
}

func (server *AdminInboxActivityServer) AdminListContactMessages(ctx context.Context, request AdminListContactMessagesRequestObject) (AdminListContactMessagesResponseObject, error) {
	requestID := publicRequestID(request.Params.XRequestID)
	var status *string
	if request.Params.Status != nil {
		value := string(*request.Params.Status)
		status = &value
	}
	result, err := server.inbox.List(ctx, request.Params.Cursor, optionalLimit(request.Params.Limit), status)
	if errors.Is(err, catalogapp.ErrInvalidCursor) || errors.Is(err, catalogapp.ErrInvalidPageLimit) || errors.Is(err, contactapp.ErrAdminMessageInvalid) {
		return AdminListContactMessages400ApplicationProblemPlusJSONResponse{adminBadRequest(requestID, "/api/v1/admin/contact-messages")}, nil
	}
	if err != nil {
		return AdminListContactMessages500ApplicationProblemPlusJSONResponse{adminInternal(requestID, "/api/v1/admin/contact-messages")}, nil
	}
	data := make([]AdminContactMessage, 0, len(result.Messages))
	for _, message := range result.Messages {
		data = append(data, contactMessageDTO(message))
	}
	meta, err := NewListResponseMeta(requestID, result.NextCursor, result.HasMore, result.Limit)
	if err != nil {
		return nil, err
	}
	return AdminListContactMessages200JSONResponse{AdminContactListOKJSONResponse{Body: AdminContactListResponse{Data: data, Meta: meta}, Headers: AdminContactListOKResponseHeaders{CacheControl: "no-store", XRequestID: requestID}}}, nil
}

func (server *AdminInboxActivityServer) AdminGetContactMessage(ctx context.Context, request AdminGetContactMessageRequestObject) (AdminGetContactMessageResponseObject, error) {
	requestID, id := publicRequestID(request.Params.XRequestID), request.Id.String()
	message, err := server.inbox.Get(ctx, id)
	if errors.Is(err, contactapp.ErrAdminMessageNotFound) {
		return AdminGetContactMessage404ApplicationProblemPlusJSONResponse{adminNotFound(requestID, "/api/v1/admin/contact-messages/"+id)}, nil
	}
	if err != nil {
		return AdminGetContactMessage500ApplicationProblemPlusJSONResponse{adminInternal(requestID, "/api/v1/admin/contact-messages/"+id)}, nil
	}
	return AdminGetContactMessage200JSONResponse{AdminContactOKJSONResponse{Body: AdminContactResponse{Data: contactMessageDTO(message), Meta: mustResponseMeta(requestID)}, Headers: AdminContactOKResponseHeaders{CacheControl: "no-store", ETag: message.ETag(), XRequestID: requestID}}}, nil
}

func (server *AdminInboxActivityServer) AdminUpdateContactMessageStatus(ctx context.Context, request AdminUpdateContactMessageStatusRequestObject) (AdminUpdateContactMessageStatusResponseObject, error) {
	requestID, id := publicRequestID(request.Params.XRequestID), request.Id.String()
	instance := "/api/v1/admin/contact-messages/" + id + "/status"
	if request.Body == nil {
		return AdminUpdateContactMessageStatus422ApplicationProblemPlusJSONResponse{adminValidation(requestID, instance, "Status wajib diisi.")}, nil
	}
	message, err := server.inbox.UpdateStatus(ctx, id, request.Params.IfMatch, string(request.Body.Status), currentPrincipalFromContext(ctx))
	if errors.Is(err, contactapp.ErrAdminMessageNotFound) {
		return AdminUpdateContactMessageStatus404ApplicationProblemPlusJSONResponse{adminNotFound(requestID, instance)}, nil
	}
	if errors.Is(err, contactapp.ErrAdminMessageConflict) {
		return AdminUpdateContactMessageStatus412ApplicationProblemPlusJSONResponse{contentPrecondition(requestID, instance)}, nil
	}
	if errors.Is(err, contactapp.ErrAdminMessageInvalid) {
		return AdminUpdateContactMessageStatus422ApplicationProblemPlusJSONResponse{adminValidation(requestID, instance, "Status tidak valid.")}, nil
	}
	if err != nil {
		return AdminUpdateContactMessageStatus500ApplicationProblemPlusJSONResponse{adminInternal(requestID, instance)}, nil
	}
	return AdminUpdateContactMessageStatus200JSONResponse{AdminContactOKJSONResponse{Body: AdminContactResponse{Data: contactMessageDTO(message), Meta: mustResponseMeta(requestID)}, Headers: AdminContactOKResponseHeaders{CacheControl: "no-store", ETag: message.ETag(), XRequestID: requestID}}}, nil
}

func (server *AdminInboxActivityServer) AdminListActivityLogs(ctx context.Context, request AdminListActivityLogsRequestObject) (AdminListActivityLogsResponseObject, error) {
	requestID := publicRequestID(request.Params.XRequestID)
	filter := auditapp.ActivityFilter{}
	if request.Params.EventType != nil {
		value := string(*request.Params.EventType)
		filter.EventType = &value
	}
	if request.Params.EntityType != nil {
		value := string(*request.Params.EntityType)
		filter.EntityType = &value
	}
	result, err := server.activity.List(ctx, request.Params.Cursor, optionalLimit(request.Params.Limit), filter)
	if errors.Is(err, catalogapp.ErrInvalidCursor) || errors.Is(err, catalogapp.ErrInvalidPageLimit) || errors.Is(err, auditapp.ErrInvalidActivityFilter) {
		return AdminListActivityLogs400ApplicationProblemPlusJSONResponse{adminBadRequest(requestID, "/api/v1/admin/activity-logs")}, nil
	}
	if err != nil {
		return AdminListActivityLogs500ApplicationProblemPlusJSONResponse{adminInternal(requestID, "/api/v1/admin/activity-logs")}, nil
	}
	data := make([]AdminActivity, 0, len(result.Activities))
	for _, activity := range result.Activities {
		data = append(data, activityDTO(activity))
	}
	meta, err := NewListResponseMeta(requestID, result.NextCursor, result.HasMore, result.Limit)
	if err != nil {
		return nil, err
	}
	return AdminListActivityLogs200JSONResponse{AdminActivityListOKJSONResponse{Body: AdminActivityListResponse{Data: data, Meta: meta}, Headers: AdminActivityListOKResponseHeaders{CacheControl: "no-store", XRequestID: requestID}}}, nil
}

func contactMessageDTO(message contactapp.AdminMessage) AdminContactMessage {
	id, _ := uuid.Parse(message.ID)
	return AdminContactMessage{Id: id, Name: message.Name, Email: openapi_types.Email(message.Email), Phone: message.Phone, Subject: message.Subject, Message: message.Message, Status: ContactMessageStatus(message.Status), CreatedAt: message.CreatedAt, UpdatedAt: message.UpdatedAt}
}

func activityDTO(activity auditapp.Activity) AdminActivity {
	id, _ := uuid.Parse(activity.ID)
	result := AdminActivity{Id: id, EventType: activity.EventType, EntityType: activity.EntityType, ActorType: AdminActivityActorType(activity.ActorType), RequestId: activity.RequestID, CreatedAt: activity.CreatedAt}
	if activity.EntityID != nil {
		value, _ := uuid.Parse(*activity.EntityID)
		result.EntityId = &value
	}
	if activity.ActorID != nil {
		value, _ := uuid.Parse(*activity.ActorID)
		result.ActorId = &value
	}
	return result
}
