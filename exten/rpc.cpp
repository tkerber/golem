#include <jubatus/msgpack/rpc/client.h>
#include <jubatus/msgpack/rpc/server.h>
#include "socket.hpp"
extern "C" {
#include <unistd.h>
#include <gio/gio.h>
#include <gio/gunixsocketaddress.h>
#include <glib.h>
#include <webkit2/webkit-web-extension.h>
#include "rpc.h"
#include "hints.h"
#include "libgolem.h"
}

#define ERR_NULL_SCROLL_ELEM 0

#define GOLEM_ERROR golem_error_quark()

G_DEFINE_QUARK("golem-error-quark", golem_error);

#define GOLEM_ERROR_GENERIC 0

namespace rpc {
    using namespace msgpack;
    using namespace msgpack::rpc;
}  // namespace rpc

template <typename R> struct mc_cb_data {
    R                  *ret;
    std::function<R()>  func;
    GCond              *cond;
};

struct mc_cb_data_void {
    std::function<void()>  func;
    GCond                 *cond;
};

template <typename R>
gboolean glib_mc_cb(gpointer user_data)
{
    struct mc_cb_data<R> *data = (struct mc_cb_data<R> *)user_data;
    *(data->ret) = data->func();
    g_cond_broadcast(data->cond);
    return FALSE;
}

gboolean glib_mc_cb_void(gpointer user_data)
{
    struct mc_cb_data_void *data = (struct mc_cb_data_void*)user_data;
    data->func();
    g_cond_broadcast(data->cond);
    return FALSE;
}

template <typename R>
R main_context_call(std::function<R()> func)
{
    GMutex mutex;
    g_mutex_init(&mutex);
    g_mutex_lock(&mutex);
    GCond cond;
    g_cond_init(&cond);
    R ret;
    struct mc_cb_data<R> data = {.ret = &ret, .func = func, .cond = &cond};
    g_main_context_invoke(NULL, glib_mc_cb<R>, &data);
    g_cond_wait(&cond, &mutex);
    g_mutex_unlock(&mutex);
    return ret;
}

void main_context_call_void(std::function<void()> func)
{
    GMutex mutex;
    g_mutex_init(&mutex);
    g_mutex_lock(&mutex);
    GCond cond;
    g_cond_init(&cond);
    struct mc_cb_data_void data = {.func = func, .cond = &cond};
    g_main_context_invoke(NULL, glib_mc_cb_void, &data);
    g_cond_wait(&cond, &mutex);
    g_mutex_unlock(&mutex);
}

typedef rpc::request request;

class golem_dispatcher: public rpc::dispatcher {
private:
    Exten *exten;
public:
    golem_dispatcher(Exten *exten);
public:

    void dispatch(request req);
};

golem_dispatcher::golem_dispatcher(Exten *exten) {
    this->exten = exten;
}

static void
set_call(std::string method, gint64 param, int *err, Exten *exten)
{
    WebKitWebPage *wp = exten->web_page;
    WebKitDOMDocument *dom = webkit_web_page_get_dom_document(wp);
    WebKitDOMElement *e = NULL;
    if(method == "GolemWebExtension.SetScrollTop" ||
            method == "GolemWebExtension.SetScrollLeft") {
        e = WEBKIT_DOM_ELEMENT(webkit_dom_document_get_body(dom));
    } else if(method == "GolemWebExtension.SetScrollTargetTop" ||
            method == "GolemWebExtension.SetScrollTargetLeft") {
        e = exten->scroll_target;
    }
    if(e == NULL) {
        *err = ERR_NULL_SCROLL_ELEM;
        return;
    }
    if(method == "GolemWebExtension.SetScrollTop" ||
            method == "GolemWebExtension.SetScrollTargetTop") {
        webkit_dom_element_set_scroll_top(e, param);
    } else if(method == "GolemWebExtension.SetScrollLeft" ||
            method == "GolemWebExtension.SetScrollTargetLeft") {
        webkit_dom_element_set_scroll_left(e, param);
    }
}

