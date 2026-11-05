BINARY  = istary
PREFIX  ?= /usr/local

.PHONY: build install uninstall clean gui

build:
	go build -o $(BINARY) ./cmd/istary/

install: build
	install -d $(PREFIX)/bin
	install -m 755 $(BINARY) $(PREFIX)/bin/$(BINARY)

uninstall:
	rm -f $(PREFIX)/bin/$(BINARY)

clean:
	rm -f $(BINARY)

# Desktop GUI prototype (needs the Wails CLI; see gui/README.md)
gui:
	cd gui && wails build -platform darwin/arm64
