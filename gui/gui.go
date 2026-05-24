package gui

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/AllenDang/giu"
	g "github.com/AllenDang/giu"
	"github.com/unknown321/fuse/message"
)

type AppState struct {
	MessageLog     []message.Message
	Rules          []Rule
	MessageChannel chan int
}

type privateState struct {
	MessageStrings   []string
	ruleTableWidgets []*g.TableRowWidget
}

type Rule struct {
	Enabled   bool   `json:"enabled"`
	Block     bool   `json:"block"`
	Name      string `json:"name"`
	Request   bool   `json:"request"`
	Cmd       string `json:"cmd"`
	JsonPatch string `json:"jsonPatch"`
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
	editor           *g.CodeEditorWidget
	ruleEditor       *g.CodeEditorWidget
	treeListElements []*g.TreeTableRowWidget

	sashPos                = float32(320)
	currentSelectedContent int
)

var currentRule Rule

func saveRule() {
	openRuleWindow = false
	currentRule.Enabled = true
	shouldCreate := true
	currentRule.JsonPatch = ruleEditor.GetText()
	ruleEditor.Text("")
	for i, rule := range appstate.Rules {
		if currentRule.Name == rule.Name {
			//update
			appstate.Rules[i] = currentRule
			shouldCreate = false
		}
	}
	if shouldCreate {
		appstate.Rules = append(appstate.Rules, currentRule)
	}
	currentRule = Rule{}

	saveConfig()
	rebuildRuleTable()
}

func rebuildRuleTable() {
	// lastRule := privatestate.Rules[len(privatestate.Rules)-1]
	privatestate.ruleTableWidgets = privatestate.ruleTableWidgets[0:0] //clear array

	for _, rule := range appstate.Rules {
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
	for _, rule := range appstate.Rules {
		if rule.Name == ruleName {
			currentRule = rule
		}
	}
	ruleEditor.Text(currentRule.JsonPatch)
	openRuleWindow = true
}

func openRuleWindowFromContent() {
	message := appstate.MessageLog[currentSelectedContent]
	prettyPrintData, _ := prettyPrint(message)

	rule := Rule{
		Enabled: true,
		Block:   false,
		Name:    message.MsgID.String(),
		Request: message.IsRequest,
		Cmd:     message.MsgID.String(),
	}
	ruleEditor.Text(prettyPrintData)

	currentRule = rule
	openRuleWindow = true
}

func InitGui(Appstate *AppState) {
	appstate = Appstate
	privatestate = privateState{}
	loadConfig()
	go handleMessages(appstate, &privatestate)

	w := g.NewMasterWindow("Overview", 1000, 800, 0)
	g.Context.FontAtlas.SetDefaultFont("Calibri", 16)

	editor = g.CodeEditor().ShowWhitespaces(true).LanguageDefinition(g.LanguageDefinitionJSON).Border(true)
	ruleEditor = g.CodeEditor().ShowWhitespaces(true).LanguageDefinition(g.LanguageDefinitionJSON).Border(true)

	asdRule := Rule{
		Enabled: true,
		Block:   false,
		Name:    "CMD_GET_INFORMATIONLIST2",
		Cmd:     "CMD_GET_INFORMATIONLIST2",
		Request: false,
		JsonPatch: `{
    "info_list": [
      {
        "date": 1779116400,
        "important": "TRUE",
        "info_id": 13000,
        "mes_body": "\u003cI=C=cmn-col-special|** rules are working!! **\u003e\n\nyup\u003e",
        "mes_subject": ""
      }
    ]
  }`,
	}
	appstate.Rules = append(appstate.Rules, asdRule)

	giu.Update()

	w.Run(loop)
}

func traverseJson(messageData string) {
	treeListElements = treeListElements[0:0]

	var data any
	if err := json.Unmarshal([]byte(messageData), &data); err != nil {
		treeListElements = append(treeListElements, g.TreeTableRow("invalid JSON", g.Label(err.Error())))
		return
	}

	switch v := data.(type) {
	case map[string]any:
		// stable ordering
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			treeListElements = append(treeListElements, buildRowFor(k, v[k]))
		}
	default:
		treeListElements = append(treeListElements, buildRowFor("value", v))
	}

}

// buildRowFor converts a key/value into a TreeTableRowWidget (recursively)
func buildRowFor(key string, val any) *g.TreeTableRowWidget {
	switch t := val.(type) {
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		children := make([]*g.TreeTableRowWidget, 0, len(keys))
		for _, k := range keys {
			children = append(children, buildRowFor(k, t[k]))
		}
		return g.TreeTableRow(key, g.Label("")).Children(children...)
	case []any:
		children := make([]*g.TreeTableRowWidget, 0, len(t))
		for i, item := range t {
			children = append(children, buildRowFor(fmt.Sprintf("[%d]", i), item))
		}
		return g.TreeTableRow(key, g.Label(fmt.Sprintf("len=%d", len(t)))).Children(children...)
	case string:
		return g.TreeTableRow(key, g.Label(t))
	case float64:
		return g.TreeTableRow(key, g.Label(fmt.Sprintf("%v", t)))
	case bool:
		return g.TreeTableRow(key, g.Label(fmt.Sprintf("%t", t)))
	case nil:
		return g.TreeTableRow(key, g.Label("null"))
	default:
		return g.TreeTableRow(key, g.Label(fmt.Sprintf("%v", t)))
	}
}

func loop() {
	g.SingleWindowWithMenuBar().Layout(
		g.TabBar().TabItems(
			g.TabItem("ListBox").Layout(
				giu.SplitLayout(giu.DirectionVertical, &sashPos,
					g.ListBox(privatestate.MessageStrings).OnChange(func(selectedIndex int) {
						fmt.Printf("selected index: %d\n", selectedIndex)
						// fmt.Println("val:", appstate.MessageLog[selectedIndex])
						currentSelectedContent = selectedIndex

						content, _ := prettyPrint(appstate.MessageLog[selectedIndex])
						editor.Text(content)

						traverseJson(content)
						giu.Update()
					}),
					g.Column(
						g.Row(
							g.Button("create rule from CMD").OnClick(openRuleWindowFromContent),
						),
						g.TabBar().TabItems(
							g.TabItem("JSON").Layout(
								editor,
							),
							g.TabItem("Tree").Layout(
								g.TreeTable().
									Columns(g.TableColumn("Name"), g.TableColumn("Value")).
									Rows(treeListElements...).
									Size(g.Auto, g.Auto),
							),
						),
					),
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
				// g.Checkbox("block CMD", &currentRule.Block),
				g.Label("patch:"),
				ruleEditor,
			),
		)
	}
}
