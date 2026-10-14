include .env
export
.PHONY: help up down logs logs-t logs-cf logs-cf-t logs-all logs-all-t logs-errors logs-500 logs-4xx logs-important logs-rate logs-slow restart rebuild run build test cover lint tidy clean clean-docker check migrate-up migrate-down migrate-version migrate-new migrate-force

help:
	@echo "RepoReef Backend — Komutlar:"
	@echo ""
	@echo "  ── DOCKER ──"
	@echo "  up             Stack'i baslat"
	@echo "  down           Stack'i durdur"
	@echo "  restart        Backend'i yeniden baslat"
	@echo "  rebuild        Kod degisti, yeniden derle"
	@echo ""
	@echo "  ── LOGS (bu terminal) ──"
	@echo "  logs           Backend loglari (tek satir, renkli degil)"
	@echo "  logs-cf        Cloudflared loglari"
	@echo "  logs-all       Tum loglar"
	@echo ""
	@echo "  ── LOGS (yeni tab) ──"
	@echo "  logs-t         Backend loglari (yeni tab)"
	@echo "  logs-cf-t      Cloudflared loglari (yeni tab)"
	@echo "  logs-all-t     Tum loglar (yeni tab)"
	@echo ""
	@echo "  ── LOGS (filtreli, canli) ──"
	@echo "  logs-errors    Sadece error seviyesi (JSON)"
	@echo "  logs-500       Sadece 5xx donen istekler"
	@echo "  logs-4xx       Sadece 4xx donen istekler"
	@echo "  logs-important Error + warn + 5xx"
	@echo "  logs-rate      Rate limit olaylari"
	@echo "  logs-slow      500ms ustu yavas istekler"
	@echo ""
	@echo "  ── KONTROL ──"
	@echo "  check          build + vet + lint (+ test)"
	@echo "  build          Binary uret"
	@echo "  test           Testleri calistir"
	@echo "  cover          Coverage raporu (test sonrasi)"
	@echo "  lint           golangci-lint"
	@echo "  tidy           go mod tidy"
	@echo "  clean          Binary + coverage temizle"
	@echo "  clean-docker   Docker cache + unused image/volume temizle"
	@echo ""
	@echo "  ── MIGRATIONS ──"
	@echo "  migrate-up     Bekleyen migration'ları uygula"
	@echo "  migrate-down   Son migration'ı geri al"
	@echo "  migrate-version Aktif sürüm"
	@echo "  migrate-new    Yeni migration dosyası oluştur"
	@echo ""
	@echo "  cd ~/Masaüstü/RepoReef/Backend"

# ═══════════════════════════════════════════════════════════
# DOCKER
# ═══════════════════════════════════════════════════════════

up:
	docker compose up -d --build
	@echo ""
	@echo "Stack baslatildi. Loglar icin: make logs-all  (veya make logs-all-t)"

down:
	docker compose down

# ═══════════════════════════════════════════════════════════
# LOGS — Genel
# ═══════════════════════════════════════════════════════════

logs:
	@docker compose logs --no-log-prefix -f --tail 20 backend | python3 scripts/pretty-logs.py

logs-t:
	gnome-terminal --tab --title="RepoReef - Backend Logs" -- bash -c "cd $(PWD) && docker compose logs -f --tail 20 backend; exec bash"

logs-cf:
	docker compose logs -f --tail 20 cloudflared

logs-cf-t:
	gnome-terminal --tab --title="RepoReef - Cloudflared Logs" -- bash -c "cd $(PWD) && docker compose logs -f --tail 20 cloudflared; exec bash"

logs-all:
	docker compose logs -f --tail 20 backend cloudflared

logs-all-t:
	gnome-terminal --tab --title="RepoReef - All Logs" -- bash -c "cd $(PWD) && docker compose logs -f --tail 20 backend cloudflared; exec bash"

# ═══════════════════════════════════════════════════════════
# LOGS — Filtreli (canli, sadece ilgili satirlar)
# ═══════════════════════════════════════════════════════════

