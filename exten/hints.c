#include <webkit2/webkit-web-extension.h>
#include <glib.h>
#include <stdio.h>
#include <libsoup/soup.h>
#include <string.h>
#include "hints.h"

typedef struct _Point {
    gdouble x;
    gdouble y;
} Point;

// Returns a point. The point is owned by h, and must not be freed.
static Point *
dom_get_absolute_position(
        GHashTable *h,
        WebKitDOMElement *e)
{
    if(g_hash_table_contains(h, e)) {
        return g_hash_table_lookup(h, e);
    }
    Point *point = g_malloc(sizeof(Point));
    if(e == NULL || WEBKIT_DOM_IS_HTML_BODY_ELEMENT(e)) {
        point->x = 0;
        point->y = 0;
    } else {
        WebKitDOMElement *parent = webkit_dom_element_get_offset_parent(e);
        Point *p = dom_get_absolute_position(h, parent);
        point->x = p->x + webkit_dom_element_get_offset_left(e);
        point->y = p->y + webkit_dom_element_get_offset_top(e);
    }
    g_hash_table_insert(h, e, point);
    return point;
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

// calls a hint with the given string.
//
// Ownership of the string is transferred to this function, and it will be
// freed.
static gboolean
hint_call(gchar *str, Exten *exten)
{
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

gboolean
hint_call_by_form_variable_get(WebKitDOMNode *n, Exten *exten)
{
    if(!WEBKIT_DOM_IS_HTML_INPUT_ELEMENT(n)) {
        return TRUE;
    }
    WebKitDOMHTMLInputElement *input = WEBKIT_DOM_HTML_INPUT_ELEMENT(n);
    WebKitDOMHTMLFormElement *form = webkit_dom_html_input_element_get_form(input);
    if(form == NULL) {
        return TRUE;
    }
    gchar *doc_url = webkit_dom_document_get_url(webkit_dom_node_get_owner_document(n));
    SoupURI *uri_base = soup_uri_new(doc_url);
    g_free(doc_url);
    gchar *action = webkit_dom_html_form_element_get_action(form);
    SoupURI *uri = soup_uri_new_with_base(uri_base, action);
    g_free(action);
    soup_uri_free(uri_base);

    WebKitDOMHTMLCollection *coll = webkit_dom_html_form_element_get_elements(form);
    gulong len = webkit_dom_html_collection_get_length(coll);
    gchar **names = g_malloc(sizeof(gchar*) * len);
    gchar **values = g_malloc(sizeof(gchar*) * len);
    gulong i;
    for(i = 0; i < len; i++) {
        WebKitDOMNode *node = webkit_dom_html_collection_item(coll, i);
        if(node == n) {
            names[i] = webkit_dom_html_input_element_get_name(input);
            values[i] = g_strdup("__golem_form_variable");
        } else if(WEBKIT_DOM_IS_HTML_INPUT_ELEMENT(node)) {
            gchar *type = webkit_dom_html_input_element_get_input_type(
                    WEBKIT_DOM_HTML_INPUT_ELEMENT(node));
            if(g_strcmp0(type, "submit") == 0 ||
                    g_strcmp0(type, "button") == 0) {
                g_free(type);
                goto skip;
            }
            g_free(type);
            names[i] = webkit_dom_html_input_element_get_name(
                    WEBKIT_DOM_HTML_INPUT_ELEMENT(node));
            values[i] = webkit_dom_html_input_element_get_value(
                    WEBKIT_DOM_HTML_INPUT_ELEMENT(node));
        } else if(WEBKIT_DOM_IS_HTML_SELECT_ELEMENT(node)) {
            names[i] = webkit_dom_html_select_element_get_name(
                    WEBKIT_DOM_HTML_SELECT_ELEMENT(node));
            values[i] = webkit_dom_html_select_element_get_value(
                    WEBKIT_DOM_HTML_SELECT_ELEMENT(node));
        } else if(WEBKIT_DOM_IS_HTML_TEXT_AREA_ELEMENT(node)) {
            names[i] = webkit_dom_html_text_area_element_get_name(
                    WEBKIT_DOM_HTML_TEXT_AREA_ELEMENT(node));
            values[i] = webkit_dom_html_text_area_element_get_value(
                    WEBKIT_DOM_HTML_TEXT_AREA_ELEMENT(node));
        } else {
skip:
            names[i] = NULL;
            values[i] = NULL;
        }
        gchar *tmp = names[i];
        if(tmp != NULL) {
            names[i] = soup_uri_encode(tmp, NULL);
            g_free(tmp);
        }
        tmp = values[i];
        if(tmp != NULL) {
            values[i] = soup_uri_encode(tmp, NULL);
            g_free(tmp);
        }
    }
    // Length: 2 for each form element (? for first, & for all others), and =
    // + length of name, + length of value (except for [input] which uses
    // __golem_form_variable instead of value.
    guint opts_len = 0;
    for(i = 0; i < len; i++) {
        if(names[i] != NULL && *names[i] != '\0' && values[i] != NULL) {
            opts_len += 2 + strlen(names[i]) + strlen(values[i]);
        }
    }
    gchar *get_opts = g_malloc(sizeof(gchar) * (opts_len + 1));
    gboolean first = TRUE;
    gchar *get_opts_at = get_opts;
    for(i = 0; i < len; i++) {
        if(names[i] != NULL && *names[i] != '\0' && values[i] != NULL) {
            if(first) {
                *(get_opts_at++) = '?';
                first = FALSE;
            } else {
                *(get_opts_at++) = '&';
            }
            get_opts_at = g_stpcpy(get_opts_at, names[i]);
            *(get_opts_at++) = '=';
            get_opts_at = g_stpcpy(get_opts_at, values[i]);
        }
        g_free(names[i]);
        g_free(values[i]);
    }
    g_free(names);
    g_free(values);
    SoupURI *final_uri = soup_uri_new_with_base(uri, get_opts);
    g_free(get_opts);
    soup_uri_free(uri);
    char *str = soup_uri_to_string(final_uri, false);
    soup_uri_free(final_uri);

    return hint_call(str, exten);
}

gboolean
hint_call_by_href(WebKitDOMNode *n, Exten *exten)
{
    if(!WEBKIT_DOM_IS_ELEMENT(n)) {
        return TRUE;
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

    return hint_call(str, exten);
}

// Clicks the passed node.
gboolean
hint_call_by_click(WebKitDOMNode *n, Exten *exten)
{
    if(WEBKIT_DOM_IS_HTML_INPUT_ELEMENT(n)) {
        webkit_dom_html_input_element_select(
                WEBKIT_DOM_HTML_INPUT_ELEMENT(n));
    } else if(WEBKIT_DOM_IS_HTML_TEXT_AREA_ELEMENT(n)) {
        webkit_dom_html_text_area_element_select(
                WEBKIT_DOM_HTML_TEXT_AREA_ELEMENT(n));
    } else if(WEBKIT_DOM_IS_HTML_ELEMENT(n)) {
        webkit_dom_html_element_click(WEBKIT_DOM_HTML_ELEMENT(n));
    } else if(WEBKIT_DOM_IS_ELEMENT(n)) {
        webkit_dom_element_focus(WEBKIT_DOM_ELEMENT(n));
    }
    return FALSE;
}

// TODO: For now, this is limited to check if it is visible within its own
// document.
static gboolean
is_visible(GHashTable *h, WebKitDOMNode *n) {
    if(!WEBKIT_DOM_IS_ELEMENT(n)) {
        return FALSE;
    }
    WebKitDOMElement *e = WEBKIT_DOM_ELEMENT(n);
    Point *p = dom_get_absolute_position(h, e);

    glong vp_width, vp_height, vp_x_offset, vp_y_offset;
    WebKitDOMDOMWindow *vp = webkit_dom_document_get_default_view(
            webkit_dom_node_get_owner_document(n));
    g_object_get(vp,
            "inner-width", &vp_width,
            "inner-height", &vp_height,
            "page-x-offset", &vp_x_offset,
            "page-y-offset", &vp_y_offset,
            NULL);
    return
        p->x >= vp_x_offset &&
        p->x <= vp_x_offset + vp_width &&
        p->y >= vp_y_offset &&
        p->y <= vp_y_offset + vp_height;
}

static void
scan_documents(WebKitDOMDocument *doc, GList **l, Exten *exten)
{
    *l = g_list_prepend(*l, doc);
    WebKitDOMNodeList *iframes = webkit_dom_document_get_elements_by_tag_name(doc, "IFRAME");
    gulong len = webkit_dom_node_list_get_length(iframes);
    gulong i;
    for(i = 0; i < len; i++) {
        scan_documents(webkit_dom_html_iframe_element_get_content_document(
                WEBKIT_DOM_HTML_IFRAME_ELEMENT(
                    webkit_dom_node_list_item(iframes, i))),
                l,
                exten);
    }
}

// Selects text input elements of forms.
GList *
select_form_text_variables(GHashTable *h, Exten *exten)
{
    GList *ret = NULL;
    GList *docs = NULL;
    scan_documents(exten->document, &docs, exten);
    GList *l;
    for(l = docs; l != NULL; l = l->next) {
        WebKitDOMNodeList *nl = webkit_dom_document_get_elements_by_tag_name(
                l->data, "INPUT");
        gulong len = webkit_dom_node_list_get_length(nl);
        gulong i;
        for(i = 0; i < len; i++) {
            WebKitDOMNode *item = webkit_dom_node_list_item(nl, i);
            if(!is_visible(h, item) || !WEBKIT_DOM_IS_HTML_INPUT_ELEMENT(item)) {
                continue;
            }
            WebKitDOMHTMLInputElement *input = WEBKIT_DOM_HTML_INPUT_ELEMENT(item);
            // Filter non-text input types.
            gchar *input_type = webkit_dom_html_input_element_get_input_type(input);
            if(input_type != NULL &&
                    *input_type != '\0' &&
                    g_strcmp0(input_type, "text") != 0 &&
                    g_strcmp0(input_type, "search") != 0) {
                g_free(input_type);
                continue;
            }
            g_free(input_type);
            WebKitDOMHTMLFormElement *form = webkit_dom_html_input_element_get_form(input);
            if(form == NULL) {
                continue;
            }
            // Filter non-get forms.
            gchar *method = webkit_dom_html_form_element_get_method(form);
            if(method != NULL && *method != '\0' && g_strcmp0(method, "get") != 0) {
                g_free(method);
                continue;
            }
            g_free(method);
            g_object_ref(item);
            ret = g_list_prepend(ret, item);
        }
    }
    g_list_free(docs);
    return ret;
}

// Selects all elements which may normally be clicked.
// 
// - Anchor elements
// - Input elements
// - Embed elements
// - Button elements
// - TextArea elements
// - Select elements
GList *
select_clickable(GHashTable *h, Exten *exten)
{
    const char* const tags[] = {
        "A",
        "INPUT",
        "EMBED",
        "BUTTON",
        "TEXTAREA",
        "SELECT",
        NULL};
    GList *ret = NULL;
    GList *docs = NULL;
    scan_documents(exten->document, &docs, exten);
    GList *l;
    for(l = docs; l != NULL; l = l->next) {
        guint i;
        for(i = 0; tags[i] != NULL; i++) {
            WebKitDOMNodeList *nl =
                webkit_dom_document_get_elements_by_tag_name(l->data, tags[i]);
            gulong len = webkit_dom_node_list_get_length(nl);
            gulong j;
            for(j = 0; j < len; j++) {
                WebKitDOMNode *item = webkit_dom_node_list_item(nl, j);
                if(!is_visible(h, item)) {
                    continue;
                }
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
select_links(GHashTable *h, Exten *exten)
{
    GList *ret = NULL;
    GList *docs = NULL;
    scan_documents(exten->document, &docs, exten);
    GList *l;
    for(l = docs; l != NULL; l = l->next) {
        WebKitDOMHTMLCollection *coll = webkit_dom_document_get_links(l->data);
        gulong len = webkit_dom_html_collection_get_length(coll);
        gulong i;
        for(i = 0; i < len; i++) {
            WebKitDOMNode *item = webkit_dom_html_collection_item(coll, i);
            if(!is_visible(h, item)) {
                continue;
            }
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
    GHashTable *ht = g_hash_table_new_full(NULL, NULL, NULL, g_free);
    GList *nodes = ns(ht, exten);
    guint len = g_list_length(nodes);
    gchar **hints_texts = get_hints_texts(len, exten, &err);
    if(err != NULL) {
        printf("Failed to get hints texts: %s\n", err->message);
        g_error_free(err);
        g_hash_table_unref(ht);
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
        Point *point = dom_get_absolute_position(ht, l->data);
        gchar *style = g_strdup_printf("left:%fpx;top:%fpx",
                point->x,
                point->y);
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
    g_hash_table_unref(ht);
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
                    filter_hints_mode("", exten);
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
