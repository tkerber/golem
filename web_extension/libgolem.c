#include <gio/gio.h>
#include <webkit2/webkit-web-extension.h>
#include <glib.h>
#include <stdlib.h>

static const gchar introspection_xml[] =
    "<node>"
    "    <interface name='com.github.tkerber.golem.WebExtension'>"
    "        <method name='ScrollDelta'>"
    "            <arg type='x' name='delta' direction='in' />"
    "            <arg type='b' name='vertical' direction='in' />"
    "        </method>"
    "        <method name='ScrollToTop' />"
    "        <method name='ScrollToBottom' />"
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
    if(g_strcmp0(method_name, "ScrollDelta") == 0){
        const gint64 delta;
        const gboolean vertical;
        g_variant_get(parameters, "(xb)", &delta, &vertical);
        scroll_delta(user_data, delta, vertical);
        g_dbus_method_invocation_return_value(invocation, NULL);
    } else if(g_strcmp0(method_name, "ScrollToTop") == 0) {
        scroll_to_top(user_data);
        g_dbus_method_invocation_return_value(invocation, NULL);
    } else if(g_strcmp0(method_name, "ScrollToBottom") == 0) {
        scroll_to_bottom(user_data);
        g_dbus_method_invocation_return_value(invocation, NULL);
    }
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
    // Currently no properties exist.
    return NULL;
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
    // Currently no properties exist.
    return 0;
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
scroll_delta(gpointer web_page_p, gint64 delta, gboolean vertical)
{
    WebKitWebPage *web_page = web_page_p;
    WebKitDOMDocument *dom = webkit_web_page_get_dom_document(web_page);
    WebKitDOMElement *e = webkit_dom_document_get_active_element(dom);
    if(vertical) {
        webkit_dom_element_set_scroll_top(e, webkit_dom_element_get_scroll_top(e) + delta);
    } else {
        webkit_dom_element_set_scroll_left(e, webkit_dom_element_get_scroll_left(e) + delta);
    }
}

static void
scroll_to_top(gpointer web_page_p)
{
    WebKitWebPage *web_page = web_page_p;
    WebKitDOMDocument *dom = webkit_web_page_get_dom_document(web_page);
    WebKitDOMElement *e = webkit_dom_document_get_active_element(dom);
    webkit_dom_element_set_scroll_top(e, 0);
}

static void
scroll_to_bottom(gpointer web_page_p)
{
    WebKitWebPage *web_page = web_page_p;
    WebKitDOMDocument *dom = webkit_web_page_get_dom_document(web_page);
    WebKitDOMElement *e = webkit_dom_document_get_active_element(dom);
    webkit_dom_element_set_scroll_top(e, webkit_dom_element_get_scroll_height(e));
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