# jq gerekir: sudo apt install jq
# JSON log satırlarını renkli, filtrelenmiş gösterir.

logs-errors:
	docker compose logs -f --tail 0 backend | jq -R -r 'fromjson? | select(.level=="ERROR")'

logs-500:
	docker compose logs -f --tail 0 backend | jq -R -r 'fromjson? | select(.status >= 500)'

logs-4xx:
	docker compose logs -f --tail 0 backend | jq -R -r 'fromjson? | select(.status >= 400 and .status < 500)'

logs-important:
	docker compose logs -f --tail 0 backend | jq -R -r 'fromjson? | select((.level=="ERROR" or .level=="WARN") or .status >= 500)'

logs-rate:
	docker compose logs -f --tail 0 backend | jq -R -r 'fromjson? | select(.msg=="rate limit exceeded")'

logs-slow:
	docker compose logs -f --tail 0 backend | jq -R -r 'fromjson? | select(.duration_ms > 500)'

# ═══════════════════════════════════════════════════════════
# DOCKER — Restart / Rebuild
# ═══════════════════════════════════════════════════════════

restart:
	docker compose restart backend

rebuild:
	docker compose up -d --build backend

# ═══════════════════════════════════════════════════════════
# GO
# ═══════════════════════════════════════════════════════════

run:
	go run ./cmd/server

# ═══════════════════════════════════════════════════════════
# KONTROL
# ═══════════════════════════════════════════════════════════

check:
	@echo "═══ 1/4  build ═══"
	go build ./...
	@echo ""
	@echo "═══ 2/4  vet ═══"
	go vet ./...
	@echo ""
	@echo "═══ 3/4  lint ═══"
	golangci-lint run --timeout=5m
	@echo ""
	@echo "═══ 4/4  test ═══"
	go test ./internal/... -race -coverprofile=coverage.out
	@echo ""
	@echo "✔ Tum kontroller gecti."

build:
	go build -o reporeef-api ./cmd/server

test:
	go test ./internal/... -race -coverprofile=coverage.out

cover: ## Coverage raporu (test sonrası)
	@go tool cover -func=coverage.out

lint:
	golangci-lint run --timeout=5m

tidy:
	go mod tidy

clean:
	rm -f reporeef-api coverage.out

clean-docker: ## Docker build cache + unused image/volume temizle
	@echo "═══ Docker temizlik ═══"
	@echo ""
	@echo "Önce:"
	@docker system df
	@echo ""
	docker builder prune -a -f
	docker image prune -a -f
	docker volume prune -f
	docker container prune -f
	@echo ""
	@echo "Sonra:"
	@docker system df
	@echo ""
	@echo "✔ Temizlik tamamlandı."

	# ═══════════════════════════════════════════════════════════
# MIGRATIONS
# ═══════════════════════════════════════════════════════════

migrate-up:      ## Bekleyen migration'ları uygula
	@migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:    ## Son migration'ı geri al
	@migrate -path migrations -database "$(DATABASE_URL)" down 1

migrate-version: ## Aktif migration sürümü
	@migrate -path migrations -database "$(DATABASE_URL)" version

migrate-force:   ## Baseline (kullanım: make migrate-force V=2)
	migrate -path migrations -database "$(DATABASE_URL)" force $(V)

migrate-new:     ## Yeni migration (kullanım: make migrate-new NAME=add_foo)
	@if [ -z "$(NAME)" ]; then echo "Kullanım: make migrate-new NAME=add_foo"; exit 1; fi
	@N=$$(printf "%03d" $$(($$(migrate -path migrations -database "$(DATABASE_URL)" version 2>/dev/null | tail -1) + 1))); \
	touch migrations/$${N}_$(NAME).up.sql migrations/$${N}_$(NAME).down.sql; \
	echo "✔ Oluşturuldu: migrations/$${N}_$(NAME).{up,down}.sql"