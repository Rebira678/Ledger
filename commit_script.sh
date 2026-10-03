#!/bin/bash
set -e

# Remove scratch files
rm -f test_cbe.go test_cbe2.go test_parsers.go

# 1. Domain Types
git add internal/domain/types.go
git commit -m "feat(domain): extend transaction models with exact balance tracking"

# 2. Database Transactions
git add internal/repo/transactions.go
git commit -m "feat(repo): implement sql persistence for transaction balances"

# 3. Reports Refactoring
git add internal/repo/reports.go
git commit -m "refactor(repo): optimize balance estimation to query most recent transaction"

# 4. SMS Parsers
git add internal/parsers/cbe.go internal/parsers/telebirr.go
git commit -m "feat(parsers): implement regex-based balance extraction for CBE and Telebirr"

# 5. Device Repo
git add internal/repo/users.go
git commit -m "feat(repo): implement dynamic user device list querying"

# 6. API Handlers & Routes
git add internal/api/handlers.go internal/api/routes.go
git commit -m "feat(api): wire balance ingestion logic and expose /v1/profile endpoint"

# 7. Frontend Dashboard
git add web-client/src/components/Dashboard.tsx
git commit -m "feat(ui): replace static dashboard metrics with real dynamic balances"

# 8. Frontend Transactions
git add web-client/src/components/Transactions.tsx
git commit -m "style(ui): apply contextual semantic colors to transaction list"

# 9. Frontend Clarifications
git add web-client/src/components/Clarifications.tsx
git commit -m "refactor(ui): condense clarifications into streamlined notification layout"

# 10. Frontend Profile & App Shell
git add web-client/src/components/Profile.tsx web-client/src/App.tsx web-client/src/index.css
git commit -m "feat(ui): implement premium split-pane profile and settings layout"

# 11. Frontend Landing
git add web-client/src/components/Landing.tsx
git commit -m "style(ui): harmonize landing page aesthetics with global design system"

# 12. Android Web App integration
git add android/
git commit -m "feat(android): integrate native webview for unified cross-platform UX"

echo "Commits completed successfully!"
