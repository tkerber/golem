#include <webkit2/webkit-web-extension.h>
#include <glib.h>
#include <stdio.h>
#include <libsoup/soup.h>
#include <string.h>
#include "hints.h"

static void
dom_get_absolute_position(WebKitDOMElement *e, gdouble *left, gdouble *top)
{
    if(e == NULL || WEBKIT_DOM_IS_HTML_BODY_ELEMENT(e)) {
        *left = 0;
        *top = 0;
        return;
    }
    gdouble parent_left, parent_top;
    dom_get_absolute_position(
            webkit_dom_element_get_offset_parent(e),
            &parent_left,
            &parent_top);
    *left = parent_left + webkit_dom_element_get_offset_left(e);
    *top = parent_top + webkit_dom_element_get_offset_top(e);
}

static void
highlight(WebKitDOMElement *e)
{
    gchar *class_name = webkit_dom_element_get_class_name(e);
    // not perfect, as classes containing __golem-highlight are also matched.
    // I assume the class name is sufficiently unique for this not to matter.
    if(class_name != NULL && strstr(class_name, "__golem-highlight") != NULL) {
        g_free(class_name);
        return;
    }
    if(class_name == NULL || strlen(class_name) == 0) {
        g_free(class_name);
        webkit_dom_element_set_class_name(e, "__golem-highlight");
    } else {
        gchar *new_class_name = g_strconcat(class_name, " __golem-highlight", NULL);
        g_free(class_name);
        webkit_dom_element_set_class_name(e, new_class_name);
        g_free(new_class_name);
    }
}

static void
unhighlight(WebKitDOMElement *e)
{
    gchar *class_name = webkit_dom_element_get_class_name(e);
    // not perfect, as seperators apart from space may be used. It will do
    // for our purposes.
    gchar **classes = g_strsplit(class_name, " ", 0);
    g_free(class_name);
    gchar **class;
    for(class = classes; *class != NULL; class++) {
        if(g_strcmp0(*class, "__golem-highlight") == 0) {
            // remove element
            g_free(*class);
            gchar **class2;
            // shift array
            for(class2 = class; *class2 != NULL; class2++) {
                *class2 = *(class2 + 1);
            }
            // repeat
            class--;
        }
    }
    gchar *new_class_name = g_strjoinv(" ", classes);
    g_strfreev(classes);
    webkit_dom_element_set_class_name(e, new_class_name);
    g_free(new_class_name);
}

