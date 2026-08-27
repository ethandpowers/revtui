# revtui

The year was 2026...

I discovered TUIs and my life was changed.  It turns out that using
terminals for applications is like, the best thing ever.  Why you ask?
I'll tell you why.  Because terminal applications work flawlessly
over SSH and serial (sometimes), and can be organized in about a
billion different ways to fit into your workflow.  You like tmux?  Use
a TUI.  Like herdr?  Use a TUI.  Like ghostty or wezterm?  You guessed
it... TUI!  Not to mention that they just look so dang cool!

So, is this just my prideful attempt to build a TUI?  You bet!  Also,
I got tired of going to my web browser to check on the status of my
commits and those of my teammates.  Why couldn't I just do code review
from where I already work?  The terminal!  Until I get tmux in a web
browser, I'll be sticking to revtui for code review.

## Installation

Thanks to the *incredible* tooling that comes out of the box with go,
we can build a standalone binary by executing the following command in
the root of the repository (assuming you have go, of course):

```bash
go build -o revtui ./cmd/
```

Then, you can put it wherever you would like.  Note that this has only
tested on Linux as of yet.

## Getting Started

While this application *in theory* supports multiple backends for code
review, the only one implemented so far is for the code review server
*Gerrit*.  If anyone wants to implement support for any other platforms
before I get the chance, feel free!  To integrate with your gerrit
account, you will need to set the following environment variables.  I
have them set in my `.bashrc` though there are a handful of places to
set them:
```
GERRIT_HOST="<your_gerrit_host"
GERRIT_USER="<your_username>"
```

Note that there is also an enviroment override called
`GERRIT_SKIP_VERIFY_TLS` which is handy if you are self-hosting a gerrit
server and are using self-signed certificates.

Once your environment variables are set, either manually or in an rc
file, you can run the revtui command with the login parameter to get
started.  For gerrit, you'll need to have generated HTTP credentials
for basic auth, which can be passed in like so:
```bash
revtui login <your-http-auth-code>
```
