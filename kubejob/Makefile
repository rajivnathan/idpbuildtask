PROJECT := idpbuildtask/kubejob
GITCOMMIT := $(shell git rev-parse --short HEAD 2>/dev/null)
PKGS := $(shell go list  ./... | grep -v $(PROJECT)/vendor | grep -v $(PROJECT)/tests )
COMMON_FLAGS := -X $(PROJECT)/pkg/kdo/cli/version.GITCOMMIT=$(GITCOMMIT)
BUILD_FLAGS := -ldflags="-w $(COMMON_FLAGS)"
DEBUG_BUILD_FLAGS := -ldflags="$(COMMON_FLAGS)"
FILES := idcbuildtask

default: bin

.PHONY: bin
bin:
	go build ${BUILD_FLAGS} -o idpbuildtask

# .PHONY: install
# install:
# 	go install ${BUILD_FLAGS} ./cmd/kdo/

.PHONY: clean
clean:
	@rm -rf $(FILES)

.PHONY: test
test:
	go test -race $(PKGS)