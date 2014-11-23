CC = gcc
CFLAGS =
CFLAGS += `pkg-config --cflags webkit2gtk-web-extension-4.0 json-glib-1.0`
LFLAGS += `pkg-config --libs json-glib-1.0`

%.o: %.c
	$(CC) -c -fPIC -o $@ $< $(CFLAGS)

libgolem.so: web_extension.o
	$(CC) -shared -o $@ $^ $(LFLAGS)

.PHONY: clean

clean:
	rm *.o
