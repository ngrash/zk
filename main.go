package main

import (
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
)

var addr = flag.String("addr", "localhost:8000", "Network address to listen on")

type Page struct {
	Title string
	Body  []byte
}

func (p *Page) save() error {
	filename := dataDir + p.Title + ".txt"
	return ioutil.WriteFile(filename, p.Body, 0600)
}

func loadPage(title string) (*Page, error) {
	filename := dataDir + title + ".txt"
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return &Page{Title: title, Body: body}, nil
}

const dataDir = "data/"
const tmplDir = "tmpl/"
const pagePrefix = "/p/"
const editPrefix = "/e/"
const savePrefix = "/s/"

var validPath = regexp.MustCompile(
	fmt.Sprintf("^(%s|%s|%s)([a-zA-Z0-9]+)$", pagePrefix, editPrefix, savePrefix))

var pageLink = regexp.MustCompile("@[a-zA-Z0-9]+")

var templates = template.Must(template.ParseGlob(tmplDir + "*.html"))

func main() {
	flag.Parse()

	http.HandleFunc(pagePrefix, makeHandler(handlePageRequest))
	http.HandleFunc(editPrefix, makeHandler(handleEditRequest))
	http.HandleFunc(savePrefix, makeHandler(handleSaveRequest))
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func handlePageRequest(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		http.Redirect(w, r, editPrefix+title, http.StatusFound)
		return
	}

	body := pageLink.ReplaceAllFunc(p.Body, func(match []byte) []byte {
		linkTitle := string(match[1:]) // cut off leading @
		return []byte(fmt.Sprintf(`<a href="%s">%s</a>`, linkTitle, linkTitle))
	})

	html := strings.ReplaceAll(string(body), "\n", "<br>")

	m := struct {
		Title string
		Body  template.HTML
	}{title, template.HTML(html)}

	renderTemplate(w, "page.html", m)
}

func handleEditRequest(w http.ResponseWriter, r *http.Request, title string) {
	action := "Editing"
	p, err := loadPage(title)
	if err != nil {
		action = "Creating"
		p = &Page{Title: title}
	}

	m := struct {
		Action string
		Title  string
		Body   template.HTML
	}{action, p.Title, template.HTML(p.Body)}

	renderTemplate(w, "edit.html", m)
}

func handleSaveRequest(w http.ResponseWriter, r *http.Request, title string) {
	body := r.FormValue("body")
	p := &Page{Title: title, Body: []byte(body)}
	if err := p.save(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, pagePrefix+title, http.StatusFound)
}

func getTitle(r *http.Request) (string, error) {
	m := validPath.FindStringSubmatch(r.URL.Path)
	if m == nil {
		return "", errors.New("invalid page title")
	}
	return m[2], nil
}

func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		title, err := getTitle(r)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		fn(w, r, title)
	}
}

func renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	err := templates.ExecuteTemplate(w, name, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
