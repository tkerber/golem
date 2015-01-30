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
    WebKitDOMElement *hl_span;
} Hint;

struct _HintsMode {
    NodeExecuter executer;
    GHashTable  *hints;
};

void
start_hints_mode(NodeSelecter ns, NodeExecuter ne, Exten *exten);

void
hints_mode_filter(const gchar *hints, Exten *exten);

void
end_hints_mode(Exten *exten);

#endif /* HINTS_H */
