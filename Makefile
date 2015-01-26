TAG := $(shell git describe --tags)
LAST_TAG := $(shell git describe --tags --abbrev=0)
NEW_TAG := $(shell echo $(LAST_TAG) | perl -lpe 's/v//; $$_ += 0.01; $$_ = sprintf("v%.2f", $$_)')

snapshot-manager:
	@go version
	go install -v ./...

test:	snapshot-manager
	go test ./...

doc:	snapshot-manager.1 README.html README.txt

snapshot-manager.1:	README.md Makefile
	pandoc -M "title=SNAPSHOT-MANAGER(1)" -M "date="`date -Id`  -M "author=Memset Ltd" -s --from markdown --to man README.md -o snapshot-manager.1

README.html:	README.md Makefile
	pandoc -s --from markdown_github --to html README.md -o README.html

README.txt:	README.md Makefile
	pandoc -s --from markdown_github --to plain README.md -o README.txt

install: snapshot-manager
	install -d ${DESTDIR}/usr/bin
	install -t ${DESTDIR}/usr/bin ${GOPATH}/bin/snapshot-manager

clean:
	go clean ./...
	find . -name \*~ | xargs -r rm -f
	rm -rf build
	rm -f snapshot-manager snapshot-manager.1 README.html README.txt

.PHONY:	upload

upload:
	./upload $(TAG)

cross:	doc
	./cross-compile $(TAG)

tag:
	@echo "Old tag is $(LAST_TAG)"
	@echo "New tag is $(NEW_TAG)"
	echo -e "package main\n const Version = \"$(NEW_TAG)\"\n" | gofmt > version.go
	git tag $(NEW_TAG)
	@echo "Add this to changelog in README.md"
	@echo "  * $(NEW_TAG) -" `date -I`
	@git log $(LAST_TAG)..$(NEW_TAG) --oneline
	@echo "Then commit the changes"
	@echo git commit -m "Version $(NEW_TAG)" -a -v
	@echo "And finally run make retag before make cross etc"

retag:
	git tag -f $(LAST_TAG)
