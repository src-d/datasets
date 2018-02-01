# Package configuration
PROJECT = core-retrieval
COMMANDS =
GOFLAGS = -tags norwfs

# Including ci Makefile
MAKEFILE = Makefile.main
CI_REPOSITORY = https://github.com/src-d/ci.git
CI_FOLDER = .ci

$(MAKEFILE):
	@git clone --quiet $(CI_REPOSITORY) $(CI_FOLDER); \
	cp $(CI_FOLDER)/$(MAKEFILE) .;

-include $(MAKEFILE)

ensure-models-generated: generate-models ensure-no-changes

generate-models:
	@go get -v -u `go list -f '{{ join .Deps  "\n"}}' . | grep kallax | grep -v types`/...; \
	go generate ./...; \

ensure-no-changes:
	@git --no-pager diff && \
	if [ `git status | grep 'Changes not staged for commit' | wc -l` != '0' ]; then \
		echo 'There are differences between the commited files and the one(s) generated right now'; \
		exit 2; \
	fi; \

ensure-schema-generated: schema ensure-no-changes

schema:
	go get github.com/jteeuwen/go-bindata/... && \
	go-bindata -nometadata -pkg schema -o schema/bindata.go ./schema/sql

test-coverage: test-with-hdfs
test: test-with-hdfs
test-with-hdfs:
	sh setup_hdfs.sh

.PHONY: schema
