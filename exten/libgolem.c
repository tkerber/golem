#include <gio/gio.h>
#include <webkit2/webkit-web-extension.h>
#include <glib.h>
#include <stdlib.h>
#include <stdio.h>
#include "libgolem.h"
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
    "        <signal name='VerticalPositionChanged'>"
    "            <arg type='x' name='ScrollTop' />"
    "            <arg type='x' name='ScrollHeight' />"
    "        </signal>"
    "        <signal name='InputFocusChanged'>"
    "            <arg type='b' name='InputFocused' />"
    "        </signal>"
    "        <method name='LinkHintsMode'>"
    "            <arg type='b' name='Empty' direction='out' />"
    "        </method>"
    "        <method name='FormVariableHintsMode'>"
    "            <arg type='b' name='Empty' direction='out' />"
    "        </method>"
    "        <method name='ClickHintsMode'>"
    "            <arg type='b' name='Empty' direction='out' />"
    "        </method>"
    "        <method name='EndHintsMode' />"
    "        <method name='FilterHintsMode'>"
    "            <arg type='s' name='Prefix' direction='in' />"
    "            <arg type='b' name='HitAndEnd' direction='out' />"
    "        </method>"
    "    </interface>"
    "</node>";

// Adblock constants
#define ADBLOCK_SCRIPT            (1<<0)
#define ADBLOCK_IMAGE             (1<<1)
#define ADBLOCK_STYLE_SHEET       (1<<2)
#define ADBLOCK_OBJECT            (1<<3)
#define ADBLOCK_XML_HTTP_REQUEST  (1<<4)
#define ADBLOCK_OBJECT_SUBREQUEST (1<<5)
#define ADBLOCK_SUBDOCUMENT       (1<<6)
#define ADBLOCK_DOCUMENT          (1<<7)
#define ADBLOCK_ELEMHIDE          (1<<8)
#define ADBLOCK_OTHER             (1<<9)

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

// frame_document_loaded watches signals emitted from the given document.
static void
frame_document_loaded(WebKitDOMDocument *doc,
                      Exten             *exten);

