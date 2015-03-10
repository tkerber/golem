#ifndef GOLEM_LIB_GOLEM_H
#define GOLEM_LIB_GOLEM_H

#include <gio/gio.h>
#include <webkit2/webkit-web-extension.h>
#include <glib.h>

typedef struct _HintsMode HintsMode;

typedef struct _RPCSession RPCSession;

// Exten contains all data the web extension requires.
typedef struct _Exten {
    HintsMode         *hints;
    WebKitWebPage     *web_page;
    guint64            page_id;
    WebKitDOMDocument *document;
    WebKitDOMElement  *active;
    WebKitDOMElement  *scroll_target;
    RPCSession        *rpc_session;
    glong              last_top;
    glong              last_height;
    gboolean           last_input_focus;
    gchar             *profile;
    // Used as a set for documents which have had handlers added.
    GHashTable        *registered_documents;
} Exten;

#endif /* GOLEM_LIB_GOLEM_H */
