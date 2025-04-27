package notification

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/e-commerce/platform/internal/common/config"
	"github.com/e-commerce/platform/internal/common/db"
	"github.com/e-commerce/platform/internal/common/models"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"gorm.io/gorm"
)

// API represents the API server for the notification service
type API struct {
	echo    *echo.Echo
	db      *db.Database
	config  *config.Config
	service *Service
	upgrader websocket.Upgrader
}

// NewAPI creates a new API server
func NewAPI(db *db.Database, config *config.Config, service *Service) *API {
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	api := &API{
		echo:    e,
		db:      db,
		config:  config,
		service: service,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for WebSocket connections
			},
		},
	}

	// Routes
	api.registerRoutes()

	return api
}

// registerRoutes registers API routes
func (api *API) registerRoutes() {
	// Health check
	api.echo.GET("/health", api.healthCheck)

	// API group
	v1 := api.echo.Group("/api/v1/notifications")

	// Notification routes
	v1.GET("", api.getNotifications)
	v1.GET("/unread", api.getUnreadNotifications)
	v1.PUT("/:id/read", api.markAsRead)
	v1.PUT("/read-all", api.markAllAsRead)

	// WebSocket route for real-time notifications
	v1.GET("/ws/:user_id", api.handleWebSocket)
}

// Start starts the API server
func (api *API) Start(ctx context.Context) error {
	// Server
	go func() {
		address := ":" + strconv.Itoa(api.config.Services.NotificationServicePort)
		if err := api.echo.Start(address); err != nil && err != http.ErrServerClosed {
			api.echo.Logger.Fatal("shutting down the server")
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	// Shutdown gracefully
	shutdownCtx, cancel := context.WithTimeout(context.Background(), api.config.Server.IdleTimeout)
	defer cancel()

	return api.echo.Shutdown(shutdownCtx)
}

// healthCheck is a health check endpoint
func (api *API) healthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "notification",
	})
}

// getNotifications returns notifications with pagination
func (api *API) getNotifications(c echo.Context) error {
	// Parse user ID from query
	userIDStr := c.QueryParam("user_id")
	if userIDStr == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "User ID is required")
	}

	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid user ID")
	}

	// Pagination
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page <= 0 {
		page = 1
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	// Query
	var notifications []models.Notification
	var total int64

	query := api.db.Model(&models.Notification{}).Where("user_id = ?", userID)

	// Count total
	query.Count(&total)

	// Get paginated results
	if err := query.Order("delivered_at DESC").
		Limit(limit).Offset(offset).
		Find(&notifications).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch notifications")
	}

	// Response
	return c.JSON(http.StatusOK, map[string]interface{}{
		"notifications": notifications,
		"total":         total,
		"page":          page,
		"limit":         limit,
	})
}

// getUnreadNotifications returns unread notifications for a user
func (api *API) getUnreadNotifications(c echo.Context) error {
	// Parse user ID from query
	userIDStr := c.QueryParam("user_id")
	if userIDStr == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "User ID is required")
	}

	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid user ID")
	}

	// Pagination
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	if offset < 0 {
		offset = 0
	}

	// Get unread notifications
	notifications, total, err := api.service.GetUnreadNotifications(uint(userID), limit, offset)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch unread notifications")
	}

	// Response
	return c.JSON(http.StatusOK, map[string]interface{}{
		"notifications": notifications,
		"total":         total,
		"limit":         limit,
		"offset":        offset,
	})
}

// markAsRead marks a notification as read
func (api *API) markAsRead(c echo.Context) error {
	// Parse notification ID from path
	id := c.Param("id")
	notificationID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid notification ID")
	}

	// Check if notification exists
	var notification models.Notification
	if err := api.db.First(&notification, notificationID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "Notification not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch notification")
	}

	// Mark as read
	if err := api.service.MarkNotificationAsRead(uint(notificationID)); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to mark notification as read")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Notification marked as read",
	})
}

// markAllAsRead marks all notifications as read for a user
func (api *API) markAllAsRead(c echo.Context) error {
	// Parse user ID from body
	var request struct {
		UserID uint `json:"user_id" validate:"required"`
	}

	if err := c.Bind(&request); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	// Mark all as read
	if err := api.service.MarkAllNotificationsAsRead(request.UserID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to mark all notifications as read")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "All notifications marked as read",
	})
}

// handleWebSocket handles WebSocket connections for real-time notifications
func (api *API) handleWebSocket(c echo.Context) error {
	// Parse user ID from path
	userIDStr := c.Param("user_id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid user ID")
	}

	// Check if user exists
	var user models.User
	if err := api.db.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "User not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch user")
	}

	// Upgrade to WebSocket connection
	ws, err := api.upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "WebSocket upgrade error")
	}
	defer ws.Close()

	// Register channel for notifications
	notificationCh := api.service.RegisterUserChannel(uint(userID))
	defer api.service.UnregisterUserChannel(uint(userID))

	// Send initial unread count
	var unreadCount int64
	api.db.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Count(&unreadCount)

	initialMessage := map[string]interface{}{
		"type":         "init",
		"unread_count": unreadCount,
		"connected_at": time.Now(),
	}
	if err := ws.WriteJSON(initialMessage); err != nil {
		return err
	}

	// Set up ping/pong
	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	// Create a context that's cancelled when this handler returns
	ctx, cancel := context.WithCancel(c.Request().Context())
	defer cancel()

	// Start a goroutine to read messages from the WebSocket
	go func() {
		defer cancel()
		for {
			_, _, err := ws.ReadMessage()
			if err != nil {
				break
			}
		}
	}()

	// Main loop to handle notifications and ping
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-notificationCh:
			if !ok {
				return nil
			}
			notification := map[string]interface{}{
				"type":    "notification",
				"message": msg,
				"time":    time.Now(),
			}
			if err := ws.WriteJSON(notification); err != nil {
				return err
			}
		case <-pingTicker.C:
			if err := ws.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(5*time.Second)); err != nil {
				return err
			}
		}
	}
}