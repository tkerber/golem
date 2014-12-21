#include <gio/gio.h>
#include <webkit2/webkit-web-extension.h>
#include <glib.h>
#include <stdlib.h>
#include <stdio.h>

#define GOLEM_WEB_ERROR golem_web_error_quark()

GQuark
golem_web_error_quark()
{
    return g_quark_from_static_string("golem-web-error-quark");
}

#define GOLEM_WEB_ERROR_NULL_BODY 0

static const gchar introspection_xml[] =
    "<node>"
    "    <interface name='com.github.tkerber.golem.WebExtension'>"
    "        <property type='x' name='ScrollTop' access='readwrite' />"
    "        <property type='x' name='ScrollLeft' access='readwrite' />"
    "        <property type='x' name='ScrollHeight' access='read' />"
    "        <property type='x' name='ScrollWidth' access='read' />"
    "        <signal name='VerticalPositionChanged'>"
    "            <arg type='x' name='ScrollTop' />"
    "            <arg type='x' name='ScrollHeight' />"
    "        </signal>"
    "    </interface>"
    "</node>";

struct Exten {
    WebKitWebPage *web_page;
    GDBusConnection *connection;
    glong last_top;
    glong last_height;
    gchar *object_path;
};

static void
handle_method_call(GDBusConnection       *connection,
                   const gchar           *sender,
                   const gchar           *object_path,
                   const gchar           *interface_name,
                   const gchar           *method_name,
                   GVariant              *parameters,
                   GDBusMethodInvocation *invocation,
                   gpointer               user_data);

static GVariant *
handle_get_property(GDBusConnection *connection,
                    const gchar     *sender,
                    const gchar     *object_path,
                    const gchar     *interface_name,
                    const gchar     *property_name,
                    GError         **error,
                    gpointer         user_data);

static gboolean
handle_set_property(GDBusConnection *connection,
                    const gchar     *sender,
                    const gchar     *object_path,
                    const gchar     *interface_name,
                    const gchar     *property_name,
                    GVariant        *value,
                    GError         **error,
                    gpointer         user_data);

static void
scroll_delta(gpointer web_page_p, gint64 delta, gboolean vertical);

static void
scroll_to_top(gpointer web_page_p);

static void
scroll_to_bottom(gpointer web_page_p);

static GDBusNodeInfo *introspection_data = NULL;
static const GDBusInterfaceVTable interface_vtable =
{
    handle_method_call,
    handle_get_property,
    handle_set_property
};

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
    // No methods currently.
}

static GVariant *
handle_get_property(GDBusConnection *connection,
                    const gchar     *sender,
                    const gchar     *object_path,
                    const gchar     *interface_name,
                    const gchar     *property_name,
                    GError         **error,
                    gpointer         user_data)
{
    struct Exten *exten = user_data;
    GVariant *ret = NULL;
    WebKitWebPage *wp = exten->web_page;
    WebKitDOMDocument *dom = webkit_web_page_get_dom_document(wp);
    WebKitDOMElement *e = NULL;
    if(dom != NULL) {
        e = WEBKIT_DOM_ELEMENT(webkit_dom_document_get_body(dom));
    }
    if(e == NULL) {
        g_set_error(
                error,
                GOLEM_WEB_ERROR,
                GOLEM_WEB_ERROR_NULL_BODY,
                "Body element is NULL.");
        return NULL;
    }

    if(g_strcmp0(property_name, "ScrollTop") == 0) {
        ret = g_variant_new_int64(
                webkit_dom_element_get_scroll_top(e));
    } else if(g_strcmp0(property_name, "ScrollLeft") == 0) {
        ret = g_variant_new_int64(
                webkit_dom_element_get_scroll_left(e));
    } else if(g_strcmp0(property_name, "ScrollHeight") == 0) {
        ret = g_variant_new_int64(
                webkit_dom_element_get_scroll_height(e));
    } else if(g_strcmp0(property_name, "ScrollWidth") == 0) {
        ret = g_variant_new_int64(
                webkit_dom_element_get_scroll_width(e));
    }
    return ret;
}

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
    struct Exten *exten = user_data;
    WebKitDOMDocument *dom = webkit_web_page_get_dom_document(exten->web_page);
    WebKitDOMElement *e = NULL;
    if(dom != NULL) {
        e = WEBKIT_DOM_ELEMENT(webkit_dom_document_get_body(dom));
    }
    if(e == NULL) {
        g_set_error(
                error,
                GOLEM_WEB_ERROR,
                GOLEM_WEB_ERROR_NULL_BODY,
                "Body element is NULL.");
        return FALSE;
    }

    if(g_strcmp0(property_name, "ScrollTop") == 0) {
        webkit_dom_element_set_scroll_top(e, g_variant_get_int64(value));
        return TRUE;
    } else if(g_strcmp0(property_name, "ScrollLeft") == 0) {
        webkit_dom_element_set_scroll_left(e, g_variant_get_int64(value));
        return TRUE;
    }
    // Currently no properties exist.
    return FALSE;
}

