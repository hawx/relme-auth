# relme-auth

An implementation of <http://microformats.org/wiki/RelMeAuth>.


## What?

Sign-in to websites using your own domain.

For example, say you own `https://john.example.com`. First you would need to set
this domain on your Flickr/GitHub/Twitter profile(s). Then add a `<a rel="me"
href="...">` link to those profiles from `https://john.example.com`.

Now you can go to _relme-auth_, enter `https://john.example.com` and hit
sign-in. You can then select which provider you want to authenticate with.


## Running the code

This should be pretty standard for a Go project. It requires modules to pin
specific versions of packages.

```
$ go get hawx.me/code/relme-auth
```

Go to each of Flickr, GitHub and Twitter and setup a new app. Take the id/apiKey
and secret given and put in a `config.toml` file like so,

```toml
[flickr]
id = "..."
secret = "..."

[github]
id = "..."
secret = "..."

[twitter]
id = "..."
secret = "..."
```

Then run the app and go to `http://localhost:8080`.

```
$ relme-auth --cookie-secret something
```
