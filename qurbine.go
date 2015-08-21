package sg

import ()

// Offspring handles stuff.
type Offspring struct {
	Channels map[*RequestXML]chan *RequestXML // Stores the channel of a given request.
}
