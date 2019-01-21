package microformats

import (
	"hawx.me/code/relme"
)

type EventType uint

const (
	Error EventType = iota
	Found
	Verified
	Unverified
)

type Event struct {
	Type EventType
	Link string
	Err  error
}

func Me(profile string) <-chan Event {
	eventCh := make(chan Event)

	go func() {
		profileLinks, err := relme.Find(profile)
		if err != nil {
			eventCh <- Event{Type: Error, Err: err}
			close(eventCh)
			return
		}

		for _, link := range profileLinks {
			eventCh <- Event{Type: Found, Link: link}
		}

		for _, link := range profileLinks {
			ok, err := relme.LinksTo(link, profile)
			if err != nil {
				eventCh <- Event{Type: Error, Link: link, Err: err}
			} else if ok {
				eventCh <- Event{Type: Verified, Link: link}
			} else {
				eventCh <- Event{Type: Unverified, Link: link}
			}
		}

		close(eventCh)
	}()

	return eventCh
}
