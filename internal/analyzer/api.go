package analyzer

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/e-commerce/platform/internal/common/config"
	"github.com/e-commerce/platform/internal/common/db"
	"github.com/e-commerce/platform/internal/common/models"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"gorm.io/gorm"
)

// API represents the API server for the analyzer service
type API struct {
	echo    *echo.Echo
	db      *db.Database
	config  *config.Config
	service *Service
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
	v1 := api.echo.Group("/api/v1/analyzer")

	// Stats routes
	v1.GET("/stats/products", api.getProductStats)
	v1.GET("/stats/prices", api.getPriceStats)
	v1.GET("/stats/favorites", api.getFavoriteStats)

	// Trend routes
	v1.GET("/trends/prices", api.getPriceTrends)
	v1.GET("/trends/stock", api.getStockTrends)

	// History routes
	v1.GET("/history/prices/:id", api.getPriceHistory)
	v1.GET("/history/stock/:id", api.getStockHistory)

	// Alert routes
	v1.POST("/alerts/price", api.createPriceAlert)
	v1.GET("/alerts/price/user/:id", api.getUserPriceAlerts)
	v1.DELETE("/alerts/price/:id", api.deletePriceAlert)
}

// Start starts the API server
func (api *API) Start(ctx context.Context) error {
	// Server
	go func() {
		address := ":" + strconv.Itoa(api.config.Services.AnalyzerServicePort)
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
		"service": "analyzer",
	})
}

// getProductStats returns product statistics
func (api *API) getProductStats(c echo.Context) error {
	// Count total products
	var totalProducts int64
	if err := api.db.Model(&models.Product{}).Count(&totalProducts).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to count products")
	}

	// Count active products
	var activeProducts int64
	if err := api.db.Model(&models.Product{}).Where("is_active = ?", true).Count(&activeProducts).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to count active products")
	}

	// Count products added in the last 24 hours
	var newProducts int64
	if err := api.db.Model(&models.Product{}).Where("created_at > ?", time.Now().Add(-24*time.Hour)).Count(&newProducts).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to count new products")
	}

	// Count products updated in the last 24 hours
	var updatedProducts int64
	if err := api.db.Model(&models.Product{}).Where("updated_at > ? AND updated_at != created_at", time.Now().Add(-24*time.Hour)).Count(&updatedProducts).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to count updated products")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"total":   totalProducts,
		"active":  activeProducts,
		"new":     newProducts,
		"updated": updatedProducts,
	})
}

// getPriceStats returns price statistics
func (api *API) getPriceStats(c echo.Context) error {
	// Average price change percentage
	var avgPriceChange struct {
		AvgChange float64 `json:"avg_change"`
	}
	if err := api.db.Model(&models.PriceHistory{}).Select("AVG(change_percent) as avg_change").Scan(&avgPriceChange).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to calculate average price change")
	}

	// Count price increases
	var priceIncreases int64
	if err := api.db.Model(&models.PriceHistory{}).Where("change_percent > 0").Count(&priceIncreases).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to count price increases")
	}

	// Count price decreases
	var priceDecreases int64
	if err := api.db.Model(&models.PriceHistory{}).Where("change_percent < 0").Count(&priceDecreases).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to count price decreases")
	}

	// Top 5 biggest price drops in the last 24 hours
	type PriceDrop struct {
		ProductID     uint    `json:"product_id"`
		ProductName   string  `json:"product_name"`
		VariantID     uint    `json:"variant_id"`
		PreviousPrice float64 `json:"previous_price"`
		NewPrice      float64 `json:"new_price"`
		ChangePercent float64 `json:"change_percent"`
	}
	var biggestDrops []PriceDrop
	if err := api.db.Raw(`
		SELECT ph.product_id, p.name as product_name, ph.variant_id, ph.previous_price, ph.new_price, ph.change_percent
		FROM price_histories ph
		JOIN products p ON ph.product_id = p.id
		WHERE ph.created_at > NOW() - INTERVAL '24 hours'
		AND ph.change_percent < 0
		ORDER BY ph.change_percent ASC
		LIMIT 5
	`).Scan(&biggestDrops).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find biggest price drops")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"avg_change":     avgPriceChange.AvgChange,
		"increases":      priceIncreases,
		"decreases":      priceDecreases,
		"biggest_drops":  biggestDrops,
	})
}