static gchar **
get_hints_texts(guint length, Exten *exten, GError **err) {
    GVariant *retv = g_dbus_connection_call_sync(
            exten->connection,
            exten->golem_name,
            "/com/github/tkerber/Golem",
            "com.github.tkerber.Golem",
            "GetHintsLabels",
            g_variant_new(
                "(x)",
                (gint64)length),
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

gboolean
hint_call_by_href(WebKitDOMNode *n, Exten *exten)
{
    if(!WEBKIT_DOM_IS_ELEMENT(n)) {
        return true;
    }
    WebKitDOMElement *e = WEBKIT_DOM_ELEMENT(n);
    gchar *doc_url = webkit_dom_document_get_url(webkit_dom_node_get_owner_document(n));
    SoupURI *uri_base = soup_uri_new(doc_url);
    g_free(doc_url);
    gchar *href = webkit_dom_element_get_attribute(e, "HREF");
    SoupURI *uri = soup_uri_new_with_base(uri_base, href);
    g_free(href);
    soup_uri_free(uri_base);
    char *str = soup_uri_to_string(uri, false);
    soup_uri_free(uri);

    GError *err = NULL;
    GVariant *retv = g_dbus_connection_call_sync(
            exten->connection,
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
            &err);
    g_free(str);
    gboolean ret = FALSE;
    if(err != NULL) {
        printf("Failed to call hint: %s\n", err->message);
        g_error_free(err);
    } else {
        g_variant_get(retv, "(b)", &ret);
        g_variant_unref(retv);
    }
    return ret;
}

// Clicks the passed node.
gboolean
hint_call_by_click(WebKitDOMNode *n, Exten *exten)
{
    WebKitDOMDocument *doc = webkit_dom_node_get_owner_document(n);
    GError *err = NULL;
    WebKitDOMEvent *e = webkit_dom_document_create_event(doc, "MouseEvents", &err);
    if(err != NULL) {
        printf("Failed to click element: %s\n", err->message);
        g_error_free(err);
        return FALSE;
    }
    webkit_dom_mouse_event_init_mouse_event(
            WEBKIT_DOM_MOUSE_EVENT(e),
            "click",
            true,
            true,
            webkit_dom_document_get_default_view(doc),
            0,
            0,
            0,
            0,
            0,
            false,
            false,
            false,
            false,
            0,
            WEBKIT_DOM_EVENT_TARGET(n));
    webkit_dom_event_target_dispatch_event(
            WEBKIT_DOM_EVENT_TARGET(n),
            e,
            &err);
    if(err != NULL) {
        printf("Failed to click element: %s\n", err->message);
        g_error_free(err);
    }
    g_object_unref(e);
    return FALSE;
}

// Selects all elements which may normally be clicked.
// 
// - Anchor elements
// - Input elements
// - Embed elements
// - Button elements
// - TextArea elements
GList *
select_clickable(Exten *exten)
{
    const char* const tags[] = {"A", "INPUT", "EMBED", "BUTTON", "TEXTAREA", NULL};
    GList *ret = NULL;
    GList *docs = g_hash_table_get_keys(exten->registered_documents);
    GList *l;
    for(l = docs; l != NULL; l = l->next) {
        guint i;
        for(i = 0; tags[i] != NULL; i++) {
            // Special case for A elements: we select links instead of by tag name
            WebKitDOMNodeList *nl =
                webkit_dom_document_get_elements_by_tag_name(l->data, tags[i]);
            gulong len = webkit_dom_node_list_get_length(nl);
            gulong i;
            for(i = 0; i < len; i++) {
                WebKitDOMNode *item = webkit_dom_node_list_item(nl, i);
                // special case for A elements: ignore those without href.
                if(i == 0) {
                    if(!WEBKIT_DOM_IS_HTML_ANCHOR_ELEMENT(item)) {
                        continue;
                    }
                    gchar *href = webkit_dom_html_anchor_element_get_href(
                            WEBKIT_DOM_HTML_ANCHOR_ELEMENT(item));
                    if(href == NULL || *href == '\0') {
                        g_free(href);
                        continue;
                    }
                    g_free(href);
                }
                g_object_ref(item);
                ret = g_list_prepend(ret, item);
            }
        }
    }
    g_list_free(docs);
    return ret;
}

GList *
select_links(Exten *exten)
{
    GList *ret = NULL;
    GList *docs = g_hash_table_get_keys(exten->registered_documents);
    GList *l;
    for(l = docs; l != NULL; l = l->next) {
        WebKitDOMHTMLCollection *coll = webkit_dom_document_get_links(l->data);
        gulong len = webkit_dom_html_collection_get_length(coll);
        gulong i;
        for(i = 0; i < len; i++) {
            WebKitDOMNode *item = webkit_dom_html_collection_item(coll, i);
            g_object_ref(item);
            ret = g_list_prepend(ret, item);
        }
    }
    g_list_free(docs);
    return ret;
}

void
start_hints_mode(NodeSelecter ns, NodeExecuter ne, Exten *exten)
{
    if(exten->hints) {
        end_hints_mode(exten);
    }
    GError *err = NULL;
    GList *nodes = ns(exten);
    guint len = g_list_length(nodes);
    gchar **hints_texts = get_hints_texts(len, exten, &err);
    if(err != NULL) {
        printf("Failed to get hints texts: %s\n", err->message);
        g_error_free(err);
        return;
    }
    GHashTable *hints = g_hash_table_new(NULL, NULL);
    GList *l;
    guint i = 0;
    for(l = nodes; l != NULL; l = l->next) {
        Hint *h = g_malloc(sizeof(Hint));
        h->text = *(hints_texts + i++);
        WebKitDOMElement *div = NULL;
        WebKitDOMText *text = NULL;
        WebKitDOMDocument *doc = webkit_dom_node_get_owner_document(l->data);
        // create new hint div.
        div =
            webkit_dom_document_create_element(doc, "DIV", &err);
        if(err != NULL) {
            printf("Failed to create hint div: %s\n", err->message);
            goto err;
        }
        text =
            webkit_dom_document_create_text_node(doc, h->text);
        webkit_dom_node_append_child(
                WEBKIT_DOM_NODE(div),
                WEBKIT_DOM_NODE(text),
                &err);
        if(err != NULL) {
            printf("Failed to create hint div: %s\n", err->message);
            goto err;
        }
        g_object_unref(text);
        text = NULL;
        // set hint div position
        gdouble left, top;
        dom_get_absolute_position(l->data, &left, &top);
        gchar *style = g_strdup_printf("left:%fpx;top:%fpx",
                left,
                top);
        webkit_dom_element_set_attribute(div, "style", style, &err);
        g_free(style);
        if(err != NULL) {
            printf("Failed to set hint div position: %s\n", err->message);
            goto err;
        }
        // add hint div to DOM at the document body
        WebKitDOMNode *p = WEBKIT_DOM_NODE(webkit_dom_document_get_body(doc));
        if(p == NULL) {
            printf("Failed to attach hint div: NULL body\n");
            goto err;
        }
        webkit_dom_node_append_child(p, WEBKIT_DOM_NODE(div), &err);
        if(err != NULL) {
            printf("Failed to attach hint div: %s\n", err->message);
            goto err;
        }
        webkit_dom_element_set_class_name(div, "__golem-hint");
        // highlight the element by adding it to the __golem-highlight class.
        highlight(l->data);
        // add to hash table
        h->div = div;
        g_hash_table_insert(hints, l->data, h);
        continue;
err:
        g_object_unref(l->data);
        if(err != NULL) {
            g_error_free(err);
            err = NULL;
        }
        g_free(h->text);
        g_free(h);
        if(div != NULL) {
            g_object_unref(div);
        }
        if(text != NULL) {
            g_object_unref(text);
        }
    }
    g_list_free(nodes);
    g_free(hints_texts);
    HintsMode *hm = g_malloc(sizeof(HintsMode));
    hm->executer = ne;
    hm->hints = hints;
    exten->hints = hm;
}

gboolean
filter_hints_mode(const gchar *hints, Exten *exten)
{
    gchar *hints_ci = g_utf8_casefold(hints, -1);
    if(exten->hints == NULL) {
        return;
    }
    GList *nodes = g_hash_table_get_keys(exten->hints->hints);
    GList *l;
    for(l = nodes; l != NULL; l = l->next) {
        Hint *h = g_hash_table_lookup(exten->hints->hints, l->data);
        gchar *text_ci = g_utf8_casefold(h->text, -1);
        if(g_str_has_prefix(text_ci, hints_ci)) {
            // If the hints exactly match, execute it.
            if(g_strcmp0(text_ci, hints_ci) == 0) {
                if(exten->hints->executer(l->data, exten)) {
                    hints_mode_filter("", exten);
                    g_free(hints_ci);
                    g_free(text_ci);
                    return FALSE;
                } else {
                    end_hints_mode(exten);
                    g_free(hints_ci);
                    g_free(text_ci);
                    return TRUE;
                }
            }
            webkit_dom_element_set_class_name(h->div, "__golem-hint");
            highlight(l->data);
        } else {
            webkit_dom_element_set_class_name(h->div, "__golem-hide");
            unhighlight(l->data);
        }
        g_free(text_ci);
    }
    g_list_free(nodes);
    g_free(hints_ci);
    return FALSE;
}

void
end_hints_mode(Exten *exten)
{
    if(exten->hints == NULL) {
        return;
    }
    GList *nodes = g_hash_table_get_keys(exten->hints->hints);
    GList *l;
    for(l = nodes; l != NULL; l = l->next) {
        Hint *h = g_hash_table_lookup(exten->hints->hints, l->data);
        GError *err = NULL;
        g_free(h->text);
        // remove div
        WebKitDOMNode *p = webkit_dom_node_get_parent_node(WEBKIT_DOM_NODE(h->div));
        if(p != NULL) {
            webkit_dom_node_remove_child(p, WEBKIT_DOM_NODE(h->div), &err);
            if(err != NULL) {
                printf("Failed to remove hint div: %s\n", err->message);
                g_error_free(err);
            }
        }
        g_object_unref(h->div);

        unhighlight(l->data);
        g_free(h);
    }
    g_list_free(nodes);
    g_hash_table_unref(exten->hints->hints);
    g_free(exten->hints);
    exten->hints = NULL;
}