static void
get_call(std::string method, gint64 *ret, int *err, Exten *exten)
{
    WebKitWebPage *wp = exten->web_page;
    WebKitDOMDocument *dom = webkit_web_page_get_dom_document(wp);
    WebKitDOMElement *e = NULL;
    if(method == "GolemWebExtension.GetScrollTop" ||
            method == "GolemWebExtension.GetScrollLeft" ||
            method == "GolemWebExtension.GetScrollHeight" ||
            method == "GolemWebExtension.GetScrollWidth") {
        e = WEBKIT_DOM_ELEMENT(webkit_dom_document_get_body(dom));
    } else if(method == "GolemWebExtension.GetScrollTargetTop" ||
            method == "GolemWebExtension.GetScrollTargetLeft" ||
            method == "GolemWebExtension.GetScrollTargetHeight" ||
            method == "GolemWebExtension.GetScrollTargetWidth") {
        e = exten->scroll_target;
    }
    if(e == NULL) {
        *err = ERR_NULL_SCROLL_ELEM;
        return;
    }
    if(method == "GolemWebExtension.GetScrollTop" ||
            method == "GolemWebExtension.GetScrollTargetTop") {
        *ret = webkit_dom_element_get_scroll_top(e);
    } else if(method == "GolemWebExtension.GetScrollLeft" ||
            method == "GolemWebExtension.GetScrollTargetLeft") {
        *ret = webkit_dom_element_get_scroll_left(e);
    } else if(method == "GolemWebExtension.GetScrollWidth" ||
            method == "GolemWebExtension.GetScrollTargetWidth") {
        *ret = webkit_dom_element_get_scroll_width(e);
    } else if(method == "GolemWebExtension.GetScrollHeight" ||
            method == "GolemWebExtension.GetScrollTargetHeight") {
        *ret = webkit_dom_element_get_scroll_height(e);
    }
}

// must run in main context.
void golem_dispatcher::dispatch(request req)
// FIXME: run extension stuff in glib main context.
try {
    std::string method;
    req.method().convert(&method);
    if(method == "GolemWebExtension.GetPageID") {
        req.result((unsigned long)exten->page_id);
    } else if(method == "GolemWebExtension.LinkHintsMode") {
        req.result((long)main_context_call<gint64>(std::bind(
                        start_hints_mode,
                        select_links,
                        hint_call_by_href,
                        exten)));
    } else if(method == "GolemWebExtension.FormVariableHintsMode") {
        req.result((long)main_context_call<gint64>(std::bind(
                        start_hints_mode,
                        select_form_text_variables,
                        hint_call_by_form_variable_get,
                        exten)));
    } else if(method == "GolemWebExtension.ClickHintsMode") {
        req.result((long)main_context_call<gint64>(std::bind(
                        start_hints_mode,
                        select_clickable,
                        hint_call_by_click,
                        exten)));
    } else if(method == "GolemWebExtension.EndHintsMode") {
        main_context_call_void(std::bind(end_hints_mode, exten));
        req.result(NULL);
    } else if(method == "GolemWebExtension.FilterHintsMode") {
        msgpack::type::tuple<std::string> params;
        req.params().convert(&params);
        req.result(main_context_call<gboolean>(std::bind(
                    filter_hints_mode,
                    params.get<0>().c_str(),
                    exten)) == TRUE);
    } else if(method == "GolemWebExtension.GetScrollTop" ||
            method == "GolemWebExtension.GetScrollLeft" ||
            method == "GolemWebExtension.GetScrollHeight" ||
            method == "GolemWebExtension.GetScrollWidth" ||
            method == "GolemWebExtension.GetScrollTargetTop" ||
            method == "GolemWebExtension.GetScrollTargetHeight" ||
            method == "GolemWebExtension.GetScrollTargetWidth") {
        gint64 ret;
        // Error codes defined:
        // 0: Scroll element is NULL
        int err = -1;
        main_context_call_void(std::bind(
                    get_call,
                    method,
                    &ret,
                    &err,
                    exten));
        if(err != -1) {
            switch(err) {
            case ERR_NULL_SCROLL_ELEM:
                req.error(std::string("Scroll element is NULL."));
            default:
                req.error(std::string("Unknown error code."));
            }
        } else {
            req.result((long)ret);
        }
    } else if(method == "GolemWebExtension.SetScrollTop" ||
            method == "GolemWebExtension.SetScrollLeft" ||
            method == "GolemWebExtension.SetScrollTargetTop" ||
            method == "GolemWebExtension.SetScrollTargetLeft") {
        msgpack::type::tuple<long> params;
        req.params().convert(&params);
        gint64 param = (gint64)params.get<0>();
        // Error codes defined:
        // 0: Scroll element is NULL
        int err = -1;
        main_context_call_void(std::bind(
                    set_call,
                    method,
                    param,
                    &err,
                    exten));
        if(err != -1) {
            switch(err) {
            case ERR_NULL_SCROLL_ELEM:
                req.error(std::string("Scroll element is NULL."));
            default:
                req.error(std::string("Unknown error code."));
            }
        } else {
            req.result(NULL);
        }
        req.result(NULL);
    } else {
        req.error(msgpack::rpc::NO_METHOD_ERROR);
    }

} catch (msgpack::type_error& e) {
    req.error(msgpack::rpc::ARGUMENT_ERROR);
    return;

} catch (std::exception& e) {
    req.error(std::string(e.what()));
    return;
}

