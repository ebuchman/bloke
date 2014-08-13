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
     cmd := exec.Command("git", "pull", "origin", "master")
     var out bytes.Buffer
     cmd.Stdout = &out
     cmd.Run()
     log.Println(out.String())
     if !strings.Contains(out.String(), "already up-to-date"){
        g.Refresh()
     }
}

// check if given dir is a bloke
func IsBloke(pathname string) bool{
    _, err := os.Stat(path.Join(pathname, ".isbloke"))
    if err != nil{
        return false
    }
    return true
}



