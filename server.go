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
)

/* TODO
    - add tls support
    - clean up js bubbles so they follow user as they scroll
    - meta info (pages, posts, bubbles)
    - add "technical explanation" part to bubbles
    - validate serve assets
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

// config struct - corresponds to config.json
type ConfigType struct{
    SiteName string `json:"site_name"`
    Email string `json:"email"`
    Site string `json:"site"`
    Repo string `json:"github_repo"`
}

type MetaInfoType struct {
    Title string `json:"title"`
}

// main site struct
type Globals struct{
    Projects [][]string // list of pairs (url/filename, display name)
    SubProjects map[string][][]string // map from project-filename to list of pairs (subproject filenames, subproject displayname). these generate the dropdowns
    // all proj/subproj references should be with url_name!

    Posts map[string]map[string]map[string][]string // year, month, day, title
    RecentPosts [][]string // [](title, date_name)

    Text string // text of current page
    Title string // title of current page
    MetaInfo MetaInfoType // struct of meta info

    Config ConfigType

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
                err := g.LoadPage(path.Join(SiteRoot, "posts"), r.URL.Path[1:])
                if err != nil{
                    g.errorPage(w, err)
                    return
                }
            //pages
            }else if g.IsPage(r.URL.Path[1:]){
                err := g.LoadPage(path.Join(SiteRoot, "pages"), r.URL.Path[1:])
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
            err := g.LoadPage(path.Join(SiteRoot, "posts"), g.RecentPosts[0][1])
            if err != nil{
                g.errorPage(w, err)
                return 
            }
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
        log.Println("error on bubble ", r.URL.Path[1:], err)
    }
    fmt.Fprintf(w, g.ParseBubbles(b))
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

// serve static files (assets)
func serveFile(w http.ResponseWriter, r *http.Request){
    // if img, load from SiteRoot
    // if js/css, load from BlokePath

    //TODO: better validation

    if !strings.Contains(r.URL.Path, "."){
        //s.handleIndex(w, r)
    }else{
        subs := strings.Split(r.URL.Path, ".")
        ext := subs[len(subs)-1]
        if ext == "js" || ext == "css"{
            http.ServeFile(w, r, path.Join(BlokePath, r.URL.Path[1:]))
        }else if ext == "png" || ext == "jpg" || ext == "pdf" {
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

func StartServer(){
    // load config, compile lists of site contents
    g := Globals{}
    g.LoadConfig()
    g.AssembleSite()

    // routing functions
    http.HandleFunc("/", g.handleIndex) // main page (/, /posts, /pages)
    http.HandleFunc("/imgs/", serveFile) // static images (png, jpg)
    http.HandleFunc("/files/", serveFile) // static documents (pdfs)
    http.HandleFunc("/assets/", serveFile) // static js, css files
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


    StartServer()
}
