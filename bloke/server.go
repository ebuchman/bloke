package bloke

import (
    "net/http"
    "strings"
    "log"
    "os"
    "path"
    "io/ioutil"
    "github.com/howeyc/fsnotify"
)

/* TODO
    - Glossary/bubbles page
    - better home/blog definition
    - pdfs in bubbles?
    - add tls redirects
    - clean up js bubbles so they follow user as they scroll
    - add "technical explanation" part to bubbles - + meta info?
*/

/*
    Global variables
        - Bloke path variable
        - NewBubbleString
*/

// bloke should be launched from the sites root
// should be installed in gopath/src/github/ebuchman/bloke...
var GoPath = os.Getenv("GOPATH")
var BlokePath = GoPath + "/src/github.com/ebuchman/bloke" // is there a nicer way to get this?

var NewBubbleString = "This bubble has not been written yet" // this will be changed to refer you to the github repo once it's configured :)

// main site struct
type Globals struct{
    Projects [][]string // list of pairs (url/filename, display name)
    SubProjects map[string][][]string // map from project-filename to list of pairs (subproject filenames, subproject displayname). these generate the dropdowns
    // all proj/subproj references should be with url_name!

    Posts map[string]map[string]map[string][]string // year, month, day, title
    RecentPosts [][]string // [](title, date_name)

    Config ConfigType // config struct loaded from config.json
    SiteRoot string // path to the site
    webhookSecret []byte // secret key for authenticating github webhook requests

    Close chan bool // close server channel

    mux *http.ServeMux // when many blokes are run behind one serve, give each a routing mux (instead of standalone server)
}

// new ServeMux. 
func (g *Globals) NewServeMux(){
    g.mux = http.NewServeMux()
    ApplyRouting(g.mux, g)
}

// serve over the mux
func (g *Globals) ServeHTTP(w http.ResponseWriter, r *http.Request){
    g.mux.ServeHTTP(w, r)
}

// launch a new live bloke
func LiveBloke(SitePath string) Globals{
    var g = Globals{}
    g.LoadConfig(SitePath) // load config
    g.AssembleSite() // assemble site composition from dir contents
    g.NewWatcher(SitePath) // watch the root directory
    g.NewServeMux() // creates g.mux and applies standard routing rules
    return g
}

// create new globals, copy over (eg. after git pull)
func (g *Globals) Refresh(){
    // TODO: close all old g things!
    *g = LiveBloke(g.SiteRoot)
}

// serve static files (assets: js, css)
func serveBlokeFile(w http.ResponseWriter, r *http.Request){
    if strings.Contains(r.URL.Path, "."){
        subs := strings.Split(r.URL.Path, ".")
        ext := subs[len(subs)-1]
        if ext == "js" || ext == "css"{
            p := path.Join(BlokePath, r.URL.Path[1:])
            _, err := os.Stat(p)
            if err == nil{
                http.ServeFile(w, r, p)
            }
        }
    }
}

// serve static files (imgs, files)
func (g *Globals) serveFile(w http.ResponseWriter, r *http.Request){
    if strings.Contains(r.URL.Path, "."){
        subs := strings.Split(r.URL.Path, ".")
        ext := subs[len(subs)-1]
        if ext == "png" || ext == "jpg" || ext == "pdf" {
            p := path.Join(g.SiteRoot, r.URL.Path[1:])
            _, err := os.Stat(p)
            if err == nil{
                http.ServeFile(w, r, p)
            }
        }
    }
}

// serve a single html page
func (g *Globals) servePage(w http.ResponseWriter, r *http.Request){
    if !strings.Contains(r.URL.Path, "."){
        http.ServeFile(w, r, g.SiteRoot+"/sites/main.html") //+r.URL.Path[1:])
        //s.handleIndex(w, r)
    }else{
        http.ServeFile(w, r, g.SiteRoot+"/sites/main.html") //+r.URL.Path[1:])
    }
}

// watch directory callback
// kind of messy since it fires soo much
func (g *Globals) WatchDirCallback(watcher *fsnotify.Watcher){
    for {
        select {
        case ev := <-watcher.Event:
            log.Println("event:", ev)
            if ev != nil{
                // only refresh if name is not hidden
                split := strings.Split(ev.Name, "/")
                name := split[len(split)-1]
                if !strings.HasPrefix(name, ".") && !strings.HasSuffix(name, "~"){
                    g.Refresh()            
                }
            }
        case err := <-watcher.Error:
            log.Println("error:", err)
        }
    }
    defer watcher.Close()
}

// recursive watch all directories
func (g *Globals) WatchDirs(watcher *fsnotify.Watcher, dir string){
    err := watcher.Watch(dir)
    if err != nil {
        log.Println("Could'nt watch dir", dir, err)
    }
    files := ReadDir(dir)
    if err != nil{
        log.Println("Couldn't read dir", dir, err)
    }
    
    for _, f := range files{
        if f.IsDir(){
            g.WatchDirs(watcher, path.Join(dir, f.Name()))
        }
    }
}

// create new watcher for directory
// cleanup/close!
func (g *Globals) NewWatcher(SiteRoot string){
    // set up new watcher (should only be used for local changes (otherwise use github))
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        log.Fatal(err)
    }
    // watch dir callback
    go g.WatchDirCallback(watcher)
    // recursive watch dirs. when an event fires in any dir, it'll hit the callback
    g.WatchDirs(watcher, SiteRoot)
}

// apply a set of routing rules to a mux using a bloke globals struct
func ApplyRouting(mux *http.ServeMux, g *Globals){
    mux.HandleFunc("/", g.handleIndex) // main page (/, /posts, /pages)
    mux.HandleFunc("/imgs/", g.serveFile) // static images (png, jpg)
    mux.HandleFunc("/files/", g.serveFile) // static documents (pdfs)
    mux.HandleFunc("/assets/", serveBlokeFile) // static js, css files
    mux.HandleFunc("/bubbles/", g.ajaxResponse) // async bubbles
    mux.HandleFunc("/git/", g.gitResponse) // github webhook
}

func RedirectTLS(w http.ResponseWriter, r *http.Request){
    host := r.Host
    log.Println("https://"+host)
    http.Redirect(w, r, "https://"+host, 301)
}

func RedirectServer(){
    mux := http.NewServeMux()
    mux.HandleFunc("/", RedirectTLS)
    http.ListenAndServe(":80", mux)
}


// start a http or https server listening on addr routing with the mux
func StartServer(addr string, mux *http.ServeMux, tls bool){
    if tls{
        _, err := ioutil.ReadDir("certs")
        if err != nil{
            log.Fatal("could not find certs dir", err)
        }
        err = http.ListenAndServeTLS(addr, "certs/ssl.crt", "certs/ssl.key", mux)
        if err != nil{
            log.Println("err on tls", err)
        }
    } else{
        err := http.ListenAndServe(addr,  mux)
        if err != nil{
            log.Println("err on http server", err)
        }
    }
}

// standalone server for running your own bloke
func StartBloke(addr, SiteRoot string, tls bool) {
    g := LiveBloke(SiteRoot)
    StartServer(addr, g.mux, tls)
}

