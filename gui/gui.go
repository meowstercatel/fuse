package gui

import (
	"encoding/json"
	"fmt"
	"image/color"
	"time"

	"github.com/AllenDang/giu"
	g "github.com/AllenDang/giu"
	"github.com/unknown321/fuse/message"
)

type AppState struct {
	MessageLog     []message.Message
	MessageChannel chan int
}

type privateState struct {
	MessageStrings   []string
	rules            []Rule
	ruleTableWidgets []*g.TableRowWidget
}

type Rule struct {
	Enabled   bool
	Name      string
	Request   bool
	Cmd       string
	JsonPatch string
}

var privatestate privateState
var appstate *AppState
var openRuleWindow bool = false

func prettyPrint(msg message.Message) (string, error) {
	// Outer struct as map so we can replace Data with parsed JSON
	outer := map[string]any{
		"compress":       msg.Compress,
		"data":           msg.Data,
		"original_size":  msg.OriginalSize,
		"session_crypto": msg.SessionCrypto,
		"session_key":    msg.SessionKey,
	}

	// Try to parse Data as nested JSON
	var nested any
	if err := json.Unmarshal([]byte(msg.Data), &nested); err == nil {
		outer["data"] = nested
	}

	b, err := json.MarshalIndent(outer, "", "  ")
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func handleMessages(appstate *AppState, privatestate *privateState) {
	for {
		msgType := <-appstate.MessageChannel
		latestMessage := appstate.MessageLog[len(appstate.MessageLog)-1]

		if msgType == 0 {
			privatestate.MessageStrings = append(privatestate.MessageStrings, "req - "+latestMessage.MsgID.String())
		} else {
			privatestate.MessageStrings = append(privatestate.MessageStrings, "res - "+latestMessage.MsgID.String())
		}
		giu.Update()
	}
}

var (
	editor     *g.CodeEditorWidget
	ruleEditor *g.CodeEditorWidget

	name                   string
	items                  []string
	itemSelected           int32
	checked                bool
	checked2               bool
	dragInt                int32
	multiline              string
	radioOp                int
	autoCompleteCandidates = []string{"hello", "hello world"}
	date                   = time.Now()
	col                    = &color.RGBA{}
	sashPos                = float32(320)
)

var currentRule Rule

func saveRule() {
	currentRule.Enabled = true
	privatestate.rules = append(privatestate.rules, currentRule)
	currentRule = Rule{}
}

func rebuildRuleTable() {
	lastRule := privatestate.rules[len(privatestate.rules)-1]
	tableRow := g.TableRow()
}

func InitGui(Appstate *AppState) {
	appstate = Appstate
	privatestate = privateState{}
	go handleMessages(appstate, &privatestate)

	w := g.NewMasterWindow("Overview", 1000, 800, 0)

	// courierNew := g.Context.FontAtlas.AddFont("Courier New", 16)
	for _, font := range g.Context.FontAtlas.GetDefaultFonts() {
		fmt.Println(font.String())
	}
	g.Context.FontAtlas.SetDefaultFont("Calibri", 16)
	// g.Style().SetFont(courierNew)

	editor = g.CodeEditor().ShowWhitespaces(true).LanguageDefinition(g.LanguageDefinitionJSON).Border(true)
	ruleEditor = g.CodeEditor().ShowWhitespaces(true).LanguageDefinition(g.LanguageDefinitionJSON).Border(true)

	w.Run(loop)
}

func loop() {
	g.SingleWindowWithMenuBar().Layout(
		g.TabBar().TabItems(
			g.TabItem("ListBox").Layout(
				giu.SplitLayout(giu.DirectionVertical, &sashPos,
					g.ListBox(privatestate.MessageStrings).OnChange(func(selectedIndex int) {
						fmt.Printf("selected index: %d\n", selectedIndex)
						fmt.Println("val:", appstate.MessageLog[selectedIndex])

						content, _ := prettyPrint(appstate.MessageLog[selectedIndex])
						editor.Text(string(content))
					}),
					editor,
				)),
			g.TabItem("Rules").Layout(
				g.Column(
					g.Button("create rule").OnClick(func() {
						openRuleWindow = true
					}),
					g.Table().
						Columns(
							g.TableColumn("Name"),
							g.TableColumn("CMD"),
							g.TableColumn("Enabled"),
							g.TableColumn("req/res"),
						).
						Rows(
							g.TableRow(g.Label("Loooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooog").Wrapped(true), g.Label("Age"), g.Label("Loc")),
							g.TableRow(g.Label("Second Loooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooog").Wrapped(true), g.Label("Age"), g.Label("Loc")),
							g.TableRow(g.Label("Name"), g.Label("Age"), g.Label("Location")),
							g.TableRow(g.Label("Allen"), g.Label("33"), g.Label("Shanghai/China")),
							g.TableRow(g.Checkbox("check me", &checked), g.Button("click me"), g.Label("Anything")),
						),
				)),
		),
	)

	if openRuleWindow {
		g.Window("Create Rule").Pos(100, 100).Size(600, 400).Layout(
			g.Column(
				g.Row(
					g.Button("save").OnClick(saveRule),
					g.Button("abandon").OnClick(func() {
						openRuleWindow = false
						currentRule = Rule{}
					}),
				),
				g.Row(g.Label("rule name: "), g.InputText(&currentRule.Name)),
				g.Row(g.Label("rule trigger (cmd name): "), g.InputText(&currentRule.Cmd)),
				g.Row(
					g.Label("rule type: "),
					g.RadioButton("request", currentRule.Request).OnChange(func() { currentRule.Request = !currentRule.Request }),
					g.RadioButton("response", !currentRule.Request).OnChange(func() { currentRule.Request = !currentRule.Request }),
				),
				g.Label("patch:"),
				ruleEditor,
			),
		)
	}
}
