Bloke
===
### Forget about your blog - time for a bloke!

#### *Bloke, you say? Yes, bloke!*
- Bloke (*proper noun*)  - This software!
- A bloke (*noun*) - a better kind of blog!
- To bloke (*verb*) - a better kind of blogging!

What is this?
---
Bloke is everything you wanted from a blog and more. With a built in server and AJAX functionality, a bloke gives you blog posts, project pages, and dynamic wiki-bubbles, all generated on the fly from simple markdown files. Use the wiki-bubbles to document important terms, or to build up colossal glossaries. Easily monitor and manage content and contributions with github integration. Connect with millions of internet communities through the Disqus commenting plugin. Bloke's open-source and written in Go (wtih some javascript), so it's fast, easy to install/configure/run, and open to contributions and improvements from anyone!

If you like, Bloke can stand for `Bubbles, Love, and Open Knowledge Everywhere`. But really it's just a better blog ;)

Install bloke with `go get github.com/ebuchman/bloke`. To create a new site, use `bloke --init site-name`, and start adding some content!

Features
---
- http server with ajax support for wiki-bubbles
- automatically generate pages, blog posts, and wiki-bubbles from markdown files
- serve images and pdfs
- use subdirectories for pages to create dropdown lists in the navbar
- convenient testing: automatically detect and serve edits and new files without restarting the server
- simple production deploy: automatically update on push to github (uses webhooks, requires configuration - see below)
- manage community edits using github!
- disqus commenting system connects you with millions of other internet communities worldwide
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

**Build and Serve**: To build and serve the site, cd into the site's root diretory and run `bloke --port 8080`.  Your site is now live on port 8080 (default port without the flag is 9099). Any changes pushed to the github repo will take effect immediately, as with any changes made locally. There is no need to restart the server.

Markdown Refresher
---
- italic: \*some text\*
- bold: \*\*some text\*\*
- links: \[some text](http://somesite.info)
- files: \[some text](/files/filename.pdf)
- images: !\[img-text](/imgs/img.png)
- bubbles: [[some text] [wiki-bubble-name]]

URL Rules
---
- when linking, only use file extensions for images/files, not for md/html (eg. click \[here](/MyProject) for more images like this [img-name](img-name.png))
- all posts and pages are located at `/PageName` or `/yyyy-mm-dd-PostName`
- pages in a subdirectory are specified with their parent: `/PageDir/SubPage`
- bubbles are linked simply with their name (no path), eg. `[[something cool] [bubble-name]]`





