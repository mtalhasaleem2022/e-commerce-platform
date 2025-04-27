package crawler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/e-commerce/platform/internal/common/config"
	"github.com/e-commerce/platform/internal/common/db"
	"github.com/e-commerce/platform/internal/common/models"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"gorm.io/gorm"
)

// API represents the API server for the crawler service
type API struct {
	echo   *echo.Echo
	db     *db.Database
	config *config.Config
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
		echo:   e,
		db:     db,
		config: config,
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
	v1 := api.echo.Group("/api/v1/crawler")
	
	// Category routes
	v1.GET("/categories", api.getCategories)
	v1.GET("/categories/:id", api.getCategoryByID)
	
	// Product routes
	v1.GET("/products", api.getProducts)
	v1.GET("/products/:id", api.getProductByID)
	v1.POST("/products/:id/priority", api.updateProductPriority)
	
	// Crawler control
	v1.POST("/crawl/category/:id", api.crawlCategory)
	v1.POST("/crawl/product/:id", api.crawlProduct)
}

// Start starts the API server
func (api *API) Start(ctx context.Context) error {
	// Server
	go func() {
		address := ":" + strconv.Itoa(api.config.Server.Port)
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
		"status": "ok",
		"service": "crawler",
	})
}

// getCategories returns all categories
func (api *API) getCategories(c echo.Context) error {
	var categories []models.Category
	
	if err := api.db.Find(&categories).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch categories")
	}
	
	return c.JSON(http.StatusOK, categories)
}

// getCategoryByID returns a category by ID
func (api *API) getCategoryByID(c echo.Context) error {
	id := c.Param("id")
	
	var category models.Category
	if err := api.db.Where("external_id = ?", id).First(&category).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "Category not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch category")
	}
	
	return c.JSON(http.StatusOK, category)
}

// getProducts returns products with pagination
func (api *API) getProducts(c echo.Context) error {
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
	var products []models.Product
	var total int64
	
	query := api.db.Model(&models.Product{})
	
	// Apply filters
	if category := c.QueryParam("category"); category != "" {
		query = query.Where("category_id = ?", category)
	}
	
	if brand := c.QueryParam("brand"); brand != "" {
		query = query.Where("brand_id = ?", brand)
	}
	
	if active := c.QueryParam("active"); active != "" {
		isActive := active == "true"
		query = query.Where("is_active = ?", isActive)
	}
	
	// Count total
	query.Count(&total)
	
	// Get paginated results
	if err := query.Limit(limit).Offset(offset).Find(&products).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch products")
	}
	
	// Response
	return c.JSON(http.StatusOK, map[string]interface{}{
		"products": products,
		"total":    total,
		"page":     page,
		"limit":    limit,
	})
}

// getProductByID returns a product by ID
func (api *API) getProductByID(c echo.Context) error {
	id := c.Param("id")
	
	var product models.Product
	if err := api.db.
		Preload("Category").
		Preload("Brand").
		Preload("Seller").
		Preload("Images").
		Preload("Videos").
		Preload("Variants").
		Preload("Attributes").
		Where("external_id = ?", id).
		First(&product).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "Product not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch product")
	}
	
	return c.JSON(http.StatusOK, product)
}

// updateProductPriority updates the priority of a product
func (api *API) updateProductPriority(c echo.Context) error {
	id := c.Param("id")
	
	// Parse priority from request body
	var request struct {
		Priority int `json:"priority"`
	}
	
	if err := c.Bind(&request); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}
	
	// Validate priority
	if request.Priority < 0 || request.Priority > 10 {
		return echo.NewHTTPError(http.StatusBadRequest, "Priority must be between 0 and 10")
	}
	
	// Check if product exists
	var product models.Product
	if err := api.db.Where("external_id = ?", id).First(&product).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "Product not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch product")
	}
	
	// Update priority in service
	api.service.priorityMux.Lock()
	api.service.priorityList[id] = request.Priority
	api.service.priorityMux.Unlock()
	
	return c.JSON(http.StatusOK, map[string]interface{}{
		"success":  true,
		"message":  "Priority updated successfully",
		"product":  id,
		"priority": request.Priority,
	})
}

// crawlCategory triggers crawling for a specific category
func (api *API) crawlCategory(c echo.Context) error {
	id := c.Param("id")
	
	// Check if category exists
	var category models.Category
	if err := api.db.Where("external_id = ?", id).First(&category).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "Category not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch category")
	}
	
	// Trigger crawling in background
	go func() {
		productIDs, err := api.service.scraper.GetProductIDsByCategory(id)
		if err != nil {
			api.echo.Logger.Errorf("Error crawling category %s: %v", id, err)
			return
		}
		
		// Process each product
		for _, productID := range productIDs {
			product, err := api.service.scraper.GetProductDetails(productID)
			if err != nil {
				api.echo.Logger.Errorf("Error getting product details for ID %s: %v", productID, err)
				continue
			}
			
			// Save product
			if err := api.service.saveProduct(product); err != nil {
				api.echo.Logger.Errorf("Error saving product: %v", err)
			}
			
			// Publish product update
			if err := api.service.publishProductUpdate(context.Background(), product); err != nil {
				api.echo.Logger.Errorf("Error publishing product update: %v", err)
			}
		}
	}()
	
	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Crawling started for category: " + category.Name,
	})
}

// crawlProduct triggers crawling for a specific product
func (api *API) crawlProduct(c echo.Context) error {
	id := c.Param("id")
	
	// Trigger crawling in background
	go func() {
		product, err := api.service.scraper.GetProductDetails(id)
		if err != nil {
			api.echo.Logger.Errorf("Error getting product details for ID %s: %v", id, err)
			return
		}
		
		// Save product
		if err := api.service.saveProduct(product); err != nil {
			api.echo.Logger.Errorf("Error saving product: %v", err)
		}
		
		// Publish product update
		if err := api.service.publishProductUpdate(context.Background(), product); err != nil {
			api.echo.Logger.Errorf("Error publishing product update: %v", err)
		}
	}()
	
	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Crawling started for product: " + id,
	})
}