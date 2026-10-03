#!/bin/bash
set -e

# 1
git add web-client/src/components/Transactions.tsx
git commit -m "frontend: overhaul transactions dashboard" -m "Implements bento-card aesthetic and functional client-side search filter."

# 2
git add migrations/000004_user_profile_fields.up.sql migrations/000004_user_profile_fields.down.sql
git commit -m "db: add user profile fields to schema" -m "Adds display_name and avatar_url to the users table."

# 3
git add internal/domain/types.go
git commit -m "domain: extend User model" -m "Adds DisplayName and AvatarURL properties to the domain User entity."

# 4
git add internal/repo/users.go
git commit -m "repo: implement UpdateUserProfile method" -m "Adds repository support for persisting user profile modifications."

# 5
git add web-client/src/components/Profile.tsx
git commit -m "frontend: wire profile settings UI to backend" -m "Adds state management, toast notifications, and avatar base64 upload."

# 6
git add android/app/src/main/java/app/ledger/client/MainActivity.kt
git commit -m "android: fix webview persistent auto-login and routing" -m "Injects access token from EncryptedSharedPreferences on webview boot and correctly routes to /dashboard instead of landing."

# 7
git add internal/parsers/cbe.go
git commit -m "parser: fix CBE balance regex extraction" -m "Supports CBE SMS strings containing 'is' or lacking colon delimiters to prevent NULL balances."

# 8
git add internal/agent/llm/client.go
git commit -m "llm: add ParseSMS fallback method" -m "Implements Gemini AI fallback for unmatched bank SMS formats with a strict schema."

# 9
git add internal/api/routes.go internal/api/handlers.go
git commit -m "api: wire profile updates and SMS AI fallback" -m "Registers PATCH /v1/profile and routes unmatched SMS formats to Gemini before queueing for manual review."

# 10
git add cmd/ledger-server/main.go
git commit -m "cmd: wire LLM parser in server initialization" -m "Connects Gemini client to the API server fallback parsing interface."

# 11
git add android/app/src/main/AndroidManifest.xml android/app/src/main/res/mipmap-*
git commit -m "android: apply ledger branding" -m "Updates Android manifest and mipmap resources to use the new Ledger brand icon."
