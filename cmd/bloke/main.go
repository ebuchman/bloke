package main

import (
    "flag"
    "log"
    "fmt"
    "os"
    "strconv"
    "github.com/ebuchman/bloke"
)

/*
    Run a stand-alone bloke server
*/

func main(){
    // flag variables for cli
    var InitSite = flag.String("init", "", "path to new site dir")
    var ListenPort = flag.Int("port", 9099, "port to listen for incoming connections")
    var WebHook = flag.Bool("webhook", false, "create a new secret token for use with github webhook")
    var NewBubbles = flag.Bool("bubbles", false, "give all referenced bubbles a markdown file")
    var SSLEnable = flag.Bool("ssl", false, "enable ssl/tls (https)") 
    var NoHTML = flag.Bool("nohtml", false, "serve html pages") 

    flag.Parse()

    SiteRoot, err := os.Getwd()
    if err !=nil{
        log.Fatal("could not get site root", err)
    }
    
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

    // host standalone bloke
    if *SSLEnable{
        // run a server on 80 to redirect all traffic to 443
        go bloke.RedirectServer() 
    }
    bloke.StartBloke(addr, SiteRoot, *SSLEnable, *NoHTML)
}
