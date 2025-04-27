package crawler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/e-commerce/platform/internal/common/config"
	"github.com/e-commerce/platform/internal/common/models"
)

// Scraper is responsible for scraping product data from Trendyol
type Scraper struct {
	client          *http.Client
	config          *config.ScraperConfig
	currentProxyIdx int
	proxies         []string
	rateLimiter     <-chan time.Time
}

// NewScraper creates a new scraper instance
func NewScraper(cfg *config.ScraperConfig) *Scraper {
	client := &http.Client{
		Timeout: cfg.RequestTimeout,
	}

	// Create a rate limiter to avoid getting banned
	rateLimiter := time.Tick(cfg.RequestDelay)

	return &Scraper{
		client:      client,
		config:      cfg,
		rateLimiter: rateLimiter,
		proxies:     []string{}, // Add proxies if needed
	}
}

// GetCategories fetches all product categories
func (s *Scraper) GetCategories() ([]models.Category, error) {
	<-s.rateLimiter // Rate limiting

	// Create a request to fetch categories
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/categories", s.config.BaseURL), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", s.config.UserAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse the JSON response
	var result struct {
		Categories []struct {
			ID          int    `json:"id"`
			Name        string `json:"name"`
			ParentID    *int   `json:"parentId"`
			DisplayOrder int    `json:"displayOrder"`
			Level       int    `json:"level"`
			URL         string `json:"url"`
		} `json:"categories"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to Category models
	categories := make([]models.Category, 0, len(result.Categories))
	for _, cat := range result.Categories {
		var parentID *uint
		if cat.ParentID != nil {
			parentUint := uint(*cat.ParentID)
			parentID = &parentUint
		}

		category := models.Category{
			Name:       cat.Name,
			ExternalID: strconv.Itoa(cat.ID),
			ParentID:   parentID,
			Level:      cat.Level,
			IsActive:   true,
		}
		categories = append(categories, category)
	}

	return categories, nil
}

// GetProductIDsByCategory fetches product IDs for a specific category
func (s *Scraper) GetProductIDsByCategory(categoryID string) ([]string, error) {
	<-s.rateLimiter // Rate limiting

	// Create a request to fetch products in a category
	reqURL := fmt.Sprintf("%s/api/category/%s/products?page=1&limit=100", s.config.BaseURL, categoryID)
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", s.config.UserAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse the JSON response
	var result struct {
		Products []struct {
			ID string `json:"id"`
		} `json:"products"`
		TotalCount int `json:"totalCount"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract product IDs
	productIDs := make([]string, 0, len(result.Products))
	for _, product := range result.Products {
		productIDs = append(productIDs, product.ID)
	}

	return productIDs, nil
}

// GetProductDetails fetches detailed information for a specific product
func (s *Scraper) GetProductDetails(productID string) (*models.Product, error) {
	<-s.rateLimiter // Rate limiting

	// Create a request to fetch product details
	reqURL := fmt.Sprintf("%s/api/product/%s", s.config.BaseURL, productID)
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", s.config.UserAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse the JSON response
	var result struct {
		ID               string  `json:"id"`
		Name             string  `json:"name"`
		Description      string  `json:"description"`
		URL              string  `json:"url"`
		CategoryID       string  `json:"categoryId"`
		CategoryName     string  `json:"categoryName"`
		BrandID          string  `json:"brandId"`
		BrandName        string  `json:"brandName"`
		BrandLogoURL     string  `json:"brandLogoUrl"`
		SellerID         string  `json:"sellerId"`
		SellerName       string  `json:"sellerName"`
		SellerRating     float64 `json:"sellerRating"`
		PositiveRatio    float64 `json:"positiveRatio"`
		Rating           float64 `json:"rating"`
		RatingCount      int     `json:"ratingCount"`
		FavoriteCount    int     `json:"favoriteCount"`
		CommentCount     int     `json:"commentCount"`
		IsInStock        bool    `json:"isInStock"`
		DiscountRate     int     `json:"discountRate"`
		HasVideo         bool    `json:"hasVideo"`
		InstallmentCount int     `json:"installmentCount"`
		Images           []struct {
			ID     string `json:"id"`
			URL    string `json:"url"`
			IsMain bool   `json:"isMain"`
		} `json:"images"`
		Videos []struct {
			ID  string `json:"id"`
			URL string `json:"url"`
		} `json:"videos"`
		Variants []struct {
			ID            string  `json:"id"`
			Price         float64 `json:"price"`
			OriginalPrice float64 `json:"originalPrice"`
			DiscountRate  int     `json:"discountRate"`
			StockCount    int     `json:"stockCount"`
			IsInStock     bool    `json:"isInStock"`
			Attributes    []struct {
				Name  string `json:"name"`
				ID    string `json:"id"`
				Value string `json:"value"`
			} `json:"attributes"`
		} `json:"variants"`
		Attributes []struct {
			Name  string `json:"name"`
			ID    string `json:"id"`
			Value string `json:"value"`
		} `json:"attributes"`
		RelatedProducts []string `json:"relatedProductIds"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Create brand model
	brand := models.Brand{
		Name:       result.BrandName,
		ExternalID: result.BrandID,
		LogoURL:    result.BrandLogoURL,
		IsActive:   true,
	}

	// Create seller model
	seller := models.Seller{
		Name:          result.SellerName,
		ExternalID:    result.SellerID,
		Rating:        result.SellerRating,
		PositiveRatio: result.PositiveRatio,
		IsActive:      true,
	}

	// Create category model
	category := models.Category{
		Name:       result.CategoryName,
		ExternalID: result.CategoryID,
		IsActive:   true,
	}

	// Create product model
	product := &models.Product{
		ExternalID:    result.ID,
		Name:          result.Name,
		Description:   result.Description,
		URL:           result.URL,
		IsActive:      result.IsInStock,
		Brand:         brand,
		Seller:        seller,
		Category:      category,
		Rating:        result.Rating,
		RatingCount:   result.RatingCount,
		FavoriteCount: result.FavoriteCount,
		CommentCount:  result.CommentCount,
		LastUpdated:   time.Now(),
	}

	// Add images
	for _, img := range result.Images {
		image := models.Image{
			URL:        img.URL,
			IsMain:     img.IsMain,
			ExternalID: img.ID,
		}
		product.Images = append(product.Images, image)
	}

	// Add videos
	for _, vid := range result.Videos {
		video := models.Video{
			URL:        vid.URL,
			ExternalID: vid.ID,
		}
		product.Videos = append(product.Videos, video)
	}

	// Create attribute map
	attributeMap := make(map[string]models.Attribute)
	attributeValueMap := make(map[string]models.AttributeValue)

	// Add product attributes
	for _, attr := range result.Attributes {
		attribute := models.Attribute{
			Name:       attr.Name,
			ExternalID: attr.ID,
		}

		attrValue := models.AttributeValue{
			Value:      attr.Value,
			ExternalID: fmt.Sprintf("%s-%s", attr.ID, url.QueryEscape(attr.Value)),
		}
		attributeValueMap[attrValue.ExternalID] = attrValue

		attribute.Values = append(attribute.Values, attrValue)
		attributeMap[attribute.ExternalID] = attribute
		product.Attributes = append(product.Attributes, attribute)
	}

	// Add variants
	for _, v := range result.Variants {
		installmentInfo := models.InstallmentOptions{
			Available: result.InstallmentCount > 0,
			MaxMonths: result.InstallmentCount,
		}

		variant := models.Variant{
			ExternalID:      v.ID,
			Price:           v.Price,
			OriginalPrice:   v.OriginalPrice,
			DiscountRate:    v.DiscountRate,
			StockCount:      v.StockCount,
			IsActive:        v.IsInStock,
			InstallmentInfo: installmentInfo,
		}

		// Add variant attributes
		for _, attr := range v.Attributes {
			attrValue := models.AttributeValue{
				Value:      attr.Value,
				ExternalID: fmt.Sprintf("%s-%s", attr.ID, url.QueryEscape(attr.Value)),
			}

			if existingValue, exists := attributeValueMap[attrValue.ExternalID]; !exists {
				attributeValueMap[attrValue.ExternalID] = attrValue
				variant.AttributeValues = append(variant.AttributeValues, attrValue)
			} else {
				variant.AttributeValues = append(variant.AttributeValues, existingValue)
			}
		}

		product.Variants = append(product.Variants, variant)
	}

	return product, nil
}

// scrapeHTML parses HTML content using goquery
func (s *Scraper) scrapeHTML(url string) (*goquery.Document, error) {
	<-s.rateLimiter // Rate limiting

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", s.config.UserAgent)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Read and convert the response body to UTF-8
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Convert response body string to reader
	bodyReader := strings.NewReader(string(body))

	// Parse HTML
	doc, err := goquery.NewDocumentFromReader(bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	return doc, nil
}

// rotateProxy rotates to the next proxy in the list
func (s *Scraper) rotateProxy() {
	if len(s.proxies) == 0 {
		return
	}

	s.currentProxyIdx = (s.currentProxyIdx + 1) % len(s.proxies)
	proxyURL, _ := url.Parse(s.proxies[s.currentProxyIdx])
	s.client.Transport = &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}
}