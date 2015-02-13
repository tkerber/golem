#include <glib.h>
#include <gio/gio.h>
#include <stdlib.h>
#include "rpc.h"
#include "hints.h"

// The DBus introspection xml for the WebExtension interface.
static const gchar introspection_xml[] =
    "<node>"
    "    <interface name='com.github.tkerber.golem.WebExtension'>"
    "        <property type='x' name='ScrollTop' access='readwrite' />"
    "        <property type='x' name='ScrollLeft' access='readwrite' />"
    "        <property type='x' name='ScrollHeight' access='read' />"
    "        <property type='x' name='ScrollWidth' access='read' />"
    "        <property type='x' name='ScrollTargetTop' access='readwrite' />"
    "        <property type='x' name='ScrollTargetLeft' access='readwrite' />"
    "        <property type='x' name='ScrollTargetHeight' access='read' />"
    "        <property type='x' name='ScrollTargetWidth' access='read' />"
    "        <method name='LinkHintsMode'>"
    "            <arg type='x' name='Empty' direction='out' />"
    "        </method>"
    "        <method name='FormVariableHintsMode'>"
    "            <arg type='x' name='Empty' direction='out' />"
    "        </method>"
    "        <method name='ClickHintsMode'>"
    "            <arg type='x' name='Empty' direction='out' />"
    "        </method>"
    "        <method name='EndHintsMode' />"
    "        <method name='FilterHintsMode'>"
    "            <arg type='s' name='Prefix' direction='in' />"
    "            <arg type='b' name='HitAndEnd' direction='out' />"
    "        </method>"
    "    </interface>"
    "</node>";

// Error stuff

#define GOLEM_WEB_ERROR golem_web_error_quark()

G_DEFINE_QUARK("golem-web-error-quark", golem_web_error);

#define GOLEM_WEB_ERROR_NULL_BODY 0

// handle_method_call handles a DBus method call on the WebExtension.
static void
handle_method_call(GDBusConnection       *connection,
                   const gchar           *sender,
                   const gchar           *object_path,
                   const gchar           *interface_name,
                   const gchar           *method_name,
                   GVariant              *parameters,
                   GDBusMethodInvocation *invocation,
                   gpointer               user_data);

// handle_get_property handles a DBus property get call.
static GVariant *
handle_get_property(GDBusConnection *connection,
                    const gchar     *sender,
                    const gchar     *object_path,
                    const gchar     *interface_name,
                    const gchar     *property_name,
                    GError         **error,
                    gpointer         user_data);

// handle_set_property handles a DBus property set call.
static gboolean
handle_set_property(GDBusConnection *connection,
                    const gchar     *sender,
                    const gchar     *object_path,
                    const gchar     *interface_name,
                    const gchar     *property_name,
                    GVariant        *value,
                    GError         **error,
                    gpointer         user_data);
// introspection_data contains the DBus introspection data for the
// WebExtension.
static GDBusNodeInfo *introspection_data = NULL;

// interface_vtable references the methods used for DBus calls to the
// WebExtension.
static const GDBusInterfaceVTable interface_vtable =
{
    handle_method_call,
    handle_get_property,
    handle_set_property
};

// handle_method_call handles a DBus method call on the WebExtension.
static void
handle_method_call(GDBusConnection       *connection,
                   const gchar           *sender,
                   const gchar           *object_path,
                   const gchar           *interface_name,
                   const gchar           *method_name,
                   GVariant              *parameters,
                   GDBusMethodInvocation *invocation,
                   gpointer               user_data)
{
    Exten *exten = user_data;
    if(g_strcmp0(method_name, "LinkHintsMode") == 0) {
        gint64 ret = start_hints_mode(select_links, hint_call_by_href, exten);
        g_dbus_method_invocation_return_value(invocation,
                g_variant_new("(x)", ret));
    } else if(g_strcmp0(method_name, "FormVariableHintsMode") == 0) {
        gint64 ret = start_hints_mode(
                select_form_text_variables,
                hint_call_by_form_variable_get,
                exten);
        g_dbus_method_invocation_return_value(invocation,
                g_variant_new("(x)", ret));
    } else if(g_strcmp0(method_name, "ClickHintsMode") == 0) {
        gint64 ret = start_hints_mode(
                select_clickable,
                hint_call_by_click,
                exten);
        g_dbus_method_invocation_return_value(invocation,
                g_variant_new("(x)", ret));
    } else if(g_strcmp0(method_name, "EndHintsMode") == 0) {
        end_hints_mode(exten);
        g_dbus_method_invocation_return_value(invocation, NULL);
    } else if(g_strcmp0(method_name, "FilterHintsMode") == 0) {
        const gchar *str;
        g_variant_get(parameters, "(&s)", &str);
        gboolean ret = filter_hints_mode(str, exten);
        g_dbus_method_invocation_return_value(invocation,
                g_variant_new("(b)", ret));
    }
}

