#include "jsonrpc_plugin_g_io_channel.h"
#include <glib.h>

typedef struct queue_item_t {
    struct queue_item_t *next;
    char                *data;
    size_t               length;
} queue_item_t;

typedef struct queue_t {
    queue_item_t *head;
    queue_item_t *tail;
    GMutex        mutex;
} queue_t;

typedef struct jsonrpc_g_io_channel_t {
    GIOChannel   *channel;
    queue_t      *w_queue;
    queue_t      *r_queue;
    queue_item_t *garbage;
    GMutex        mutex;
    GCond         recv_cond;
} jsonrpc_g_io_channel_t;

static queue_item_t *
queue_pop(queue_t *queue)
{
    g_mutex_lock(&queue->mutex);
    queue_item_t *ret = queue->head;
    queue->head = ret->next;
    if(ret == queue->tail) {
        queue->tail = NULL;
    }
    g_mutex_unlock(&queue->mutex);
    return ret;
}

static void
queue_push(queue_t *queue, const char *data, size_t len)
{
    g_mutex_lock(&queue->mutex);
    queue_item_t *item = g_malloc(sizeof(queue_item_t));
    char *data_cp = g_malloc(sizeof(char) * (len + 1));
    data_cp[len] = '\0';
    memcpy(data_cp, data, len);
    item->next = queue->tail;
    item->data = data_cp;
    item->length = len;
    if(!queue->head) {
        queue->head = item;
    }
    queue->tail = item;
    g_mutex_unlock(&queue->mutex);
}

// TODO: in progress.
static void
gc(jsonrpc_g_io_channel_t *handle)
{
    queue_item_t *garbage = handle->garbage;
    handle->garbage = NULL;
    while(garbage) {
        queue_item_t *next = garbage->next;
        g_free(garbage->data);
        g_free(garbage);
        garbage = next;
    }
}

static gboolean
read_chan(GIOChannel  *chan,
          GIOCondition cond,
          gpointer     conn)
{
    gchar *str;
    gsize len;
    jsonrpc_g_io_channel_t *handle = (jsonrpc_g_io_channel_t*)conn;
    g_mutex_lock(&handle->mutex);
    GIOStatus status = g_io_channel_read_to_end(chan, &str, &len, NULL);
    if(status == G_IO_STATUS_NORMAL) {
        queue_push(handle->r_queue, str, len);
        g_free(str);
    }
    g_mutex_unlock(&handle->mutex);
    return TRUE;
}

// creates a new handle for the json rpc client/server, with the given
// arguments (variable).
static jsonrpc_handle_t
open(va_list ap)
{
    jsonrpc_g_io_channel_t *ret = g_malloc(sizeof(jsonrpc_g_io_channel_t));
    ret->channel = va_arg(ap, GIOChannel*);
    g_io_channel_ref(ret->channel);
    g_io_add_watch(
            ret->channel,
            G_IO_IN,
            read_chan,
            ret);
    ret->w_queue = malloc(sizeof(queue_t));
    ret->w_queue->head = NULL;
    ret->w_queue->tail = NULL;
    ret->r_queue = malloc(sizeof(queue_t));
    ret->r_queue->head = NULL;
    ret->r_queue->tail = NULL;
    ret->garbage = NULL;
    return ret;
}

// close closes the handle for the json rpc client/server (but doesn't free the
// handle itself)
static void
close(jsonrpc_handle_t conn)
{
    jsonrpc_g_io_channel_t *handle = (jsonrpc_g_io_channel_t*)conn;
    g_mutex_lock(&handle->mutex);
    gc(handle);
    queue_t *queues[2] = {handle->w_queue, handle->r_queue};
    handle->w_queue = NULL;
    handle->r_queue = NULL;
    int i;
    for(i = 0; i < 2; i++) {
        queue_item_t *head = queues[i]->head;
        g_free(queues[i]);
        while(head) {
            queue_item_t *next = head->next;
            g_free(head);
            head = next;
        }
    }
    g_io_channel_unref(handle->channel);
    handle->channel = NULL;
    g_mutex_unlock(&handle->mutex);
}

static const char *
pop_data(jsonrpc_g_io_channel_t *handle, queue_t *queue)
{
    queue_item_t *recv = queue_pop(queue);
    if(recv) {
        char *data = recv->data;
        recv->next = handle->garbage;
        handle->garbage = recv;
        return data;
    }
    return NULL;
}

static const char *
recv(jsonrpc_handle_t conn,
     unsigned int     timeout,
     void           **user_data_return)
{
    jsonrpc_g_io_channel_t *handle = (jsonrpc_g_io_channel_t*)conn;
    g_mutex_lock(&handle->mutex);

    gc(handle);
    const char *data;
    gint64 end_time = g_get_monotonic_time () + timeout * G_TIME_SPAN_MILLISECOND;
    while(!(data = pop_data(handle, handle->r_queue))) {
        if(!g_cond_wait_until(&handle->recv_cond, &handle->mutex, end_time)) {
            g_mutex_unlock(&handle->mutex);
            return NULL;
        }
    }
    g_mutex_unlock(&handle->mutex);
    return data;
}

static gboolean
write_chan(GIOChannel  *src,
           GIOCondition cond,
           gpointer     conn)
{
    jsonrpc_g_io_channel_t *handle = (jsonrpc_g_io_channel_t*)conn;
    g_mutex_lock(&handle->mutex);
    
    queue_item_t *data = queue_pop(handle->w_queue);

    if(data) {
        g_io_channel_write_chars(src, data->data, data->length, NULL, NULL);
        // TODO: possibly capture error scenarios. There's not much that can
        // be done in the case of an error though.
        g_free(data->data);
        g_free(data);
        g_mutex_unlock(&handle->mutex);
        return TRUE;
    } else {
        g_mutex_unlock(&handle->mutex);
        return FALSE;
    }
}

static jsonrpc_error_t
send(jsonrpc_handle_t conn,
     const char      *data,
     void *desc)
{
    jsonrpc_g_io_channel_t *handle = (jsonrpc_g_io_channel_t*)conn;
    g_mutex_lock(&handle->mutex);

    queue_push(handle->w_queue, data, strlen(data));

    g_io_add_watch(
            handle->channel,
            G_IO_OUT,
            write_chan,
            conn);

    g_mutex_unlock(&handle->mutex);
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
