package handler

import "io"

type Templates interface {
	ExecuteTemplate(w io.Writer, tmpl string, data interface{}) error
}
