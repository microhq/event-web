package handler

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"sort"
	"time"

	"github.com/gorilla/mux"
	"github.com/yosssi/ace"
	"golang.org/x/net/context"

	"github.com/micro/event-srv/proto/event"
)

var (
	templateDir = "templates"
	opts        *ace.Options

	EventClient event.EventClient
)

func init() {
	opts = ace.InitializeOptions(nil)
	opts.BaseDir = templateDir
	opts.DynamicReload = true
	opts.FuncMap = template.FuncMap{
		"TimeAgo": func(t int64) string {
			return timeAgo(t)
		},
		"Timestamp": func(t int64) string {
			return time.Unix(t, 0).Format(time.RFC822)
		},
		"Colour": func(s string) string {
			return colour(s)
		},
	}
}

func render(w http.ResponseWriter, r *http.Request, tmpl string, data map[string]interface{}) {
	basePath := hostPath(r)

	opts.FuncMap["URL"] = func(path string) string {
		return filepath.Join(basePath, path)
	}

	tpl, err := ace.Load("layout", tmpl, opts)
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/", 302)
		return
	}

	if err := tpl.Execute(w, data); err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/", 302)
	}
}

// The index page
func Index(w http.ResponseWriter, r *http.Request) {
	rsp, err := EventClient.Search(context.TODO(), &event.SearchRequest{
		Reverse: true,
	})
	if err != nil {
		http.Redirect(w, r, "/", 302)
		return
	}

	sort.Sort(sortedRecords{rsp.Records})

	render(w, r, "index", map[string]interface{}{
		"Latest": rsp.Records,
	})
}

func Latest(w http.ResponseWriter, r *http.Request) {
	rsp, err := EventClient.Search(context.TODO(), &event.SearchRequest{
		Reverse: true,
	})
	if err != nil {
		http.Redirect(w, r, "/", 302)
		return
	}

	sort.Sort(sortedRecords{rsp.Records})

	render(w, r, "latest", map[string]interface{}{
		"Latest": rsp.Records,
	})
}

func Search(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		r.ParseForm()
		id := r.Form.Get("id")

		if len(id) > 0 {
			http.Redirect(w, r, filepath.Join(hostPath(r), "event/"+id), 302)
			return
		}

		rid := r.Form.Get("rid")
		typ := r.Form.Get("type")

		if len(rid) == 0 && len(typ) == 0 {
			http.Redirect(w, r, filepath.Join(hostPath(r), "search"), 302)
			return
		}

		rsp, err := EventClient.Search(context.TODO(), &event.SearchRequest{
			Id:      rid,
			Type:    typ,
			Reverse: true,
		})
		if err != nil {
			http.Redirect(w, r, filepath.Join(hostPath(r), "search"), 302)
			return
		}

		query := "ID: " + rid + " Type: " + typ

		render(w, r, "results", map[string]interface{}{
			"Query":   query,
			"Results": rsp.Records,
		})
		return
	}
	render(w, r, "search", map[string]interface{}{})
}

func Event(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if len(id) == 0 {
		http.Redirect(w, r, "/", 302)
		return
	}
	// TODO: limit/offset
	rsp, err := EventClient.Read(context.TODO(), &event.ReadRequest{
		Id: id,
	})
	if err != nil {
		http.Redirect(w, r, "/", 302)
		return
	}

	render(w, r, "event", map[string]interface{}{
		"Id":     id,
		"Record": rsp.Record,
	})
}
