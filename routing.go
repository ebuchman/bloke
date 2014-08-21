package bloke

import (
    "net/http"
    "strings"
    "log"
    "io/ioutil"
    "path"
    "fmt"
    "encoding/hex"
    "errors"
    "encoding/json"
)


// this guy gets passed to the go templates. simply has pointers to the globals and the page
// every request sees the same globals, but a different page
type ViewType struct{
    Page *PageType
    Globals *Globals
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
        log.Println("handle Index", r.URL.Path, r.Host)

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
                if g.html{
                   http.ServeFile(w, r, path.Join(g.SiteRoot, "_site", "posts", r.URL.Path[1:]+".html")) 
                   return
                }
                err := g.LoadPage(path.Join(g.SiteRoot, "posts"), r.URL.Path[1:], page)
                if err != nil{
                    g.errorPage(w, err)
                    return
                }
            //pages
            }else if g.IsPage(r.URL.Path[1:]){
                if g.html{
                   http.ServeFile(w, r, path.Join(g.SiteRoot, "_site", "pages", r.URL.Path[1:]+".html")) 
                   return
                }
                err := g.LoadPage(path.Join(g.SiteRoot, "pages"), r.URL.Path[1:], page)
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
            if g.html{
               http.ServeFile(w, r, path.Join(g.SiteRoot, "_site", "posts", g.RecentPosts[0][1]+".html")) 
               return
            }
            err := g.LoadPage(path.Join(g.SiteRoot, "posts"), g.RecentPosts[0][1], page)
            if err != nil{
                g.errorPage(w, err)
                return 
            }
        }
        g.RenderTemplate(w, "page", ViewType{Page:page, Globals:g})
}

type Bubble struct{
    Title string `json:"title"`
    Content string `json:"content"`
}

type AjaxBubbleResponseType struct{
    Bubbles []Bubble `json:"bubbles"`
}

// ajax bubble response
// if bubblename.md doesnt exist or is blank, return the NewBubbleString
// r.URL.Path should be /bubbles/bubble-name or just /bubbles/ to return all entries
func (g *Globals) ajaxBubbleResponse(w http.ResponseWriter, r *http.Request){
    split := strings.Split(r.URL.Path[1:], "/")
    log.Println(split)
    response := AjaxBubbleResponseType{[]Bubble{}}
    if len(split) > 1 && split[1] != ""{
        // TODO: assert length is 2, file name. this should probably be in the post request not url...
        log.Println("this is a single bubble")
        // if the url came with a name, return that bubble
        // maybe we should pass the name in a post request instead of url?
        name := split[1]
        bubble_content := LoadBubble(g.SiteRoot, r.URL.Path[1:])
        response.Bubbles = append(response.Bubbles, Bubble{Title:name, Content:bubble_content})
    } else {
    // else, return all bubbles
    log.Println("all bubbles")
        files, err := ioutil.ReadDir(path.Join(g.SiteRoot, "bubbles"))
        if err != nil{
            log.Println("couldn't read bubble dir", err);
            return
        }
        for _, f := range files{
            name := f.Name()
            split := strings.Split(name, ".")
            name = split[0]
            bubble_content := LoadBubble(g.SiteRoot, path.Join("bubbles", name))
            response.Bubbles = append(response.Bubbles, Bubble{name, bubble_content})
        }
    }
    b, err := json.Marshal(response)
    if err != nil{
        log.Println("could not marshal response to json", err)
    }
    fmt.Fprintf(w, string(b))
}

func (g *Globals) ajaxPagesResponse(w http.ResponseWriter, r *http.Request){
    split := strings.Split(r.URL.Path[1:], "/")
    log.Println("shit son", split)
    page := new(PageType)
    if len(split) > 1 && split[1] != ""{
        err := g.LoadPage(g.SiteRoot, r.URL.Path[1:], page)
        if err != nil{
            log.Println("error reading page request")
            return 
        }
    } else {
        log.Println("error reading page request")
        return 
    }
    b, err := json.Marshal(page)
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
