# Makefile
.PHONY: help install dev build test clean

# SETUP COMMANDS
install:
	@echo "Установка инструментов разработки..."
	go install github.com/go-kratos/kratos/cmd/kratos/v2@latest
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest
	@echo "Инструменты установлены"

setup-auth0:
	@echo "Настройка Auth0..."
	@./scripts/setup/setup-auth0.sh
	@echo "Auth0 настроен"

setup-kong:
	@echo "Настройка Kong Gateway..."
	@./scripts/setup/setup-kong.sh
	@echo "Kong настроен"

# DEVELOPMENT
dev:
	@echo "Запуск среды разработки..."
	@make infra-up
	@make services-up
	@echo "Среда разработки готова"

infra-up:
	@echo "Запуск инфраструктуры..."
	docker-compose -f docker-compose.infra.yml up -d
	@echo "Ожидание запуска сервисов..."
	@sleep 10

services-up:
	@echo "Запуск микросервисов..."
	@cd services/user && make run &
	@cd services/bot && make run &
	@cd services/strategy && make run &
	@echo "Все сервисы запущены"

# BUILDING
proto:
	@echo "Генерация protobuf файлов..."
	@for service in services/*; do \
		if [ -d "$$service" ]; then \
			echo "Генерация proto для $$service..."; \
			cd $$service && make api && cd ../..; \
		fi \
	done
	@echo "Protobuf файлы сгенерированы"

build:
	@echo "Сборка всех сервисов..."
	@for service in services/*; do \
		if [ -d "$$service" ]; then \
			echo "Сборка $$service..."; \
			cd $$service && make build && cd ../..; \
		fi \
	done
	@echo "Сборка завершена"

docker-build:
	@echo "Сборка Docker образов..."
	@for service in services/*; do \
		if [ -d "$$service" ]; then \
			service_name=$$(basename $$service); \
			echo "Сборка Docker образа для $$service_name..."; \
			docker build -t tradingbot/$$service_name:latest $$service; \
		fi \
	done
	@echo "Docker образы собраны"

# TESTING
test:
	@echo "Запуск тестов..."
	@for service in services/*; do \
		if [ -d "$$service" ]; then \
			echo "Тестирование $$service..."; \
			cd $$service && go test ./... && cd ../..; \
		fi \
	done
	@echo "Тесты завершены"

test-api:
	@echo "Запуск API тестов..."
	@chmod +x scripts/test/api-test.sh
	@./scripts/test/api-test.sh
	@echo "API тесты завершены"

test-load:
	@echo "Запуск нагрузочных тестов..."
	@if command -v k6 >/dev/null 2>&1; then \
		k6 run scripts/test/load-test.js; \
	else \
		echo "k6 не установлен. Установите с https://k6.io"; \
	fi

# DEPLOYMENT
deploy-dev:
	@echo "Деплой в среду разработки..."
	@./scripts/deploy/deploy-dev.sh
	@echo "Деплой в разработку завершен"

deploy-prod:
	@echo "Деплой в продакшен..."
	@./scripts/deploy/deploy-prod.sh
	@echo "Деплой в продакшен завершен"

# UTILITIES
logs:
	@echo "Просмотр логов..."
	docker-compose logs -f

clean:
	@echo "Очистка..."
	docker-compose down -v
	docker system prune -f
	@for service in services/*; do \
		if [ -d "$$service" ]; then \
			cd $$service && make clean && cd ../..; \
		fi \
	done
	@echo "Очистка завершена"

status:
	@echo "Статус системы:"
	@echo "Kong Gateway:"
	@curl -s http://localhost:8001/status | jq .
	@echo "Сервисы:"
	@docker-compose ps

help:
	@echo "TradingBot Development Commands"
	@echo ""
	@echo "Setup:"
	@echo "  install       - Установить инструменты разработки"
	@echo "  setup-auth0   - Настроить Auth0"
	@echo "  setup-kong    - Настроить Kong Gateway"
	@echo ""
	@echo "Development:"
	@echo "  dev           - Запустить полную среду разработки"
	@echo "  infra-up      - Запустить только инфраструктуру"
	@echo "  services-up   - Запустить микросервисы"
	@echo ""
	@echo "Build:"
	@echo "  proto         - Генерировать protobuf файлы"
	@echo "  build         - Собрать все сервисы"
	@echo "  docker-build  - Собрать Docker образы"
	@echo ""
	@echo "Testing:"
	@echo "  test          - Запустить unit тесты"
	@echo "  test-api      - Запустить API интеграционные тесты"
	@echo "  test-load     - Запустить нагрузочные тесты"
	@echo ""
	@echo "Deploy:"
	@echo "  deploy-dev    - Деплой в среду разработки"
	@echo "  deploy-prod   - Деплой в продакшен"
	@echo ""
	@echo "Utils:"
	@echo "  logs          - Показать логи"
	@echo "  clean         - Очистить все"
	@echo "  status        - Показать статус системы"

.DEFAULT_GOAL := help