package main

import (
    "github.com/russross/blackfriday" // parsing markdown
    "net/http"
    "text/template"
    "strings"
    "log"
    "io/ioutil"
    "os"
    "path"
    "regexp"
    "fmt"
)

// bloke should be launched from the sites root
// should be installed in gopath/src/github/ebuchman/bloke...
var SiteRoot = "."
var GoPath = os.Getenv("GOPATH")
var BlokePath = GoPath + "/src/github.com/ebuchman/bloke"

//parse template files
var templates = template.Must(template.ParseFiles(BlokePath+"/views/page.html", BlokePath+"/views/nav.html", BlokePath+"/views/footer.html", BlokePath+"/views/bubbles.html"))

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

// get name from blogpost url
func GetNameFromPost(s string) string{
       date_name := strings.Split(strings.Split(s, ".")[0], "-")
       title := date_name[3]
       return title
}

// parse and replace for bubbles and markdown to js/html
// takes the raw txt.md bytes
func DataTransform(s []byte) string{
    r, _ := regexp.Compile(`\[\[(.+?)\] \[(.+?)\]\]?`)
    s = blackfriday.MarkdownCommon(s)
    return r.ReplaceAllString(string(s), `<a href="#/" onClick="get_entry_data('$2')">$1</a>`)
}

func (g *Globals) handleIndex(w http.ResponseWriter, r *http.Request){
        log.Println("handle Index", r.URL.Path)
        /* url is either
            /
            /posts/Date-PostName
            /ProjectName
        */ 
        if len(r.URL.Path[1:]) > 0{
            path_elements := strings.Split(r.URL.Path[1:], "/")
            log.Println(path_elements)
            // posts
            if path_elements[0] == "posts"{
                b, err := ioutil.ReadFile(path.Join(SiteRoot,r.URL.Path[1:]))
                if err != nil{
                    log.Fatal("error acessing data", err)
                }
                g.Text = DataTransform(b) //string(blackfriday.MarkdownCommon(b))
                g.Title = GetNameFromPost(r.URL.Path[1:])
            // pages
            }else{
                b, err := ioutil.ReadFile(SiteRoot+"/pages/"+r.URL.Path[1:]+".md")
                if err != nil{
                    log.Fatal("error acessing data", err)
                }
                g.Text = DataTransform(b) //string(blackfriday.MarkdownCommon(b))
                log.Println(SiteRoot+"/pages/"+r.URL.Path[1:]+".md")
                split_path := strings.Split(r.URL.Path[1:], "/")
                g.Title = split_path[len(split_path)-1]
            }
        // home
        } else {
            b, err := ioutil.ReadFile(SiteRoot+"/posts/"+g.RecentPosts[0][1])
            if err != nil{
                log.Fatal("error opening post", err)
            }
            g.Text = string(blackfriday.MarkdownCommon(b))
            g.Title = g.RecentPosts[0][0]
        }
        renderTemplate(w, "page", g)
}


// ajax bubble response
func (g *Globals) ajaxResponse(w http.ResponseWriter, r *http.Request){
    path_split := strings.Split(r.URL.Path[1:], "/")
    // path_split [0] should be bubble
    bubble := path_split[1]
    b, err := ioutil.ReadFile(path.Join(SiteRoot, r.URL.Path[1:]))
    if err != nil{
        log.Fatal("error on bubble", r.URL.Path[1:], err)
    }
    g.Text = DataTransform(b) //string(blackfriday.MarkdownCommon(b))
    g.Title = bubble

    // return json
    fmt.Fprintf(w, DataTransform(b))


}

// serve static files (assets)
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


func (g *Globals) AssemblePages(){
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
}

func (g *Globals) AssemblePosts(){
    // posts dir should be fill with files like 2014-06-12-Name.md
    // No directories
    files, err := ioutil.ReadDir(SiteRoot+"/posts")
    if err != nil {
        log.Fatal("error reading pages")
    }
    for _, f := range files {
        if !f.IsDir(){
           date_name := strings.Split(strings.Split(f.Name(), ".")[0], "-")
           //year := date_name[0]
           //month := date_name[1]
           //day := date_name[2]
           title := date_name[3]
           g.RecentPosts = append(g.RecentPosts, []string{title, f.Name()})
        }
    }

}



func (g *Globals) AssembleSite(){
    // go through pages and posts and entries
    //RenderTemplateToFile("page", "main", g)
    g.AssemblePages()
    g.AssemblePosts()
    g.NumProjects = len(g.Projects)
    log.Println(g)


}

type Bubble struct{
    Title string
    Text string
}

type Globals struct{
    NumProjects int
    Projects []string // names of projects
    SubProjects map[string][]string // subprojects are either list of strings or empty. these generate the dropdowns
    Posts map[string]map[string]map[string][]string // year, month, day, title
    RecentPosts [][]string // [](title, date_name)
    Text string
    Title string

    Bubbles []Bubble
}

func StartServer(){

    g := Globals{}

    g.AssembleSite()

    // pages
    http.HandleFunc("/", g.handleIndex) // main page
    //http.HandleFunc("/", servePage) // main page
    http.HandleFunc("/imgs/", serveFile)
    http.HandleFunc("/assets/", serveFile) // static files
    http.HandleFunc("/bubbles/", g.ajaxResponse) // async bubbles

    // sockets
    //http.Handle("/chat_sock", websocket.Handler(g.chatSocketHandler))
    //http.Handle("/ethereum", websocket.Handler(g.ethereumSocketHandler))

    http.ListenAndServe(":9099", nil)
}

func main(){
    StartServer()
}
