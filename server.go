package main

import (
    "net/http"
    "strings"
    "log"
    "io/ioutil"
    "os"
    "path"
    "fmt"
    "flag"
    "os/exec"
    "bytes"
    "strconv"
    "encoding/hex"
    "crypto/hmac"
    "crypto/sha1"
    "errors"
    "github.com/howeyc/fsnotify"
    "encoding/json"
)

/* TODO
    - Glossary/bubbles page
    - better home/blog definition
    - pdfs in bubbles?
    - add tls support
    - clean up js bubbles so they follow user as they scroll
    - add "technical explanation" part to bubbles - + meta info?
*/

/*
    Global variables
        - Path variables
        - Flags
        - NewBubbleString
*/

// bloke should be launched from the sites root
// should be installed in gopath/src/github/ebuchman/bloke...
var SiteRoot = "."
var GoPath = os.Getenv("GOPATH")
var BlokePath = GoPath + "/src/github.com/ebuchman/bloke" // is there a nicer way to get this?

// flag variables for cli
var InitSite = flag.String("init", "", "path to new site dir")
var ListenPort = flag.Int("port", 9099, "port to listen for incoming connections")
var WebHook = flag.Bool("webhook", false, "create a new secret token for use with github webhook")
var NewBubbles = flag.Bool("bubbles", false, "give all referenced bubbles a markdown file")

var NewBubbleString = "This bubble has not been written yet" // this will be changed to refer you to the github repo once it's configured :)

// config struct - corresponds to config.json
type ConfigType struct{
    SiteName string `json:"site_name"`
    Email string `json:"email"`
    Site string `json:"site"`
    Repo string `json:"github_repo"`
    Glossary string `json:"glossary_file"`
}

// meta info struct. read from json
type MetaInfoType struct {
    Title string `json:"title"`
}

// this guy gets passed to the go templates. simply has pointers to the globals and the page
// every request sees the same globals, but a different page
type ViewType struct{
    Page *PageType
    Globals *Globals
}

// info specific to the page requested by a client
type PageType struct{
    Name string //URL name of this page
    Text string // text of current page
    Title string // title of current page
    MetaInfo MetaInfoType // struct of meta info for current page

    // bloke flags (trigger specialized html/templating)
    IsGlossary bool
}

// main site struct
type Globals struct{
    Projects [][]string // list of pairs (url/filename, display name)
    SubProjects map[string][][]string // map from project-filename to list of pairs (subproject filenames, subproject displayname). these generate the dropdowns
    // all proj/subproj references should be with url_name!

    Posts map[string]map[string]map[string][]string // year, month, day, title
    RecentPosts [][]string // [](title, date_name)

    Config ConfigType // config struct loaded from config.json
    webhookSecret []byte // secret key for authenticating github webhook requests
}

/* 
    main routing function - validate, render, server pages
        url is either
            /                                   home page (recent blog posts)  
            /Date-PostName                      a specific blog post
            /ProjectName                        a particular project page
            /ProjectName/SubProjectName         a particular subproject page
*/ 
func (g *Globals) handleIndex(w http.ResponseWriter, r *http.Request){
        log.Println("handle Index", r.URL.Path)
        page := new(PageType)
        // is URL is empty, serve main page, else validate URL and LoadPage
        if len(r.URL.Path[1:]) > 0{
            path_elements := strings.Split(r.URL.Path[1:], "/")
            // currently, a URL can only have 2 parts (ie. if its a subproject)
            if len(path_elements) > 2{
                g.errorPage(w, errors.New("Invalid URL"))
                return
            }
            //posts
            if IsPost(path_elements[0]){
                err := g.LoadPage(path.Join(SiteRoot, "posts"), r.URL.Path[1:], page)
                if err != nil{
                    g.errorPage(w, err)
                    return
                }
            //pages
            }else if g.IsPage(r.URL.Path[1:]){
                err := g.LoadPage(path.Join(SiteRoot, "pages"), r.URL.Path[1:], page)
                if err != nil{
                    g.errorPage(w, err)
                    return 
                }
            } else{
                g.errorPage(w, errors.New("Invalid URL"))
                return
            }
        //home
        } else {
            err := g.LoadPage(path.Join(SiteRoot, "posts"), g.RecentPosts[0][1], page)
            if err != nil{
                g.errorPage(w, err)
                return 
            }
        }
        renderTemplate(w, "page", ViewType{Page:page, Globals:g})
}

type AjaxResponseType struct{
    Bubbles [][]string `json:"bubbles"`
}

// ajax bubble response
// if bubblename.md doesnt exist or is blank, return the NewBubbleString
// r.URL.Path should be /bubbles/bubble-name or just /bubbles/ to return all entries
func (g *Globals) ajaxResponse(w http.ResponseWriter, r *http.Request){
    split := strings.Split(r.URL.Path[1:], "/")
    log.Println(split)
    response := AjaxResponseType{[][]string{}}
    if len(split) > 1 && split[1] != ""{
        // TODO: assert length is 2, file name. this should probably be in the post request not url...
        log.Println("this is a single bubble")
        // if the url came with a name, return that bubble
        // maybe we should pass the name in a post request instead of url?
        bubble_content := LoadBubble(r.URL.Path[1:])
        response.Bubbles = append(response.Bubbles, []string{split[1], bubble_content})
    } else {
    // else, return all bubbles
    log.Println("all bubbles")
        files, err := ioutil.ReadDir(path.Join(SiteRoot, "bubbles"))
        if err != nil{
            log.Println("couldn't read bubble dir", err);
            return
        }
        for _, f := range files{
            name := f.Name()
            bubble_content := LoadBubble(path.Join("bubbles", name))
            response.Bubbles = append(response.Bubbles, []string{name, bubble_content})
        }
    }
    b, err := json.Marshal(response)
    if err != nil{
        log.Println("could not marshal response to json", err)
    }
    fmt.Fprintf(w, string(b))
}

