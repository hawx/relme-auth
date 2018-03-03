# relme-auth

An (in-progress) implementation of <http://microformats.org/wiki/RelMeAuth>.


## What?

Sign-in to websites using your own domain.

1. Own a domain, for example `https://john.example.com`.

2. Set this domain for your profiles on Flickr/GitHub/Twitter.

3. Add `<a rel="me" href="...">` tags pointing to your profile pages from
   `https://john.example.com`.

4. Go to _relme-auth_, enter `https://john.example.com` and hit sign-in. You
   will be redirected to sign-in with one of the 3rd parties your site points
   to.


## Trying it out

This project will currently only work locally on `localhost:8080` (this is good
because I wouldn't want anyone using this on the internet yet). But assuming you
want to try it out locally,

1. `$ go get hawx.me/code/relme-auth`

2. Go to each of Flickr, GitHub and Twitter and setup a new app. Take the
   id/apiKey and secret given and put in a `config.toml` file like so,

   ```
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

3. Generate a set of keys using `ssh-keygen` in some local folder, say `./credentials`.

4. Now run it with `$ relme-auth --private-key ./credentials/id_rsa`.

5. Go to `http://localhost:8080` and try signing-in with your domain. If
   everything works you will get a nice JWT printed to the screen (obviously
   this will eventually become something more useful).
