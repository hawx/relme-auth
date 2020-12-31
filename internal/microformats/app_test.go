package microformats

import (
	"strings"
	"testing"

	"hawx.me/code/assert"
)

type testCase struct {
	Name         string
	HTML         string
	ExpectedName string
	ExpectedURL  string
}

func TestParseApp(t *testing.T) {
	testCases := []testCase{
		{
			Name: "x",
			HTML: `<div class="h-x-app">
  <a class="p-name u-url" href="http://host">my app</a>
</div>`,
			ExpectedName: "my app",
			ExpectedURL:  "http://host",
		},
		{
			Name: "same element",
			HTML: `<div class="h-app">
  <a class="p-name u-url" href="http://host">my app</a>
</div>`,
			ExpectedName: "my app",
			ExpectedURL:  "http://host",
		},
		{
			Name: "nested elements",
			HTML: `<div class="h-app">
  <h1 class="p-name"><a class="u-url" href="http://host">my app</a></h1>
</div>`,
			ExpectedName: "my app",
			ExpectedURL:  "http://host",
		},
		{
			Name: "different elements",
			HTML: `<div class="h-app">
  <h1 class="p-name">my app</h1>
  <a class="u-url" href="http://host">[here]</a>
</div>`,
			ExpectedName: "my app",
			ExpectedURL:  "http://host",
		},
		{
			Name: "no explicit name",
			HTML: `<div class="h-app">
  <a class="u-url" href="http://host">[here]</a>
</div>`,
			ExpectedName: "[here]",
			ExpectedURL:  "http://host",
		},
		{
			Name: "no url",
			HTML: `<div class="h-app">
  <span class="p-name">my app</span>
</div>`,
			ExpectedName: "my app",
			ExpectedURL:  "",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			r := strings.NewReader(testCase.HTML)

			app, err := ParseApp(r, nil)
			assert.Nil(t, err)
			assert.Equal(t, testCase.ExpectedName, app.Name)
			assert.Equal(t, testCase.ExpectedURL, app.URL)
		})
	}
}

func TestParseAppWhenNone(t *testing.T) {
	r := strings.NewReader("")

	_, err := ParseApp(r, nil)
	assert.Equal(t, ErrNoApp, err)
}