static gboolean
poll_scroll_position(gpointer user_data)
{
    struct Exten *exten = user_data;

    WebKitDOMDocument *dom = webkit_web_page_get_dom_document(exten->web_page);
    WebKitDOMElement *e = NULL;
    if(dom != NULL) {
        e = WEBKIT_DOM_ELEMENT(webkit_dom_document_get_body(dom));
    }
    if(e == NULL) {
        return G_SOURCE_CONTINUE;
    }
    glong top = webkit_dom_element_get_scroll_top(e);
    glong height = webkit_dom_element_get_scroll_height(e);
    if(top != exten->last_top || height != exten->last_height) {
        exten->last_top = top;
        exten->last_height = height;
        g_dbus_connection_emit_signal(
                exten->connection,
                NULL,
                exten->object_path,
                "com.github.tkerber.golem.WebExtension",
                "VerticalPositionChanged",
                g_variant_new("(xx)", top, height),
                NULL);
    }

    return G_SOURCE_CONTINUE;
}

static void
on_bus_acquired(GDBusConnection *connection,
                const gchar     *name,
                gpointer         user_data)
{
    struct Exten *exten = malloc(sizeof(struct Exten));
    exten->connection = connection;
    exten->web_page = user_data;
    exten->last_top = 0;
    exten->last_height = 0;
    exten->object_path = g_strdup_printf(
            "/com/github/tkerber/golem/WebExtension/page%d", 
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
    // Register 100ms loop polling scroll positions and sending updates
    // as required.
    g_timeout_add(100, poll_scroll_position, exten);
}

static void
on_name_lost(GDBusConnection *connection,
             const gchar     *name,
             gpointer         user_data)
{
    g_printerr("Lost DBus connection to main proccess.\n");
    exit(1);
}

// NOTE: There appears to be no way to attach to a web page being destroyed.
// I'm not sure if this means they *aren't* destroyed, or just that it wasn't
// planned for. Either way, it spews errors on the regular update if used
// with a destroyed page.
//
// As there is only one page per process, this isn't a problem, however it is
// worthy of note should this ever change for any reason.
static void
web_page_created_callback(WebKitWebExtension *extension,
                          WebKitWebPage      *web_page,
                          gpointer            user_data)
{
    guint owner_id;

    introspection_data = g_dbus_node_info_new_for_xml(introspection_xml, NULL);
    g_assert(introspection_data != NULL);
    gchar *bus_name = g_strdup_printf(
            "com.github.tkerber.golem.WebExtension.Page%d", 
            webkit_web_page_get_id(web_page));
    owner_id = g_bus_own_name(G_BUS_TYPE_SESSION,
            bus_name,
            G_BUS_NAME_OWNER_FLAGS_NONE,
            on_bus_acquired,
            NULL,
            on_name_lost,
            web_page,
            NULL);
    free(bus_name);
}

G_MODULE_EXPORT void
webkit_web_extension_initialize(WebKitWebExtension *extension)
{
    g_signal_connect(extension, "page-created",
        G_CALLBACK(web_page_created_callback), NULL);
}
