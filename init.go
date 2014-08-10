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
        }
    }
}

func (g *Globals) LoadSecret(){
    file, e := ioutil.ReadFile(path.Join(SiteRoot, ".secret"))
    if e != nil{
        log.Println("no secret, github webhooks not enabled")
        g.webhookSecret = []byte("")
        return
    }
    g.webhookSecret = file
}

// main server startup function
// compile lists of pages and posts and prepare globals struct
func (g *Globals) AssembleSite(){
    // go through pages and posts and entries
    //RenderTemplateToFile("page", "main", g)
    g.AssemblePages()
    g.AssemblePosts()
    //g.NumProjects = len(g.Projects)
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
           date_name := strings.Split(strings.Split(f.Name(), ".")[0], "-")
           //year := date_name[0]
           //month := date_name[1]
           //day := date_name[2]
           title := date_name[3]
           g.RecentPosts = append(g.RecentPosts, []string{title, f.Name()})
        }
    }

}

// compile list of pages and prepare Globals struct (mostly for filling in the nav bar with pages links)
// in future, write everything out to static .html files for serving later (so we don't have to render template each time)
func (g *Globals) AssemblePages(){
    files, err := ioutil.ReadDir(SiteRoot+"/pages")
    if err != nil {
        log.Fatal("error reading pages")
    }
    log.Println(files)
    g.SubProjects = make(map[string][]string)
    for _, f := range files {
        if !f.IsDir(){
            name := strings.Split(f.Name(), ".")[0]
            g.Projects = append(g.Projects, name)
            g.SubProjects[name] = []string{}
        } else{
            subfiles, err := ioutil.ReadDir(SiteRoot+"/pages/"+f.Name())
            if err != nil {
                log.Fatal("error reading sub pages")
            }
            var list []string
            for _, ff := range subfiles{
                name := strings.Split(ff.Name(), ".")[0]
                list = append(list, name)
            }
            g.Projects = append(g.Projects, f.Name())
            g.SubProjects[f.Name()] = list
        }
    }
}

// get name from blogpost url
func GetNameFromPost(s string) string{
       date_name := strings.Split(strings.Split(s, ".")[0], "-")
       title := date_name[3]
       return title
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

