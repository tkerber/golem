#include <webkit2/webkit-web-extension.h>
#include <glib.h>
#include <stdlib.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <errno.h>
#include <fcntl.h>

static gboolean
read_fifo(GIOChannel *src, GIOCondition cond, gpointer web_page_p)
{
    gchar *str;
    GError *err = NULL;
    GIOStatus status = g_io_channel_read_line(src, &str, NULL, NULL, &err);
    // TODO error
    WebKitWebPage *web_page = web_page_p;
    WebKitDOMDocument *dom = webkit_web_page_get_dom_document(web_page);
    WebKitDOMElement *e = webkit_dom_document_get_active_element(dom);
    if(str[0] == 1)
        webkit_dom_element_set_scroll_top(e, webkit_dom_element_get_scroll_top(e) + 40);
    else
        webkit_dom_element_set_scroll_top(e, webkit_dom_element_get_scroll_top(e) - 40);
    //g_print(str);
    free(str);
    // read null-seperated JSON
    // parse said JSON
    // do what it says
    // TODO
    
    return true;
}

static void
web_page_created_callback(WebKitWebExtension *extension,
                          WebKitWebPage      *web_page,
                          gpointer            user_data)
{
    gchar *dir = getenv("GOLEM_TMP");
    // Create fifo, at $GOLEM_TMP/webkitfifo
    gchar *fifo_path = g_build_path(G_DIR_SEPARATOR_S, dir, "webkitfifo", NULL);
    struct stat sb;
    int err = stat(fifo_path, &sb);
    if(!err && !S_ISFIFO(sb.st_mode)) {
        g_printerr("Failed to create fifo: %s - Non-fifo file already exists.\n", fifo_path);
        exit(1);
    } else if(err && errno == ENOENT) {
        // File permissions: rw-------
        // As this is an IPC pipe, other users should have nothing to say here.
        err = mkfifo(fifo_path, S_IRUSR | S_IWUSR);
        if(err) {
            g_printerr("Failed to create fifo: %s\n", fifo_path);
            exit(1);
        }
    } else {
        g_printerr("Failed to create fifo: %s - Failed to stat.\n", fifo_path);
        exit(1);
    }
    // Create GIOChannel and watch it for available reads.
    GError *g_err = NULL;
    // We need to open the fifo in nonblocking mode, otherwise the browser
    // freezes until a command is sent.
    int fd = open(fifo_path, O_RDONLY | O_NONBLOCK);
    free(fifo_path);
    if(fd == -1) {
        g_printerr("Failed to read fifo: %s\n", fifo_path);
        exit(1);
    }
    GIOChannel *c = g_io_channel_unix_new(fd);
    // As we communicate by json, we can null seperate each message.
    // (Just in case several come in at the same time)
    g_io_channel_set_line_term(c, "\0", -1);
    g_io_add_watch(c, G_IO_IN, read_fifo, web_page);
}

G_MODULE_EXPORT void
webkit_web_extension_initialize(WebKitWebExtension *extension)
{
    g_signal_connect(extension, "page-created",
        G_CALLBACK(web_page_created_callback), NULL);
}
