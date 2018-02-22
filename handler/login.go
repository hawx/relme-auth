package handler

import (
	"fmt"
	"net/http"
)

func Login() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `
<!DOCTYPE html>
<html>
<body>
  <form action="/authenticate" method="POST">
    <label for="me">Web Address:</label>
    <input type="url" id="me" name="me" />
    <button type="submit">Sign-in</button>
  </form>
</body>
</html>
`)
	})
}
