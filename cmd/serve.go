package cmd

import (
	"flag"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/svandragt/park/internal/park"
)

const serveCSS = `
body { font-family: system-ui, sans-serif; max-width: 900px; margin: 2rem auto; padding: 0 1rem; color: #222; }
h1 { font-size: 1.4rem; margin-bottom: 1rem; }
nav { display: flex; align-items: center; gap: 0.75rem; flex-wrap: wrap; margin-bottom: 1.5rem; }
nav a { text-decoration: none; color: #555; }
nav a.active { font-weight: bold; color: #000; border-bottom: 2px solid #000; }
.search-form { display: flex; gap: 0.4rem; margin-left: auto; }
.search-form input { border: 1px solid #ccc; border-radius: 4px; padding: 3px 8px; font-size: 0.85rem; width: 180px; }
.search-form button { border: 1px solid #ccc; border-radius: 4px; padding: 3px 8px; font-size: 0.85rem; background: #f5f5f5; cursor: pointer; }
.item { border: 1px solid #ddd; border-radius: 6px; padding: 1rem; margin-bottom: 1rem; }
.item-header { display: flex; align-items: baseline; gap: 0.75rem; }
.item-id { color: #888; font-size: 0.85rem; }
.item-name { font-weight: bold; font-size: 1rem; }
.item-name a { text-decoration: none; color: inherit; }
.item-name a:hover { text-decoration: underline; }
.badge { font-size: 0.75rem; padding: 2px 6px; border-radius: 4px; background: #eee; color: #555; }
.badge a { color: inherit; text-decoration: none; }
.badge a:hover { text-decoration: underline; }
.badge.active { background: #d4edda; color: #155724; }
.badge.resolved { background: #cce5ff; color: #004085; }
.badge.archived { background: #e2e3e5; color: #383d41; }
.item-desc { color: #555; font-size: 0.9rem; margin-top: 0.3rem; }
.item-meta { font-size: 0.8rem; color: #888; margin-top: 0.4rem; }
.detail-body { white-space: pre-wrap; background: #f8f8f8; padding: 0.75rem; border-radius: 4px; margin-top: 0.5rem; }
.field { margin-top: 0.6rem; font-size: 0.9rem; }
.field-label { font-weight: bold; color: #555; }
a.back { display: inline-block; margin-bottom: 1.5rem; color: #555; text-decoration: none; }
a.back:hover { text-decoration: underline; }
.repo-link { margin-top: 1.5rem; font-size: 0.9rem; }
.empty { color: #888; }
`

func RunServe(store *park.Store, args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	addr := fs.String("addr", "127.0.0.1:7654", "address to listen on")
	if err := fs.Parse(args); err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/item/", func(w http.ResponseWriter, r *http.Request) {
		idStr := strings.TrimPrefix(r.URL.Path, "/item/")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		it, err := store.Get(id)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		serveDetail(w, it)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		q := r.URL.Query()
		status := q.Get("status")
		if status == "" {
			status = "active"
		}
		filterStatus := status
		if filterStatus == "all" {
			filterStatus = ""
		}
		tag := q.Get("tag")
		typ := q.Get("type")
		remote := q.Get("remote")
		search := q.Get("q")

		f := park.ListFilter{Status: filterStatus, Tag: tag, Type: typ, Remote: remote}
		var items []park.Item
		var err error
		if search != "" {
			items, err = store.Search(search, f)
		} else {
			items, err = store.List(f)
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		serveIndex(w, items, status, tag, typ, remote, search)
	})

	fmt.Printf("park web UI: http://%s\n", *addr)
	return http.ListenAndServe(*addr, mux)
}

func htmlPage(title, body string) string {
	return `<!DOCTYPE html><html><head><meta charset="utf-8"><title>` + html.EscapeString(title) + `</title>` +
		`<style>` + serveCSS + `</style></head><body>` + body + `</body></html>`
}

func indexURL(status, tag, typ, remote, search string) string {
	v := url.Values{}
	v.Set("status", status)
	if tag != "" {
		v.Set("tag", tag)
	}
	if typ != "" {
		v.Set("type", typ)
	}
	if remote != "" {
		v.Set("remote", remote)
	}
	if search != "" {
		v.Set("q", search)
	}
	return "/?" + v.Encode()
}

func typeBadge(typ, status string) string {
	if typ == "" {
		return ""
	}
	href := indexURL(status, "", typ, "", "")
	return `<span class="badge"><a href="` + html.EscapeString(href) + `">` + html.EscapeString(typ) + `</a></span>`
}

func tagBadges(tags, status string) string {
	if tags == "" {
		return ""
	}
	var parts []string
	for _, t := range strings.Split(tags, ",") {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		href := indexURL(status, t, "", "", "")
		parts = append(parts, `<span class="badge"><a href="`+html.EscapeString(href)+`">`+html.EscapeString(t)+`</a></span>`)
	}
	return strings.Join(parts, " ")
}

