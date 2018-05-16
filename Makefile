# Package configuration
PROJECT = datasets
COMMANDS = PublicGitArchive/borges-indexer/cmd/borges-indexer PublicGitArchive/multitool PublicGitArchive/pga

# Including ci Makefile
CI_REPOSITORY ?= https://github.com/src-d/ci.git
CI_PATH ?= $(shell pwd)/.ci
CI_VERSION ?= v1
PKG_OS = linux darwin windows

MAKEFILE := $(CI_PATH)/Makefile.main
$(MAKEFILE):
	git clone --quiet --branch $(CI_VERSION) --depth 1 $(CI_REPOSITORY) $(CI_PATH);

-include $(MAKEFILE)
