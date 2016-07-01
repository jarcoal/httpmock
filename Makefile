# Makefile for thingful/httpmock
#
# Targets:
# 	test: runs tests
#
GOCMD=go
GOTEST=$(GOCMD) test -v
GOCOVER=$(GOCMD) tool cover
COVERAGE=coverage.out

.PHONY: test
test:
	$(GOTEST) -coverprofile=$(COVERAGE)

.PHONY: clean
clean:
	rm -f $(COVERAGE)

.PHONY: cover
cover: test
	$(GOCOVER) -func=$(COVERAGE)