// github webhook response (confirm valid post request, git pull)
func (g *Globals) gitResponse(w http.ResponseWriter, r *http.Request){
    log.Println("githook!")
    log.Println(r.Header)
    header := r.Header
    agent := header["User-Agent"][0]
    event := header["X-Github-Event"][0]
    sig := header["X-Hub-Signature"][0]
    // assert GitHub agent
    if !strings.Contains(agent, "GitHub"){
        log.Println("git request from non Github agent")
        return
    }
    // assert event type
    if !(strings.Contains(event, "push") || strings.Contains(event, "ping")){
        log.Println("git request for non push or ping event")
        return
    }
    // check HMAC
    p := make([]byte, r.ContentLength)    
    sum := 0
    // read http req - there is almost certainly a oneline for this...
    for sum < int(r.ContentLength){
        n, err := r.Body.Read(p[sum:])
        if err != nil{
            log.Println("error reading http.req", err)
            return
        }
        sum += n
    }
    key := g.webhookSecret
    sigbytes, err := hex.DecodeString(sig[5:]) // sig begins with "sha1:"
    if err != nil{
        log.Println("no hex to bytes!", err)
    }

    if !CheckMAC(p, sigbytes, key){
        log.Println("git request with invalid signature")
        return
    }

    // all checks passed
    g.GitPull()
}

// CheckMAC returns true if messageMAC is a valid HMAC tag for message.
func CheckMAC(message, messageMAC, key []byte) bool {
    mac := hmac.New(sha1.New, key)
    mac.Write(message)
    expectedMAC := mac.Sum(nil)
    log.Println(hex.EncodeToString(expectedMAC), hex.EncodeToString(messageMAC))
    return hmac.Equal(messageMAC, expectedMAC)
}

// if git pull not up to date, refresh Globals
// how do we pull safely, without messing up a user?!
func (g *Globals) GitPull(){
     cmd := exec.Command("git", "pull", "origin", "master")
     var out bytes.Buffer
     cmd.Stdout = &out
     cmd.Run()
     log.Println(out.String())
     if !strings.Contains(out.String(), "already up-to-date"){
        g.Refresh()
     }
}

// create new globals, copy over (eg. after git pull)
func (g *Globals) Refresh(){
    gg := Globals{}
    gg.LoadConfig()
    gg.AssembleSite()
    *g = gg
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
func serveFile(w http.ResponseWriter, r *http.Request){
    if strings.Contains(r.URL.Path, "."){
        subs := strings.Split(r.URL.Path, ".")
        ext := subs[len(subs)-1]
        if ext == "png" || ext == "jpg" || ext == "pdf" {
            p := path.Join(SiteRoot, r.URL.Path[1:])
            _, err := os.Stat(p)
            if err == nil{
                http.ServeFile(w, r, p)
            }
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

// watch directory callback
// kind of messy since it fires soo much
func (g *Globals) WatchDirCallback(watcher *fsnotify.Watcher){
    for {
        select {
        case ev := <-watcher.Event:
            log.Println("event:", ev)
            // only refresh if name is not hidden
            split := strings.Split(ev.Name, "/")
            name := split[len(split)-1]
            if !strings.HasPrefix(name, ".") && !strings.HasSuffix(name, "~"){
                g.Refresh()            
            }
        case err := <-watcher.Error:
            log.Println("error:", err)
        }
    }
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

func StartServer(){
    // load config, compile lists of site contents
    var g = Globals{}
    g.LoadConfig()
    g.AssembleSite()

    // set up new watcher
    watcher, err := fsnotify.NewWatcher()
    defer watcher.Close()
    if err != nil {
        log.Fatal(err)
    }
    // watch dir callback
    go g.WatchDirCallback(watcher)
    // recursive watch dirs. when an event fires in any dir, it'll hit the callback
    g.WatchDirs(watcher, ".")

    // routing functions
    http.HandleFunc("/", g.handleIndex) // main page (/, /posts, /pages)
    http.HandleFunc("/imgs/", serveFile) // static images (png, jpg)
    http.HandleFunc("/files/", serveFile) // static documents (pdfs)
    http.HandleFunc("/assets/", serveBlokeFile) // static js, css files
    http.HandleFunc("/bubbles/", g.ajaxResponse) // async bubbles
    http.HandleFunc("/git/", g.gitResponse) // github webhook

    // listen and serve!
    http.ListenAndServe(":"+strconv.Itoa(*ListenPort), nil)
}

func main(){
    flag.Parse()
    
    if *InitSite != ""{
        CreateNewSite()
        os.Exit(0)
    }
  
    if *WebHook{
        CreateSecretToken()
        os.Exit(0)
    }

    if *NewBubbles{
        var g = Globals{}
        g.LoadConfig()
        new_bubbles := ParseForNewBubbles()
        WriteSetToFile("empty_bubbles.txt", new_bubbles)
        log.Println(new_bubbles)
        os.Exit(0)
    }

    StartServer()
}
