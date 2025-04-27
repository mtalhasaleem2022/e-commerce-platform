package crawler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/e-commerce/platform/internal/common/config"
	"github.com/e-commerce/platform/internal/common/db"
	"github.com/e-commerce/platform/internal/common/messaging"
	"github.com/e-commerce/platform/internal/common/models"
)

// Service represents the crawler service
type Service struct {
	db           *db.Database
	kafka        *messaging.KafkaClient
	config       *config.Config
	scraper      *Scraper
	priorityList map[string]int // Maps productID to priority level
	priorityMux  sync.RWMutex   // Mutex for the priority list
}

// NewCrawlerService creates a new crawler service
func NewCrawlerService(db *db.Database, kafka *messaging.KafkaClient, cfg *config.Config) *Service {
	return &Service{
		db:           db,
		kafka:        kafka,
		config:       cfg,
		scraper:      NewScraper(&cfg.Scraper),
		priorityList: make(map[string]int),
	}
}

// Start starts the crawler service
func (s *Service) Start(ctx context.Context) error {
	// Create Kafka producer for product updates
	if err := s.kafka.CreateProducer(s.config.Kafka.ProductTopic); err != nil {
		return fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	// Load priority list from user favorites
	if err := s.loadPriorityList(); err != nil {
		log.Printf("Warning: failed to load priority list: %v", err)
	}

	// Start periodic crawling
	go s.periodicCrawling(ctx)

	// Listen for priority update requests
	go s.listenForPriorityUpdates(ctx)

	return nil
}

// loadPriorityList loads the priority list from user favorites
func (s *Service) loadPriorityList() error {
	var userFavorites []models.UserFavorite
	if err := s.db.Preload("Product").Find(&userFavorites).Error; err != nil {
		return fmt.Errorf("failed to load user favorites: %w", err)
	}

	s.priorityMux.Lock()
	defer s.priorityMux.Unlock()

	for _, favorite := range userFavorites {
		// Set higher priority for favorited products
		s.priorityList[favorite.Product.ExternalID] = 10
	}

	return nil
}

// periodicCrawling performs periodic crawling based on priority
func (s *Service) periodicCrawling(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Minute) // Default crawl interval
	highPriorityTicker := time.NewTicker(5 * time.Minute) // Higher priority crawl interval

	// Get category list to start crawling
	categories, err := s.scraper.GetCategories()
	if err != nil {
		log.Printf("Error getting categories: %v", err)
		return
	}

	// Store categories in the database
	for _, category := range categories {
		var existingCategory models.Category
		result := s.db.Where("external_id = ?", category.ExternalID).First(&existingCategory)
		if result.Error != nil {
			// Create new category
			if err := s.db.Create(&category).Error; err != nil {
				log.Printf("Error creating category: %v", err)
			}
		} else {
			// Update existing category
			category.ID = existingCategory.ID
			if err := s.db.Save(&category).Error; err != nil {
				log.Printf("Error updating category: %v", err)
			}
		}
	}

	// Start crawling products by category
	go s.crawlProductsByCategory(ctx, categories)

	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			highPriorityTicker.Stop()
			return
		case <-ticker.C:
			// Regular priority crawling
			go s.crawlRegularPriorityProducts(ctx)
		case <-highPriorityTicker.C:
			// High priority crawling
			go s.crawlHighPriorityProducts(ctx)
		}
	}
}

// crawlProductsByCategory crawls products by category
func (s *Service) crawlProductsByCategory(ctx context.Context, categories []models.Category) {
	for _, category := range categories {
		select {
		case <-ctx.Done():
			return
		default:
			productIDs, err := s.scraper.GetProductIDsByCategory(category.ExternalID)
			if err != nil {
				log.Printf("Error getting product IDs for category %s: %v", category.Name, err)
				continue
			}

			for _, productID := range productIDs {
				select {
				case <-ctx.Done():
					return
				default:
					// Check if product already exists in the database
					var existingProduct models.Product
					result := s.db.Where("external_id = ?", productID).First(&existingProduct)
					if result.Error == nil {
						// Product exists, update its priority in the list
						s.priorityMux.Lock()
						if _, exists := s.priorityList[productID]; !exists {
							s.priorityList[productID] = 1 // Default priority
						}
						s.priorityMux.Unlock()
					} else {
						// New product, crawl it immediately
						product, err := s.scraper.GetProductDetails(productID)
						if err != nil {
							log.Printf("Error getting product details for ID %s: %v", productID, err)
							continue
						}

						// Save the product to the database
						if err := s.saveProduct(product); err != nil {
							log.Printf("Error saving product: %v", err)
							continue
						}

						// Publish product to Kafka
						if err := s.publishProductUpdate(ctx, product); err != nil {
							log.Printf("Error publishing product update: %v", err)
						}
					}
				}
			}
		}
	}
}

// crawlRegularPriorityProducts crawls regular priority products
func (s *Service) crawlRegularPriorityProducts(ctx context.Context) {
	s.priorityMux.RLock()
	regularPriorityProducts := make([]string, 0)
	for productID, priority := range s.priorityList {
		if priority < 5 {
			regularPriorityProducts = append(regularPriorityProducts, productID)
		}
	}
	s.priorityMux.RUnlock()

	// Limit the number of products to crawl
	maxProducts := 100
	if len(regularPriorityProducts) > maxProducts {
		regularPriorityProducts = regularPriorityProducts[:maxProducts]
	}

	for _, productID := range regularPriorityProducts {
		select {
		case <-ctx.Done():
			return
		default:
			product, err := s.scraper.GetProductDetails(productID)
			if err != nil {
				log.Printf("Error getting product details for ID %s: %v", productID, err)
				continue
			}

			// Save the product to the database
			if err := s.saveProduct(product); err != nil {
				log.Printf("Error saving product: %v", err)
				continue
			}

			// Publish product to Kafka
			if err := s.publishProductUpdate(ctx, product); err != nil {
				log.Printf("Error publishing product update: %v", err)
			}
		}
	}
}

