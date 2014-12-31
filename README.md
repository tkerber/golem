# golem web browser

## Current State

Go away. Nothing to see here. Yet.

## `go get` / `go generate`

Golem does not follow go conventions that `go generate` is to be run by the
author and generated code stored in the repository.

Reason for this is that platform the generated code includes a platform-
dependant shared object file.

As golem is not a library, the fact that it doesn't directly function with
`go get` is not a huge loss.

To install golem with the `go` command, instead execute:

    go get -d github.com/tkerber/golem
    go generate .../golem
    go install .../golem

Note that `go-bindata` is a dependency for golems `go generate` command.
Run `go get github.com/jteeuwen/go-bindata` to install it. The executable
is expected to with in the `PATH`.

### pdf.js

Golem has builtin support for using pdf.js to render pdf files. At the time
of writing, the pdf.js files are not in this repository in order to keep it
clean, and aren't built automatically.

See #8

## Naming

The name `golem` was chosen to remind people of what this browser should not
be: Slow and cumbersome.

## Design goals

* A keyboard driven, minimalistic browser.
* Powerful adblocking and noscript support.
  * Fully configurable.
  * Powerful whitelisting support.
    * In particular it should be possible to whitelist particular users
      on services such as youtube and twitch.
* Written in go.
* Capable of running multiple windows within one instance.
* One process per tab.
* *Everything* runs in it's own goroutine.
* No crashes. All goroutines should fail gracefully.
* In-browser pdf support.
* Multi-profile support.
* Vim-like keybindings.
* Fully reconfigurable keybindings.
* Public extensions API.
* Must exit quickly and gracefully on SIGTERM.
* Should exit cleanly on SIGKILL (in particular should not corrupt cookies).
* Browser should always be responsive. The UI goroutine should not handle
  any meaningful computation.
* Vertical and horizontal tab bar support
* Option to limit to one tab per window.
* Per-tab, per-window *and* global private browsing mode
  * Clear visual indicator
* Restore pages *option* upon restarting browser.
* Optional lazy tab loading

## Choice of layout engine

The options I considered for the choice of layout engine were Gecko and
Webkit. Having using webkit-based browsers much in the past, and having
seen many bugs attributed to them in fact being webkit bugs I was originally
inclined to use gecko.

Unfortunately gecko is (no longer) built for embedding into applications,
and it has no stable API to do so. As a result a gecko-based browser would
be very difficult to maintain.

I chose to use webkit instead, and to focus the energy on ensuring that
webkit can crash at most one tab at a time.

## Inspiration

* [DWB](http://portix.bitbucket.org/dwb/) - This browser was the immediate
  inspiration of this project.
* [Vimperator](http://www.vimperator.org/vimperator/) - A firefox plugin with
  similar goals.
* [Vim](http://www.vim.org/) - How keyboard control should be done.
