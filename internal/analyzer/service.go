package analyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/e-commerce/platform/internal/common/config"
	"github.com/e-commerce/platform/internal/common/db"
	"github.com/e-commerce/platform/internal/common/messaging"
	"github.com/e-commerce/platform/internal/common/models"
)

// Service represents the product analyzer service
type Service struct {
	db          *db.Database
	kafka       *messaging.KafkaClient
	config      *config.Config
	priceAlerts map[uint][]priceAlert
}

// priceAlert represents a price alert configuration
type priceAlert struct {
	UserID           uint
	ProductID        uint
	VariantID        uint
	DiscountPercent  float64
	LastNotification time.Time
}

// NewAnalyzerService creates a new product analyzer service
func NewAnalyzerService(db *db.Database, kafka *messaging.KafkaClient, cfg *config.Config) *Service {
	return &Service{
		db:          db,
		kafka:       kafka,
		config:      cfg,
		priceAlerts: make(map[uint][]priceAlert),
	}
}

// Start starts the product analyzer service
func (s *Service) Start(ctx context.Context) error {
	// Create Kafka consumer for product updates
	if err := s.kafka.CreateConsumer(s.config.Kafka.ProductTopic); err != nil {
		return fmt.Errorf("failed to create Kafka consumer: %w", err)
	}

	// Create Kafka producer for notifications
	if err := s.kafka.CreateProducer(s.config.Kafka.NotificationTopic); err != nil {
		return fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	// Load price alerts for favorited products
	if err := s.loadPriceAlerts(); err != nil {
		log.Printf("Warning: failed to load price alerts: %v", err)
	}

	// Start consuming product updates
	go s.consumeProductUpdates(ctx)

	// Start periodic analysis
	go s.periodicAnalysis(ctx)

	return nil
}

// loadPriceAlerts loads price alerts for favorited products
func (s *Service) loadPriceAlerts() error {
	var userFavorites []models.UserFavorite
	if err := s.db.Preload("Product").Find(&userFavorites).Error; err != nil {
		return fmt.Errorf("failed to load user favorites: %w", err)
	}

	for _, favorite := range userFavorites {
		// Create a price alert with 10% discount threshold
		alert := priceAlert{
			UserID:          favorite.UserID,
			ProductID:       favorite.ProductID,
			DiscountPercent: 10.0, // Default 10% discount threshold
		}

		// Add to the price alerts map
		s.priceAlerts[favorite.ProductID] = append(s.priceAlerts[favorite.ProductID], alert)
	}

	return nil
}

// consumeProductUpdates consumes product update messages from Kafka
func (s *Service) consumeProductUpdates(ctx context.Context) {
	s.kafka.ConsumeMessages(ctx, s.config.Kafka.ProductTopic, func(message []byte) error {
		var update struct {
			ExternalID  string    `json:"external_id"`
			LastUpdated time.Time `json:"last_updated"`
		}
		if err := json.Unmarshal(message, &update); err != nil {
			return fmt.Errorf("failed to unmarshal product update: %w", err)
		}

		// Process the product update
		return s.processProductUpdate(ctx, update.ExternalID)
	})
}

// processProductUpdate processes a product update
func (s *Service) processProductUpdate(ctx context.Context, externalID string) error {
	// Fetch the product from the database
	var product models.Product
	if err := s.db.Where("external_id = ?", externalID).First(&product).Error; err != nil {
		return fmt.Errorf("failed to fetch product: %w", err)
	}

	// Check for price history entries
	var priceHistories []models.PriceHistory
	if err := s.db.Where("product_id = ?", product.ID).
		Order("created_at DESC").
		Limit(10).
		Find(&priceHistories).Error; err != nil {
		return fmt.Errorf("failed to fetch price histories: %w", err)
	}

	// Check if the product has price alerts
	if alerts, hasAlerts := s.priceAlerts[product.ID]; hasAlerts && len(priceHistories) > 0 {
		// Process each price alert
		for _, alert := range alerts {
			// Check if the price drop exceeds the threshold
			for _, history := range priceHistories {
				if history.ChangePercent <= -alert.DiscountPercent && 
				   time.Since(alert.LastNotification) > 24*time.Hour {
					// Fetch variant details
					var variant models.Variant
					if err := s.db.First(&variant, history.VariantID).Error; err != nil {
						log.Printf("Failed to fetch variant: %v", err)
						continue
					}

					// Create notification message
					notification := struct {
						UserID          uint    `json:"user_id"`
						ProductID       uint    `json:"product_id"`
						VariantID       uint    `json:"variant_id"`
						PreviousPrice   float64 `json:"previous_price"`
						NewPrice        float64 `json:"new_price"`
						DiscountPercent float64 `json:"discount_percent"`
						ProductName     string  `json:"product_name"`
						ProductURL      string  `json:"product_url"`
					}{
						UserID:          alert.UserID,
						ProductID:       product.ID,
						VariantID:       variant.ID,
						PreviousPrice:   history.PreviousPrice,
						NewPrice:        history.NewPrice,
						DiscountPercent: -history.ChangePercent,
						ProductName:     product.Name,
						ProductURL:      product.URL,
					}

					// Publish notification
					if err := s.kafka.PublishMessage(ctx, s.config.Kafka.NotificationTopic, 
													fmt.Sprintf("price-drop-%d-%d", alert.UserID, product.ID), 
													notification); err != nil {
						log.Printf("Failed to publish notification: %v", err)
					} else {
						// Update last notification time
						for i := range s.priceAlerts[product.ID] {
							if s.priceAlerts[product.ID][i].UserID == alert.UserID {
								s.priceAlerts[product.ID][i].LastNotification = time.Now()
							}
						}
					}
				}
			}
		}
	}

	return nil
}

// periodicAnalysis performs periodic analysis of products
func (s *Service) periodicAnalysis(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.analyzeTrends()
			s.detectAnomalies()
			s.updatePriorities(ctx)
		}
	}
}

