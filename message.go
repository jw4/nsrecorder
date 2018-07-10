package nsrecorder // import "jw4.us/nsrecorder"

import "time"

// Header is the simplified structure for deserialization.
type Header struct {
	Name string
}

// Question is the simplified structure for deserialization.
type Question struct {
	Name string
}

// Answer is the simplified structure for deserialization.
type Answer struct {
	Hdr  Header
	A    string
	AAAA string
}

// Msg is the simplified structure for deserialization.
type Msg struct {
	ID       int
	Question []Question
	Answer   []Answer
}

// Message is the simplified structure for deserialization.
type Message struct {
	ClientIP string
	Time     time.Time
	Msg      Msg
}
