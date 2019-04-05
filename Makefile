DEP := bin/dep
MAGE := go run build.go

.PHONY: setup
setup:
	mkdir -p bin
ifeq (,$(wildcard $(DEP)))
	curl -s https://raw.githubusercontent.com/golang/dep/master/install.sh | INSTALL_DIRECTORY=bin sh
endif
	$(DEP) ensure -v

.PHONY: all
all: setup
	$(MAGE) build:all

.PHONY: build
build:
	$(MAGE) build:build

.PHONY: publish
publish:
	$(MAGE) publish:all

.PHONY: git-check
git-check:
	$(MAGE) git:checkStatus

.PHONY: test
test:
	$(MAGE) test

.PHONY: clean
clean:
	$(MAGE) build:clean

.PHONY: generate
generate:
	$(MAGE) generate:all

.PHONY: version
version:
	$(MAGE) version:print

.PHONY: version-set
version-set:
	$(MAGE) version:set
