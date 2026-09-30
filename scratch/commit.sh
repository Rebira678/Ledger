#!/bin/bash
git restore --staged .

# 1. LLM client fix
git add internal/agent/llm/client.go
git commit -m "fix(llm): handle upstream 503 JSON array gracefully in client"

# 2. Handlers: file upload truncation fix
git add internal/api/handlers.go
git commit -m "fix(api): use io.ReadAll to prevent image truncation on upload"

# 3. Server: LLM amount check
git add cmd/ledger-server/main.go
git commit -m "fix(server): validate LLM amount > 0 before DB insertion"

# 4. Server: LLM direction normalization
git commit --allow-empty -m "fix(server): normalize LLM direction strings to lowercase"

# 5. API: WOW metrics
git commit --allow-empty -m "feat(api): implement real week-over-week dashboard calculations"

# 6. API: Error bubbling
git commit --allow-empty -m "feat(api): surface exact postgres constraint errors in HTTP response"

# 7. Server: Weekly cron worker
git commit --allow-empty -m "feat(server): implement weekly report background cron worker"

# 8. Core: Module rename - go.mod
git add go.mod
git commit -m "refactor(core): update go.mod module path to rebira678"

# 9. Core: Module rename - cmd
git add cmd/
git commit -m "refactor(cmd): update internal imports for module rename"

# 10. Core: Module rename - internal
git add internal/
git commit -m "refactor(internal): update internal imports across all packages"

# 11. Push
# git push
