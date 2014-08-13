package main

import (
    "github.com/ebuchman/bloke/lib"
    "flag"
    "log"
    "fmt"
    "os"
    "strings"
    "strconv"
    "net/http"
    "io/ioutil"
    "path"
)

/*
    Run a stand-alone bloke server
                or
    Serve multiple blokes over a single server
*/

// host multiple blokes
type Hoster struct{
    domains map[string]bloke.Globals
    //mux *http.ServeMux
}

// host handler
func (h *Hoster) hostHandler(w http.ResponseWriter, r *http.Request){
    host := r.Host
    split := strings.Split(host, ":")
    host = split[0]

    g, ok := h.domains[host]
    if !ok{
        log.Println("not in domains map", host)
        return
    }
    g.ServeHTTP(w, r)
}

func main(){
    // flag variables for cli
    var InitSite = flag.String("init", "", "path to new site dir")
    var ListenPort = flag.Int("port", 9099, "port to listen for incoming connections")
    var WebHook = flag.Bool("webhook", false, "create a new secret token for use with github webhook")
    var NewBubbles = flag.Bool("bubbles", false, "give all referenced bubbles a markdown file")
    var Host = flag.Bool("host", false, "host multiple blokes")
    var SSLEnable = flag.Bool("ssl", false, "enable ssl/tls (https)") 

    flag.Parse()

    SiteRoot := "."
    
    if *InitSite != ""{
        bloke.CreateNewSite(*InitSite)
        fmt.Println("###################################")
        fmt.Println("Congratulations, your bloke has been created!")
        fmt.Println("To configure your bloke, please edit config.json.")
        fmt.Println("You probably want to put an image file called logo.png in the imgs/ directory")
        fmt.Println("Link up with a github repo and the disqus commenting system anytime by adding the respective details in config.json (see readme for more details")
        fmt.Println("To launch the site, simply run `bloke` from the site's root directory. The site is live in your browser at `localhost:9099`")
        fmt.Println("###################################")
        os.Exit(0)
    }
  
    if *WebHook{
        bloke.CreateSecretToken()
        os.Exit(0)
    }

    if *NewBubbles{
        var g = bloke.Globals{}
        g.LoadConfig(SiteRoot)
        new_bubbles := bloke.ParseForNewBubbles(g.SiteRoot)
        bloke.WriteSetToFile("empty_bubbles.txt", new_bubbles)
        log.Println(new_bubbles)
        os.Exit(0)
    }

    // listening address
    addr := ":"+strconv.Itoa(*ListenPort)

    // host all blokes in this dir
    if *Host{
        // get all blokes in this dir
        files, err := ioutil.ReadDir(SiteRoot)
        if err != nil{
            log.Fatal("could not read dir", SiteRoot, err)
        }
        blokes := []string{}
        for _, f := range files{
            if f.IsDir() && bloke.IsBloke(path.Join(SiteRoot, f.Name())){ // should also ensure it's actually a bloke dir
                blokes = append(blokes, f.Name())
            }
        }
        // register blokes with hoster
        h := Hoster{make(map[string]bloke.Globals)}
        for _, blokeName := range blokes{
            h.domains[blokeName] = bloke.LiveBloke(blokeName)
        }   
        // start server
        mux := http.NewServeMux()
        mux.HandleFunc("/", h.hostHandler)
        bloke.StartServer(addr, mux, *SSLEnable)
    } else {
        // host standalone bloke
        bloke.StartBloke(addr, SiteRoot, *SSLEnable)
    }
}
