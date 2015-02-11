package golem

import (
	"fmt"
	"reflect"
)

// A cfg is one of the below config levels, which all implement basic
// boilerplate accessor methods.
type cfg interface {
	typeOf(cfg string) (reflect.Kind, error)
	get(cfg string) interface{}
	set(cfg string, v interface{})
	getSettings(t reflect.Kind) []string
}

// globalCfg contains the configuration for the entire golem session.
type globalCfg struct {
	*windowCfg

	// A map from sanitized keystring (i.e. parsed and stringified again) to
	// a struct of uri and name.
	quickmarks   map[string]uriEntry
	hasQuickmark map[string]bool

	bookmarks  []uriEntry
	isBookmark map[string]bool

	searchEngines *searchEngines

	profile      string
	pdfjsEnabled bool
	maxHistLen   uint
}

// typeOf gets the reflect.Kind associated with the given setting.
func (c *globalCfg) typeOf(cfg string) (reflect.Kind, error) {
	switch cfg {
	case "profile":
		return reflect.String, nil
	case "pdf.js-enabled":
		return reflect.Bool, nil
	case "max-history-length":
		return reflect.Uint, nil
	default:
		return c.windowCfg.typeOf(cfg)
	}
}

// get retrieves a setting with the given key.
func (c *globalCfg) get(cfg string) interface{} {
	switch cfg {
	case "profile":
		return c.profile
	case "pdf.js-enabled":
		return c.pdfjsEnabled
	case "max-history-length":
		return c.maxHistLen
	default:
		return c.windowCfg.get(cfg)
	}
}

// set sets a setting with the given key to the given value.
func (c *globalCfg) set(cfg string, v interface{}) {
	switch cfg {
	case "profile":
		c.profile = v.(string)
	case "pdf.js-enabled":
		c.pdfjsEnabled = v.(bool)
	case "max-history-length":
		c.maxHistLen = v.(uint)
	default:
		c.windowCfg.set(cfg, v)
	}
}

// getSettings retrieves the names of all settings of the given kind.
func (c *globalCfg) getSettings(t reflect.Kind) []string {
	children := c.windowCfg.getSettings(t)
	switch t {
	case reflect.String:
		return append(children, "profile")
	case reflect.Bool:
		return append(children, "pdf.js-enabled")
	case reflect.Uint:
		return append(children, "max-history-length")
	default:
		return children
	}
}

// windowCfg contains the configuration for a single window.
type windowCfg struct {
	*tabCfg

	newTabPage string
}

// typeOf gets the reflect.Kind associated with the given setting.
func (c *windowCfg) typeOf(cfg string) (reflect.Kind, error) {
	switch cfg {
	case "new-tab-page":
		return reflect.String, nil
	default:
		return c.tabCfg.typeOf(cfg)
	}
}

// get retrieves a setting with the given key.
func (c *windowCfg) get(cfg string) interface{} {
	switch cfg {
	case "new-tab-page":
		return c.newTabPage
	default:
		return c.tabCfg.get(cfg)
	}
}

// set sets a setting with the given key to the given value.
func (c *windowCfg) set(cfg string, v interface{}) {
	switch cfg {
	case "new-tab-page":
		c.newTabPage = v.(string)
	default:
		c.tabCfg.set(cfg, v)
	}
}

// getSettings retrieves the names of all settings of the given kind.
func (c *windowCfg) getSettings(t reflect.Kind) []string {
	children := c.tabCfg.getSettings(t)
	switch t {
	case reflect.String:
		return append(children, "new-tab-page")
	default:
		return children
	}
}

// clone clones the config.
func (c *windowCfg) clone() *windowCfg {
	return &windowCfg{
		c.tabCfg.clone(),
		c.newTabPage,
	}
}

// tabCfg contains the configuration for a single tab.
type tabCfg struct {
	scrollDelta uint
}

// typeOf gets the reflect.Kind associated with the given setting.
func (c *tabCfg) typeOf(cfg string) (reflect.Kind, error) {
	switch cfg {
	case "scroll-delta":
		return reflect.Uint, nil
	default:
		return reflect.Invalid, fmt.Errorf("Unknown setting: %s", cfg)
	}
}

// get retrieves a setting with the given key.
func (c *tabCfg) get(cfg string) interface{} {
	switch cfg {
	case "scroll-delta":
		return c.scrollDelta
	default:
		panic(fmt.Sprintf("Unknown setting: %s", cfg))
	}
}

// set sets a setting with the given key to the given value.
func (c *tabCfg) set(cfg string, v interface{}) {
	switch cfg {
	case "scroll-delta":
		c.scrollDelta = v.(uint)
	default:
		panic(fmt.Sprintf("Unknown setting: %s", cfg))
	}
}

// getSettings retrieves the names of all settings of the given kind.
func (c *tabCfg) getSettings(t reflect.Kind) []string {
	switch t {
	case reflect.Uint:
		return []string{"scroll-delta"}
	default:
		return nil
	}
}

// clone clones the config.
func (c *tabCfg) clone() *tabCfg {
	return &tabCfg{
		c.scrollDelta,
	}
}

// The defaultCfg is used when golem is started, and typically overwritten
// with rc commands.
var defaultCfg *globalCfg

// init initializes the default config.
func init() {
	_, err := Asset("srv/pdf.js/enabled")
	defaultCfg = &globalCfg{
		&windowCfg{
			&tabCfg{
				40,
			},
			"http://github.com/tkerber/golem",
		},
		make(map[string]uriEntry, 20),
		make(map[string]bool, 20),
		make([]uriEntry, 0, 100),
		make(map[string]bool, 100),
		&searchEngines{make(map[string]*searchEngine, 10), nil},
		"default",
		err == nil,
		500,
	}
}
