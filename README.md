# Goofy Ahh Expenses Tracker

A Telegram bot with mini app for tracking daily expenses and calculating spending budgets.

## Features

- 📱 **Telegram Mini App** - Add expenses through a beautiful web interface
- 💰 **Daily Budget Tracking** - Monitor spending against daily limits
- 📊 **Daily Reports** - Get spending summaries at 7pm daily
- 📁 **CSV Import/Export** - Upload existing data or export for backup
- 🔒 **HTTPS Support** - Production-ready with SSL certificates
- 🐳 **Docker Ready** - Easy deployment with Docker

## Quick Start

### 1. Setup Environment

```bash
# Copy environment template
cp env.example .env

# Edit .env with your Telegram bot token
nano .env
```

### 2. Build and Run with Docker

```bash
# Build the Docker image
make docker-build

# Run with SSL certificates (production)
make docker-run

# View logs
make docker-logs

# Stop the container
make docker-stop
```

### 3. Local Development

```bash
# Install dependencies
make deps

# Build locally
make build

# Run locally
./bin/goofy-ahh-expenses-tracker
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `TELEGRAM_BOT_TOKEN` | Your Telegram bot token | Required |
| `WEB_ADDRESS` | Web server address | `0.0.0.0:8080` |
| `DATA_PATH` | Path to CSV data file | `/app/data/data.csv` |
| `DAILY_REPORT_TIME` | Time for daily reports | `19:00` |
| `DAILY_REPORT_TIMEZONE` | Timezone for reports | `Europe/Moscow` |

### SSL Certificates

For production deployment, SSL certificates are automatically detected from:
- `/app/certs/fullchain.pem`
- `/app/certs/privkey.pem`

These are mounted from your Let's Encrypt certificates in the Docker container.

## Telegram Bot Commands

- `/start` - Welcome message and mini app access
- `/report` - Get today's spending summary
- `/csv` - Upload CSV file with expenses
- `/help` - Show help information

## CSV Format

The application expects CSV files with this exact header:
```csv
Date,Category,Description,Amount
2024-01-15,Food,Lunch,500.00
2024-01-15,Transport,Bus,50.00
```

## Project Structure

```
├── cmd/main.go              # Application entry point
├── config/config.go         # Configuration management
├── internal/
│   ├── bot/bot.go          # Telegram bot logic
│   ├── data/csv.go         # CSV data management
│   └── web/server.go       # Web server and API
├── static/                  # Web app assets
│   ├── index.html          # Mini app interface
│   ├── styles.css          # Styling
│   └── script.js           # Frontend logic
├── Dockerfile              # Docker configuration
├── Makefile                # Build and deployment commands
└── env.example             # Environment template
```

## Development

### Prerequisites

- Go 1.21+
- Docker (for production deployment)
- Telegram Bot Token

### Testing

```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./internal/data -v
```

### Building

```bash
# Build for local development
make build

# Build Docker image
make docker-build
```

## Production Deployment

1. **Set up SSL certificates** with Let's Encrypt
2. **Create `.env` file** with your bot token
3. **Update domain** in Makefile (replace `your-domain.com`)
4. **Run with Docker**:
   ```bash
   make docker-run
   ```

## Data Portability

All data is stored in CSV format, making it easy to:
- Export your data for backup
- Import into other applications
- Migrate to different systems

## License

This project is open source and available under the MIT License. 