prefix = /usr/local
bindir = $(DESTDIR)$(prefix)/bin
sharedir = $(DESTDIR)$(prefix)/share
docdir = $(DESTDIR)$(prefix)/share/doc

BUILD_ID = $(shell ./utils/build-id)

EVWEB_INPUT_DIRS = . ../data # relative to the evweb directory

BIN_DIR ?= $(CURDIR)/bin

GO_PKGS =					\
  github.com/exograd/eventline/cmd/eventline	\
  $(EVCLI_PKG)

EVCLI_PKG = github.com/exograd/eventline/cmd/evcli

GO_TEST_OPTIONS ?= -count 1

define evweb_make1
$(MAKE) -C evweb --no-print-directory		\
  $2 INPUT_DIR=$1 OUTPUT_DIR=../data/assets
endef

define evweb_make
$(foreach dir,$(EVWEB_INPUT_DIRS),$(call evweb_make1,$(dir),$1)
)
endef

define go_make1
CGO_ENABLED=0 \
go build -o $(BIN_DIR) \
  -ldflags="-X 'main.buildId=$(BUILD_ID)'" \
  $1
endef

define go_make
$(foreach dir,$(GO_PKGS),$(call go_make1,$(dir))
)
endef

DOC_PDF = $(CURDIR)/doc/handbook.pdf
DOC_HTML = $(CURDIR)/doc/handbook.html

all: build

assets:
	$(call evweb_make,build)

build: assets FORCE
	$(call go_make)

evcli: FORCE
	$(call go_make1,$(EVCLI_PKG))

check: vet

vet:
	go vet $(CURDIR)/...

test:
	go test $(GO_TEST_OPTIONS) $(CURDIR)/...

doc: doc-html doc-pdf

doc-html: FORCE
	$(call doc_genhtml)

doc-pdf: $(DOC_PDF) $(DOC_HTML) FORCE

doc/%.html: FORCE
	asciidoctor --backend html \
	            --destination-dir $(CURDIR)/doc/ \
	            $(basename $@)/$(basename $(notdir $@)).adoc

doc/%.pdf: FORCE
	asciidoctor-pdf --backend pdf \
	                --destination-dir $(CURDIR)/doc/ \
	                $(basename $@)/$(basename $(notdir $@)).adoc

install: build
	mkdir -p $(bindir)
	cp $(CURDIR)/bin/eventline $(bindir)/
	cp $(CURDIR)/bin/evcli $(bindir)/
	mkdir -p $(sharedir)
	cp -r $(CURDIR)/data $(sharedir)/eventline
	mkdir -p $(docdir)/eventline/
	mkdir -p $(docdir)/eventline/html
	cp -r $(DOC_HTML) $(docdir)/eventline/html
	cp -r $(DOC_PDF) $(docdir)/eventline

clean:
	$(RM) $(BIN_DIR)/*
	$(RM) $(DOC_PDF) $(DOC_HTML)

FORCE:

.PHONY: all assets build evcli clean
.PHONY: check vet test
.PHONY: doc doc-html doc-pdf
.PHONY: install
