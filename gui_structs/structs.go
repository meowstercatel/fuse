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
	Name string `json:"name"`
	// Cmd           string
	Body          string `json:"body"`
	SessionCrypto bool   `json:"sessionCrypto"`
	Compress      bool   `json:"compress"`
}

type Tokens struct {
	SteamID string `json:"userID"` //steamID (user_name in requests)
	Hash    string `json:"hash"`
}

type AppState struct {
	MessageLog     []message.Message `json:"-"`
	Rules          []Rule            `json:"rules"`
	Requests       []Request         `json:"requests"`
	Tokens         Tokens            `json:"tokens"`
	MessageChannel chan int          `json:"-"`
}

type PrivateState struct {
	MessageStrings   []string
	RuleTableWidgets []*g.TableRowWidget

	RequestStrings []string
}
