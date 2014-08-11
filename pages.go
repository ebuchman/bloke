package main 

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
    page := new(PageType)
    page.Title = "Error"
    page.Text = err.Error()
    renderTemplate(w, "page", ViewType{Page: page, Globals: g})
}

// load and parse a page and relevent metainfo 
func (page *PageType) LoadPage(dirPath, name string) error{
    // read markdown file
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
    return nil
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

// check if bubble exists. if not, create, return true
func CheckCreateBubble(name string) bool{
    _, err := os.Stat(path.Join("bubbles", name+".md"))
    if err != nil{
        f, err := os.Create(path.Join("bubbles", name+".md"))
        if err != nil{
            log.Println("could not create new bubble file")
        } else{
            f.WriteString(NewBubbleString)
        }
        return true
    }
    return false
}

// parse and replace for bubbles and markdown to js/html
// takes the raw txt.md bytes
// creates new bubble entries if they are referenced but don't exist
func ParseBubbles(s []byte) string{
    r, _ := regexp.Compile(`\[\[(.+?)\] \[(.+?)\]\]?`)
    s = blackfriday.MarkdownCommon(s)

    // get all matches, check if they exist, add them if not...
    for _, match := range r.FindAllStringSubmatch(string(s), -1){
        name := match[2]
        CheckCreateBubble(name)
    }

    return r.ReplaceAllString(string(s), `<a href="#/" onClick="get_entry_data('$2')">$1</a>`)
}



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

