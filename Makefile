CC = gcc
CFLAGS =
CFLAGS += `pkg-config --cflags webkit2gtk-web-extension-4.0 glib-2.0 gio-2.0`
LFLAGS += `pkg-config --libs webkit2gtk-web-extension-4.0 glib-2.0 gio-2.0`
# Path to google's closure compiler for javascript.
CLOSURE_COMPILER=/usr/share/java/closure-compiler/closure-compiler.jar

all: data/libgolem.so data/srv/pdf.js/

.PHONY: all clean

%.o: %.c
	$(CC) -c -fPIC -o $@ $< $(CFLAGS)

data/libgolem.so: exten/libgolem.o exten/hints.o
	$(CC) -shared -o $@ $^ $(LFLAGS)

data/srv/pdf.js/enabled:
	cd pdf.js && CLOSURE_COMPILER=$(CLOSURE_COMPILER) node make minified
	mkdir -p data/srv
	mv pdf.js/build/minified/web -T data/srv/pdf.js/web
	mv pdf.js/build/minified/build -T data/srv/pdf.js/build
	touch $@

all: data/srv/pdf.js/enabled data/libgolem.so

clean:
	rm exten/*.o
	rm -rf pdf.js