// handle_get_property handles a DBus property get call.
static GVariant *
handle_get_property(GDBusConnection *connection,
                    const gchar     *sender,
                    const gchar     *object_path,
                    const gchar     *interface_name,
                    const gchar     *property_name,
                    GError         **error,
                    gpointer         user_data)
{
    Exten *exten = user_data;
    GVariant *ret = NULL;
    WebKitWebPage *wp = exten->web_page;
    WebKitDOMDocument *dom = webkit_web_page_get_dom_document(wp);
    if(dom == NULL) {
        g_set_error(
                error,
                GOLEM_WEB_ERROR,
                GOLEM_WEB_ERROR_NULL_BODY,
                "Document element is NULL.");
        return NULL;
    }
    WebKitDOMElement *e = NULL;
    if(g_strcmp0(property_name, "ScrollTop") == 0 ||
        g_strcmp0(property_name, "ScrollLeft") == 0 ||
        g_strcmp0(property_name, "ScrollHeight") == 0 ||
        g_strcmp0(property_name, "ScrollWidth") == 0) {

        e = WEBKIT_DOM_ELEMENT(webkit_dom_document_get_body(dom));
    } else if (g_strcmp0(property_name, "ScrollTargetTop") == 0 ||
        g_strcmp0(property_name, "ScrollTargetLeft") == 0 ||
        g_strcmp0(property_name, "ScrollTargetHeight") == 0||
        g_strcmp0(property_name, "ScrollTargetWidth") == 0) {

        e = exten->scroll_target;
    }
    if(e == NULL) {
        g_set_error(
                error,
                GOLEM_WEB_ERROR,
                GOLEM_WEB_ERROR_NULL_BODY,
                "Scroll element is NULL.");
        return NULL;
    }

    if(g_strcmp0(property_name, "ScrollTop") == 0 ||
            g_strcmp0(property_name, "ScrollTargetTop") == 0) {
        ret = g_variant_new_int64(
                webkit_dom_element_get_scroll_top(e));
    } else if(g_strcmp0(property_name, "ScrollLeft") == 0 ||
            g_strcmp0(property_name, "ScrollTargetLeft") == 0) {
        ret = g_variant_new_int64(
                webkit_dom_element_get_scroll_left(e));
    } else if(g_strcmp0(property_name, "ScrollHeight") == 0 ||
            g_strcmp0(property_name, "ScrollTargetHeight") == 0) {
        ret = g_variant_new_int64(
                webkit_dom_element_get_scroll_height(e));
    } else if(g_strcmp0(property_name, "ScrollWidth") == 0 ||
            g_strcmp0(property_name, "ScrollTargetWidth") == 0) {
        ret = g_variant_new_int64(
                webkit_dom_element_get_scroll_width(e));
    }
    return ret;
}

