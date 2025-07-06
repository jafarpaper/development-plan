package http

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	echoSwagger "github.com/swaggo/echo-swagger"

	_ "activity-log-service/docs"
	"activity-log-service/internal/application/usecase"
	"activity-log-service/internal/infrastructure/metrics"
)

type EchoServer struct {
	echo    *echo.Echo
	useCase *usecase.ActivityLogUseCase
	tracer  opentracing.Tracer
}

type ActivityLogResponse struct {
	ID               string    `json:"id" example:"550e8400e29b41d4a716446655440000"`
	ActivityName     string    `json:"activity_name" example:"user_created"`
	CompanyID        string    `json:"company_id" example:"company_123"`
	ObjectName       string    `json:"object_name" example:"user"`
	ObjectID         string    `json:"object_id" example:"user_456"`
	Changes          string    `json:"changes" example:"{\"name\": \"John Doe\"}"`
	FormattedMessage string    `json:"formatted_message" example:"User John Doe was created"`
	ActorID          string    `json:"actor_id" example:"actor_789"`
	ActorName        string    `json:"actor_name" example:"System Administrator"`
	ActorEmail       string    `json:"actor_email" example:"admin@company123.com"`
	CreatedAt        time.Time `json:"created_at" example:"2023-12-07T10:30:00Z"`
}

type CreateActivityLogRequest struct {
	ActivityName     string `json:"activity_name" validate:"required" example:"user_created"`
	CompanyID        string `json:"company_id" validate:"required" example:"company_123"`
	ObjectName       string `json:"object_name" validate:"required" example:"user"`
	ObjectID         string `json:"object_id" validate:"required" example:"user_456"`
	Changes          string `json:"changes,omitempty" example:"{\"name\": \"John Doe\"}"`
	FormattedMessage string `json:"formatted_message" validate:"required" example:"User John Doe was created"`
	ActorID          string `json:"actor_id" validate:"required" example:"actor_789"`
	ActorName        string `json:"actor_name" validate:"required" example:"System Administrator"`
	ActorEmail       string `json:"actor_email" validate:"required,email" example:"admin@company123.com"`
}

type ListActivityLogsResponse struct {
	ActivityLogs []*ActivityLogResponse `json:"activity_logs"`
	Total        int                    `json:"total" example:"150"`
	Page         int                    `json:"page" example:"1"`
	Limit        int                    `json:"limit" example:"10"`
}

type ErrorResponse struct {
	Error   string `json:"error" example:"Invalid request parameters"`
	Message string `json:"message,omitempty" example:"company_id is required"`
	Code    int    `json:"code" example:"400"`
}

type HealthResponse struct {
	Status  string `json:"status" example:"ok"`
	Service string `json:"service" example:"activity-log-service"`
	Version string `json:"version" example:"1.0.0"`
}

func NewEchoServer(useCase *usecase.ActivityLogUseCase, tracer opentracing.Tracer) *EchoServer {
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())
	e.Use(middleware.Secure())
	e.Use(middleware.RequestID())

	// Distributed tracing middleware
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			spanCtx, _ := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(c.Request().Header))
			span := tracer.StartSpan(c.Request().Method+" "+c.Path(), ext.RPCServerOption(spanCtx))
			defer span.Finish()

			ext.HTTPMethod.Set(span, c.Request().Method)
			ext.HTTPUrl.Set(span, c.Request().URL.String())

			c.Set("span", span)
			c.SetRequest(c.Request().WithContext(opentracing.ContextWithSpan(c.Request().Context(), span)))

			err := next(c)
			if err != nil {
				ext.Error.Set(span, true)
				span.SetTag("error.message", err.Error())
			}

			ext.HTTPStatusCode.Set(span, uint16(c.Response().Status))
			return err
		}
	})

	// Custom middleware for metrics
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			duration := time.Since(start)

			status := "success"
			if err != nil {
				status = "error"
			}

			metrics.RecordGRPCRequest(c.Request().Method+" "+c.Path(), status, duration)
			return err
		}
	})

	// Validator
	e.Validator = &CustomValidator{}

	server := &EchoServer{
		echo:    e,
		useCase: useCase,
		tracer:  tracer,
	}

	server.setupRoutes()
	return server
}

func (s *EchoServer) setupRoutes() {
	// Health check
	s.echo.GET("/health", s.healthCheck)

	// Metrics endpoint
	s.echo.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	// Swagger documentation
	s.echo.GET("/docs/*", echoSwagger.WrapHandler)

	// API routes
	api := s.echo.Group("/api/v1")

	// Activity logs routes
	api.POST("/activity-logs", s.createActivityLog)
	api.GET("/activity-logs/:id", s.getActivityLog)
	api.GET("/activity-logs", s.listActivityLogs)
}

// @Summary Health Check
// @Description Check if the service is running
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health [get]
func (s *EchoServer) healthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, HealthResponse{
		Status:  "ok",
		Service: "activity-log-service",
		Version: "1.0.0",
	})
}

