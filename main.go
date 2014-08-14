package main

import (
    "flag"
    "log"
    "fmt"
    "os"
    "strings"
    "strconv"
    "net/http"
    "io/ioutil"
    "path"
    "github.com/ebuchman/bloke/bloke"
)

/*
    Run a stand-alone bloke server
                or
    Serve multiple blokes over a single server
*/

// do you have a wildcard ssl cert for subdomains?
var WILDCARD = false

// host multiple blokes
type Hoster struct{
    mydomain string
    subdomains map[string]bloke.Globals // map to blokes registered with us
    domains map[string]bloke.Globals // map to blokes with their own domain name
    //mux *http.ServeMux
}

// new bloke hoster
func NewHoster(domainName string) *Hoster{
    h := new(Hoster)
    h.mydomain = domainName
    h.subdomains = make(map[string]bloke.Globals)
    h.domains = make(map[string]bloke.Globals)
    return h
}


// localhost handler (for testing locally)
func (h *Hoster) localhostHandler(w http.ResponseWriter, r *http.Request){
    host := r.Host
    split := strings.Split(host, ":")
    host = split[0]

    g, ok := h.subdomains[host]
    if !ok{
        log.Println("not in domains map", host)
        return
    }
    g.ServeHTTP(w, r)
}

// if tls, handle http
// either redirect to https, or use http for subdomain
func (h *Hoster) hostHTTPHandler(w http.ResponseWriter, r *http.Request){

}

// a domain is either bad, our top domain, our subdomain, a registered other host
func DomainType(host, mydomain string) int{
    split := strings.Split(host, ".")

    // TODO: test regex for alphanum

    if len(split) < 2 || len(split) > 4{
        log.Println("invalid domain", host)
        return -1
    }

    mysplit := strings.Split(mydomain, ".")

    if strings.Contains(host, mydomain){
        // main site
       if host == "www"+mydomain || host == mydomain{ 
            // serve main page        
            return 0
        } else if len(split) == len(mysplit) + 1 && strings.Join(split[1:], ".") == strings.Join(mysplit[1:], "."){
            // possible sub domain
            return 1
        } else {
            return -1
        }
    } 

    // possible other domain
    return 2
}

// host handler
//TODO: handle errors
func (h *Hoster) hostHandler(w http.ResponseWriter, r *http.Request){

    domainType := DomainType(r.Host, h.mydomain)

    switch domainType{
        case -1:
            return
        case 0:
            // serve main page
        case 1:
            // serve subdomain
            subdomain := strings.Split(r.Host, ".")[0]
            g, ok := h.subdomains[subdomain]
            if !ok{
                log.Println("invalid subdomain", subdomain)
                return
            }
            g.ServeHTTP(w, r)
        case 2:
            // serve domain
            domain := r.Host
            g, ok := h.domains[domain]
            if !ok{
                log.Println("not a valid domain name", domain)
                return
            }
            g.ServeHTTP(w, r)
    }
}

func main(){
    // flag variables for cli
    var InitSite = flag.String("init", "", "path to new site dir")
    var ListenPort = flag.Int("port", 9099, "port to listen for incoming connections")
    var WebHook = flag.Bool("webhook", false, "create a new secret token for use with github webhook")
    var NewBubbles = flag.Bool("bubbles", false, "give all referenced bubbles a markdown file")
    var HostDomain = flag.String("host", "", "host multiple blokes on subdomains of this domain")
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
    if *HostDomain != ""{
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
        h := NewHoster(*HostDomain)
        // for now these are all subdomains
        // no hosting for custom domain names yet...
        for _, blokeName := range blokes{
            h.subdomains[blokeName] = bloke.LiveBloke(blokeName)
        }   
        log.Println("blokes:", h)
        if *SSLEnable{
            // if ssl, we need to run a redirect server
            // but we also need to serve user blokes over
            // http (only our page is ssl until we get a wildcard)
            mux := http.NewServeMux()
            mux.HandleFunc("/", h.hostHTTPHandler)
            go bloke.StartServer(":80", mux, false)
        }
        // start server
        mux := http.NewServeMux()
        mux.HandleFunc("/", h.hostHandler)
        bloke.StartServer(addr, mux, *SSLEnable)
    } else {
        // host standalone bloke
        if *SSLEnable{
            // run a server on 80 to redirect all traffic to 443
            go bloke.RedirectServer() 
        }
        bloke.StartBloke(addr, SiteRoot, *SSLEnable)
    }
}
