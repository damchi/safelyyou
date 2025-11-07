package http

import (
	"errors"
	"net/http"
	coreerrors "safelyyou/internal/core/errors"
	"safelyyou/internal/core/ports"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	deviceSvc ports.DeviceService
}

// NewHandler constructs a handler that depends on the DeviceService interface.
func NewHandler(svc ports.DeviceService) *Handler {
	return &Handler{deviceSvc: svc}
}

// PostHeartbeat godoc
// @Summary Register a heartbeat from a device
// @Description Register a heartbeat from a device at the given timestamp.
// @Tags devices
// @Accept json
// @Produce json
// @Param device_id path string true "Device ID"
// @Param request body HeartbeatRequest true "Heartbeat payload"
// @Success 204 "no content"
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/devices/{device_id}/heartbeat [post]
func (h *Handler) PostHeartbeat(c *gin.Context) {
	deviceID := c.Param("device_id")

	var req HeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Msg: "invalid payload: " + err.Error(),
		})
		return
	}

	if err := h.deviceSvc.RecordHeartbeat(deviceID, req.SentAt); err != nil {
		if errors.Is(err, coreerrors.ErrDeviceNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{Msg: "device not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Msg: "internal error"})
		return
	}

	c.Status(http.StatusNoContent)
}
