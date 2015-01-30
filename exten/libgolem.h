#include <gio/gio.h>
#include <webkit2/webkit-web-extension.h>
#include <glib.h>

#ifndef LIB_GOLEM_H
#define LIB_GOLEM_H

typedef struct _HintsMode HintsMode;

// Exten contains all data the web extension requires.
typedef struct _Exten {
    HintsMode         *hints;
    WebKitWebPage     *web_page;
    WebKitDOMDocument *document;
    WebKitDOMElement  *active;
    WebKitDOMElement  *scroll_target;
    GDBusConnection   *connection;
    glong              last_top;
    glong              last_height;
    gboolean           last_input_focus;
    gchar             *object_path;
    gchar             *profile;
    gchar             *golem_name;
    // Used as a set for documents which have had handlers added.
    GHashTable        *registered_documents;
} Exten;

#endif /* LIB_GOLEM_H */