// handle_set_property handles a DBus property set call.
static gboolean
handle_set_property(GDBusConnection *connection,
                    const gchar     *sender,
                    const gchar     *object_path,
                    const gchar     *interface_name,
                    const gchar     *property_name,
                    GVariant        *value,
                    GError         **error,
                    gpointer         user_data)
{
    Exten *exten = user_data;
    WebKitDOMDocument *dom = webkit_web_page_get_dom_document(exten->web_page);
    if(dom == NULL) {
        g_set_error(
                error,
                GOLEM_WEB_ERROR,
                GOLEM_WEB_ERROR_NULL_BODY,
                "Document element is NULL.");
        return TRUE;
    }
    WebKitDOMElement *e = NULL;
    if(g_strcmp0(property_name, "ScrollTop") == 0 ||
        g_strcmp0(property_name, "ScrollLeft") == 0) {

        e = WEBKIT_DOM_ELEMENT(webkit_dom_document_get_body(dom));
    } else if (g_strcmp0(property_name, "ScrollTargetTop") == 0 ||
        g_strcmp0(property_name, "ScrollTargetLeft") == 0) {

        e = exten->scroll_target;
    }
    if(e == NULL) {
        g_set_error(
                error,
                GOLEM_WEB_ERROR,
                GOLEM_WEB_ERROR_NULL_BODY,
                "Scroll element is NULL.");
        return TRUE;
    }

    if(g_strcmp0(property_name, "ScrollTop") == 0 ||
            g_strcmp0(property_name, "ScrollTargetTop") == 0) {
        webkit_dom_element_set_scroll_top(e, g_variant_get_int64(value));
        return TRUE;
    } else if(g_strcmp0(property_name, "ScrollLeft") == 0 ||
            g_strcmp0(property_name, "ScrollTargetLeft") == 0) {
        webkit_dom_element_set_scroll_left(e, g_variant_get_int64(value));
        return TRUE;
    }
    // Currently no properties exist.
    return FALSE;
}

// get_hints_labels gets the labels for n hints.
gchar **
get_hints_labels(guint n, Exten *exten, GError **err)
{
    GVariant *retv = g_dbus_connection_call_sync(
            exten->rpc_session->connection,
            exten->golem_name,
            "/com/github/tkerber/Golem",
            "com.github.tkerber.Golem",
            "GetHintsLabels",
            g_variant_new(
                "(x)",
                (gint64)n),
            G_VARIANT_TYPE("(as)"),
            G_DBUS_CALL_FLAGS_NONE,
            -1,
            NULL,
            err);
    if(err != NULL && *err != NULL) {
        return NULL;
    }
    gchar **ret;
    g_variant_get(retv, "(^as)", &ret);
    g_variant_unref(retv);
    return ret;
}

// hint_call calls a hint with the given string.
gboolean
hint_call(const gchar *str, Exten *exten, GError **err)
{
    GVariant *retv = g_dbus_connection_call_sync(
            exten->rpc_session->connection,
            exten->golem_name,
            "/com/github/tkerber/Golem",
            "com.github.tkerber.Golem",
            "HintCall",
            g_variant_new(
                "(ts)",
                webkit_web_page_get_id(exten->web_page),
                str),
            G_VARIANT_TYPE("(b)"),
            G_DBUS_CALL_FLAGS_NONE,
            -1,
            NULL,
            err);
    gboolean ret = FALSE;
    if(err != NULL && *err != NULL) {
        return false;
    } else {
        g_variant_get(retv, "(b)", &ret);
        g_variant_unref(retv);
    }
    return ret;
}

// vertical_position_changed notifies the main process of a vertical position
// change.
void
vertical_position_changed(guint64 page_id,
                          gint64  top,
                          gint64  height,
                          Exten  *exten)
{
    g_dbus_connection_call(
        exten->rpc_session->connection,
        exten->golem_name,
        "/com/github/tkerber/Golem",
        "com.github.tkerber.Golem",
        "VerticalPositionChanged",
        g_variant_new(
            "(txx)",
            page_id,
            top,
            height),
        NULL,
        G_DBUS_CALL_FLAGS_NONE,
        -1,
        NULL,
        NULL,
        NULL);
}

// input_focus_changed notifies the main process of a change in the input
// focus.
void
input_focus_changed(guint64 page_id, gboolean input_focus, Exten *exten)
{
    g_dbus_connection_call(
        exten->rpc_session->connection,
        exten->golem_name,
        "/com/github/tkerber/Golem",
        "com.github.tkerber.Golem",
        "InputFocusChanged",
        g_variant_new(
                "(tb)",
                page_id,
                input_focus),
        NULL,
        G_DBUS_CALL_FLAGS_NONE,
        -1,
        NULL,
        NULL,
        NULL);
}