// get_hints_labels gets the labels for n hints.
gchar **
get_hints_labels(guint n, Exten *exten, GError **err)
{
    try {
        std::vector<std::string> ret = exten->rpc_session->client->call(
                "Golem.GetHintsLabels",
                (long)n).get<std::vector<std::string> >();
        gchar **cret = (gchar**)g_malloc(sizeof(gchar*) * (ret.size() + 1));
        cret[ret.size()] = NULL;
        for(int i = 0; i < ret.size(); i++) {
            std::string& at = ret.at(i);
            cret[i] = (gchar*)g_malloc(sizeof(gchar) * (at.size() + 1));
            cret[i][at.size()] = '\0';
            at.copy(cret[i], at.size());
        }
        return cret;
    } catch(std::exception& e) {
        if(err != NULL) {
            *err = g_error_new_literal(GOLEM_ERROR,
                    GOLEM_ERROR_GENERIC,
                    e.what());
        }
        return NULL;
    }
}

// hint_call calls a hint with the given string.
gboolean
hint_call(const gchar *str, Exten *exten, GError **err)
{
    try {
        msgpack::type::tuple<unsigned long, std::string> args =
            msgpack::type::tuple<unsigned long, std::string>(
                    (unsigned long)exten->page_id,
                    std::string(str));
        bool ret = exten->rpc_session->client->call(
                "Golem.HintCall",
                args).get<bool>();
        return ret ? TRUE : FALSE;
    } catch(std::exception& e) {
        if(err != NULL) {
            *err = g_error_new_literal(GOLEM_ERROR,
                    GOLEM_ERROR_GENERIC,
                    e.what());
        }
        return FALSE;
    }
}

// vertical_position_changed notifies the main process of a vertical position
// change.
void
vertical_position_changed(guint64 page_id,
                          gint64  top,
                          gint64  height,
                          Exten  *exten)
{
    try {
        msgpack::type::tuple<unsigned long, long, long> args =
            msgpack::type::tuple<unsigned long, long, long>(
                    (unsigned long)exten->page_id,
                    (long)top,
                    (long)height);
        exten->rpc_session->client->call(
                "Golem.VerticalPositionChanged",
                args);
    } catch(std::exception& e) {
        // TODO maybe print something.
    }
}

// input_focus_changed notifies the main process of a change in the input
// focus.
void
input_focus_changed(guint64 page_id, gboolean input_focus, Exten *exten)
{
    try {
        msgpack::type::tuple<unsigned long, bool> args =
            msgpack::type::tuple<unsigned long, bool>(
                    (unsigned long)exten->page_id,
                    input_focus ? true : false);
        exten->rpc_session->client->call(
                "Golem.InputFocusChanged",
                args);
    } catch(std::exception& e) {
        // TODO maybe print something.
    }
}

