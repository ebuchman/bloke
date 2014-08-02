package main

import (
    "github.com/russross/blackfriday"
    "net/http"
    "text/template"
    "strings"
    "log"
    "io/ioutil"
    "os"
)

var DataPath = "../ccblog"

//paths
//var templates = template.Must(template.ParseFiles(Home+"views/index.html", Home+"views/config.html", Home+"views/chat.html"))
var templates = template.Must(template.ParseFiles("./views/page.html", "./views/nav.html"))

func RenderTemplateToFile(tmpl, save_file string, p interface{}){
    //we already parsed the html templates
    f, err := os.Create(DataPath+"sites/"+save_file+".html")
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
        g.Title = "balls"
        if len(r.URL.Path[1:]) > 0{
            b, err := ioutil.ReadFile(DataPath+"/pages/"+r.URL.Path[1:]+".md")
            if err != nil{
                log.Fatal("error acessing data", err)
            }
            //g.Text = strings.Split(string(b), "\n\n")// string(b)
            g.Text = string(blackfriday.MarkdownCommon(b))
        } else {
            g.Text = ""
        }
        g.Title = r.URL.Path[1:]
        log.Println(g.Projects)
        log.Println(g.SubProjects)
        renderTemplate(w, "page", g)
}

// serve static files
func serveFile(w http.ResponseWriter, r *http.Request){
    // if img, load from datapath
    // if js/css, load from repo

    if !strings.Contains(r.URL.Path, "."){
        //s.handleIndex(w, r)
    }else{
        http.ServeFile(w, r, r.URL.Path[1:])
    }
}

func servePage(w http.ResponseWriter, r *http.Request){
    if !strings.Contains(r.URL.Path, "."){
        http.ServeFile(w, r, DataPath+"/sites/main.html") //+r.URL.Path[1:])
        //s.handleIndex(w, r)
    }else{
        http.ServeFile(w, r, DataPath+"/sites/main.html") //+r.URL.Path[1:])
    }
}

func (g *Globals) AssembleSite(){
    // go through pages and posts and entries
    files, err := ioutil.ReadDir(DataPath+"/pages")
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
            subfiles, err := ioutil.ReadDir(DataPath+"/pages/"+f.Name())
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
    http.HandleFunc("/assets/", serveFile) // static files

    // sockets
    //http.Handle("/chat_sock", websocket.Handler(g.chatSocketHandler))
    //http.Handle("/ethereum", websocket.Handler(g.ethereumSocketHandler))

    http.ListenAndServe(":9099", nil)
}

func main(){
    StartServer()
}
