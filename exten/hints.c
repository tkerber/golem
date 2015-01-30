#include <webkitdom/webkitdom.h>
#include <glib.h>
#include <stdio.h>
#include "hints.h"

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
    return ret;
}

gboolean
hint_call_by_href(WebKitDOMNode *n, Exten *exten)
{
    if(!WEBKIT_DOM_IS_ELEMENT(n)) {
        return false;
    }
    WebKitDOMElement *e = WEBKIT_DOM_ELEMENT(n);
    printf("Href called: %s\n", webkit_dom_element_get_attribute(e, "HREF"));
    return false;
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
        WebKitDOMElement *span = NULL;
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
        gchar *style = g_strdup_printf("left:%fpx;top:%fpx",
                webkit_dom_element_get_offset_left(l->data),
                webkit_dom_element_get_offset_top(l->data));
        webkit_dom_element_set_attribute(div, "style", style, &err);
        g_free(style);
        if(err != NULL) {
            printf("Failed to set hint div position: %s\n", err->message);
            goto err;
        }
        // add hint div to DOM at the nodes parent.
        WebKitDOMNode *p = webkit_dom_node_get_parent_node(l->data);
        if(p == NULL) {
            printf("Failed to attach hint div: NULL parent\n");
            goto err;
        }
        webkit_dom_node_append_child(p, WEBKIT_DOM_NODE(div), &err);
        if(err != NULL) {
            printf("Failed to attach hint div: %s\n", err->message);
            goto err;
        }
        webkit_dom_element_set_class_name(div, "__golem-hint");
        // create highlight span
        span =
            webkit_dom_document_create_element(doc, "SPAN", &err);
        if(err != NULL) {
            printf("Failed to create hint span: %s\n", err->message);
            goto err;
        }
        webkit_dom_element_set_class_name(span, "__golem-highlight");
        webkit_dom_node_replace_child(p, WEBKIT_DOM_NODE(span), l->data, &err);
        if(err != NULL) {
            printf("Failed to inject hint span: %s\n", err->message);
            goto err;
        }
        webkit_dom_node_append_child(WEBKIT_DOM_NODE(span), l->data, &err);
        if(err != NULL) {
            printf("Failed to inject hint span: %s\n", err->message);
            goto err;
        }
        // add to hash table
        h->div = div;
        h->hl_span = span;
        g_hash_table_insert(hints, l->data, h);
        continue;
err:
        g_object_unref(l->data);
        if(err != NULL) {
            g_error_free(err);
        }
        g_free(h->text);
        g_free(h);
        if(div != NULL) {
            g_object_unref(div);
        }
        if(text != NULL) {
            g_object_unref(text);
        }
        if(span != NULL) {
            g_object_unref(span);
        }
    }
    g_list_free(nodes);
    g_free(hints_texts);
    HintsMode *hm = g_malloc(sizeof(HintsMode));
    hm->executer = ne;
    hm->hints = hints;
    exten->hints = hm;
}

void
filter_hints_mode(const gchar *hints, Exten *exten)
{
    if(exten->hints == NULL) {
        return;
    }
    GList *nodes = g_hash_table_get_keys(exten->hints->hints);
    GList *l;
    for(l = nodes; l != NULL; l = l->next) {
        Hint *h = g_hash_table_lookup(exten->hints->hints, l->data);
        if(g_str_has_prefix(h->text, hints)) {
            // If the hints exactly match, execute it.
            if(g_strcmp0(h->text, hints) == 0) {
                if(exten->hints->executer(l->data, exten)) {
                    hints_mode_filter("", exten);
                } else {
                    end_hints_mode(exten);
                }
                return;
            }
            webkit_dom_element_set_class_name(h->div, "__golem-hint");
            webkit_dom_element_set_class_name(h->hl_span, "__golem-highlight");
        } else {
            webkit_dom_element_set_class_name(h->div, "__golem-hide");
            webkit_dom_element_set_class_name(h->hl_span, "");
        }
    }
    g_list_free(nodes);
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
        // remove span
        p = webkit_dom_node_get_parent_node(WEBKIT_DOM_NODE(h->hl_span));
        if(p != NULL) {
            webkit_dom_node_remove_child(
                    WEBKIT_DOM_NODE(h->hl_span),
                    l->data,
                    &err);
            if(err == NULL) {
                webkit_dom_node_replace_child(
                        p,
                        l->data,
                        WEBKIT_DOM_NODE(h->hl_span),
                        &err);
            }
            if(err != NULL) {
                printf("Failed to restructure span div: %s\n", err->message);
                g_error_free(err);
            }
        }
        g_object_unref(h->hl_span);
        g_free(h);
    }
    g_list_free(nodes);
    g_hash_table_unref(exten->hints->hints);
    g_free(exten->hints);
    exten->hints = NULL;
}
