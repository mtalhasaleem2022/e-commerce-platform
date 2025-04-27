package models

import (
	"time"

	"gorm.io/gorm"
)

// Product represents the main product entity
type Product struct {
	gorm.Model
	ExternalID      string         `json:"external_id" gorm:"uniqueIndex;not null"`
	Name            string         `json:"name" gorm:"not null"`
	Description     string         `json:"description"`
	URL             string         `json:"url"`
	IsActive        bool           `json:"is_active" gorm:"default:true"`
	CategoryID      uint           `json:"category_id"`
	Category        Category       `json:"category"`
	BrandID         uint           `json:"brand_id"`
	Brand           Brand          `json:"brand"`
	SellerID        uint           `json:"seller_id"`
	Seller          Seller         `json:"seller"`
	Images          []Image        `json:"images" gorm:"foreignKey:ProductID"`
	Videos          []Video        `json:"videos" gorm:"foreignKey:ProductID"`
	Variants        []Variant      `json:"variants" gorm:"foreignKey:ProductID"`
	Rating          float64        `json:"rating"`
	RatingCount     int            `json:"rating_count"`
	FavoriteCount   int            `json:"favorite_count"`
	CommentCount    int            `json:"comment_count"`
	LastUpdated     time.Time      `json:"last_updated"`
	Attributes      []Attribute    `json:"attributes" gorm:"many2many:product_attributes;"`
	RelatedProducts []Product      `json:"related_products" gorm:"many2many:product_relations;"`
	PriceHistory    []PriceHistory `json:"price_history" gorm:"foreignKey:ProductID"`
	StockHistory    []StockHistory `json:"stock_history" gorm:"foreignKey:ProductID"`
}

// Category represents product categories
type Category struct {
	gorm.Model
	Name        string     `json:"name" gorm:"not null"`
	Description string     `json:"description"`
	ExternalID  string     `json:"external_id" gorm:"uniqueIndex;not null"`
	ParentID    *uint      `json:"parent_id"`
	Parent      *Category  `json:"parent" gorm:"foreignKey:ParentID"`
	Children    []Category `json:"children" gorm:"foreignKey:ParentID"`
	Level       int        `json:"level" gorm:"not null"`
	IsActive    bool       `json:"is_active" gorm:"default:true"`
}

// Brand represents product brands
type Brand struct {
	gorm.Model
	Name       string `json:"name" gorm:"not null"`
	ExternalID string `json:"external_id" gorm:"uniqueIndex;not null"`
	LogoURL    string `json:"logo_url"`
	IsActive   bool   `json:"is_active" gorm:"default:true"`
}

// Seller represents product sellers
type Seller struct {
	gorm.Model
	Name          string  `json:"name" gorm:"not null"`
	ExternalID    string  `json:"external_id" gorm:"uniqueIndex;not null"`
	Rating        float64 `json:"rating"`
	PositiveRatio float64 `json:"positive_ratio"`
	IsActive      bool    `json:"is_active" gorm:"default:true"`
}

// Image represents product images
type Image struct {
	gorm.Model
	ProductID  uint   `json:"product_id"`
	URL        string `json:"url" gorm:"not null"`
	IsMain     bool   `json:"is_main" gorm:"default:false"`
	ExternalID string `json:"external_id" gorm:"uniqueIndex;not null"`
}

// Video represents product videos
type Video struct {
	gorm.Model
	ProductID  uint   `json:"product_id"`
	URL        string `json:"url" gorm:"not null"`
	ExternalID string `json:"external_id" gorm:"uniqueIndex;not null"`
}

// Variant represents product variants like size, color, etc.
type Variant struct {
	gorm.Model
	ProductID       uint               `json:"product_id"`
	ExternalID      string             `json:"external_id" gorm:"uniqueIndex;not null"`
	AttributeValues []AttributeValue   `json:"attribute_values" gorm:"many2many:variant_attribute_values;"`
	Price           float64            `json:"price"`
	OriginalPrice   float64            `json:"original_price"`
	DiscountRate    int                `json:"discount_rate"`
	StockCount      int                `json:"stock_count"`
	IsActive        bool               `json:"is_active" gorm:"default:true"`
	InstallmentInfo InstallmentOptions `json:"installment_info" gorm:"embedded"`
}

// Attribute represents product attributes like color, size, etc.
type Attribute struct {
	gorm.Model
	Name       string           `json:"name" gorm:"not null"`
	ExternalID string           `json:"external_id" gorm:"uniqueIndex;not null"`
	Values     []AttributeValue `json:"values" gorm:"many2many:attribute_values;"`
}

// AttributeValue represents values for attributes
type AttributeValue struct {
	gorm.Model
	Value      string `json:"value" gorm:"not null"`
	ExternalID string `json:"external_id" gorm:"uniqueIndex;not null"`
}

// InstallmentOptions represents installment payment options
type InstallmentOptions struct {
	Available   bool `json:"available"`
	MaxMonths   int  `json:"max_months"`
	BankOptions int  `json:"bank_options"`
}

// PriceHistory tracks price changes for products
type PriceHistory struct {
	gorm.Model
	ProductID     uint    `json:"product_id"`
	VariantID     uint    `json:"variant_id"`
	PreviousPrice float64 `json:"previous_price"`
	NewPrice      float64 `json:"new_price"`
	ChangePercent float64 `json:"change_percent"`
}

// StockHistory tracks stock changes for products
type StockHistory struct {
	gorm.Model
	ProductID      uint `json:"product_id"`
	VariantID      uint `json:"variant_id"`
	PreviousStock  int  `json:"previous_stock"`
	NewStock       int  `json:"new_stock"`
	ChangeQuantity int  `json:"change_quantity"`
}

// UserFavorite represents user favorites for notification priority
type UserFavorite struct {
	gorm.Model
	UserID    uint    `json:"user_id"`
	ProductID uint    `json:"product_id"`
	Product   Product `json:"product" gorm:"foreignKey:ProductID"`
}

// User represents system users
type User struct {
	gorm.Model
	Email     string         `json:"email" gorm:"uniqueIndex;not null"`
	Favorites []UserFavorite `json:"favorites" gorm:"foreignKey:UserID"`
}

// Notification represents user notifications for price drops
type Notification struct {
	gorm.Model
	UserID      uint      `json:"user_id"`
	ProductID   uint      `json:"product_id"`
	Message     string    `json:"message"`
	IsRead      bool      `json:"is_read" gorm:"default:false"`
	DeliveredAt time.Time `json:"delivered_at"`
}
