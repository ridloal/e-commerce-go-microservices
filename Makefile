.PHONY: help build up down logs ps restart stop rm prune-volumes migrate-all migrate-user migrate-product migrate-warehouse migrate-order

help:
	@echo "Available commands:"
	@echo "  build             : Build or rebuild services"
	@echo "  up                : Create and start containers in detached mode"
	@echo "  down              : Stop and remove containers, networks"
	@echo "  logs              : Follow log output"
	@echo "  logs <service>    : Follow log output for a specific service (e.g., make logs api_gateway)"
	@echo "  ps                : List containers"
	@echo "  restart           : Restart all services"
	@echo "  restart <service> : Restart specific service (e.g., make restart api_gateway)"
	@echo "  stop              : Stop services"
	@echo "  rm                : Remove stopped service containers"
	@echo "  prune-volumes     : Remove all unused local volumes (DANGEROUS for DB data if not careful)"
	@echo "  migrate-all       : Run migrations for all services (after 'up' command)"
	@echo "  migrate-user      : Run user service migrations"
	@echo "  migrate-product   : Run product service migrations"
	@echo "  migrate-warehouse : Run warehouse service migrations"
	@echo "  migrate-order     : Run order service migrations"

build:
	docker-compose build $(filter-out $@,$(MAKECMDGOALS))

up:
	docker-compose up -d --remove-orphans

down:
	docker-compose down

logs:
	docker-compose logs -f $(filter-out $@,$(MAKECMDGOALS))

ps:
	docker-compose ps

restart:
	docker-compose restart $(filter-out $@,$(MAKECMDGOALS))

stop:
	docker-compose stop

rm:
	docker-compose rm -f

prune-volumes:
	docker-compose down -v # Ini akan menghapus volume anonim dan named volume yang terkait dengan compose file ini.
	@read -p "Are you sure you want to remove ALL unused Docker volumes? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		docker volume prune -f; \
	fi

# Perintah migrasi ini menjalankan container migrasi yang didefinisikan di docker-compose
# Container ini akan exit setelah selesai, jadi kita 'up' lalu 'logs' lalu 'rm'
# Sebenarnya, `restart: on-failure` dan `depends_on` sudah cukup untuk menjalankan migrasi saat `make up`.
# Perintah migrate-* di bawah ini lebih untuk menjalankan migrasi secara manual JIKA diperlukan.
define run_migration
	@echo "Running migration for $(1)..."
	docker-compose up -d migrate_$(1)
	@echo "Waiting for migrate_$(1) to complete. Check logs with: docker-compose logs -f migrate_$(1)"
	@echo "To clean up, run: docker-compose rm -f migrate_$(1)"
endef

migrate-all: migrate-user migrate-product migrate-warehouse migrate-order
	@echo "All migrations triggered. Monitor logs."

migrate-user:
	$(call run_migration,user)

migrate-product:
	$(call run_migration,product)

migrate-warehouse:
	$(call run_migration,warehouse)

migrate-order:
	$(call run_migration,order)

# Jika ingin menjalankan service tertentu dan dependensinya
# Contoh: make service-up api_gateway
service-up:
	docker-compose up -d $(filter-out $@,$(MAKECMDGOALS))

# Jika ingin build service tertentu
# Contoh: make service-build api_gateway
service-build:
	docker-compose build $(filter-out $@,$(MAKECMDGOALS))

# Default target
all: up