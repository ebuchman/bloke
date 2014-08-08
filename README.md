Bloke
---
Bloke is a static-site generator with bubbly-wiki functionality (through ajax) written in golang. Install bloke with `go get github.com/ebuchman/bloke`. To build a site, use `bloke --init root-site-dir` to create a new root directory with the following directory structure:
```
root-site-dir/
    bubbles/
    imgs/
    pages/
    posts/
```

The pages directory should have a set of markdown files, one for each page of your site (accessible at `yoursite/pagename`). Similarly for the posts directory, but posts must begin with the date, in the format `yyyy-mm-dd-postname.md`. The bubbles directory contains the wiki entries, also as markdown. To create a bubble link (in a page, post, or bubble), use the format `[[this text] [some-bubble-entry]]`. Finally, place any images in the imgs directory.

See an example at github.com/ebuchman/ccblog

To build the site, cd into the site's root diretory and run `bloke`.  Your site is now live on port 9099!