// domain_elem_hide_css retrieves the element hider CSS for a specified
// domain.
//
// The string is transferred to the called and must be freed.
gchar *
domain_elem_hide_css(const char *domain, Exten *exten, GError **err)
{
    GVariant *ret = g_dbus_connection_call_sync(
            exten->rpc_session->connection,
            exten->golem_name,
            "/com/github/tkerber/Golem",
            "com.github.tkerber.Golem",
            "DomainElemHideCSS",
            g_variant_new("(s)", domain),
            G_VARIANT_TYPE("(s)"),
            G_DBUS_CALL_FLAGS_NONE,
            -1,
            NULL,
            err);
    if(err != NULL && *err != NULL) {
        return NULL;
    }
    gchar *retStr = g_variant_dup_string(
            g_variant_get_child_value(ret, 0),
            NULL);
    g_variant_unref(ret);
    return retStr;
}

// blocks checks if a uri is blocked.
gboolean
blocks(
        const char *uri,
        const char *page_uri,
        guint64 flags,
        Exten *exten,
        GError **err)
{
    GVariant *ret = g_dbus_connection_call_sync(
            exten->rpc_session->connection,
            exten->golem_name,
            "/com/github/tkerber/Golem",
            "com.github.tkerber.Golem",
            "Blocks",
            g_variant_new(
                    "(sst)",
                    uri,
                    webkit_web_page_get_uri(exten->web_page),
                    flags),
            G_VARIANT_TYPE("(b)"),
            G_DBUS_CALL_FLAGS_NONE,
            -1,
            NULL,
            err);
    if(err != NULL && *err != NULL) {
        return false;
    }
    gboolean retBool = g_variant_get_boolean(
            g_variant_get_child_value(ret, 0));
    g_variant_unref(ret);
    return retBool;
}

// Bundles several user data together to be passed through a callback.
struct BusAcquireCBData {
    Exten    *exten;
    GCallback cb;
    gpointer  user_data;
};

// on_bus_acquired is called when a DBus bus is acquired, and proceeds with
// starting up the web extension.
static void
on_bus_acquired(GDBusConnection *connection,
                const gchar     *name,
                gpointer         user_data)
{
    struct BusAcquireCBData *cbdata = user_data;
    Exten *exten = cbdata->exten;
    exten->rpc_session = g_malloc(sizeof(RPCSession));
    exten->rpc_session->connection = connection;
    exten->object_path = g_strdup_printf(
            "/com/github/tkerber/golem/WebExtension/%s/page%d", 
            exten->profile,
            webkit_web_page_get_id(exten->web_page));
    // Register DBus methods
    gint registration_id = g_dbus_connection_register_object(
            connection,
            exten->object_path,
            introspection_data->interfaces[0],
            &interface_vtable,
            exten,
            NULL,
            NULL);
    g_assert(registration_id > 0);

    ((void(*)(gpointer))cbdata->cb)(cbdata->user_data);
    g_free(cbdata);
}

// on_name_lost is called when a DBus name is lost, and crashes the web
// extension.
static void
on_name_lost(GDBusConnection *connection,
             const gchar     *name,
             gpointer         user_data)
{
    g_printerr("Lost DBus connection to main proccess.\n");
    exit(1);
}

// rpc_acquire acquires a RPC connection.
void
rpc_acquire(Exten *exten, GCallback cb, gpointer user_data)
{
    introspection_data = g_dbus_node_info_new_for_xml(introspection_xml, NULL);
    g_assert(introspection_data != NULL);
    gchar *bus_name = g_strdup_printf(
            "com.github.tkerber.golem.WebExtension.%s.Page%d", 
            exten->profile,
            exten->page_id);
    struct BusAcquireCBData *cbdata = g_malloc(sizeof(struct BusAcquireCBData));
    cbdata->exten = exten;
    cbdata->cb = cb;
    cbdata->user_data = user_data;
    guint owner_id = g_bus_own_name(G_BUS_TYPE_SESSION,
            bus_name,
            G_BUS_NAME_OWNER_FLAGS_NONE,
            on_bus_acquired,
            NULL,
            on_name_lost,
            cbdata,
            NULL);
    g_free(bus_name);
}
