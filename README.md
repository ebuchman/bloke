Bloke
---
Bloke is a combined static-site generator and server with bubbly-wiki functionality written in golang. It allows you to quickly build and serve a clean, attractive, informative site with little more than text files. Bloke can be easily setup to watch for new commits on github and update accordingly. Install bloke with `go get github.com/ebuchman/bloke`. To create a new site, use `bloke --init site-name`, and start adding some content!

Features
---
- http server with ajax support for wiki-bubbles (links to javascript bubbles that serve as a site-wide wiki)
- automatically generate pages, blog posts, and wiki-bubbles from markdown files
- serve images and pdfs
- convenient testing: automatically detect and serve edits and new files
- use subdirectories for pages (dropdown lists in the navbar)
- simple production deploy: automatically update on push to github (uses webhooks, requires configuration - see below)
- manage community edits using github!
- extended markdown for linking to wiki-bubbles
- meta-info for each post, page, and bubble

How-To
---
Start a new site with `bloke --init site-name`. This will create a new root directory with the following directory structure:
```
site-name/
    config.json
    bubbles/
    imgs/
    files/
    pages/
    posts/
```

**Basics**: The pages directory should have a set of markdown files, one for each page of your site (accessible at `yoursite/pagename`). A link for each page is displayed in the navbar. You can create folders in `pages/` and put markdown files in them too - each such folder will give you a dropdown list on the navbar. Blog posts go in the post directory, and must begin with the date in the format `yyyy-mm-dd-postname.md`. The bubbles directory contains the wiki entries, also as markdown. To create a bubble link (in a page, post, or bubble), use the format `[[this text] [some-bubble-entry]]` to load the text in `some-bubble-entry.md` when the `this text` link is clicked. Finally, place all images in the imgs directory, with the root image named logo.png.

**Config**: Make sure to edit config.json after creating the site and fill in all fields with the correct information. Most importantly, create a new repo on github and link to it in `config.json`. Running `bloke` will initialize a local git repo, make the first commit if it is not already there, and push to the remote. All further git events are on you.

**Examples**: See an example at https://github.com/ebuchman/ccblog.

**Note**: it is recommended not to track images and pdfs (or other non markdown files) using git. Use other means to sync such files to your server (ftp, rsync, scp, etc.).

**Syncing with GitHub**: Webhooks allow autmatic updates to your site simply by pushing changes to the github repo. Using webhooks securely involves generating a secret key and uploading it to github. This must only be done once. To create the secret key for authenticating github webhook requests, cd into the site's directory and run `bloke --webhook`. This will output a new secret key and save it to a local file `.secret`. Navigate to https://github.com/your-name/your-site-dir/settings/hooks in your browser and add a new webhook. Set the Payload URL to http://your-bloke-site-domain-name/git/ and paste the secret key just created under Secret. Then press Update webhook. Now, whenever you push to the repo, github will send a post request to your bloke site that will be authenticated via HMAC using the secret key we created. Bloke will then call `git pull origin master` and update your site. Tada!

**Build and Serve**: To build and serve the site, cd into the site's root diretory and run `bloke --port 8080`.  Your site is now live on port 8080 (default port without the flag is 9099). Any changes pushed to the github repo will take effect immediately. There is no need to restart the server.

Markdown Refresher
---
- italic: \*some text\*
- bold: \*\*some text\*\*
- links: \[some text](http://somesite.info)
- files: \[some text](/files/filename.pdf)
- images: !\[img-text](/imgs/img.png)
- bubbles: [[some text] [wiki-bubble-name]]


