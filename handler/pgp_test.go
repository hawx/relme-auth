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
	assert := assert.New(t)

	templates := &mockTemplate{}

	s := httptest.NewServer(PGP(templates))
	defer s.Close()

	http.Get(s.URL + "?client_id=my-client&state=my-state&challenge=my-challenge")

	assert.Equal(templates.Tmpl, "pgp.gotmpl")

	data, ok := templates.Data.(pgpCtx)
	if assert.True(ok) {
		assert.Equal(data.ClientID, "my-client")
		assert.Equal(data.State, "my-state")
		assert.Equal(data.Challenge, "my-challenge")
	}
}
