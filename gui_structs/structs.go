package guistructs

import (
	g "github.com/AllenDang/giu"
	"github.com/unknown321/fuse/message"
)

type Rule struct {
	Enabled   bool   `json:"enabled"`
	Block     bool   `json:"block"`
	Name      string `json:"name"`
	Request   bool   `json:"request"`
	Cmd       string `json:"cmd"`
	JsonPatch string `json:"jsonPatch"`
}

type Request struct {
	Name string
	Cmd  string
	Body string
}

type AppState struct {
	MessageLog     []message.Message
	Rules          []Rule
	Requests       []Request
	MessageChannel chan int
}

type PrivateState struct {
	MessageStrings   []string
	RuleTableWidgets []*g.TableRowWidget
}
