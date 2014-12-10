#include <gio/gio.h>
#include <webkit2/webkit-web-extension.h>
#include <glib.h>
#include <stdlib.h>

static const gchar introspection_xml[] =
    "<node>"
    "    <interface name='com.github.tkerber.golem.WebExtension'>"
    "        <property type='x' name='ScrollTop' access='readwrite' />"
    "        <property type='x' name='ScrollLeft' access='readwrite' />"
    "        <property type='x' name='ScrollHeight' access='read' />"
    "        <property type='x' name='ScrollWidth' access='read' />"
    "    </interface>"
    "</node>";

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
    GVariant *ret = NULL;
    WebKitWebPage *web_page = user_data;
    WebKitDOMDocument *dom = webkit_web_page_get_dom_document(web_page);

    if(g_strcmp0(property_name, "ScrollTop") == 0) {
        WebKitDOMElement *e = WEBKIT_DOM_ELEMENT(
                webkit_dom_document_get_body(dom));
        ret = g_variant_new_int64(
                webkit_dom_element_get_scroll_top(e));
    } else if(g_strcmp0(property_name, "ScrollLeft") == 0) {
        WebKitDOMElement *e = WEBKIT_DOM_ELEMENT(
                webkit_dom_document_get_body(dom));
        ret = g_variant_new_int64(
                webkit_dom_element_get_scroll_left(e));
    } else if(g_strcmp0(property_name, "ScrollHeight") == 0) {
        WebKitDOMElement *e = WEBKIT_DOM_ELEMENT(
                webkit_dom_document_get_body(dom));
        ret = g_variant_new_int64(
                webkit_dom_element_get_scroll_height(e));
    } else if(g_strcmp0(property_name, "ScrollWidth") == 0) {
        WebKitDOMElement *e = WEBKIT_DOM_ELEMENT(
                webkit_dom_document_get_body(dom));
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
    GVariant *ret = NULL;
    WebKitWebPage *web_page = user_data;
    WebKitDOMDocument *dom = webkit_web_page_get_dom_document(web_page);

    if(g_strcmp0(property_name, "ScrollTop") == 0) {
        WebKitDOMElement *e = WEBKIT_DOM_ELEMENT(
                webkit_dom_document_get_body(dom));
        webkit_dom_element_set_scroll_top(e, g_variant_get_int64(value));
        return TRUE;
    } else if(g_strcmp0(property_name, "ScrollLeft") == 0) {
        WebKitDOMElement *e = WEBKIT_DOM_ELEMENT(
                webkit_dom_document_get_body(dom));
        webkit_dom_element_set_scroll_left(e, g_variant_get_int64(value));
        return TRUE;
    }
    // Currently no properties exist.
    return FALSE;
}

static void
on_bus_acquired(GDBusConnection *connection,
                const gchar     *name,
                gpointer         user_data)
{
    gint registration_id = g_dbus_connection_register_object(
            connection,
            "/com/github/tkerber/golem/WebExtension",
            introspection_data->interfaces[0],
            &interface_vtable,
            user_data,
            NULL,
            NULL);
    g_assert(registration_id > 0);
}

static void
on_name_lost(GDBusConnection *connection,
             const gchar     *name,
             gpointer         user_data)
{
    g_printerr("Lost DBus connection to main proccess.\n");
    exit(1);
}

static void
web_page_created_callback(WebKitWebExtension *extension,
                          WebKitWebPage      *web_page,
                          gpointer            user_data)
{
    guint owner_id;

    introspection_data = g_dbus_node_info_new_for_xml(introspection_xml, NULL);
    g_assert(introspection_data != NULL);

    owner_id = g_bus_own_name(G_BUS_TYPE_SESSION,
            "com.github.tkerber.golem.WebExtension",
            G_BUS_NAME_OWNER_FLAGS_NONE,
            on_bus_acquired,
            NULL,
            on_name_lost,
            web_page,
            NULL);
}

G_MODULE_EXPORT void
webkit_web_extension_initialize(WebKitWebExtension *extension)
{
    g_signal_connect(extension, "page-created",
        G_CALLBACK(web_page_created_callback), NULL);
}
