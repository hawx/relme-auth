package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"hawx.me/code/assert"
)

type mockTemplate struct {
	Tmpl string
	Data interface{}
}

func (t *mockTemplate) ExecuteTemplate(w io.Writer, tmpl string, data interface{}) error {
	t.Tmpl = tmpl
	t.Data = data
	return nil
}

func TestPGP(t *testing.T) {
	assert := assert.Wrap(t)

	templates := &mockTemplate{}

	s := httptest.NewServer(PGP(templates))
	defer s.Close()

	http.Get(s.URL + "?client_id=my-client&state=my-state&challenge=my-challenge")

	assert(templates.Tmpl).Equal("app")

	data, ok := templates.Data.(pgpCtx)
	assert(ok).Must.True()
	assert("my-client").Equal(data.ClientID)
	assert("my-state").Equal(data.State)
	assert("my-challenge").Equal(data.Challenge)
}
