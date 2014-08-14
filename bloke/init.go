package bloke

import (
    "strings"
    "strconv"
    "log"
    "io/ioutil"
    "os"
    "os/exec"
    "path"
    "encoding/json"
    "encoding/hex"
    "crypto/rand"
    "bytes"
    "regexp"
    "time"
)

// config struct - corresponds to config.json
type ConfigType struct{
    SiteName string `json:"site_name"`
    Email string `json:"email"`
    Site string `json:"site"`
    Repo string `json:"github_repo"`
    Glossary string `json:"glossary_file"`
    Disqus string `json:"disqus_user"`
}

// load config struct from config.json
func (g * Globals) LoadConfig(SiteRoot string){
    g.SiteRoot = SiteRoot
    file, e := ioutil.ReadFile(path.Join(g.SiteRoot, "config.json"))
    if e != nil{
        log.Fatal("no config", e)
    }
    var c ConfigType
    json.Unmarshal(file, &c)
    g.Config = c

    g.Close = make(chan bool)

    // sync with git repo first time
    if g.Config.Repo != ""{
        log.Println("Iniitializing git repo and syncing with github remote...")
        // initialize as git repo
        _, err := os.Stat(".git")
        if err != nil{
            cmd := exec.Command("git", "init")
            cmd.Run()
        }

        // check remote host
        cmd := exec.Command("git", "remote", "--v")
        var out bytes.Buffer
        cmd.Stdout = &out
        cmd.Run()
        if !strings.Contains(out.String(), g.Config.Repo){
            cmd = exec.Command("git", "remote", "add", "origin", g.Config.Repo)
            cmd.Run()
        }
       
        // check if empty commit history, add files, commit, and push to remote
        cmd = exec.Command("git", "status")
        out = *new(bytes.Buffer)
        cmd.Stdout = &out
        cmd.Run()
        if strings.Contains(out.String(), "Initial commit"){
            log.Println("making inital git commit and pushing to remote. you may need to authenticate!")
            cmd = exec.Command("git", "add", "pages", "posts", "bubbles", "config.json")
            cmd.Run()
            cmd = exec.Command("git", "commit", "-m", `"init"`)
            cmd.Run()
            cmd = exec.Command("git", "push", "origin", "master")
            cmd.Run()
        } else {
            log.Println("\tnothing to do")
        }

        // update new bubble string to point to repo
        NewBubbleString = "This bubble hasn't been written yet! You can help us write it by submitting issues or pull requests at [our github repo]("+g.Config.Repo+")"
    }
}

// load webhooks secret from file
func (g *Globals) LoadSecret(){
    b, e := ioutil.ReadFile(path.Join(g.SiteRoot, ".secret"))
    if e != nil{
        log.Println("no secret, github webhooks not enabled")
        g.webhookSecret = []byte("")
        return
    }
    g.webhookSecret = b
}

// main server startup function
// compile lists of pages and posts and prepare globals struct
// require at least one page and one post!
func (g *Globals) AssembleSite(){
    // go through pages and posts and entries
    //RenderTemplateToFile("page", "main", g)
    g.AssemblePages()
    g.AssemblePosts()
    g.LoadSecret()
    //log.Println(g)
}

// compile list of posts and fill in Globals struct
func (g *Globals) AssemblePosts(){
    // posts dir should be fill with files like 2014-06-12-Name.md
    // No directories
    files := ReadDir(path.Join(g.SiteRoot, "posts"))
    for _, f := range files {
        if !f.IsDir(){
           fname := strings.Split(f.Name(), ".")[0]
           date_name := strings.Split(fname, "-")
           //year := date_name[0]
           //month := date_name[1]
           //day := date_name[2]
           //TODO: robustify!
           title := date_name[3]
           g.RecentPosts = append(g.RecentPosts, []string{title, fname})
        }
    }
    if len(g.RecentPosts) == 0{
        log.Fatal("Sorry, you must have at least one post in the posts/ directory.\nYou MUST use a name like posts/2014-05-06-postname.md and a valid date")
    }

}

