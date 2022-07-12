prefix = /usr/local
bindir = $(DESTDIR)$(prefix)/bin
sharedir = $(DESTDIR)$(prefix)/share
docdir = $(DESTDIR)$(prefix)/share/doc

BUILD_ID = $(shell ./utils/build-id)

EVWEB_INPUT_DIRS = . ../data # relative to the evweb directory

BIN_DIR ?= bin

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

DOC_PDF = doc/handbook.pdf
DOC_HTML = doc/handbook.html

ASCIIDOCTOR_OPTIONS = -a revnumber=$(BUILD_ID)

all: build

assets:
	$(call evweb_make,build)

build: assets FORCE
	$(call go_make)

evcli: FORCE
	$(call go_make1,$(EVCLI_PKG))

check: vet

vet:
	go vet ./...

test:
	go test $(GO_TEST_OPTIONS) ./...

doc: doc-html doc-pdf

doc-html: $(DOC_HTML)

doc-pdf: $(DOC_PDF)

.SECONDEXPANSION:
doc/%.html: $$(wildcard doc/%/*) doc/pdf-theme.yml
	asciidoctor --backend html \
	            --destination-dir doc/ \
	            $(ASCIIDOCTOR_OPTIONS) \
	            $(basename $@)/$(basename $(notdir $@)).adoc

.SECONDEXPANSION:
doc/%.pdf: $$(wildcard doc/%/*) doc/pdf-theme.yml
	asciidoctor-pdf --backend pdf \
	                --destination-dir doc/ \
	                $(ASCIIDOCTOR_OPTIONS) \
	                -a pdf-theme=doc/pdf-theme.yml \
	                -a pdf-fontsdir=doc/fonts \
	                $(basename $@)/$(basename $(notdir $@)).adoc

install: build
	mkdir -p $(bindir)
	cp $(wildcard bin/*) $(bindir)
	mkdir -p $(sharedir)/eventline
	cp -r data/assets $(sharedir)/eventline
	cp -r data/pg $(sharedir)/eventline
	cp -r data/templates $(sharedir)/eventline
	mkdir -p $(docdir)/eventline
	cp -r $(DOC_PDF) $(docdir)/eventline
	mkdir -p $(docdir)/eventline/html
	cp -r $(DOC_HTML) $(docdir)/eventline/html

install-flat: build
	@if [ -z "$(DESTDIR)" ]; then echo "DESTDIR not set" >&2; exit 1; fi
	mkdir -p $(DESTDIR)
	cp $(wildcard bin/*) $(DESTDIR)
	mkdir -p $(DESTDIR)/data
	cp -r data/assets $(DESTDIR)/data
	cp -r data/pg $(DESTDIR)/data
	cp -r data/templates $(DESTDIR)/data
	mkdir -p $(DESTDIR)/doc
	cp -r $(DOC_PDF) $(DESTDIR)/doc
	mkdir -p $(DESTDIR)/doc/html
	cp -r $(DOC_HTML) $(DESTDIR)/doc/html

clean:
	$(RM) $(BIN_DIR)/*
	$(RM) $(DOC_PDF) $(DOC_HTML)

FORCE:

.PHONY: all assets build evcli clean
.PHONY: check vet test
.PHONY: doc doc-html doc-pdf
.PHONY: install
