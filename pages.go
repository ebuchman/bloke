package main 

import (
    "github.com/russross/blackfriday" // parsing markdown
    "net/http"
    "text/template"
    "log"
    "os"
    "regexp"
    "path"
)


//parse template files
var templates = template.Must(template.ParseFiles(BlokePath+"/views/page.html", BlokePath+"/views/nav.html", BlokePath+"/views/footer.html", BlokePath+"/views/bubbles.html"))

// bring a template to life!
func renderTemplate(w http.ResponseWriter, tmpl string, p interface{}){
    //we already parsed the html templates
    err := templates.ExecuteTemplate(w, tmpl+".html", p)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}

// error function
func (g *Globals) errorPage(w http.ResponseWriter, err error){
    g.Title = "Error"
    g.Text = err.Error()
    renderTemplate(w, "page", g)
}

// parse and replace for bubbles and markdown to js/html
// takes the raw txt.md bytes
// creates new bubble entries if they are referenced but don't exist
func (g *Globals) ParseBubbles(s []byte) string{
    r, _ := regexp.Compile(`\[\[(.+?)\] \[(.+?)\]\]?`)
    s = blackfriday.MarkdownCommon(s)

    // get all matches, check if they exist, add them if not...
    for _, match := range r.FindAllStringSubmatch(string(s), -1){
        name := match[2]
        _, err := os.Stat(path.Join("bubbles", name+".md"))
        if err != nil{
            f, err := os.Create(path.Join("bubbles", name+".md"))
            if err != nil{
                log.Println("could not create new bubble file")
            } else{
                f.WriteString("This bubble hasn't been written yet! You can help us write it by submitting issues or pull requests at [our github repo!]("+g.Config.Repo+")")
            }
        }
    }

    return r.ReplaceAllString(string(s), `<a href="#/" onClick="get_entry_data('$2')">$1</a>`)
}



func RenderTemplateToFile(tmpl, save_file string, p interface{}){
    //we already parsed the html templates
    f, err := os.Create(SiteRoot+"/sites/"+save_file+".html")
    if err != nil{
        log.Fatal("err opening file:", err)
    }
    err = templates.ExecuteTemplate(f, tmpl+".html", p)
    if err != nil {
        log.Fatal("err writing template to file", err)
    }
}
