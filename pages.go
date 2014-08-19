package bloke

import (
    "github.com/russross/blackfriday" // parsing markdown
    "net/http"
    "text/template"
    "log"
    "os"
    "regexp"
    "path"
    "io/ioutil"
    "strings"
    "strconv"
    "encoding/json"
)

// meta info struct. read from json
type MetaInfoType struct {
    Title string `json:"title"`
}

// info specific to the page requested by a client
type pageType struct{
    Name string //URL name of this page
    Text string // text of current page
    Title string // title of current page
    MetaInfo MetaInfoType // struct of meta info for current page

    // bloke flags (trigger specialized html/templating)
    IsGlossary bool
    IsDisqus bool
}


//parse template files
var templates = template.Must(template.ParseFiles(BlokePath+"/views/page.html", BlokePath+"/views/nav.html", BlokePath+"/views/footer.html", BlokePath+"/views/bubbles.html"))

// bring a template to life!
func renderTemplate(w http.ResponseWriter, tmpl string, p interface{}){
    //we already parsed the html templates
    err := templates.ExecuteTemplate(w, tmpl+".html", p)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}

// error function
func (g *Globals) errorPage(w http.ResponseWriter, err error){
    page := new(pageType)
    page.Title = "Error"
    page.Text = err.Error()
    renderTemplate(w, "page", viewType{Page: page, Globals: g})
}

// load and parse a page and relevent metainfo 
func (g *Globals) LoadPage(dirPath, name string, page *pageType) error{
    // read markdown file
//    log.Println(path.Join(dirPath, name+".md"))
    b, err := ioutil.ReadFile(path.Join(dirPath,name+".md"))
    if err != nil{
        return err
    }
    // set meta-info and text
    page.MetaInfo, b = ParseMetaInfo(b)
    page.Text = ParseBubbles(b) 
    // set title
    if page.MetaInfo.Title == ""{
        page.Title = GetTitleFromUrl(name)
    }else {
        page.Title = page.MetaInfo.Title
    }
    page.Name = name

    //set flags
    page.IsGlossary = page.Name == g.Config.Glossary ||  page.Name == "pages/"+g.Config.Glossary
    log.Println(page.IsGlossary, page.Name)
    page.IsDisqus = g.Config.Disqus != ""

    return nil
}

// load bubble, parse text, return html string
// will need an upgrade to json for metainfo...
func LoadBubble(SiteRoot, name string) string{
    _, err := os.Stat(path.Join(SiteRoot, name+".md"))
    bubble_content := ""
    if err == nil{
        b, err := ioutil.ReadFile(path.Join(SiteRoot, name+".md"))
        if err != nil{
            log.Println("error on bubble ", name, err)
            b = []byte("there was an error reading this bubble")
        }
        if len(b) == 0{
            bubble_content = ParseBubbles([]byte(NewBubbleString))
        }else {
            bubble_content = ParseBubbles(b)
        }
    } else{
        bubble_content = ParseBubbles([]byte(NewBubbleString))
    }
    return bubble_content
}

// parse metainfo. return metainfo struct and remaining bytes
func ParseMetaInfo(s []byte) (MetaInfoType, []byte){
    var m MetaInfoType
    r, err := regexp.Compile(`---\n((?s).+?)\n---`)
    if err != nil{
        log.Println("shitty regexp bro..")
    }
    match := r.FindSubmatch(s)
    if len(match) > 0{
        full_match := match[0]  // 0 is the match, 1 is the first submatch
        meta_info_bytes := match[1]
        json.Unmarshal(meta_info_bytes, &m)
        s = s[len(full_match):]
    }
    return m, s
}

// parse and replace for bubbles and markdown to js/html
// takes the raw txt.md bytes
// creates new bubble entries if they are referenced but don't exist
func ParseBubbles(s []byte) string{
    r, _ := regexp.Compile(`\[\[(.+?)\] \[(.+?)\]\]?`)
    s = blackfriday.MarkdownCommon(s)
    return r.ReplaceAllString(string(s), `<a href="#/" onClick="get_entry_data('$2')">$1</a>`)
}


func (g *Globals) SaveSite(){
    CheckFatal(os.MkdirAll(path.Join(g.SiteRoot, "_site"), 0777))
    CheckFatal(os.MkdirAll(path.Join(g.SiteRoot, "_site", "pages"), 0777))
    CheckFatal(os.MkdirAll(path.Join(g.SiteRoot, "_site", "posts"), 0777))

    // generate project html files
    for _, p := range g.Projects{
        name := p[0]
        if _, ok := g.SubProjects[name]; !ok{
            page := new(pageType)
            CheckFatal(g.LoadPage(path.Join(g.SiteRoot, "pages"), name, page))
            RenderTemplateToFile("page", path.Join(g.SiteRoot, "_site", "pages"), name, viewType{page, g})
        } else{
            // deal with subprojects!
            subprojs := g.SubProjects[name]
            for _, sp := range subprojs{
                sp_name := sp[0]
                page := new(pageType)
                CheckFatal(os.MkdirAll(path.Join(g.SiteRoot, "_site", "pages", name), 0777))
                CheckFatal(g.LoadPage(path.Join(g.SiteRoot, "pages", name), sp_name, page))
                RenderTemplateToFile("page", path.Join(g.SiteRoot, "_site", "pages", name), sp_name, viewType{page, g})
            }

        }

    }

    // generate post html files
    for y, _ := range g.Posts{
        for m, _ := range g.Posts[y]{
            for d, _ := range g.Posts[y][m]{
                for _, t := range g.Posts[y][m][d]{
                    name := y+"-"+m+"-"+d+"-"+t
                    page := new(pageType)
                    CheckFatal(g.LoadPage(path.Join(g.SiteRoot, "posts"), name, page)) 
                    RenderTemplateToFile("page", path.Join(g.SiteRoot, "_site", "posts"), name, viewType{page, g})
                }
            }
        }
    }


}


func RenderTemplateToFile(tmpl, SiteRoot, save_file string, p interface{}){
    //we already parsed the html templates
    f, err := os.Create(path.Join(SiteRoot, save_file+".html"))
    if err != nil{
        log.Fatal("err opening file:", err)
    }
    err = templates.ExecuteTemplate(f, tmpl+".html", p)
    if err != nil {
        log.Fatal("err writing template to file", err)
    }
    f.Close()
}

// it's a blog post if it is of the form yyyy-mm-dd-name-of-post.md and the date is quasi valid
func IsPost(name string) bool{
    parts := strings.Split(name, "-")
    if len(parts) >= 4{
        n1, err1 := strconv.Atoi(parts[0])
        n2, err2 := strconv.Atoi(parts[1])
        n3, err3 := strconv.Atoi(parts[2])
        return err1==nil && err2==nil && err3==nil && n1>1970 && 0<n2 && n2<13 && 0<n3 && n3<32
    }
    return false
}

// it's a page (subpage) if its in blokes list of known pages (subpages)
func (g *Globals) IsPage(name string) bool{
    parts := strings.Split(name, "/")
    if len(parts) > 2{
        return false
    }
    isPage := -1
    //find index of project
    for i, k := range g.Projects{
        if parts[0] == k[0]{
            isPage = i
            break
        }
    }
    // if project exists and request for subproject, check subproject exists
    if isPage>-1 && len(parts) == 2{
        _, ok := g.SubProjects[g.Projects[isPage][0]]
        if !ok{
            isPage = -1
        }
    }

    return isPage > -1
}