// analyzeTrends analyzes product trends
func (s *Service) analyzeTrends() {
	log.Println("Analyzing product trends...")

	// Example: Find products with increasing price trend
	var products []models.Product
	s.db.Raw(`
		SELECT p.* FROM products p
		JOIN price_histories ph ON p.id = ph.product_id
		GROUP BY p.id
		HAVING AVG(ph.change_percent) > 5
		LIMIT 100
	`).Scan(&products)

	log.Printf("Found %d products with increasing price trend", len(products))

	// Example: Find products with decreasing stock trend
	s.db.Raw(`
		SELECT p.* FROM products p
		JOIN stock_histories sh ON p.id = sh.product_id
		GROUP BY p.id
		HAVING AVG(sh.change_quantity) < -10
		LIMIT 100
	`).Scan(&products)

	log.Printf("Found %d products with decreasing stock trend", len(products))
}

// detectAnomalies detects price or stock anomalies
func (s *Service) detectAnomalies() {
	log.Println("Detecting product anomalies...")

	// Example: Find products with sudden price drops
	var products []models.Product
	s.db.Raw(`
		SELECT p.* FROM products p
		JOIN price_histories ph ON p.id = ph.product_id
		WHERE ph.change_percent < -30
		AND ph.created_at > NOW() - INTERVAL '24 hours'
		GROUP BY p.id
		LIMIT 100
	`).Scan(&products)

	log.Printf("Found %d products with sudden price drops", len(products))

	// Example: Find products with sudden stock increases
	s.db.Raw(`
		SELECT p.* FROM products p
		JOIN stock_histories sh ON p.id = sh.product_id
		WHERE sh.change_quantity > 100
		AND sh.created_at > NOW() - INTERVAL '24 hours'
		GROUP BY p.id
		LIMIT 100
	`).Scan(&products)

	log.Printf("Found %d products with sudden stock increases", len(products))
}

// updatePriorities updates product crawling priorities based on analysis
func (s *Service) updatePriorities(ctx context.Context) {
	log.Println("Updating product priorities...")

	// Example: Increase priority for trending products
	var trendingProducts []models.Product
	s.db.Raw(`
		SELECT p.* FROM products p
		WHERE p.favorite_count > 100
		AND p.is_active = true
		ORDER BY p.favorite_count DESC
		LIMIT 100
	`).Scan(&trendingProducts)

	// Update priorities
	for _, product := range trendingProducts {
		priorityUpdate := struct {
			ProductID string `json:"product_id"`
			Priority  int    `json:"priority"`
		}{
			ProductID: product.ExternalID,
			Priority:  8, // High priority
		}

		// Publish priority update
		if err := s.kafka.PublishMessage(ctx, "product-priorities", product.ExternalID, priorityUpdate); err != nil {
			log.Printf("Failed to publish priority update: %v", err)
		}
	}

	log.Printf("Updated priorities for %d trending products", len(trendingProducts))
}