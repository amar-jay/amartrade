VERSION ?= dev
DATE ?= unknown

LDFLAGS := -X main.version=$(VERSION) -X main.date=$(DATE)

.PHONY: build

build:
	go build -ldflags "$(LDFLAGS)" -o bin/amartrade .