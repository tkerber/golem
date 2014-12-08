CC = gcc
CFLAGS =
CFLAGS += `pkg-config --cflags webkit2gtk-web-extension-4.0 glib-2.0 gio-2.0`
LFLAGS += `pkg-config --libs webkit2gtk-web-extension-4.0 glib-2.0 gio-2.0`

%.o: %.c
	$(CC) -c -fPIC -o $@ $< $(CFLAGS)

libgolem.so: libgolem.o
	$(CC) -shared -o $@ $^ $(LFLAGS)

.PHONY: clean

clean:
	rm *.o
