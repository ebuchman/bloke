package main

import (
    "github.com/russross/blackfriday"
    "net/http"
    "text/template"
    "strings"
    "log"
    "io/ioutil"
    "os"
    "path"
)

// bloke should be launched from the sites root
// should be installed in gopath/src/github/ebuchman/bloke...
var SiteRoot = "."
var GoPath = os.Getenv("GOPATH")
var BlokePath = GoPath + "/src/github.com/ebuchman/bloke"

//parse template files
var templates = template.Must(template.ParseFiles(BlokePath+"/views/page.html", BlokePath+"/views/nav.html"))

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

func renderTemplate(w http.ResponseWriter, tmpl string, p interface{}){
    //we already parsed the html templates
    err := templates.ExecuteTemplate(w, tmpl+".html", p)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}


func (g *Globals) handleIndex(w http.ResponseWriter, r *http.Request){
        log.Println("handle Index", r.URL.Path)
        if len(r.URL.Path[1:]) > 0{
            b, err := ioutil.ReadFile(SiteRoot+"/pages/"+r.URL.Path[1:]+".md")
            if err != nil{
                log.Fatal("error acessing data", err)
            }
            //g.Text = strings.Split(string(b), "\n\n")// string(b)
            g.Text = string(blackfriday.MarkdownCommon(b))
            log.Println(SiteRoot+"/pages/"+r.URL.Path[1:]+".md")
        } else {
            g.Text = ""
        }
        g.Title = r.URL.Path[1:]
        renderTemplate(w, "page", g)
}

// serve static files
func serveFile(w http.ResponseWriter, r *http.Request){
    // if img, load from SiteRoot
    // if js/css, load from BlokePath

    if !strings.Contains(r.URL.Path, "."){
        //s.handleIndex(w, r)
    }else{
        subs := strings.Split(r.URL.Path, ".")
        ext := subs[len(subs)-1]
        if ext == "js" || ext == "css"{
            http.ServeFile(w, r, path.Join(BlokePath, r.URL.Path[1:]))
        }else if ext == "png" || ext == "jpg"{
            http.ServeFile(w, r, path.Join(SiteRoot, r.URL.Path[1:]))
        }
    }
}

func servePage(w http.ResponseWriter, r *http.Request){
    if !strings.Contains(r.URL.Path, "."){
        http.ServeFile(w, r, SiteRoot+"/sites/main.html") //+r.URL.Path[1:])
        //s.handleIndex(w, r)
    }else{
        http.ServeFile(w, r, SiteRoot+"/sites/main.html") //+r.URL.Path[1:])
    }
}

func (g *Globals) AssembleSite(){
    // go through pages and posts and entries
    files, err := ioutil.ReadDir(SiteRoot+"/pages")
    if err != nil {
        log.Fatal("error reading pages")
    }
    g.SubProjects = make(map[string][]string)
    for _, f := range files {
        if !f.IsDir(){
            name := strings.Split(f.Name(), ".")[0]
            g.Projects = append(g.Projects, name)
            g.SubProjects[name] = []string{}
        } else{
            subfiles, err := ioutil.ReadDir(SiteRoot+"/pages/"+f.Name())
            if err != nil {
                log.Fatal("error reading sub pages")
            }
            var list []string
            for _, ff := range subfiles{
                name := strings.Split(ff.Name(), ".")[0]
                list = append(list, name)
            }
            g.Projects = append(g.Projects, f.Name())
            g.SubProjects[f.Name()] = list
        }
    }
    //RenderTemplateToFile("page", "main", g)
    g.NumProjects = len(g.Projects)
    log.Println(g)


}

type Globals struct{
    NumProjects int
    Projects []string // names of projects
    SubProjects map[string][]string // subprojects are either list of strings or empty. these generate the dropdowns
    Text string
    Title string
}

func StartServer(){

    g := Globals{}

    g.AssembleSite()

    // pages
    http.HandleFunc("/", g.handleIndex) // main page
    //http.HandleFunc("/", servePage) // main page
    http.HandleFunc("/imgs/", serveFile)
    http.HandleFunc("/assets/", serveFile) // static files

    // sockets
    //http.Handle("/chat_sock", websocket.Handler(g.chatSocketHandler))
    //http.Handle("/ethereum", websocket.Handler(g.ethereumSocketHandler))

    http.ListenAndServe(":9099", nil)
}

func main(){
    StartServer()
}
