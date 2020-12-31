package handler

import "io"

type tmpl interface {
	ExecuteTemplate(w io.Writer, tmpl string, data interface{}) error
}
