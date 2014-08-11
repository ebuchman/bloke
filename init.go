package main

import (
    "strings"
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
)

// load config struct from config.json
func (g * Globals) LoadConfig(){
    file, e := ioutil.ReadFile(path.Join(SiteRoot, "config.json"))
    if e != nil{
        log.Fatal("no config", e)
    }
    var c ConfigType
    json.Unmarshal(file, &c)
    g.Config = c

    // sync with git repo first time
    if g.Config.Repo != ""{
        log.Println("Iniitializing git repo and syncing with github remote...")
        // initialize as git repo
        cmd := exec.Command("git", "init")
        cmd.Run()

        // check remote host
        cmd = exec.Command("git", "remote", "--v")
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
    }
}

// load webhooks secret from file
func (g *Globals) LoadSecret(){
    b, e := ioutil.ReadFile(path.Join(SiteRoot, ".secret"))
    if e != nil{
        log.Println("no secret, github webhooks not enabled")
        g.webhookSecret = []byte("")
        return
    }
    g.webhookSecret = b
}

// main server startup function
// compile lists of pages and posts and prepare globals struct
func (g *Globals) AssembleSite(){
    // go through pages and posts and entries
    //RenderTemplateToFile("page", "main", g)
    g.AssemblePages()
    g.AssemblePosts()
    g.LoadSecret()
    log.Println(g)
}

// compile list of posts and fill in Globals struct
func (g *Globals) AssemblePosts(){
    // posts dir should be fill with files like 2014-06-12-Name.md
    // No directories
    files, err := ioutil.ReadDir(SiteRoot+"/posts")
    if err != nil {
        log.Fatal("error reading pages")
    }
    for _, f := range files {
        if !f.IsDir(){
           fname := strings.Split(f.Name(), ".")[0]
           date_name := strings.Split(fname, "-")
           //year := date_name[0]
           //month := date_name[1]
           //day := date_name[2]
           title := date_name[3]
           g.RecentPosts = append(g.RecentPosts, []string{title, fname})
        }
    }

}

// search every file for new bubbles, create them, return list of new_bubbles
func ParseForNewBubbles() []string{
    new_bubbles := []string{}
    folders := []string{"bubbles", "posts", "pages"}
    // compose list of bubbles to make
    for _, folder := range folders{
        // read folder
        files, err := ioutil.ReadDir(path.Join(SiteRoot, folder))
        if err != nil{
            log.Fatal("error reading", folder, err)
        }   
        // for each file in folder, parse for new bubbles
        for _, file := range files{
            if !file.IsDir(){
                bubbles := ParseFileForNewBubbles(path.Join(folder, file.Name()))
                if len(bubbles) > 0{
                    new_bubbles = append(new_bubbles, bubbles...)
                }
            } else{
                // if file is a dir, read dir, for each subfile, parse for new bubbles
                subfiles, err := ioutil.ReadDir(path.Join(SiteRoot, folder, file.Name()))
                if err != nil{
                    log.Fatal("error reading", folder, file, err)
                }
                for _, subfile := range subfiles{
                    bubbles := ParseFileForNewBubbles(path.Join(folder, file.Name(), subfile.Name()))
                    if len(bubbles) > 0{
                        new_bubbles = append(new_bubbles, bubbles...)
                    }
                }
            }
        }
    }
    return new_bubbles
}

// Find all bubbles, check against old_bubbles, return new bubbles
func ParseFileForNewBubbles(pathname string) []string{
    b, err := ioutil.ReadFile(pathname+".md")
    if err != nil{
        return []string{}
    }
    r, _ := regexp.Compile(`\[\[(.+?)\] \[(.+?)\]\]?`)

    new_bubbles := []string{}
    for _, match := range r.FindAllStringSubmatch(string(b), -1){
        name := match[2]
        new_bubbles = append(new_bubbles, name)
        CheckCreateBubble(name)
    }
    return new_bubbles 
}

// compile list of pages and prepare Globals struct (mostly for filling in the nav bar with pages links)
// in future, write everything out to static .html files for serving later (so we don't have to render template each time)
func (g *Globals) AssemblePages(){
    // g.Projects is a list of strings
    // g.SubProjects maps projcets to a list of subprojectes (pairs (urlname, displayname))
    g.SubProjects = make(map[string][][]string)

    // get list of files in pages dir
    files, err := ioutil.ReadDir(SiteRoot+"/pages")
    if err != nil {
        log.Fatal("error reading pages")
    }

    for _, f := range files {
        // if project is not a directory, attempt get name from meta info
        if !f.IsDir(){
            url_name := strings.Split(f.Name(), ".")[0]
            display_name := GetTitleFromMetaInfo("pages", url_name)
            g.Projects = append(g.Projects, []string{url_name, display_name})
        } else{
            // project is a directory, and has subprojects
            // get name from meta-info.json, or fall back to dirname
            subfiles, err := ioutil.ReadDir(path.Join(SiteRoot, "pages", f.Name()))
            if err != nil {
                log.Fatal("error reading sub pages")
            }
            // go through list of subfiles, get names
            var subproj_list [][]string // list of pairs (urlname, displayname)
            parent := f.Name()
            for _, ff := range subfiles{
                url_name := strings.Split(ff.Name(), ".")[0]
                display_name := GetTitleFromMetaInfo(path.Join(SiteRoot, "pages", parent), url_name)
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

// called on `bloke --init _InitSite`
func CreateNewSite(){
    os.Mkdir(*InitSite, 0777) // apparently 6s aren't sufficient here?
    os.Mkdir(path.Join(*InitSite, "bubbles"), 0666)
    os.MkdirAll(path.Join(*InitSite, "imgs"), 0666)
    os.MkdirAll(path.Join(*InitSite, "files"), 0666)
    os.MkdirAll(path.Join(*InitSite, "pages"), 0666)
    os.MkdirAll(path.Join(*InitSite, "posts"), 0666)

    f, err := os.Create(path.Join(*InitSite, "config.json"))
    defer f.Close()
    if err != nil{
     log.Println("Could not create config file:", err)
    }else{
        /*
        c := ConfigType{SiteName: *InitSite}
        jc, _ := json.Marshal(c)
        enc := json.NewEncoder(f)
        err := enc.Encode(jc)
        if err != nil{
            log.Fatal(err)
        }
        */ // why can't I write a clean config file?
        f.WriteString("{\n")
        f.WriteString("\t\"site_name\": \""+*InitSite+"\",\n")
        f.WriteString("\t\"email\": \"\",\n")
        f.WriteString("\t\"site\": \"\",\n")
        f.WriteString("\t\"github_repo\": \"\"\n")
        f.WriteString("}")
    }
    log.Println("Your site has been created!")
    log.Println("To configure your site, please edit config.json. Then, run bloke")
}

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

