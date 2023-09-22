ifndef VERBOSE
.SILENT:
endif

TAG = $(strip `git describe --tag 2>/dev/null`)

REV = git.`git rev-parse HEAD | cut -c1-7`

ifeq ($(shell echo ${TAG}),)
	VER := $(REV)
else
	VER := $(TAG)
endif

package-all: win-package linux-package

.PHONY: win-package
win-package: win-binary-x86_64
	mkdir -p os2cb
	cp target/os2cb.exe os2cb/
	cp README.md os2cb/
	cp LICENSE.txt os2cb/
	zip os2cb-$(VER)_win64.zip os2cb/* >/dev/null
	rm -rf os2cb || true

.PHONY: linux-package
linux-package: linux-binary-x86_64
	mkdir -p os2cb
	cp target/os2cb os2cb/
	cp README.md os2cb/
	cp LICENSE.txt os2cb/
	tar -czvf os2cb-$(VER)_linux.tar.gz os2cb/ >/dev/null
	rm -rf os2cb || true

binary-all: win-binary-x86_64 linux-binary-x86_64

.PHONY: win-binary-x86_64
win-binary-x86_64:
	mkdir -p target
	GOOS=windows GOARCH=amd64 go build -o target/os2cb.exe -ldflags '-w -s' .

.PHONY: linux-binary-x86_64
linux-binary-x86_64:
	mkdir -p target
	GOOS=linux GOARCH=amd64 go build -o target/os2cb -ldflags '-w -s' .

.PHONY: clean
clean:
	rm -rf target || true
	rm -rf os2cb 2>/dev/null || true
	rm *_win64.zip 2>/dev/null || true
	rm *_linux.tar.gz 2>/dev/null || true