func serveIndex(w http.ResponseWriter, items []park.Item, status, activeTag, activeType, activeRemote, activeSearch string) {
	statuses := []string{"active", "resolved", "archived", "all"}
	nav := `<nav>`
	for _, s := range statuses {
		cls := ""
		if s == status {
			cls = ` class="active"`
		}
		nav += `<a href="` + indexURL(s, activeTag, activeType, activeRemote, activeSearch) + `"` + cls + `>` + s + `</a>`
	}
	if activeTag != "" {
		nav += ` <span class="badge">tag: ` + html.EscapeString(activeTag) + ` <a href="` + indexURL(status, "", activeType, activeRemote, activeSearch) + `">×</a></span>`
	}
	if activeType != "" {
		nav += ` <span class="badge">type: ` + html.EscapeString(activeType) + ` <a href="` + indexURL(status, activeTag, "", activeRemote, activeSearch) + `">×</a></span>`
	}
	if activeRemote != "" {
		label := strings.TrimPrefix(activeRemote, "https://")
		label = strings.TrimPrefix(label, "git@")
		label = strings.TrimSuffix(label, ".git")
		nav += ` <span class="badge">repo: ` + html.EscapeString(label) + ` <a href="` + indexURL(status, activeTag, activeType, "", activeSearch) + `">×</a></span>`
	}

	// search box
	searchVal := html.EscapeString(activeSearch)
	nav += `<form class="search-form" method="get" action="/">` +
		`<input type="hidden" name="status" value="` + html.EscapeString(status) + `">` +
		`<input type="search" name="q" value="` + searchVal + `" placeholder="search…">` +
		`<button type="submit">go</button>` +
		`</form>`
	nav += `</nav>`

	var b strings.Builder
	b.WriteString(`<h1>parked items</h1>`)
	b.WriteString(nav)

	if len(items) == 0 {
		b.WriteString(`<p class="empty">no items</p>`)
	}
	for _, it := range items {
		b.WriteString(`<div class="item">`)
		b.WriteString(`<div class="item-header">`)
		b.WriteString(`<span class="item-id">#` + fmt.Sprintf("%d", it.ID) + `</span>`)
		b.WriteString(`<span class="item-name"><a href="/item/` + fmt.Sprintf("%d", it.ID) + `">` + html.EscapeString(it.Name) + `</a></span>`)
		b.WriteString(`<span class="badge ` + it.Status + `">` + it.Status + `</span>`)
		b.WriteString(typeBadge(it.Type, status))
		b.WriteString(`</div>`)
		if it.Description != "" {
			b.WriteString(`<div class="item-desc">` + html.EscapeString(it.Description) + `</div>`)
		}
		meta := []string{}
		if it.GitRemote != "" {
			label := strings.TrimPrefix(it.GitRemote, "https://")
			label = strings.TrimPrefix(label, "git@")
			label = strings.TrimSuffix(label, ".git")
			meta = append(meta, html.EscapeString(label))
		}
		if it.Branch != "" {
			meta = append(meta, html.EscapeString(it.Branch))
		}
		if len(meta) > 0 {
			b.WriteString(`<div class="item-meta">` + strings.Join(meta, " · ") + `</div>`)
		}
		if it.Tags != "" {
			b.WriteString(`<div class="item-meta">` + tagBadges(it.Tags, status) + `</div>`)
		}
		b.WriteString(`</div>`)
	}

	fmt.Fprint(w, htmlPage("park", b.String()))
}

func serveDetail(w http.ResponseWriter, it *park.Item) {
	var b strings.Builder
	b.WriteString(`<a class="back" href="/">← back</a>`)
	b.WriteString(`<div class="item">`)
	b.WriteString(`<div class="item-header">`)
	b.WriteString(`<span class="item-id">#` + fmt.Sprintf("%d", it.ID) + `</span>`)
	b.WriteString(`<span class="item-name">` + html.EscapeString(it.Name) + `</span>`)
	b.WriteString(`<span class="badge ` + it.Status + `">` + it.Status + `</span>`)
	b.WriteString(typeBadge(it.Type, it.Status))
	b.WriteString(`</div>`)

	if it.Description != "" {
		b.WriteString(`<div class="field"><span class="field-label">Description:</span> ` + html.EscapeString(it.Description) + `</div>`)
	}
	if it.Body != "" {
		b.WriteString(`<div class="field"><span class="field-label">Body:</span><div class="detail-body">` + html.EscapeString(it.Body) + `</div></div>`)
	}
	if it.Why != "" {
		b.WriteString(`<div class="field"><span class="field-label">Why:</span> ` + html.EscapeString(it.Why) + `</div>`)
	}
	if it.HowToApply != "" {
		b.WriteString(`<div class="field"><span class="field-label">How to apply:</span> ` + html.EscapeString(it.HowToApply) + `</div>`)
	}
	if it.Tags != "" {
		b.WriteString(`<div class="field"><span class="field-label">Tags:</span> ` + tagBadges(it.Tags, it.Status) + `</div>`)
	}
	if it.GitRemote != "" {
		b.WriteString(`<div class="field"><span class="field-label">Repo:</span> ` + html.EscapeString(it.GitRemote) + ` · <span class="field-label">Branch:</span> ` + html.EscapeString(it.Branch) + `</div>`)
	}
	b.WriteString(`<div class="field item-meta">Device: ` + html.EscapeString(it.Device) + ` · Parked: ` + it.CreatedAt.Format("2006-01-02 15:04") + `</div>`)
	b.WriteString(`</div>`)

	if it.GitRemote != "" {
		repoURL := indexURL("all", "", "", it.GitRemote, "")
		b.WriteString(`<div class="repo-link"><a href="` + html.EscapeString(repoURL) + `">All items in this repo →</a></div>`)
	}

	fmt.Fprint(w, htmlPage("park — "+it.Name, b.String()))
}