static void
inject_adblock_css(WebKitDOMDocument *doc,
                   Exten             *exten);

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
        start_hints_mode(select_links, hint_call_by_href, exten);
        g_dbus_method_invocation_return_value(invocation,
                g_variant_new("(b)", FALSE));
    } else if(g_strcmp0(method_name, "FormVariableHintsMode") == 0) {
        start_hints_mode(select_form_text_variables, hint_call_by_form_variable_get, exten);
        g_dbus_method_invocation_return_value(invocation,
                g_variant_new("(b)", FALSE));
    } else if(g_strcmp0(method_name, "ClickHintsMode") == 0) {
        start_hints_mode(select_clickable, hint_call_by_click, exten);
        g_dbus_method_invocation_return_value(invocation,
                g_variant_new("(b)", FALSE));
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

// uri_is_blocked queries if a uri is blocked.
static gboolean
uri_is_blocked(const char *uri, guint64 flags, Exten *exten)
{
    GError *err = NULL;
    GVariant *ret = g_dbus_connection_call_sync(
            exten->connection,
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
            &err);
    if(err != NULL) {
        printf("Failed to check if uri is blocked: %s\n", err->message);
        g_error_free(err);
        return false;
    }
    gboolean blocked = g_variant_get_boolean(g_variant_get_child_value(ret, 0));
    g_variant_unref(ret);
    return blocked;
}

// uri_request_cb is called when a uri request is issued, and determines
// whether to allow it to proceed or not.
static void
uri_request_cb(WebKitWebPage     *page,
               WebKitURIRequest  *req,
               WebKitURIResponse *resp,
               gpointer           exten)
{
    const gchar *uri = webkit_uri_request_get_uri(req);
    if(uri_is_blocked(uri, ADBLOCK_OTHER, exten)) {
        webkit_uri_request_set_uri(req, "about:blank");
    }
}

// is_scroll_target checks if a DOM element can be scrolled in.
//
// TODO: The way this is done now doesn't *really* work in all cases.
static gboolean
is_scroll_target(WebKitDOMElement *elem)
{
    WebKitDOMElement *parent = webkit_dom_element_get_offset_parent(elem);
    if(parent == NULL) {
        return true;
    }
    glong height = webkit_dom_element_get_scroll_height(elem);
    glong width = webkit_dom_element_get_scroll_width(elem);
    glong parentHeight = webkit_dom_element_get_scroll_height(parent);
    glong parentWidth = webkit_dom_element_get_scroll_width(parent);
    return parentHeight < height || parentWidth < width;
}

// get_scroll_target gets the first parent of the passed element which is a
// scroll target.
static WebKitDOMElement *
get_scroll_target(WebKitDOMElement *elem)
{
    WebKitDOMElement *prev = elem;
    while(!is_scroll_target(elem)) {
        elem = webkit_dom_element_get_offset_parent(elem);
    }
    return elem;
}

// document_scroll_cb is called when the document is scrolled, and updates
// the main processes knowledge of the document.
static void
document_scroll_cb(WebKitDOMEventTarget *target,
                   WebKitDOMEvent       *event,
                   gpointer              user_data)
{
    Exten *exten = user_data;
    WebKitDOMDocument *dom = WEBKIT_DOM_DOCUMENT(target);
    WebKitDOMElement *e = NULL;
    if(dom != NULL) {
        e = WEBKIT_DOM_ELEMENT(webkit_dom_document_get_body(dom));
    }

    // Check for current scroll position. If it has changed, signal DBus.
    if(e != NULL) {
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
    }

    if(dom != NULL) {
        e = webkit_dom_document_get_active_element(dom);
    }
}

// active_element_change_cb is called when the active element is changed, and
// updates bookkeeping and the main process.
static void
active_element_change_cb(WebKitDOMEventTarget *target,
                         WebKitDOMEvent       *event,
                         gpointer              user_data)
{
    Exten *exten = user_data;
    WebKitDOMDocument *document;
    if(WEBKIT_DOM_IS_DOCUMENT(target)) {
        document = WEBKIT_DOM_DOCUMENT(target);
    } else {
        // target is a window.
        g_object_get(target, "document", &document, NULL);
    }
    WebKitDOMElement *active = webkit_dom_document_get_active_element(document);
    if(active == NULL || active == exten->active) {
        return;
    }
    if(WEBKIT_DOM_IS_HTML_IFRAME_ELEMENT(active)) {
        // The iframe document handles this.
        return;
    }
    exten->active = active;
    exten->scroll_target = get_scroll_target(active);

    // Check whether the currently active element is an input element.
    // If this has changed, signal DBus.
    //
    // Input elements:
    //
    // WebKitDOMHTMLAppletElement
    // WebKitDOMHTMLEmbedElement
    // WebKitDOMHTMLInputElement
    // WebKitDOMHTMLTextAreaElement
    gboolean input_focus = (
            WEBKIT_DOM_IS_HTML_APPLET_ELEMENT(active) ||
            WEBKIT_DOM_IS_HTML_EMBED_ELEMENT(active) ||
            WEBKIT_DOM_IS_HTML_INPUT_ELEMENT(active) ||
            WEBKIT_DOM_IS_HTML_TEXT_AREA_ELEMENT(active));
    if(input_focus != exten->last_input_focus) {
        exten->last_input_focus = input_focus;
        g_dbus_connection_emit_signal(
                exten->connection,
                NULL,
                exten->object_path,
                "com.github.tkerber.golem.WebExtension",
                "InputFocusChanged",
                g_variant_new("(b)", input_focus),
                NULL);
    }
}

// adblock_before_load_cb is triggered when 
static void
adblock_before_load_cb(WebKitDOMEventTarget *doc,
                       WebKitDOMEvent       *event,
                       gpointer              user_data)
{
    WebKitDOMEventTarget *target = webkit_dom_event_get_target(event);

    guint64 flags = 0;
    gchar *uri = NULL;
    if(WEBKIT_DOM_IS_HTML_LINK_ELEMENT(target)) {
        WebKitDOMHTMLLinkElement *e = WEBKIT_DOM_HTML_LINK_ELEMENT(target);
        gboolean isCSS = 0;
        isCSS |= g_strcmp0(
                webkit_dom_html_link_element_get_rel(e),
                "stylesheet") == 0;
        isCSS |= g_strcmp0(
                webkit_dom_html_link_element_get_type_attr(e),
                "text/css") == 0;
        if(!isCSS) {
            return;
        }
        uri = webkit_dom_html_link_element_get_href(e);
        flags |= ADBLOCK_STYLE_SHEET;
    } else if(WEBKIT_DOM_IS_HTML_OBJECT_ELEMENT(target)) {
        WebKitDOMHTMLObjectElement *e = WEBKIT_DOM_HTML_OBJECT_ELEMENT(target);
        uri = webkit_dom_html_object_element_get_data(e);
        flags |= ADBLOCK_OBJECT;
    } else if(WEBKIT_DOM_IS_HTML_EMBED_ELEMENT(target)) {
        WebKitDOMHTMLEmbedElement *e = WEBKIT_DOM_HTML_EMBED_ELEMENT(target);
        uri = webkit_dom_html_embed_element_get_src(e);
        flags |= ADBLOCK_OBJECT;
    } else if(WEBKIT_DOM_IS_HTML_IMAGE_ELEMENT(target)) {
        WebKitDOMHTMLImageElement *e = WEBKIT_DOM_HTML_IMAGE_ELEMENT(target);
        uri = webkit_dom_html_image_element_get_src(e);
        flags |= ADBLOCK_IMAGE;
    } else if(WEBKIT_DOM_IS_HTML_SCRIPT_ELEMENT(target)) {
        WebKitDOMHTMLScriptElement *e = WEBKIT_DOM_HTML_SCRIPT_ELEMENT(target);
        uri = webkit_dom_html_script_element_get_src(e);
        flags |= ADBLOCK_SCRIPT;
    } else if(WEBKIT_DOM_IS_HTML_IFRAME_ELEMENT(target)) {
        WebKitDOMHTMLIFrameElement *e = WEBKIT_DOM_HTML_IFRAME_ELEMENT(target);
        uri = webkit_dom_html_iframe_element_get_src(e);
        flags |= ADBLOCK_SUBDOCUMENT;
        if(uri_is_blocked(uri, flags, user_data)) {
            webkit_dom_event_prevent_default(event);
        } else {
            frame_document_loaded(
                    webkit_dom_html_iframe_element_get_content_document(e),
                    user_data);
        }
        g_free(uri);
        return;
    }
    if(uri == NULL) {
        return;
    }
    if(uri_is_blocked(uri, flags, user_data)) {
        webkit_dom_event_prevent_default(event);
    }
    g_free(uri);
}

static void
inject_adblock_css(WebKitDOMDocument *doc,
                   Exten             *exten)
{
    // Get css rules
    gchar *domain = webkit_dom_document_get_domain(doc);
    GError *err = NULL;
    GVariant *ret = g_dbus_connection_call_sync(
            exten->connection,
            exten->golem_name,
            "/com/github/tkerber/Golem",
            "com.github.tkerber.Golem",
            "DomainElemHideCSS",
            g_variant_new("(s)", domain),
            G_VARIANT_TYPE("(s)"),
            G_DBUS_CALL_FLAGS_NONE,
            -1,
            NULL,
            &err);
    if(err != NULL) {
        printf("Failed to retrieve element hide CSS: %s\n", err->message);
        g_error_free(err);
        return;
    }
    gchar *css = g_variant_dup_string(
            g_variant_get_child_value(ret, 0),
            NULL);
    g_variant_unref(ret);

    // Add CSS
    WebKitDOMElement *style_elem = webkit_dom_document_create_element(
            doc,
            "STYLE",
            &err);
    if(err != NULL) {
        printf("Failed to inject style: %s\n", err->message);
        g_error_free(err);
        g_free(domain);
        g_free(css);
        return;
    }
    webkit_dom_html_element_set_inner_html(
            WEBKIT_DOM_HTML_ELEMENT(style_elem),
            css,
            &err);
    if(err != NULL) {
        printf("Failed to inject style: %s\n", err->message);
        g_error_free(err);
        g_free(domain);
        g_free(css);
        return;
    }
    WebKitDOMHTMLHeadElement *head = webkit_dom_document_get_head(doc);
    webkit_dom_node_append_child(
            WEBKIT_DOM_NODE(head),
            WEBKIT_DOM_NODE(style_elem),
            &err);
    if(err != NULL) {
        printf("Failed to inject style: %s\n", err->message);
        g_error_free(err);
        g_free(domain);
        g_free(css);
        return;
    }
}

// frame_document_loaded watches signals emitted from the given document.
static void
frame_document_loaded(WebKitDOMDocument *doc,
                      Exten             *exten)
{
    // Track document, and don't register multiple times.
    if(!g_hash_table_add(exten->registered_documents, doc)) {
        return;
    }
    WebKitDOMEventTarget *target = WEBKIT_DOM_EVENT_TARGET(
            webkit_dom_document_get_default_view(doc));
    // listen for focus changes
    webkit_dom_event_target_add_event_listener(
            target,
            "blur",
            G_CALLBACK(active_element_change_cb),
            true,
            exten);
    webkit_dom_event_target_add_event_listener(
            target,
            "focus",
            G_CALLBACK(active_element_change_cb),
            true,
            exten);

    // listen for resource loads.
    webkit_dom_event_target_add_event_listener(
            target,
            "beforeload",
            G_CALLBACK(adblock_before_load_cb),
            true,
            exten);

    // Scan for existing iframes, and add them as new frames.
    WebKitDOMNodeList *nodes = webkit_dom_document_get_elements_by_tag_name(
            WEBKIT_DOM_DOCUMENT(doc),
            "IFRAME");
    gulong i;
    gulong len = webkit_dom_node_list_get_length(nodes);
    for(i = 0; i < len; i++) {
        WebKitDOMDocument *subdoc =
            webkit_dom_html_iframe_element_get_content_document(
                    WEBKIT_DOM_HTML_IFRAME_ELEMENT(
                        webkit_dom_node_list_item(nodes, i)));
        frame_document_loaded(subdoc, exten);
    }
    // Element hider
    inject_adblock_css(doc, exten);
}

// document_loaded_cb is called when a document is loaded, and updates
// internal bookkeeping and attaches to signals from the document.
static void
document_loaded_cb(WebKitWebPage *page,
                   gpointer       user_data)
{
    Exten *exten = user_data;
    if(exten->registered_documents) {
        g_hash_table_unref(exten->registered_documents);
    }
    exten->registered_documents = g_hash_table_new(NULL, NULL);
    exten->document = webkit_web_page_get_dom_document(page);
    frame_document_loaded(exten->document, exten);
    active_element_change_cb(
            WEBKIT_DOM_EVENT_TARGET(exten->document),
            NULL,
            exten);
    // listen for scroll changes.
    webkit_dom_event_target_add_event_listener(
            WEBKIT_DOM_EVENT_TARGET(exten->document),
            "scroll",
            G_CALLBACK(document_scroll_cb),
            false,
            exten);
}

// on_bus_acquired is called when a DBus bus is acquired, and proceeds with
// starting up the web extension.
static void
on_bus_acquired(GDBusConnection *connection,
                const gchar     *name,
                gpointer         user_data)
{
    Exten *exten = user_data;
    exten->connection = connection;
    exten->last_top = 0;
    exten->last_height = 0;
    exten->last_input_focus = FALSE;
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

    g_signal_connect(
            exten->web_page,
            "document-loaded",
            G_CALLBACK(document_loaded_cb),
            exten);
    // Register the request signal...
    g_signal_connect(
            exten->web_page,
            "send-request",
            G_CALLBACK(uri_request_cb),
            exten);
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

// web_page_created_callback is called when a web page is created, and creates
// a DBus connection for this page.
//
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
    Exten *exten = malloc(sizeof(Exten));
    exten->hints = NULL;
    exten->web_page = web_page;
    exten->document = NULL;
    exten->active = NULL;
    exten->scroll_target = NULL;
    exten->profile = user_data;
    exten->golem_name = g_strdup_printf(
            "com.github.tkerber.Golem.%s", exten->profile);
    exten->registered_documents = NULL;
    guint owner_id;

    introspection_data = g_dbus_node_info_new_for_xml(introspection_xml, NULL);
    g_assert(introspection_data != NULL);
    gchar *bus_name = g_strdup_printf(
            "com.github.tkerber.golem.WebExtension.%s.Page%d", 
            exten->profile,
            webkit_web_page_get_id(web_page));
    owner_id = g_bus_own_name(G_BUS_TYPE_SESSION,
            bus_name,
            G_BUS_NAME_OWNER_FLAGS_NONE,
            on_bus_acquired,
            NULL,
            on_name_lost,
            exten,
            NULL);
    g_free(bus_name);
}

// webkit_web_extension_initialize_with_user_data initializes the web extension
//
// The profile name should be passed as the user data.
G_MODULE_EXPORT void
webkit_web_extension_initialize_with_user_data(WebKitWebExtension *extension,
                                               GVariant           *data)
{
    gchar *profile = g_variant_dup_string(data, NULL);
    g_signal_connect(extension, "page-created",
        G_CALLBACK(web_page_created_callback), profile);
}
