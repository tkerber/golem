#include "jsonrpc_plugin_g_io_channel.h"

// TODO: in progress.

// creates a new handle for the json rpc client/server, with the given
// arguments (variable).
static jsonrpc_handle_t
open(va_list ap)
{
    // TODO
}

// close closes the handle for the json rpc client/server (but doesn't free the
// handle itself)
static void
close(jsonrpc_handle_t conn)
{
    // TODO
}

// cond_was_timedout allows the timeout condition to give the information
// that it timed out. (Allowing removal of the condition in all other cases.)
struct cond_was_timedout {
    GCond    *cond;
    gboolean *timed_out;
}

// recv_cond_timeout waits for a timeout to occur, and
static gboolean
recv_cond_timeout(gpointer cond)
{
    struct cond_was_timedout *c = (struct cond_was_timedout*)cond;
    *(c->timeout_out) = TRUE;
    g_cond_broadcast(c->cond);
}

static const char *
recv(jsonrpc_handle_t conn,
     unsigned int     timeout,
     void           **user_data_return);
{
    jsonrpc_g_io_channel_t *handle = (jsonrpc_g_io_channel_t*)conn;

    queue_data_t *recv = queue_pop(handle->r_queue);
    if(recv) {
        char *data = recv->data;
        g_free(recv);
        return data;
    }
    // Timeout and wait for data. This is a little tricky. The gist of it is
    // that we wait on a GCond, which gets broadcast either when new data is
    // available, or when the timeout expired. Each recv creates its own GCond,
    // and they are stored in a hash table set in the handle.
    GCond *cond = g_cond_new();
    gboolean timed_out = FALSE;
    struct cond_was_timedout *cwt = g_malloc(sizeof(struct cond_was_timedout));
    cwt->cond = cond;
    cwr->timed_out = &timed_out;
    g_hash_table_add(handle->recv_cond_set, cond);

    guint timeout = g_timeout_add(timeout, recv_cond_timeout, cond);

    g_cond_wait(cond);

    if(!timed_out) {
        g_source_remove(timeout);
    }
    g_hash_table_remove(cond);
    g_cond_free(cond);
    g_free(cond_was_timedout);

    queue_data_t *recv = queue_pop(handle->r_queue);
    if(recv) {
        char *data = recv->data;
        g_free(recv);
        return data;
    }
    return NULL;
}

static gboolean
jsonrpc_g_io_channel_write(GIOChannel  *src,
                           GIOCondition cond,
                           gpointer     conn)
{
    jsonrpc_g_io_channel_t *handle = (jsonrpc_g_io_channel_t*)conn;
    
    queue_data_t *data = queue_pop(handle->q_queue);

    if(data) {
        g_io_channel_write_chars(src, data->data, data->length, NULL, NULL);
        // TODO: possibly capture error scenarios. There's not much that can
        // be done in the case of an error though.
        g_free(data->data);
        g_free(data);
        return TRUE;
    } else {
        return FALSE;
    }
}

static jsonrpc_error_t
send(jsonrpc_handle_t conn,
     const char      *data)
{
    jsonrpc_g_io_channel_t *handle = (jsonrpc_g_io_channel_t*)conn;

    queue_push(handle->w_queue, data);

    g_io_channel_add_watch(
            conn->channel,
            G_IO_OUT,
            jsonrpc_g_io_channel_write,
            conn);

    return JSONRPC_ERROR_OK;
}

static jsonrpc_error_t
error(jsonrpc_handle_t net)
{
    // This method is poorly documented. I don't know what to put here.
    return JSONRPC_ERROR_OK;
}

const jsonrpc_net_plugin_t *jsonrpc_plugin_g_io_channel()
{
    static const jsonrpc_net_plugin_t plugin_g_io_channel = {
        open,
        close,
        recv,
        send,
        error
    };
    return &plugin_g_io_channel;
}