// getFavoriteStats returns favorite statistics
func (api *API) getFavoriteStats(c echo.Context) error {
	// Count total favorites
	var totalFavorites int64
	if err := api.db.Model(&models.UserFavorite{}).Count(&totalFavorites).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to count favorites")
	}

	// Count users with favorites
	var usersWithFavorites int64
	if err := api.db.Model(&models.UserFavorite{}).Distinct("user_id").Count(&usersWithFavorites).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to count users with favorites")
	}

	// Top 5 most favorited products
	type PopularProduct struct {
		ProductID uint   `json:"product_id"`
		Name      string `json:"name"`
		Count     int    `json:"count"`
	}
	var popularProducts []PopularProduct
	if err := api.db.Raw(`
		SELECT p.id as product_id, p.name, COUNT(uf.id) as count
		FROM products p
		JOIN user_favorites uf ON p.id = uf.product_id
		GROUP BY p.id, p.name
		ORDER BY count DESC
		LIMIT 5
	`).Scan(&popularProducts).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find popular products")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"total_favorites":    totalFavorites,
		"users_with_favorites": usersWithFavorites,
		"popular_products":   popularProducts,
	})
}

// getPriceTrends returns price trends
func (api *API) getPriceTrends(c echo.Context) error {
	// Get days parameter
	days, err := strconv.Atoi(c.QueryParam("days"))
	if err != nil || days <= 0 {
		days = 30 // Default to 30 days
	}

	// Get price trends for the specified period
	type DailyPriceTrend struct {
		Date        time.Time `json:"date"`
		AvgChange   float64   `json:"avg_change"`
		Increases   int       `json:"increases"`
		Decreases   int       `json:"decreases"`
		NoChange    int       `json:"no_change"`
		TotalChanges int      `json:"total_changes"`
	}
	var trends []DailyPriceTrend
	if err := api.db.Raw(`
		SELECT
			DATE(created_at) as date,
			AVG(change_percent) as avg_change,
			COUNT(CASE WHEN change_percent > 0 THEN 1 END) as increases,
			COUNT(CASE WHEN change_percent < 0 THEN 1 END) as decreases,
			COUNT(CASE WHEN change_percent = 0 THEN 1 END) as no_change,
			COUNT(*) as total_changes
		FROM price_histories
		WHERE created_at > NOW() - INTERVAL '?' days
		GROUP BY DATE(created_at)
		ORDER BY date DESC
	`, days).Scan(&trends).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get price trends")
	}

	return c.JSON(http.StatusOK, trends)
}

// getStockTrends returns stock trends
func (api *API) getStockTrends(c echo.Context) error {
	// Get days parameter
	days, err := strconv.Atoi(c.QueryParam("days"))
	if err != nil || days <= 0 {
		days = 30 // Default to 30 days
	}

	// Get stock trends for the specified period
	type DailyStockTrend struct {
		Date        time.Time `json:"date"`
		AvgChange   float64   `json:"avg_change"`
		Increases   int       `json:"increases"`
		Decreases   int       `json:"decreases"`
		NoChange    int       `json:"no_change"`
		TotalChanges int      `json:"total_changes"`
	}
	var trends []DailyStockTrend
	if err := api.db.Raw(`
		SELECT
			DATE(created_at) as date,
			AVG(change_quantity) as avg_change,
			COUNT(CASE WHEN change_quantity > 0 THEN 1 END) as increases,
			COUNT(CASE WHEN change_quantity < 0 THEN 1 END) as decreases,
			COUNT(CASE WHEN change_quantity = 0 THEN 1 END) as no_change,
			COUNT(*) as total_changes
		FROM stock_histories
		WHERE created_at > NOW() - INTERVAL '?' days
		GROUP BY DATE(created_at)
		ORDER BY date DESC
	`, days).Scan(&trends).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get stock trends")
	}

	return c.JSON(http.StatusOK, trends)
}

// getPriceHistory returns the price history for a product
func (api *API) getPriceHistory(c echo.Context) error {
	id := c.Param("id")
	
	// Convert ID to uint
	productID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid product ID")
	}

	// Get price history
	var priceHistory []models.PriceHistory
	if err := api.db.Where("product_id = ?", productID).
		Order("created_at DESC").
		Find(&priceHistory).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get price history")
	}

	return c.JSON(http.StatusOK, priceHistory)
}

