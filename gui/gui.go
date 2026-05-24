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
	Block     bool
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
	openRuleWindow = false
	currentRule.Enabled = true
	shouldCreate := true
	currentRule.JsonPatch = ruleEditor.GetText()
	ruleEditor.Text("")
	for i, rule := range privatestate.rules {
		if currentRule.Name == rule.Name {
			//update
			privatestate.rules[i] = currentRule
			shouldCreate = false
		}
	}
	if shouldCreate {
		privatestate.rules = append(privatestate.rules, currentRule)
	}
	currentRule = Rule{}

	rebuildRuleTable()
}

func rebuildRuleTable() {
	// lastRule := privatestate.rules[len(privatestate.rules)-1]
	privatestate.ruleTableWidgets = privatestate.ruleTableWidgets[0:0] //clear array

	for _, rule := range privatestate.rules {
		var ruleType string
		if rule.Request {
			ruleType = "req"
		} else {
			ruleType = "res"
		}

		privatestate.ruleTableWidgets = append(privatestate.ruleTableWidgets,
			g.TableRow(g.Label(rule.Name), g.Label(rule.Cmd), g.Checkbox("enabled", &rule.Enabled), g.Label(ruleType), g.Button("modify").OnClick(func() {
				openUpdateRuleWindow(rule.Name)
			})),
		)
	}

	giu.Update()
}

func openUpdateRuleWindow(ruleName string) {
	for _, rule := range privatestate.rules {
		if rule.Name == ruleName {
			currentRule = rule
		}
	}
	ruleEditor.Text(currentRule.JsonPatch)
	openRuleWindow = true
}

func InitGui(Appstate *AppState) {
	appstate = Appstate
	privatestate = privateState{}
	go handleMessages(appstate, &privatestate)

	w := g.NewMasterWindow("Overview", 1000, 800, 0)
	g.Context.FontAtlas.SetDefaultFont("Calibri", 16)

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
							g.TableColumn("modify"),
						).
						Rows(privatestate.ruleTableWidgets...),
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
				g.Row(g.Label("rule name (unique): "), g.InputText(&currentRule.Name)),
				g.Row(g.Label("rule trigger (cmd name): "), g.InputText(&currentRule.Cmd)),
				g.Row(
					g.Label("rule type: "),
					g.RadioButton("request", currentRule.Request).OnChange(func() { currentRule.Request = !currentRule.Request }),
					g.RadioButton("response", !currentRule.Request).OnChange(func() { currentRule.Request = !currentRule.Request }),
				),
				g.Checkbox("block CMD", &currentRule.Block),
				g.Label("patch:"),
				ruleEditor,
			),
		)
	}
}
