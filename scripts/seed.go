package main

import (
	"log"
	"time"

	"github.com/e-commerce/platform/internal/common/config"
	"github.com/e-commerce/platform/internal/common/db"
	"github.com/e-commerce/platform/internal/common/models"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database connection
	database, err := db.NewPostgresDB(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Seed database with sample data
	log.Println("Starting database seeding...")

	// Create sample categories
	categories := []models.Category{
		{
			Name:       "Electronics",
			ExternalID: "10001",
			Level:      1,
			IsActive:   true,
		},
		{
			Name:       "Smartphones",
			ExternalID: "10002",
			Level:      2,
			IsActive:   true,
		},
		{
			Name:       "Laptops",
			ExternalID: "10003",
			Level:      2,
			IsActive:   true,
		},
	}

	for i, _ := range categories {
		if i > 0 {
			parentID := uint(1) // Set Electronics as parent
			categories[i].ParentID = &parentID
		}
		if err := database.Create(&categories[i]).Error; err != nil {
			log.Printf("Error creating category: %v", err)
		}
	}
	log.Printf("Created %d categories", len(categories))

	// Create sample brands
	brands := []models.Brand{
		{
			Name:       "Apple",
			ExternalID: "20001",
			LogoURL:    "https://example.com/apple.png",
			IsActive:   true,
		},
		{
			Name:       "Samsung",
			ExternalID: "20002",
			LogoURL:    "https://example.com/samsung.png",
			IsActive:   true,
		},
		{
			Name:       "Dell",
			ExternalID: "20003",
			LogoURL:    "https://example.com/dell.png",
			IsActive:   true,
		},
	}

	for _, brand := range brands {
		if err := database.Create(&brand).Error; err != nil {
			log.Printf("Error creating brand: %v", err)
		}
	}
	log.Printf("Created %d brands", len(brands))

	// Create sample sellers
	sellers := []models.Seller{
		{
			Name:          "Tech Store",
			ExternalID:    "30001",
			Rating:        4.5,
			PositiveRatio: 92.5,
			IsActive:      true,
		},
		{
			Name:          "Gadget Shop",
			ExternalID:    "30002",
			Rating:        4.2,
			PositiveRatio: 88.7,
			IsActive:      true,
		},
	}

	for _, seller := range sellers {
		if err := database.Create(&seller).Error; err != nil {
			log.Printf("Error creating seller: %v", err)
		}
	}
	log.Printf("Created %d sellers", len(sellers))

	// Create sample products
	products := []models.Product{
		{
			ExternalID:    "40001",
			Name:          "iPhone 15 Pro Max",
			Description:   "Apple's latest flagship smartphone with the A17 Pro chip.",
			URL:           "https://example.com/iphone-15-pro",
			IsActive:      true,
			CategoryID:    2,
			BrandID:       1,
			SellerID:      1,
			Rating:        4.8,
			RatingCount:   352,
			FavoriteCount: 1200,
			CommentCount:  280,
			LastUpdated:   time.Now(),
		},
		{
			ExternalID:    "40002",
			Name:          "Samsung Galaxy S23 Ultra",
			Description:   "Samsung's premium smartphone with an advanced camera system.",
			URL:           "https://example.com/samsung-s23-ultra",
			IsActive:      true,
			CategoryID:    2,
			BrandID:       2,
			SellerID:      1,
			Rating:        4.7,
			RatingCount:   423,
			FavoriteCount: 980,
			CommentCount:  310,
			LastUpdated:   time.Now(),
		},
		{
			ExternalID:    "40003",
			Name:          "Dell XPS 15",
			Description:   "High-performance laptop with a stunning display.",
			URL:           "https://example.com/dell-xps-15",
			IsActive:      true,
			CategoryID:    3,
			BrandID:       3,
			SellerID:      2,
			Rating:        4.6,
			RatingCount:   187,
			FavoriteCount: 450,
			CommentCount:  120,
			LastUpdated:   time.Now(),
		},
	}

	for _, product := range products {
		if err := database.Create(&product).Error; err != nil {
			log.Printf("Error creating product: %v", err)
		}
	}
	log.Printf("Created %d products", len(products))

	// Create sample variants
	variants := []models.Variant{
		{
			ProductID:     1,
			ExternalID:    "50001",
			Price:         1299.99,
			OriginalPrice: 1399.99,
			DiscountRate:  7,
			StockCount:    50,
			IsActive:      true,
			InstallmentInfo: models.InstallmentOptions{
				Available:   true,
				MaxMonths:   12,
				BankOptions: 5,
			},
		},
		{
			ProductID:     1,
			ExternalID:    "50002",
			Price:         1499.99,
			OriginalPrice: 1599.99,
			DiscountRate:  6,
			StockCount:    35,
			IsActive:      true,
			InstallmentInfo: models.InstallmentOptions{
				Available:   true,
				MaxMonths:   12,
				BankOptions: 5,
			},
		},
		{
			ProductID:     2,
			ExternalID:    "50003",
			Price:         1199.99,
			OriginalPrice: 1299.99,
			DiscountRate:  8,
			StockCount:    60,
			IsActive:      true,
			InstallmentInfo: models.InstallmentOptions{
				Available:   true,
				MaxMonths:   10,
				BankOptions: 4,
			},
		},
		{
			ProductID:     3,
			ExternalID:    "50004",
			Price:         1799.99,
			OriginalPrice: 1899.99,
			DiscountRate:  5,
			StockCount:    20,
			IsActive:      true,
			InstallmentInfo: models.InstallmentOptions{
				Available:   true,
				MaxMonths:   18,
				BankOptions: 6,
			},
		},
	}

	for _, variant := range variants {
		if err := database.Create(&variant).Error; err != nil {
			log.Printf("Error creating variant: %v", err)
		}
	}
	log.Printf("Created %d variants", len(variants))

	// Create sample users
	users := []models.User{
		{
			Email: "user1@example.com",
		},
		{
			Email: "user2@example.com",
		},
	}

	for _, user := range users {
		if err := database.Create(&user).Error; err != nil {
			log.Printf("Error creating user: %v", err)
		}
	}
	log.Printf("Created %d users", len(users))

	// Create sample favorites
	favorites := []models.UserFavorite{
		{
			UserID:    1,
			ProductID: 1,
		},
		{
			UserID:    1,
			ProductID: 3,
		},
		{
			UserID:    2,
			ProductID: 2,
		},
	}

	for _, favorite := range favorites {
		if err := database.Create(&favorite).Error; err != nil {
			log.Printf("Error creating user favorite: %v", err)
		}
	}
	log.Printf("Created %d user favorites", len(favorites))

	// Create sample price histories
	priceHistories := []models.PriceHistory{
		{
			ProductID:     1,
			VariantID:     1,
			PreviousPrice: 1399.99,
			NewPrice:      1299.99,
			ChangePercent: -7.14,
		},
		{
			ProductID:     2,
			VariantID:     3,
			PreviousPrice: 1299.99,
			NewPrice:      1199.99,
			ChangePercent: -7.69,
		},
	}

	for _, history := range priceHistories {
		if err := database.Create(&history).Error; err != nil {
			log.Printf("Error creating price history: %v", err)
		}
	}
	log.Printf("Created %d price histories", len(priceHistories))

	log.Println("Database seeding completed successfully")
}
