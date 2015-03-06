#include <jubatus/msgpack/rpc/server.h>
extern "C" {
#include <glib.h>
#include <webkit2/webkit-web-extension.h>
#include "rpc.h"
#include "hints.h"
#include "libgolem.h"
}

#define GOLEM_ERROR golem_error_quark()

G_DEFINE_QUARK("golem-error-quark", golem_error);

#define GOLEM_ERROR_GENERIC 0

namespace rpc {
    using namespace msgpack;
    using namespace msgpack::rpc;
}  // namespace rpc

class golem_dispatcher : public rpc::dispatcher {
private:
    Exten *exten;
public:
    golem_dispatcher(Exten *exten) {
        this.exten = exten;
    }
public:
    typedef rpc::request request;

    void dispatch(request req)
    try {
        std::string method;
        req.method().convert(&method);
        if(method == "LinkHintsMode") {
            req.result((long)start_hints_mode(
                        select_links,
                        hints_call_by_href,
                        exten));
        } else if(method == "FormVariableHintsMode") {
            req.result((long)start_hints_mode(
                        select_form_text_variables,
                        hint_call_by_form_variable_get,
                        exten));
        } else if(method == "ClickHintsMode") {
            req.result((long)start_hints_mode(
                        select_clickable,
                        hint_call_by_click,
                        exten));
        } else if(method == "EndHintsMode") {
            end_hints_mode(exten);
            req.result(NULL);
        } else if(method == "FilterHintsMode") {
            msgpack::type::tuple<std::string> params;
            req.params().convert(&params);
            filter_hints_mode(params.get<0>().c_str(), exten);
        } else if(method == "GetScrollTop" ||
                method == "GetScrollLeft" ||
                method == "GetScrollHeight" ||
                method == "GetScrollWidth" ||
                method == "GetScrollTargetTop" ||
                method == "GetScrollTargetHeight" ||
                method == "GetScrollTargetWidth") {
            WebKitWebPage *wp = exten->web_page;
            WebKitDOMDocument *dom = webkit_web_page_get_dom_document(wp);
            WebKitDOMElement *e = NULL;
            if(method == "GetScrollTop" ||
                    method == "GetScrollLeft" ||
                    method == "GetScrollHeight" ||
                    method == "GetScrollWidth") {
                e = WEBKIT_DOM_ELEMENT(webkit_dom_document_get_body(dom));
            } else if(method == "GetScrollTargetTop" ||
                    method == "GetScrollTargetLeft" ||
                    method == "GetScrollTargetHeight" ||
                    method == "GetScrollTargetWidth") {
                e = exten->scroll_target;
            }
            if(e == NULL) {
                req.error(std::string("Scroll element is NULL."));
                return;
            }
            gint64 ret;
            if(method == "GetScrollTop" ||
                    method == "GetScrollTargetTop") {
                ret = webkit_dom_element_get_scroll_top(e);
            } else if(method == "GetScrollLeft" ||
                    method == "GetScrollTargetLeft") {
                ret = webkit_dom_element_get_scroll_left(e);
            } else if(method == "GetScrollWidth" ||
                    method == "GetScrollTargetWidth") {
                ret = webkit_dom_element_get_scroll_width(e);
            } else if(method == "GetScrollHeight" ||
                    method == "GetScrollTargetHeight") {
                ret = webkit_dom_element_get_scroll_height(e);
            }
            req.result((long)ret);
        } else if(method == "SetScrollTop" ||
                method == "SetScrollLeft" ||
                method == "SetScrollTargetTop" ||
                method == "SetScrollTargetLeft") {
            msgpack::type::tuple<long> params;
            req.params().convert(&params);
            gint64 param = (gint64)params.get<0>();

            WebKitWebPage *wp = exten->web_page;
            WebKitDOMDocument *dom = webkit_web_page_get_dom_document(wp);
            WebKitDOMElement *e = NULL;
            if(method == "SetScrollTop" ||
                    method == "SetScrollLeft") {
                e = WEBKIT_DOM_ELEMENT(webkit_dom_document_get_body(dom));
            } else if(method == "SetScrollTargetTop" ||
                    method == "SetScrollTargetLeft") {
                e = exten->scroll_target;
            }
            if(e == NULL) {
                req.error(std::string("Scroll element is NULL."));
                return;
            }
            if(method == "SetScrollTop" ||
                    method == "SetScrollTargetTop") {
                webkit_dom_element_set_scroll_top(e, param);
            } else if(method == "SetScrollLeft" ||
                    method == "SetScrollTargetLeft") {
                webkit_dom_element_set_scroll_left(e, param);
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

};

// get_hints_labels gets the labels for n hints.
gchar **
get_hints_labels(guint n, Exten *exten, GError **err)
{
    try {
        std::vector<std::string> ret = exten->rpc_session->client->call(
                "Golem.GetHintsLabels",
                (long)n);
        gchar **cret = g_malloc(sizeof(gchar*) * (ret.size() + 1));
        cret[ret.size()] = NULL;
        for(int i = 0; i < ret.size(); i++) {
            std::string& at = ret.at(i);
            cret[i] = g_malloc(sizeof(gchar) * (at.size() + 1));
            cret[i][at.size()] = '\0';
            at.copy(cret, at.size());
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
        bool ret = exten->rpc_session->client->call(
                "Golem.HintCall",
                (unsigned long)exten->page_id,
                std::string(str));
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
        exten->rpc_session->client->call(
                "Golem.VerticalPositionChanged",
                (unsigned long)exten->page_id,
                (long)top,
                (long)height);
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
        exten->rpc_session->client->call(
                "Golem.InputFocusChanged",
                (unsigned long)exten->page_id,
                input_focus ? true : false);
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
                std::string(domain));
        gchar *cret = g_malloc(sizeof(gchar) * (ret.size() + 1));
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
        bool ret = exten->rpc_session->client->call(
                "Golem.Blocks",
                std::string(uri),
                std::string(page_uri),
                (unsigned long) flags);
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
handshake(GSocket *sock, gchar *str, GError **err)
{
    g_socket_send(
            sock,
            str,
            // +1 to account for the \0
            strlen(str) + 1,
            NULL,
            err);
    if(err && *err) {
        return;
    }
    gchar data[3];
    gchar *data = g_malloc(sizeof(gchar) * 3);
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
                G_SOCKET_STREAM,
                G_SOCKET_PROTOCOL_DEFAULT,
                &err);
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
    // TODO INIT CLIENT
    handshake(socks[1], "msgpack-rpc-server", &err);
    if(err) {
        // DO SHIT.
    }
    // TODO INIT SERVER
    g_free(golem_socket_path);
}
