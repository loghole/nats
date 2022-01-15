APP_CMD := nats
DOCKER_COMPOSE_RUN ?= docker-compose -f docker-compose.yml

export CGO_ENABLED := 0

.PHONY: test
test:
	${DOCKER_COMPOSE_RUN} run --rm test /bin/sh -c "go test -tags=integration ./..."
	${DOCKER_COMPOSE_RUN} run --rm test /bin/sh -c "rm -r ./data"
	#${DOCKER_COMPOSE_RUN} down

.PHONY: lint
lint:
	${DOCKER_COMPOSE_RUN} run --rm linter /bin/sh -c "golangci-lint run ./... -c .golangci.yaml -v"

.PHONY: tidy
tidy: 
	go mod tidy

.PHONY: imports
imports:
	if goimports --local "github.com/loghole/$(APP_CMD)" -w -d $$(find . -type f -name '*.go' -not -path "./data/*"); then \
		echo; \
	else \
		${DOCKER_COMPOSE_RUN} run --rm -u $(USER_ID) goimports /bin/sh -c "\
			goimports --local "github.com/loghole/$(APP_CMD)" -w -d $$(find . -type f -name '*.go' -not -path "./data/*") ./ \
		"; \
	fi
