package bloke

import (
    "log"
    "path"
    "os"
    "os/exec"
    "bytes"
    "strings"
    "encoding/hex"
    "crypto/hmac"
    "crypto/sha1"
)

// CheckMAC returns true if messageMAC is a valid HMAC tag for message.
func CheckMAC(message, messageMAC, key []byte) bool {
    mac := hmac.New(sha1.New, key)
    mac.Write(message)
    expectedMAC := mac.Sum(nil)
    log.Println(hex.EncodeToString(expectedMAC), hex.EncodeToString(messageMAC))
    return hmac.Equal(messageMAC, expectedMAC)
}

// if git pull not up to date, refresh Globals
// how do we pull safely, without messing up a user?!
func (g *Globals) GitPull(){
     current, _ := os.Getwd()
     os.Chdir(g.SiteRoot)

     cmd := exec.Command("git", "pull", "origin", "master")
     var out bytes.Buffer
     cmd.Stdout = &out
     cmd.Run()
     log.Println(out.String())
     // parse output for list of new/changed bubbles
     updates := BubbleUpdates(out.String())
     if !strings.Contains(out.String(), "already up-to-date"){
        g.Refresh(updates)
     }
     os.Chdir(current)
}

func getNameFromPathSpace(head_dir, str string) string{
    splitt := strings.Split(str, " ")
    sb := ""
    for _, s := range splitt{
        if strings.Contains(s, head_dir){
            sb = s
        }
    }
    sp := strings.Split(sb, "/")
    f := sp[len(sp)-1]
    sp = strings.Split(f, ".")
    return sp[0]
}

// check if changes introduce or change bubbles. return list of them
// for now it only works on changes, adds, and rms. Rename's mess it all up!
// redo with regex for christ's sake!
func BubbleUpdates(gitpull string) map[string]int{
    updates := make(map[string]int)
    split := strings.Split(gitpull, "\n")
    log.Println("looking for bubble changes....")
    var i int
    var s string
    for i, s = range split{
        // this should be a regexp..
        if strings.Contains(s, "file") && strings.Contains(s, "insertion") && strings.Contains(s, "deletion"){
            log.Println("done checking changes")
            break
        }

        // bubbles/filename.md | 12 +++
        splitt := strings.Split(s, "|")
        if len(split) != 2{
            continue
        }
        name := splitt[0]
        info := splitt[1]
        // we only care about changes to bubbles
        if strings.Contains(name, "bubbles/"){
            // parse name to get the file name
            name := getNameFromPathSpace("bubbles/", name)
            log.Println("potential bubble", name)
            // parse info to detect if this was a change or an add/rm
            if strings.Contains(info, "+") || strings.Contains(info, "-"){
                log.Println("a change!")
                updates[name] = 0
            }// if it was add/rm, it will have no +/-

        } else {
            continue
        }
    }
    log.Println("done changes ya", i)
    // at this point, we've seen all the changes, but those that were only add/rm we ignored. now we find out create/delete
    for _, s:= range split[i:]{
        log.Println(s)
        if strings.Contains(s, " mode ") && strings.Contains(s, " bubbles/"){
            name := getNameFromPathSpace("bubbles/", s)
            log.Println("potential add/rm", name)
            if strings.Contains(s, "delete mode "){
                updates[name] = 1
            } else if strings.Contains(s, "create mode "){
                updates[name] = 2
            }
            log.Println("the upd", updates[name])
        }
    }

    return updates 
}


// check if given dir is a bloke
func IsBloke(pathname string) bool{
    _, err := os.Stat(path.Join(pathname, ".isbloke"))
    if err != nil{
        return false
    }
    return true
}



