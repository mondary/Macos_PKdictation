SHELL := /bin/bash

CONFIGURATION ?= Debug

.PHONY: build run relaunch status quit export crash test clean

build:
	./scripts/build.sh "$(CONFIGURATION)"

run:
	./scripts/run.sh "$(CONFIGURATION)"

relaunch:
	./scripts/run.sh "$(CONFIGURATION)" --relaunch

status:
	./scripts/status.sh || true

quit:
	./scripts/quit.sh

export:
	./scripts/export.sh "$(CONFIGURATION)"

crash:
	./scripts/crash_latest.sh

test:
	./scripts/test.sh "$(CONFIGURATION)"

clean:
	./scripts/clean.sh
