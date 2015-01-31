#include <webkitdom/webkitdom.h>
#include <glib.h>
#include "libgolem.h"

#ifndef HINTS_H
#define HINTS_H

// Returns a GList of WebKitDOMNodes, which have been ref'd.
typedef GList *(*NodeSelecter)(Exten*);

// Do something with a node. Return true to continue hints mode.
typedef gboolean (*NodeExecuter)(WebKitDOMNode*, Exten*);

typedef struct _Hint {
    gchar            *text;
    WebKitDOMElement *div;
} Hint;

struct _HintsMode {
    NodeExecuter executer;
    GHashTable  *hints;
};

gboolean hint_call_by_href(WebKitDOMNode*, Exten*);

GList *select_links(Exten*);

void
start_hints_mode(NodeSelecter ns, NodeExecuter ne, Exten *exten);

gboolean
filter_hints_mode(const gchar *hints, Exten *exten);

void
end_hints_mode(Exten *exten);

#endif /* HINTS_H */