// domain_elem_hide_css retrieves the element hider CSS for a specified
// domain.
//
// The string is transferred to the called and must be freed.
gchar *
domain_elem_hide_css(const char *domain, Exten *exten, GError **err)
{
    try {
        std::string ret = exten->rpc_session->client->call(
                "Golem.DomainElemHideCSS",
                std::string(domain)).get<std::string>();
        gchar *cret = (gchar*)g_malloc(sizeof(gchar) * (ret.size() + 1));
        cret[ret.size()] = '\0';
        ret.copy(cret, ret.size());
        return cret;
    } catch(std::exception& e) {
        if(err != NULL) {
            *err = g_error_new_literal(GOLEM_ERROR,
                    GOLEM_ERROR_GENERIC,
                    e.what());
        }
        return NULL;
    }
}

// blocks checks if a uri is blocked.
gboolean
blocks(const char *uri,
       const char *page_uri,
       guint64 flags,
       Exten *exten,
       GError **err)
{
    try {
        msgpack::type::tuple<std::string, std::string, unsigned long> args =
            msgpack::type::tuple<std::string, std::string, unsigned long>(
                    std::string(uri),
                    std::string(page_uri),
                    (unsigned long) flags);
        bool ret = exten->rpc_session->client->call(
                "Golem.Blocks",
                args).get<bool>();
        return ret ? TRUE : FALSE;
    } catch(std::exception& e) {
        if(err != NULL) {
            *err = g_error_new_literal(GOLEM_ERROR,
                    GOLEM_ERROR_GENERIC,
                    e.what());
        }
        return FALSE;
    }
}

static void
handshake(GSocket *sock, std::string str, GError **err)
{
    g_socket_send(
            sock,
            str.c_str(),
            // +1 to account for the \0
            str.size() + 1,
            NULL,
            err);
    if(err && *err) {
        return;
    }
    gchar data[3];
    g_socket_receive(
            sock,
            data,
            3,
            NULL,
            err);
    if(err && *err) {
        return;
    }
    if(memcmp(data, (char*)"ok\0", 3) != 0) {
        if(!err) {
            return;
        }
        *err = g_error_new_literal(GOLEM_ERROR,
                GOLEM_ERROR_GENERIC,
                "Socket handshake failed.");
        return;
    }
}

void
rpc_acquire(Exten *exten, GCallback cb, gpointer user_data)
{
    GError *err = NULL;
    exten->rpc_session = (RPCSession*)g_malloc(sizeof(RPCSession));
    gchar *golem_socket_path = g_strdup_printf(
            "%s/golem-%s",
            g_get_user_runtime_dir(),
            exten->profile);
    // i == 0: Client connection
    // i == 1: Server connection
    GSocket *socks[2];
    for(int i = 0; i < 2; i++) {
        socks[i] = g_socket_new(
                G_SOCKET_FAMILY_UNIX,
                G_SOCKET_TYPE_STREAM,
                G_SOCKET_PROTOCOL_DEFAULT,
                &err);
        g_object_ref(socks[i]);
        if(err) {
            // DO SHIT.
        }
        g_socket_connect(
                socks[i],
                g_unix_socket_address_new(golem_socket_path),
                NULL,
                &err);
        if(err) {
            // DO SHIT.
        }
    }
    handshake(socks[0], "msgpack-rpc-client", &err);
    if(err) {
        // DO SHIT.
    }
    msgpack::rpc::socket_builder sock_builder = msgpack::rpc::socket_builder(socks[0]);
    msgpack::rpc::address dummy_addr = msgpack::rpc::address();
    exten->rpc_session->client = new msgpack::rpc::client(sock_builder, dummy_addr);
    handshake(socks[1], "msgpack-rpc-server", &err);
    if(err) {
        // DO SHIT.
    }
    // TODO for some reason the server_socket gets destructed after some timeout
    // must annihalate it.
    exten->rpc_session->server = new msgpack::rpc::server();
    exten->rpc_session->server->serve(new golem_dispatcher(exten));
    exten->rpc_session->server->listen(msgpack::rpc::socket_listener(socks[1]));
    exten->rpc_session->server->start(2);
    ((void(*)(gpointer))cb)(user_data);
    g_free(golem_socket_path);
}
