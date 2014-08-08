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
    "encoding/json"
    "flag"
)

/* TODO
    - replace log.fatal with error page
    - ensure access is properly restricted
    - make sure bubble entries exist before replacing [[] []] with link
    - watch github repo and update
    - robustify posts functionality
    - clean up js bubbles so they follow user as they scroll

*/

// bloke should be launched from the sites root
// should be installed in gopath/src/github/ebuchman/bloke...
var SiteRoot = "."
var GoPath = os.Getenv("GOPATH")
var BlokePath = GoPath + "/src/github.com/ebuchman/bloke" // is there a nicer way to get this?

var InitSite = flag.String("init", "", "path to new site dir")



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

// main routing function
func (g *Globals) handleIndex(w http.ResponseWriter, r *http.Request){
        log.Println("handle Index", r.URL.Path)
        /* url is either
            /                                   home page (recent blog posts)  
            /posts/Date-PostName                a specific blog post
            /ProjectName                        a particular project page
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
    //path_split := strings.Split(r.URL.Path[1:], "/")
    // path_split [0] should be "bubble"
    //bubble := path_split[1]
    b, err := ioutil.ReadFile(path.Join(SiteRoot, r.URL.Path[1:]))
    if err != nil{
        log.Fatal("error on bubble ", r.URL.Path[1:], err)
    }
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

// serve a single html page
func servePage(w http.ResponseWriter, r *http.Request){
    if !strings.Contains(r.URL.Path, "."){
        http.ServeFile(w, r, SiteRoot+"/sites/main.html") //+r.URL.Path[1:])
        //s.handleIndex(w, r)
    }else{
        http.ServeFile(w, r, SiteRoot+"/sites/main.html") //+r.URL.Path[1:])
    }
}

// compile list of pages and prepare Globals struct (mostly for filling in the nav bar with pages links)
// in future, write everything out to static .html files for serving later (so we don't have to render template each time)
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

// compile list of posts and fill in Globals struct
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

// main server startup function
// compile lists of pages and posts and prepare globals struct
func (g *Globals) AssembleSite(){
    // go through pages and posts and entries
    //RenderTemplateToFile("page", "main", g)
    g.AssemblePages()
    g.AssemblePosts()
    //g.NumProjects = len(g.Projects)
    log.Println(g)
}

type ConfigType struct{
    SiteName string `json:"site_name"`
    Email string `json:"email"`
    Site string `json:"site"`
}

type Globals struct{
    Projects []string // names of projects
    SubProjects map[string][]string // subprojects are either list of strings or empty. these generate the dropdowns
    Posts map[string]map[string]map[string][]string // year, month, day, title
    RecentPosts [][]string // [](title, date_name)
    Text string
    Title string

    Config ConfigType
}

func (g * Globals) LoadConfig(){
    file, e := ioutil.ReadFile(path.Join(SiteRoot, "config.json"))
    if e != nil{
        log.Fatal("no config", e)
    }
    log.Println("file", string(file))
    var c ConfigType
    json.Unmarshal(file, &c)
    log.Println(c)
    g.Config = c
    log.Println("config", g.Config)
}

func CreateNewSite(){
    os.Mkdir(*InitSite, 0777) // apparently 6s aren't sufficient here?
    os.Mkdir(path.Join(*InitSite, "bubbles"), 0666)
    os.MkdirAll(path.Join(*InitSite, "imgs"), 0666)
    os.MkdirAll(path.Join(*InitSite, "pages"), 0666)
    os.MkdirAll(path.Join(*InitSite, "posts"), 0666)

    f, err := os.Create(path.Join(*InitSite, "config.json"))
    defer f.Close()
    if err != nil{
     log.Println("Could not create config file:", err)
    }else{
        /*
        c := ConfigType{SiteName: *InitSite}
        jc, _ := json.Marshal(c)
        enc := json.NewEncoder(f)
        err := enc.Encode(jc)
        if err != nil{
            log.Fatal(err)
        }
        */ // why can't I write a clean config file?
        f.WriteString("{\n")
        f.WriteString("\t\"site_name\": \""+*InitSite+"\",\n")
        f.WriteString("\t\"email\": \"\",\n")
        f.WriteString("\t\"site\": \"\"\n")
        f.WriteString("}")

    }
}

func StartServer(){
    flag.Parse()
    
    if *InitSite != ""{
        CreateNewSite()
        os.Exit(0)
    }

    g := Globals{}
    g.LoadConfig()
    g.AssembleSite()

    http.HandleFunc("/", g.handleIndex) // main page
    http.HandleFunc("/imgs/", serveFile)
    http.HandleFunc("/assets/", serveFile) // static files
    http.HandleFunc("/bubbles/", g.ajaxResponse) // async bubbles

    http.ListenAndServe(":9099", nil)
}

func main(){
    StartServer()
}
