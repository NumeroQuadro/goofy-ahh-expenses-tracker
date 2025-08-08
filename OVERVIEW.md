### Goofy Ahh Expenses Tracker — Overview

Brief Telegram bot + Mini App for daily expense tracking with CSV storage. Runs well behind Nginx under a subpath.

### What it does
- **Mini App UI**: Add expenses with date, category, description, amount.
- **Daily report**: `/report` shows a per‑day summary (timezone aware) and attaches a full CSV export.
- **CSV import**: Validate and import user CSV with strict header.
- **CSV export**: `/export` returns all data as a CSV file.
- **Budgeting**: Daily saldo/allowance derived from monthly budget (evenly distributed across the month).

### Key technical details
- **Tech stack**: Go + Gin HTTP server, Telegram Bot API v5.
- **Data model**: Flat CSV with header `Date,Category,Description,Amount`. Concurrency guarded by a mutex; every write rewrites the file to keep it simple and portable.
- **Routes (behind subpath)**:
  - UI: `GET /expenses/` (serves `static/index.html`)
  - Static: `GET /expenses/static/*`
  - API: `POST /expenses/transaction`, `POST /expenses/upload-csv`, `GET /expenses/transactions[?date=YYYY-MM-DD]`
- **Reverse proxy aware**: Assets are served under `/expenses/static`; URLs in HTML/JS are subpath‑safe.
- **Duplicate prevention**: When a request carries `chat_id`, persistence is delegated to the bot handler to avoid double‑saving (API + bot).
- **Timezone**: Respects `DAILY_REPORT_TIMEZONE` (requires `tzdata` in the container).
- **TLS**: App can run plain HTTP and sit behind Nginx TLS, or terminate TLS inside the container if certs are mounted at `/app/certs`.

### Environment variables
- **TELEGRAM_BOT_TOKEN**: Bot token (required)
- **WEB_ADDRESS**: Bind address, default `0.0.0.0:8088`
- **DATA_PATH**: CSV path (default `/app/data/data.csv` in Docker)
- **DAILY_REPORT_TIME**: HH:MM for scheduled sending (placeholder hook)
- **DAILY_REPORT_TIMEZONE**: e.g., `Europe/Moscow`
- **MONTHLY_BUDGET_RUB**: Float, monthly budget used for saldo math (default 12000)

### Docker and Nginx
- **Container**: exposes `8088` by default. Image includes `tzdata` for timezone support.
- **Example run**:
```bash
docker run -d --name goofy-ahh-expenses-tracker \
  -p 8088:8088 \
  -e TELEGRAM_BOT_TOKEN=xxx \
  -e WEB_ADDRESS=0.0.0.0:8088 \
  -e DATA_PATH=data.csv \
  -e DAILY_REPORT_TIME=19:00 \
  -e DAILY_REPORT_TIMEZONE=Europe/Moscow \
  -e MONTHLY_BUDGET_RUB=12000 \
  goofy-ahh-expenses-tracker
```
- **Nginx snippet** (serve under `/expenses/`):
```nginx
location = / { return 302 /expenses/; }
location /expenses/ {
  proxy_pass http://127.0.0.1:8088/expenses/;
  proxy_set_header Host $host;
  proxy_set_header X-Real-IP $remote_addr;
  proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
  proxy_set_header X-Forwarded-Proto $scheme;
}
```

### Bot commands
- `/start` open Mini App
- `/report` or `/report YYYY-MM-DD` daily summary + CSV attachment
- `/csv` CSV upload instructions
- `/export` CSV with all expenses
- `/help` quick help

### Notes
- Gin currently runs in debug; set `GIN_MODE=release` in production.
- CSV header is strict; imports must match exactly.
- App logs may warn about trusted proxies; set `SetTrustedProxies` if you want to restrict.


