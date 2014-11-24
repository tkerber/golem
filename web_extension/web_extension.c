#include <webkit2/webkit-web-extension.h>
#include <glib.h>
#include <json-glib/json-glib.h>
#include <stdlib.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <errno.h>
#include <fcntl.h>

static void scroll_by_delta(gpointer, gboolean, gint64);
static void scroll_to_top(gpointer);
static void scroll_to_bottom(gpointer);

static gboolean
read_fifo(GIOChannel *src, GIOCondition cond, gpointer web_page_p)
{
    // Read one JSON object.
    gchar *str;
    GError *err = NULL;
    GIOStatus status = g_io_channel_read_line(src, &str, NULL, NULL, &err);
    if(err != NULL) {
        g_printerr("Read from IPC fifo failed!\n");
        exit(1);
    }

    // Parse it.
    JsonParser *parser = json_parser_new();
    json_parser_load_from_data(parser, str, -1, &err);
    // If an error occurs here, we consider it more of a warning; as the IPC
    // infrastructure is still intact.
    if(err != NULL) {
        g_printerr("Failed to parse JSON object for IPC:\n\n%s\n\n", str);
        goto cleanup;
    }
    // Handle it.
    JsonNode *root = json_parser_get_root(parser);
    // The instruction must be at root/instruction
    if(json_node_get_node_type(root) != JSON_NODE_OBJECT) {
        g_printerr("Unexpected JSON format for IPC:\n\n%s\n\n", str);
        goto cleanup;
    }
    JsonObject *rootObj = json_node_get_object(root);
    if(!json_object_has_member(rootObj, "instruction")) {
        g_printerr("Unexpected JSON format for IPC:\n\n%s\n\n", str);
        goto cleanup;
    }
    JsonNode *instrNode = json_object_get_member(rootObj, "instruction");
    if(json_node_get_node_type(instrNode) != JSON_NODE_VALUE ||
            json_node_get_value_type(instrNode) != G_TYPE_STRING) {
        g_printerr("Unexpected JSON format for IPC:\n\n%s\n\n", str);
        goto cleanup;
    }
    const gchar *instr = json_node_get_string(instrNode);

    // Scroll instruction
    if(!strcmp(instr, "scroll")) {
        // root/direction contains the direction string, either
        // vertical or horizontal.
        //
        // root/delta contains an integer delta; the amount to scroll.

        // get vertical part.
        gboolean vertical;
        if(!json_object_has_member(rootObj, "direction")) {
            g_printerr("Unexpected JSON format for IPC:\n\n%s\n\n", str);
            goto cleanup;
        }
        JsonNode *directionNode = json_object_get_member(rootObj, "direction");
        if(json_node_get_node_type(directionNode) != JSON_NODE_VALUE ||
                json_node_get_value_type(directionNode) != G_TYPE_STRING) {
            g_printerr("Unexpected JSON format for IPC:\n\n%s\n\n", str);
            goto cleanup;
        }
        const gchar *directionStr = json_node_get_string(directionNode);
        if(!strcmp(directionStr, "vertical")) {
            vertical = true;
        } else if (!strcmp(directionStr, "horizontal")) {
            vertical = false;
        } else {
            g_printerr("Unexpected JSON format for IPC:\n\n%s\n\n", str);
            goto cleanup;
        }
        
        // get delta part.
        gint64 delta;
        if(!json_object_has_member(rootObj, "delta")) {
            goto cleanup;
        }
        JsonNode *deltaNode = json_object_get_member(rootObj, "delta");
        if(json_node_get_node_type(deltaNode) != JSON_NODE_VALUE ||
                json_node_get_value_type(deltaNode) != G_TYPE_INT64) {
            g_printerr("Unexpected JSON format for IPC:\n\n%s\n\n", str);
            goto cleanup;
        }
        delta = json_node_get_int(deltaNode);
        
        scroll_by_delta(web_page_p, vertical, delta);
    } else if(!strcmp(instr, "scroll_top")) {
        scroll_to_top(web_page_p);
    } else if(!strcmp(instr, "scroll_bottom")) {
        scroll_to_bottom(web_page_p);
    }
    
    // clean up.
cleanup:
    g_object_unref(parser);
    free(str);
    
    return true;
}

static void
scroll_by_delta(gpointer web_page_p, gboolean vertical, gint64 delta)
{
    WebKitWebPage *web_page = web_page_p;
    WebKitDOMDocument *dom = webkit_web_page_get_dom_document(web_page);
    WebKitDOMElement *e = webkit_dom_document_get_active_element(dom);
    if(vertical) {
        webkit_dom_element_set_scroll_top(e, webkit_dom_element_get_scroll_top(e) + delta);
    } else {
        webkit_dom_element_set_scroll_left(e, webkit_dom_element_get_scroll_left(e) + delta);
    }
}

static void
scroll_to_top(gpointer web_page_p)
{
    WebKitWebPage *web_page = web_page_p;
    WebKitDOMDocument *dom = webkit_web_page_get_dom_document(web_page);
    WebKitDOMElement *e = webkit_dom_document_get_active_element(dom);
    webkit_dom_element_set_scroll_top(e, 0);
}

static void
scroll_to_bottom(gpointer web_page_p)
{
    WebKitWebPage *web_page = web_page_p;
    WebKitDOMDocument *dom = webkit_web_page_get_dom_document(web_page);
    WebKitDOMElement *e = webkit_dom_document_get_active_element(dom);
    webkit_dom_element_set_scroll_top(e, webkit_dom_element_get_scroll_height(e));
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