// getStockHistory returns the stock history for a product
func (api *API) getStockHistory(c echo.Context) error {
	id := c.Param("id")
	
	// Convert ID to uint
	productID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid product ID")
	}

	// Get stock history
	var stockHistory []models.StockHistory
	if err := api.db.Where("product_id = ?", productID).
		Order("created_at DESC").
		Find(&stockHistory).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get stock history")
	}

	return c.JSON(http.StatusOK, stockHistory)
}

// createPriceAlert creates a new price alert
func (api *API) createPriceAlert(c echo.Context) error {
	// Parse request body
	var request struct {
		UserID          uint    `json:"user_id" validate:"required"`
		ProductID       uint    `json:"product_id" validate:"required"`
		VariantID       uint    `json:"variant_id"`
		DiscountPercent float64 `json:"discount_percent" validate:"required"`
	}
	
	if err := c.Bind(&request); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}
	
	// Validate request
	if request.DiscountPercent <= 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Discount percentage must be positive")
	}

	// Check if product exists
	var product models.Product
	if err := api.db.First(&product, request.ProductID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "Product not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch product")
	}

	// Check if user exists
	var user models.User
	if err := api.db.First(&user, request.UserID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "User not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch user")
	}

	// Check if variant exists if provided
	if request.VariantID > 0 {
		var variant models.Variant
		if err := api.db.First(&variant, request.VariantID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return echo.NewHTTPError(http.StatusNotFound, "Variant not found")
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch variant")
		}
	}

	// Create a new price alert
	alert := priceAlert{
		UserID:          request.UserID,
		ProductID:       request.ProductID,
		VariantID:       request.VariantID,
		DiscountPercent: request.DiscountPercent,
	}

	// Add to the service's price alerts
	api.service.priceAlerts[request.ProductID] = append(api.service.priceAlerts[request.ProductID], alert)

	// Also create a user favorite if it doesn't exist
	var favorite models.UserFavorite
	result := api.db.Where("user_id = ? AND product_id = ?", request.UserID, request.ProductID).First(&favorite)
	if result.Error == gorm.ErrRecordNotFound {
		favorite = models.UserFavorite{
			UserID:    request.UserID,
			ProductID: request.ProductID,
		}
		if err := api.db.Create(&favorite).Error; err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create user favorite")
		}
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"success": true,
		"message": "Price alert created successfully",
		"alert": map[string]interface{}{
			"user_id":          alert.UserID,
			"product_id":       alert.ProductID,
			"variant_id":       alert.VariantID,
			"discount_percent": alert.DiscountPercent,
		},
	})
}

// getUserPriceAlerts returns price alerts for a user
func (api *API) getUserPriceAlerts(c echo.Context) error {
	id := c.Param("id")
	
	// Convert ID to uint
	userID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid user ID")
	}

	// Get price alerts for the user
	alerts := make([]map[string]interface{}, 0)
	for productID, productAlerts := range api.service.priceAlerts {
		for _, alert := range productAlerts {
			if alert.UserID == uint(userID) {
				// Get product name
				var product models.Product
				if err := api.db.Select("name").First(&product, productID).Error; err != nil {
					continue
				}

				alerts = append(alerts, map[string]interface{}{
					"user_id":          alert.UserID,
					"product_id":       alert.ProductID,
					"product_name":     product.Name,
					"variant_id":       alert.VariantID,
					"discount_percent": alert.DiscountPercent,
				})
			}
		}
	}

	return c.JSON(http.StatusOK, alerts)
}

// deletePriceAlert deletes a price alert
func (api *API) deletePriceAlert(c echo.Context) error {
	// Parse parameters
	id := c.Param("id")
	
	// Convert ID to uint
	_, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid alert ID")
	}

	// Parse request body
	var request struct {
		UserID    uint `json:"user_id" validate:"required"`
		ProductID uint `json:"product_id" validate:"required"`
	}
	
	if err := c.Bind(&request); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	// Find and remove the alert
	found := false
	if alerts, exists := api.service.priceAlerts[request.ProductID]; exists {
		for i, alert := range alerts {
			if alert.UserID == request.UserID {
				// Remove the alert
				api.service.priceAlerts[request.ProductID] = append(alerts[:i], alerts[i+1:]...)
				found = true
				break
			}
		}
	}

	if !found {
		return echo.NewHTTPError(http.StatusNotFound, "Price alert not found")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Price alert deleted successfully",
	})
}