// crawlHighPriorityProducts crawls high priority products
func (s *Service) crawlHighPriorityProducts(ctx context.Context) {
	s.priorityMux.RLock()
	highPriorityProducts := make([]string, 0)
	for productID, priority := range s.priorityList {
		if priority >= 5 {
			highPriorityProducts = append(highPriorityProducts, productID)
		}
	}
	s.priorityMux.RUnlock()

	for _, productID := range highPriorityProducts {
		select {
		case <-ctx.Done():
			return
		default:
			product, err := s.scraper.GetProductDetails(productID)
			if err != nil {
				log.Printf("Error getting product details for ID %s: %v", productID, err)
				continue
			}

			// Save the product to the database
			if err := s.saveProduct(product); err != nil {
				log.Printf("Error saving product: %v", err)
				continue
			}

			// Publish product to Kafka
			if err := s.publishProductUpdate(ctx, product); err != nil {
				log.Printf("Error publishing product update: %v", err)
			}
		}
	}
}

// saveProduct saves a product to the database with all related entities
func (s *Service) saveProduct(product *models.Product) error {
	// Start a transaction
	tx := s.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// Check if the product already exists
	var existingProduct models.Product
	result := tx.Where("external_id = ?", product.ExternalID).First(&existingProduct)
	if result.Error == nil {
		// Product exists, check for changes
		product.ID = existingProduct.ID
		product.CreatedAt = existingProduct.CreatedAt

		// Check for price and stock changes
		var existingVariants []models.Variant
		if err := tx.Where("product_id = ?", existingProduct.ID).Find(&existingVariants).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to fetch existing variants: %w", err)
		}

		// Create a map of existing variants by external ID
		existingVariantMap := make(map[string]models.Variant)
		for _, variant := range existingVariants {
			existingVariantMap[variant.ExternalID] = variant
		}

		// Compare variants
		for i, variant := range product.Variants {
			if existingVariant, exists := existingVariantMap[variant.ExternalID]; exists {
				// Check for price changes
				if existingVariant.Price != variant.Price {
					// Record price change
					priceHistory := models.PriceHistory{
						ProductID:     existingProduct.ID,
						VariantID:     existingVariant.ID,
						PreviousPrice: existingVariant.Price,
						NewPrice:      variant.Price,
						ChangePercent: calculatePercentageChange(existingVariant.Price, variant.Price),
					}
					if err := tx.Create(&priceHistory).Error; err != nil {
						tx.Rollback()
						return fmt.Errorf("failed to create price history: %w", err)
					}
				}

				// Check for stock changes
				if existingVariant.StockCount != variant.StockCount {
					// Record stock change
					stockHistory := models.StockHistory{
						ProductID:      existingProduct.ID,
						VariantID:      existingVariant.ID,
						PreviousStock:  existingVariant.StockCount,
						NewStock:       variant.StockCount,
						ChangeQuantity: variant.StockCount - existingVariant.StockCount,
					}
					if err := tx.Create(&stockHistory).Error; err != nil {
						tx.Rollback()
						return fmt.Errorf("failed to create stock history: %w", err)
					}
				}

				// Update variant ID
				variant.ID = existingVariant.ID
				variant.ProductID = existingProduct.ID
				product.Variants[i] = variant
			}
		}
	}

	// Update or create the product
	if err := tx.Save(product).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to save product: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// calculatePercentageChange calculates the percentage change between two prices
func calculatePercentageChange(oldPrice, newPrice float64) float64 {
	if oldPrice == 0 {
		return 0
	}
	return ((newPrice - oldPrice) / oldPrice) * 100
}

// publishProductUpdate publishes a product update to Kafka
func (s *Service) publishProductUpdate(ctx context.Context, product *models.Product) error {
	// Create a simplified product for the message
	productUpdate := struct {
		ExternalID  string    `json:"external_id"`
		Name        string    `json:"name"`
		IsActive    bool      `json:"is_active"`
		LastUpdated time.Time `json:"last_updated"`
	}{
		ExternalID:  product.ExternalID,
		Name:        product.Name,
		IsActive:    product.IsActive,
		LastUpdated: time.Now(),
	}

	return s.kafka.PublishMessage(ctx, s.config.Kafka.ProductTopic, product.ExternalID, productUpdate)
}

// listenForPriorityUpdates listens for priority update requests
func (s *Service) listenForPriorityUpdates(ctx context.Context) {
	// Create a consumer for priority updates
	priorityTopic := "product-priorities"
	if err := s.kafka.CreateConsumer(priorityTopic); err != nil {
		log.Printf("Error creating consumer for priority updates: %v", err)
		return
	}

	// Process priority update messages
	s.kafka.ConsumeMessages(ctx, priorityTopic, func(message []byte) error {
		var update struct {
			ProductID string `json:"product_id"`
			Priority  int    `json:"priority"`
		}
		if err := json.Unmarshal(message, &update); err != nil {
			return fmt.Errorf("failed to unmarshal priority update: %w", err)
		}

		// Update priority list
		s.priorityMux.Lock()
		s.priorityList[update.ProductID] = update.Priority
		s.priorityMux.Unlock()

		return nil
	})
}