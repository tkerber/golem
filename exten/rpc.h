#ifndef GOLEM_RPC_H
#define GOLEM_RPC_H

#ifdef __cplusplus
#include <jubatus/msgpack/rpc/server.h>
#include <jubatus/msgpack/rpc/client.h>
extern "C" {
#endif

#include "libgolem.h"
#include <glib.h>

typedef struct _RPCSession {
#ifdef __cplusplus
    msgpack::rpc::client *client;
    msgpack::rpc::server *server;
#else
    void *client;
    void *server;
#endif
} RPCSession;

// get_hints_labels gets the labels for n hints.
gchar **
get_hints_labels(guint n, Exten *exten, GError **err);

// hint_call calls a hint with the given string.
gboolean
hint_call(const gchar *str, Exten *exten, GError **err);

// vertical_position_changed notifies the main process of a vertical position
// change.
void
vertical_position_changed(guint64 page_id,
                          gint64  top,
                          gint64  height,
                          Exten  *exten);

// input_focus_changed notifies the main process of a change in the input
// focus.
void
input_focus_changed(guint64 page_id, gboolean input_focus, Exten *exten);

// domain_elem_hide_css retrieves the element hider CSS for a specified
// domain.
//
// The string is transferred to the called and must be freed.
gchar *
domain_elem_hide_css(const char *domain, Exten *exten, GError **err);

// blocks checks if a uri is blocked.
gboolean
blocks(
        const char *uri,
        const char *page_uri,
        guint64 flags,
        Exten *exten,
        GError **err);

// rpc_acquire acquires a RPC connection.
void
rpc_acquire(Exten *exten, GCallback cb, gpointer user_data);

#endif /* GOLEM_RPC_H */

#ifdef __cplusplus
}
#endif
