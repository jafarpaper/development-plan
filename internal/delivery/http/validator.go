package http

import (
	"net/http"
	"regexp"

	"github.com/labstack/echo/v4"
)

type CustomValidator struct{}

func (cv *CustomValidator) Validate(i interface{}) error {
	switch v := i.(type) {
	case *CreateActivityLogRequest:
		return cv.validateCreateActivityLogRequest(v)
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "Unknown validation type")
	}
}

func (cv *CustomValidator) validateCreateActivityLogRequest(req *CreateActivityLogRequest) error {
	if req.ActivityName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "activity_name is required")
	}
	if req.CompanyID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "company_id is required")
	}
	if req.ObjectName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "object_name is required")
	}
	if req.ObjectID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "object_id is required")
	}
	if req.FormattedMessage == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "formatted_message is required")
	}
	if req.ActorID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "actor_id is required")
	}
	if req.ActorName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "actor_name is required")
	}
	if req.ActorEmail == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "actor_email is required")
	}
	if !cv.isValidEmail(req.ActorEmail) {
		return echo.NewHTTPError(http.StatusBadRequest, "actor_email must be a valid email address")
	}
	return nil
}

func (cv *CustomValidator) isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}
