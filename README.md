# E-Commerce Platform - Backend Services

This project implements a scalable microservices-based backend architecture for an e-commerce product data platform.

## System Architecture

The platform consists of three main microservices:

1. **Crawler Service**: Continuously fetches and updates product data from Trendyol
2. **Analyzer Service**: Analyzes product data, detects price/stock changes, and manages update priorities
3. **Notification Service**: Handles real-time notifications for users about product updates

### Technology Stack

- **Language**: Go with Echo framework
- **Database**: PostgreSQL with GORM
- **Messaging**: Kafka for event-driven communication
- **Service Communication**: gRPC for internal service communication
- **Containerization**: Docker & Docker Compose

## Getting Started

### Prerequisites

- Go (1.19+)
- Docker and Docker Compose
- PostgreSQL
- Kafka

### Installation

1. Clone the repository
```sh
git clone https://github.com/your-username/e-commerce-platform.git
cd e-commerce-platform
```

2. Initialize the Go module dependencies
```sh
go mod tidy
```

3. Set up the environment
```sh
cp .env.example .env
# Edit .env file with your configuration
```

4. Start the services using Docker Compose
```sh
make docker-up
```

### Running Locally

To run services individually:

```sh
# Run the crawler service
make run-crawler

# Run the analyzer service
make run-analyzer

# Run the notification service
make run-notification
```

## Project Structure

```
e-commerce-platform/
├── cmd/                   # Service entry points
│   ├── crawler/           # Crawler service main
│   ├── analyzer/          # Analyzer service main
│   └── notification/      # Notification service main
├── internal/              # Internal packages
│   ├── common/            # Shared code
│   │   ├── config/        # Configuration
│   │   ├── db/            # Database connection
│   │   ├── models/        # Database models
│   │   ├── proto/         # Protocol buffers
│   │   ├── messaging/     # Kafka messaging
│   │   └── util/          # Utilities
│   ├── crawler/           # Crawler service implementation
│   ├── analyzer/          # Analyzer service implementation
│   └── notification/      # Notification service implementation
├── pkg/                   # Public packages
├── scripts/               # Scripts
├── test/                  # Test files
├── docs/                  # Documentation
├── build/                 # Build files
│   ├── crawler/           # Crawler service Dockerfile
│   ├── analyzer/          # Analyzer service Dockerfile
│   └── notification/      # Notification service Dockerfile
├── docker-compose.yml     # Docker Compose configuration
├── Makefile               # Build automation
└── README.md              # Project documentation
```

## API Endpoints

### Crawler Service

- `GET /health` - Health check
- `GET /api/v1/crawler/categories` - Get all categories
- `GET /api/v1/crawler/categories/:id` - Get category by ID
- `GET /api/v1/crawler/products` - Get products with pagination
- `GET /api/v1/crawler/products/:id` - Get product details by ID
- `POST /api/v1/crawler/products/:id/priority` - Update product crawling priority
- `POST /api/v1/crawler/crawl/category/:id` - Trigger crawling for a category
- `POST /api/v1/crawler/crawl/product/:id` - Trigger crawling for a product

### Analyzer Service

- `GET /health` - Health check
- `GET /api/v1/analyzer/stats/products` - Get product statistics
- `GET /api/v1/analyzer/stats/prices` - Get price statistics
- `GET /api/v1/analyzer/stats/favorites` - Get favorite statistics
- `GET /api/v1/analyzer/trends/prices` - Get price trends
- `GET /api/v1/analyzer/trends/stock` - Get stock trends
- `GET /api/v1/analyzer/history/prices/:id` - Get price history for a product
- `GET /api/v1/analyzer/history/stock/:id` - Get stock history for a product
- `POST /api/v1/analyzer/alerts/price` - Create a price alert
- `GET /api/v1/analyzer/alerts/price/user/:id` - Get price alerts for a user
- `DELETE /api/v1/analyzer/alerts/price/:id` - Delete a price alert

### Notification Service

- `GET /health` - Health check
- `GET /api/v1/notifications` - Get notifications with pagination
- `GET /api/v1/notifications/unread` - Get unread notifications
- `PUT /api/v1/notifications/:id/read` - Mark a notification as read
- `PUT /api/v1/notifications/read-all` - Mark all notifications as read
- `GET /api/v1/notifications/ws/:user_id` - WebSocket endpoint for real-time notifications

## Development

### Making Changes

1. Create a new branch for your feature
```sh
git checkout -b feature/your-feature-name
```

2. Make your changes and run tests
```sh
make test
```

3. Run linter to ensure code quality
```sh
make lint
```

4. Build the services
```sh
make build
```

### Database Migrations

To apply database migrations:
```sh
make db-migrate
```

### Generating Protocol Buffers

After making changes to the `.proto` files, generate the Go code:
```sh
make proto
```

## License

This project is under the Muhammad Talha's  privileges.