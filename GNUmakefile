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
CGO_ENABLED=0					\
go build -o $(BIN_DIR)				\
  -ldflags="-X 'main.buildId=$(BUILD_ID)'"	\
  $1
endef

define go_make
$(foreach dir,$(GO_PKGS),$(call go_make1,$(dir))
)
endef

DOC_PDF = doc/handbook.pdf
DOC_HTML = doc/handbook/handbook.html

ASCIIDOCTOR_OPTIONS = -v -a revnumber=$(BUILD_ID)

DOCKER_IMAGES =					\
  exograd/eventline

define docker_build1
DOCKER_BUILDKIT=1							\
docker build --no-cache							\
  --label org.opencontainers.image.created=$(shell date -u +%FT%TZ)	\
  --label org.opencontainers.image.version=$(BUILD_ID)			\
  --label org.opencontainers.image.revision=$(shell git rev-parse HEAD)	\
  --tag $1:$(patsubst v%,%,$(BUILD_ID))					\
  --tag $1:latest							\
  .
endef

define docker_build
$(foreach image,$(DOCKER_IMAGES),$(call docker_build1,$(image))
)
endef

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
%.html: $$(wildcard doc/**/*.adoc) doc/html-theme.css
	asciidoctor --backend html \
	            --destination-dir $(dir $@) \
	            -a docinfo=shared \
	            -a nofooter \
	            -a rouge-style=base16.dark \
	            -a stylesdir=.. \
	            -a stylesheet=html-theme.css \
	            $(ASCIIDOCTOR_OPTIONS) \
	            $(subst .html,.adoc,$@)

# For some reason, using the compressed style breaks @font-face blocks. I have
# no idea why it breaks here but not in Eventline. I will take a patch to fix
# this if someone find the root cause.
doc/html-theme.css: doc/html-theme.scss
	sass --no-error-css --no-source-map --style expanded $<:$@

.SECONDEXPANSION:
%.pdf: $$(wildcard doc/**/*.adoc) doc/pdf-theme.yml
	asciidoctor-pdf --backend pdf \
	                --destination-dir doc/ \
	                $(ASCIIDOCTOR_OPTIONS) \
	                -a rouge-style=base16.solarized.light \
	                -a pdf-theme=doc/pdf-theme.yml \
	                -a pdf-fontsdir=doc/fonts \
	                $(basename $@)/$(basename $(notdir $@)).adoc

docker-images: build doc FORCE
	$(call docker_build)

install: build doc
	mkdir -p $(bindir)
	cp $(wildcard bin/*) $(bindir)
	mkdir -p $(sharedir)/licenses/eventline
	cp LICENSE $(sharedir)/licenses/eventline
	mkdir -p $(sharedir)/eventline
	cp -r data/assets $(sharedir)/eventline
	cp -r data/pg $(sharedir)/eventline
	cp -r data/templates $(sharedir)/eventline
	mkdir -p $(docdir)/eventline
	cp -r $(DOC_PDF) $(docdir)/eventline
	mkdir -p $(docdir)/eventline/html
	cp -r $(DOC_HTML) $(docdir)/eventline/html
	cp -r $(dir $(DOC_HTML))images $(docdir)/eventline/html
	mkdir -p $(docdir)/eventline/fonts
	cp -r doc/fonts/*.woff2 $(docdir)/eventline/fonts

install-flat: build doc
	@if [ -z "$(DESTDIR)" ]; then echo "DESTDIR not set" >&2; exit 1; fi
	mkdir -p $(DESTDIR)
	cp $(wildcard bin/*) $(DESTDIR)
	cp LICENSE $(DESTDIR)
	mkdir -p $(DESTDIR)/data
	cp -r data/assets $(DESTDIR)/data
	cp -r data/pg $(DESTDIR)/data
	cp -r data/templates $(DESTDIR)/data
	mkdir -p $(DESTDIR)/doc
	cp -r $(DOC_PDF) $(DESTDIR)/doc
	mkdir -p $(DESTDIR)/doc/html
	cp -r $(DOC_HTML) $(DESTDIR)/doc/html
	cp -r $(dir $(DOC_HTML))images $(DESTDIR)/doc/html
	mkdir -p $(DESTDIR)/doc/fonts
	cp -r doc/fonts/*.woff2 $(DESTDIR)/doc/fonts

clean:
	$(call evweb_make,clean)
	$(RM) $(BIN_DIR)/*
	$(RM) $(DOC_PDF) $(DOC_HTML) doc/html-theme.css

FORCE:

.PHONY: all assets build evcli clean
.PHONY: check vet test
.PHONY: doc doc-html doc-pdf
.PHONE: docker-images
.PHONY: install install-flat
