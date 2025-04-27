package notification

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

// Service represents the notification service
type Service struct {
	db            *db.Database
	kafka         *messaging.KafkaClient
	config        *config.Config
	userChannels  map[uint]chan string
	channelsMutex sync.RWMutex
}

// NewNotificationService creates a new notification service
func NewNotificationService(db *db.Database, kafka *messaging.KafkaClient, cfg *config.Config) *Service {
	return &Service{
		db:           db,
		kafka:        kafka,
		config:       cfg,
		userChannels: make(map[uint]chan string),
	}
}

// Start starts the notification service
func (s *Service) Start(ctx context.Context) error {
	// Create Kafka consumer for notifications
	if err := s.kafka.CreateConsumer(s.config.Kafka.NotificationTopic); err != nil {
		return fmt.Errorf("failed to create Kafka consumer: %w", err)
	}

	// Start consuming notifications
	go s.consumeNotifications(ctx)

	// Start periodic cleanup
	go s.periodicCleanup(ctx)

	return nil
}

// consumeNotifications consumes notification messages from Kafka
func (s *Service) consumeNotifications(ctx context.Context) {
	s.kafka.ConsumeMessages(ctx, s.config.Kafka.NotificationTopic, func(message []byte) error {
		// Parse notification message
		var notification struct {
			UserID          uint    `json:"user_id"`
			ProductID       uint    `json:"product_id"`
			VariantID       uint    `json:"variant_id"`
			PreviousPrice   float64 `json:"previous_price"`
			NewPrice        float64 `json:"new_price"`
			DiscountPercent float64 `json:"discount_percent"`
			ProductName     string  `json:"product_name"`
			ProductURL      string  `json:"product_url"`
		}
		if err := json.Unmarshal(message, &notification); err != nil {
			return fmt.Errorf("failed to unmarshal notification: %w", err)
		}

		// Create notification message
		notificationMsg := fmt.Sprintf(
			"Price drop alert: %s is now %.2f (was %.2f, %.1f%% discount)",
			notification.ProductName,
			notification.NewPrice,
			notification.PreviousPrice,
			notification.DiscountPercent,
		)

		// Save notification to database
		dbNotification := models.Notification{
			UserID:      notification.UserID,
			ProductID:   notification.ProductID,
			Message:     notificationMsg,
			DeliveredAt: time.Now(),
		}
		if err := s.db.Create(&dbNotification).Error; err != nil {
			log.Printf("Failed to save notification: %v", err)
		}

		// Try to deliver notification to user if they have an active channel
		s.channelsMutex.RLock()
		channel, exists := s.userChannels[notification.UserID]
		s.channelsMutex.RUnlock()
		if exists {
			select {
			case channel <- notificationMsg:
				log.Printf("Delivered notification to user %d", notification.UserID)
			default:
				log.Printf("Failed to deliver notification to user %d, channel full or closed", notification.UserID)
			}
		}

		return nil
	})
}

// periodicCleanup performs periodic cleanup of old notifications
func (s *Service) periodicCleanup(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour) // Run cleanup once a day
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.cleanupOldNotifications()
			s.cleanupInactiveChannels()
		}
	}
}

// cleanupOldNotifications removes old notifications
func (s *Service) cleanupOldNotifications() {
	// Remove notifications older than 30 days
	cutoff := time.Now().AddDate(0, 0, -30)
	result := s.db.Where("delivered_at < ?", cutoff).Delete(&models.Notification{})
	if result.Error != nil {
		log.Printf("Failed to cleanup old notifications: %v", result.Error)
		return
	}
	log.Printf("Cleaned up %d old notifications", result.RowsAffected)
}

// cleanupInactiveChannels removes inactive user channels
func (s *Service) cleanupInactiveChannels() {
	s.channelsMutex.Lock()
	defer s.channelsMutex.Unlock()

	// Simply log the number of active channels - actual cleanup is done when user disconnects
	log.Printf("Currently %d active user channels", len(s.userChannels))
}

// RegisterUserChannel registers a new user channel for notifications
func (s *Service) RegisterUserChannel(userID uint) chan string {
	s.channelsMutex.Lock()
	defer s.channelsMutex.Unlock()

	// Close existing channel if any
	if channel, exists := s.userChannels[userID]; exists {
		close(channel)
	}

	// Create a new channel for the user
	channel := make(chan string, 100) // Buffer for up to 100 notifications
	s.userChannels[userID] = channel

	return channel
}

// UnregisterUserChannel unregisters a user channel
func (s *Service) UnregisterUserChannel(userID uint) {
	s.channelsMutex.Lock()
	defer s.channelsMutex.Unlock()

	if channel, exists := s.userChannels[userID]; exists {
		close(channel)
		delete(s.userChannels, userID)
	}
}

// GetUnreadNotifications gets unread notifications for a user
func (s *Service) GetUnreadNotifications(userID uint, limit, offset int) ([]models.Notification, int64, error) {
	var notifications []models.Notification
	var total int64

	// Count total unread notifications
	if err := s.db.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count notifications: %w", err)
	}

	// Get paginated unread notifications
	if err := s.db.Where("user_id = ? AND is_read = ?", userID, false).
		Order("delivered_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&notifications).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to fetch notifications: %w", err)
	}

	return notifications, total, nil
}

// MarkNotificationAsRead marks a notification as read
func (s *Service) MarkNotificationAsRead(notificationID uint) error {
	if err := s.db.Model(&models.Notification{}).
		Where("id = ?", notificationID).
		Update("is_read", true).Error; err != nil {
		return fmt.Errorf("failed to mark notification as read: %w", err)
	}
	return nil
}

// MarkAllNotificationsAsRead marks all notifications as read for a user
func (s *Service) MarkAllNotificationsAsRead(userID uint) error {
	if err := s.db.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Update("is_read", true).Error; err != nil {
		return fmt.Errorf("failed to mark all notifications as read: %w", err)
	}
	return nil
}