// @Summary Create Activity Log
// @Description Create a new activity log entry
// @Tags Activity Logs
// @Accept json
// @Produce json
// @Param request body CreateActivityLogRequest true "Create activity log request"
// @Success 201 {object} ActivityLogResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/activity-logs [post]
func (s *EchoServer) createActivityLog(c echo.Context) error {
	var req CreateActivityLogRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
	}

	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Validation failed",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
	}

	useCaseReq := &usecase.CreateActivityLogRequest{
		ActivityName:     req.ActivityName,
		CompanyID:        req.CompanyID,
		ObjectName:       req.ObjectName,
		ObjectID:         req.ObjectID,
		Changes:          req.Changes,
		FormattedMessage: req.FormattedMessage,
		ActorID:          req.ActorID,
		ActorName:        req.ActorName,
		ActorEmail:       req.ActorEmail,
	}

	activityLog, err := s.useCase.CreateActivityLog(c.Request().Context(), useCaseReq)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to create activity log",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
	}

	response := &ActivityLogResponse{
		ID:               activityLog.ID.String(),
		ActivityName:     activityLog.ActivityName,
		CompanyID:        activityLog.CompanyID,
		ObjectName:       activityLog.ObjectName,
		ObjectID:         activityLog.ObjectID,
		Changes:          string(activityLog.Changes),
		FormattedMessage: activityLog.FormattedMessage,
		ActorID:          activityLog.ActorID,
		ActorName:        activityLog.ActorName,
		ActorEmail:       activityLog.ActorEmail,
		CreatedAt:        activityLog.CreatedAt,
	}

	return c.JSON(http.StatusCreated, response)
}

// @Summary Get Activity Log
// @Description Get an activity log by ID
// @Tags Activity Logs
// @Accept json
// @Produce json
// @Param id path string true "Activity Log ID"
// @Success 200 {object} ActivityLogResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/activity-logs/{id} [get]
func (s *EchoServer) getActivityLog(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid activity log ID",
			Message: "ID parameter is required",
			Code:    http.StatusBadRequest,
		})
	}

	activityLog, err := s.useCase.GetActivityLog(c.Request().Context(), id)
	if err != nil {
		if err.Error() == "activity log not found" {
			return c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "Activity log not found",
				Message: err.Error(),
				Code:    http.StatusNotFound,
			})
		}
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get activity log",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
	}

	response := &ActivityLogResponse{
		ID:               activityLog.ID.String(),
		ActivityName:     activityLog.ActivityName,
		CompanyID:        activityLog.CompanyID,
		ObjectName:       activityLog.ObjectName,
		ObjectID:         activityLog.ObjectID,
		Changes:          string(activityLog.Changes),
		FormattedMessage: activityLog.FormattedMessage,
		ActorID:          activityLog.ActorID,
		ActorName:        activityLog.ActorName,
		ActorEmail:       activityLog.ActorEmail,
		CreatedAt:        activityLog.CreatedAt,
	}

	return c.JSON(http.StatusOK, response)
}

// @Summary List Activity Logs
// @Description Get a paginated list of activity logs for a company
// @Tags Activity Logs
// @Accept json
// @Produce json
// @Param company_id query string true "Company ID"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Success 200 {object} ListActivityLogsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/activity-logs [get]
func (s *EchoServer) listActivityLogs(c echo.Context) error {
	companyID := c.QueryParam("company_id")
	if companyID == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request parameters",
			Message: "company_id is required",
			Code:    http.StatusBadRequest,
		})
	}

	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit < 1 || limit > 100 {
		limit = 10
	}

	activityLogs, total, err := s.useCase.ListActivityLogs(c.Request().Context(), companyID, page, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to list activity logs",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
	}

	responseItems := make([]*ActivityLogResponse, len(activityLogs))
	for i, log := range activityLogs {
		responseItems[i] = &ActivityLogResponse{
			ID:               log.ID.String(),
			ActivityName:     log.ActivityName,
			CompanyID:        log.CompanyID,
			ObjectName:       log.ObjectName,
			ObjectID:         log.ObjectID,
			Changes:          string(log.Changes),
			FormattedMessage: log.FormattedMessage,
			ActorID:          log.ActorID,
			ActorName:        log.ActorName,
			ActorEmail:       log.ActorEmail,
			CreatedAt:        log.CreatedAt,
		}
	}

	response := &ListActivityLogsResponse{
		ActivityLogs: responseItems,
		Total:        total,
		Page:         page,
		Limit:        limit,
	}

	return c.JSON(http.StatusOK, response)
}

func (s *EchoServer) Start(address string) error {
	return s.echo.Start(address)
}

func (s *EchoServer) Shutdown(ctx context.Context) error {
	return s.echo.Shutdown(ctx)
}
