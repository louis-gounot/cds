UNAME := $(shell uname)
SHA512 := $(if ifeq ${UNAME} "Darwin",shasum -a 512,sha512sum)
VERSION := $(if ${CDS_SEMVER},${CDS_SEMVER},snapshot)

clean:
	@rm -rf dist

reset:
	@rm -rf node_modules
	@rm -rf semantic
	@rm -rf semantic\\
	@rm -f package-lock.json

install: $(NG)
	npm install

NG = node_modules/@angular/cli/bin/ng

stats: $(NG)
	@node --max_old_space_size=4096 node_modules/@angular/cli/bin/ng build --prod --stats-json

ngbuild: $(NG)
	@node --max_old_space_size=4096 node_modules/@angular/cli/bin/ng build --prod

INDEX = dist/index.tmpl
$(INDEX):
	$(MAKE) ngbuild
	@cd dist && sed -i -e "s|window.cds_version = '';|window.cds_version='${VERSION}';|g" index.html && cp index.html index.tmpl

FILES_UI = dist/FILES_UI
$(FILES_UI): $(INDEX)
	$(info sha512 = ${SHA512})
	touch $(FILES_UI)
	cd dist/ && for i in `ls -p | grep -v /|grep -v FILES_UI|grep -v index.html`; do echo "$$i;`${SHA512} $$i|cut -d ' ' -f1`" >> FILES_UI; done;

build: $(FILES_UI) $(INDEX) ui.tar.gz

ui.tar.gz:
	tar cfz ui.tar.gz dist