// search every file for new bubbles, create them, return list of new_bubbles
func ParseForNewBubbles(SiteRoot string) map[string]bool{
    new_bubbles := make(map[string]bool) // a 'set' type
    folders := []string{"bubbles", "posts", "pages"}
    // compose list of bubbles to make
    for _, folder := range folders{
        // read folder
        files := ReadDir(path.Join(SiteRoot, folder))

        // for each file in folder, parse for new bubbles
        for _, file := range files{
            if !file.IsDir(){
                ParseFileForNewBubbles(path.Join(folder, file.Name()), &new_bubbles)
            } else{
                // if file is a dir, read dir, for each subfile, parse for new bubbles
                subfiles := ReadDir(path.Join(SiteRoot, folder, file.Name()))
                for _, subfile := range subfiles{
                    ParseFileForNewBubbles(path.Join(folder, file.Name(), subfile.Name()), &new_bubbles)
                }
            }
        }
    }
    return new_bubbles
}

// Find all bubbles, check against old_bubbles, return new bubbles
func ParseFileForNewBubbles(pathname string, new_bubbles *map[string]bool) {
    b, err := ioutil.ReadFile(pathname)
    if err != nil{
        log.Println("error reading file", pathname, err)
        return
    }
    r, _ := regexp.Compile(`\[\[(.+?)\] \[(.+?)\]\]?`)

    for _, match := range r.FindAllStringSubmatch(string(b), -1){
        name := match[2]
        // if file does not exist, or if file exists but is empty, its a new bubble
        _, err := os.Stat(path.Join("bubbles", name+".md"))
        if err != nil{
            (*new_bubbles)[name] = true
            _, err := os.Create(path.Join("bubbles", name+".md"))
            if err != nil{
                log.Println("could not create new bubble file")
            }
        } else {
            b, err := ioutil.ReadFile(path.Join("bubbles", name+".md"))
            if err != nil{
                log.Println("error reading file", name, err)
            }
            if len(b) == 0{
                (*new_bubbles)[name] = true
            }
        }
    }
}

// read dir, ignore hidden files/folders
func ReadDir(dir string)[] os.FileInfo{
    files, err := ioutil.ReadDir(dir)
    if err != nil {
        log.Fatal("error reading pages")
    }
    var return_files []os.FileInfo
    for _, f := range files{
        if !strings.HasPrefix(f.Name(), "."){
            return_files = append(return_files, f)
        }
    }
    return return_files
}

// compile list of pages and prepare Globals struct (mostly for filling in the nav bar with pages links)
// in future, write everything out to static .html files for serving later (so we don't have to render template each time)
func (g *Globals) AssemblePages(){
    // g.Projects is a list of strings
    // g.SubProjects maps projcets to a list of subprojectes (pairs (urlname, displayname))
    g.SubProjects = make(map[string][][]string)

    // get list of files in pages dir
    files := ReadDir(path.Join(g.SiteRoot, "pages"))

    for _, f := range files {
        // if project is not a directory, attempt get name from meta info
        if !f.IsDir(){
            url_name := strings.Split(f.Name(), ".")[0]
            display_name := GetTitleFromMetaInfo("pages", url_name)
            g.Projects = append(g.Projects, []string{url_name, display_name})
        } else{
            // project is a directory, and has subprojects
            // get name from meta-info.json, or fall back to dirname
            subfiles := ReadDir(path.Join(g.SiteRoot, "pages", f.Name()))

            // go through list of subfiles, get names
            var subproj_list [][]string // list of pairs (urlname, displayname)
            parent := f.Name()
            for _, ff := range subfiles{
                url_name := strings.Split(ff.Name(), ".")[0]
                display_name := GetTitleFromMetaInfo(path.Join(g.SiteRoot, "pages", parent), url_name)
                subproj_list = append(subproj_list, []string{url_name, display_name})
            }

            // set default project display name
            project_name := f.Name()
            // check for project meta-info
            b, err := ioutil.ReadFile(path.Join(f.Name(), "meta-info.json"))
            if err == nil{
                var m MetaInfoType
                json.Unmarshal(b, &m)
                if m.Title != ""{
                    project_name = m.Title 
                }
            }
            g.Projects = append(g.Projects, []string{f.Name(), project_name})
            g.SubProjects[f.Name()] = subproj_list
        }
    }
    if len(g.Projects) == 0{
        log.Fatal("Sorry, you must have at least one project page in the pages/ directory")
    }
}

