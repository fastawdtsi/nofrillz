COMPOSE ?= docker compose
SEED_USER_COUNT ?= 25

.PHONY: up down logs ps build migrate seed test

up:
	$(COMPOSE) up -d --build --wait

down:
	$(COMPOSE) down

logs:
	$(COMPOSE) logs -f

ps:
	$(COMPOSE) ps -a

build:
	$(COMPOSE) --profile tools build

migrate:
	$(COMPOSE) run --rm migrate

seed:
	$(COMPOSE) --profile tools run --rm --build seed /app/config.yaml $(SEED_USER_COUNT)

test:
	$(COMPOSE) run --rm --no-deps --entrypoint /bin/sh migrate -n /scripts/migrate.sh
	$(COMPOSE) config --quiet