// open a file, parse metainfo, return title
// fallback to filename if no title
func GetTitleFromMetaInfo(dirPath, name string) string{
    b, err := ioutil.ReadFile(path.Join(dirPath,name+".md"))
    if err == nil{
        metaInfo, _ := ParseMetaInfo(b)
        if metaInfo.Title != ""{
            return metaInfo.Title
        }
    }
    return GetTitleFromUrl(name)
}

// get name from url
func GetTitleFromUrl(s string) string{
       split_path := strings.Split(s, "/")
       if IsPost(split_path[0]){
           date_name := strings.Split(strings.Split(s, ".")[0], "-")
           return date_name[3]
       }
       return split_path[len(split_path)-1]
}

func CheckFatal(err error){
    if err != nil{
        log.Fatal(err)
    }
}

// called on `bloke --init _InitSite`
func CreateNewSite(InitSite string){
    // create main folder
    mode := os.FileMode(0777) // this should be better but so far I dont understand it :(
    CheckFatal(os.Mkdir(InitSite, mode))
    CheckFatal(os.Mkdir(path.Join(InitSite, "bubbles"), mode))
    CheckFatal(os.Mkdir(path.Join(InitSite, "imgs"), mode))
    CheckFatal(os.Mkdir(path.Join(InitSite, "files"), mode))
    CheckFatal(os.Mkdir(path.Join(InitSite, "pages"), mode))
    CheckFatal(os.Mkdir(path.Join(InitSite, "posts"), mode))

    // create glossary page
    f, err := os.Create(path.Join(InitSite, "pages", "Glossary.md"))
    gloss_success := false
    if err != nil{
     log.Println("Could not create glossary file:", err)
    }else{
        f.WriteString("Below you will find a glossary of all bubbles")
        gloss_success = true
    }
    f.Close()

    // create first post
    yy, mm, dd := time.Now().Date()
    postname := strconv.Itoa(int(yy))+"-"+strconv.Itoa(int(mm))+"-"+strconv.Itoa(int(dd))+"-FirstPost.md"
    f, err = os.Create(path.Join(InitSite, "posts", postname))
    if err != nil{
     log.Println("Could not create first post file:", err)
    }else{
        f.WriteString("Welcome to your new bloke!")
    }
    f.Close()

    // create and initialize config file
    f, err = os.Create(path.Join(InitSite, "config.json"))
    defer f.Close()
    if err != nil{
     log.Println("Could not create config file:", err)
    }else{
        /*
        c := ConfigType{SiteName: InitSite}
        jc, _ := json.Marshal(c)
        enc := json.NewEncoder(f)
        err := enc.Encode(jc)
        if err != nil{
            log.Fatal(err)
        }
        */ // why can't I write a clean config file?
        f.WriteString("{\n")
        f.WriteString("\t\"site_name\": \""+InitSite+"\",\n")
        f.WriteString("\t\"email\": \"\",\n")
        f.WriteString("\t\"site\": \"\",\n")
        f.WriteString("\t\"github_repo\": \"\"\n")
        if gloss_success{
            f.WriteString("\t\"glossary_file\": \"Glossary.md\"\n")
        } else{
            f.WriteString("\t\"glossary_file\": \"\"\n")
        }
        f.WriteString("\t\"disqus_user\": \"\"\n")
        f.WriteString("}")
    }
    f.Close()

    _, err = os.Create(path.Join(InitSite, ".isbloke"))
    if err != nil{
        log.Println("could not init as bloke site. weird")
    }
}

// called on bloke --webhook
func CreateSecretToken(){
    f, err := os.Create(".secret")
    defer f.Close()
    if err != nil{
        log.Fatal("Could not create secret file", err)
    }
    secret_bytes := make([]byte, 20)
    _, err = rand.Read(secret_bytes)
    if err != nil{
        log.Fatal("could not generate random secret", err)
    }
    secret := hex.EncodeToString(secret_bytes)
    log.Println("copy the following secret into your webhook on github")
    log.Println("new secret:", secret)
    f.WriteString(secret)
}

// for writing list of empty bubbles to file
func WriteSetToFile(filename string, set map[string]bool){
    f, err := os.Create(filename)
    defer f.Close()
    if err != nil{
        log.Println("Could not create new file", err)
    }
    for k, _ := range set{
        f.WriteString(k+"\n")
    }
}


