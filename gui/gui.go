package gui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"sort"

	"github.com/AllenDang/giu"
	g "github.com/AllenDang/giu"
	"github.com/unknown321/fuse/coder"
	guistructs "github.com/unknown321/fuse/gui_structs"
	"github.com/unknown321/fuse/handlers"
	"github.com/unknown321/fuse/message"
)

var privatestate guistructs.PrivateState
var appstate *guistructs.AppState
var openRuleWindow bool = false
var session_key string
var coderClass *coder.Coder

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
		if showRawMessage {
			outer["data"] = nested
		} else {
			outer = nested.(map[string]any)
		}
	}

	b, err := json.MarshalIndent(outer, "", "  ")
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func handleMessages(appstate *guistructs.AppState, privatestate *guistructs.PrivateState) {
	for {
		msgType := <-appstate.MessageChannel
		latestMessage := appstate.MessageLog[len(appstate.MessageLog)-1]

		if latestMessage.SessionKey != nil {
			session_key = *latestMessage.SessionKey
			coderClass.WithKey([]byte(session_key))
		}

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
	creatorEditor    *g.CodeEditorWidget
	responseEditor   *g.CodeEditorWidget
	ruleEditor       *g.CodeEditorWidget
	treeListElements []*g.TreeTableRowWidget

	sashPos                = float32(320)
	showRawMessage         = false
	currentSelectedContent int
)

var currentRule guistructs.Rule

func syncCoderFromSessionKey() error {
	if coderClass == nil {
		coderClass = &coder.Coder{}
	}

	if err := coderClass.WithKey([]byte(session_key)); err != nil {
		return fmt.Errorf("cannot initialize coder: %w", err)
	}

	return nil
}

func parseWireMessage(content string) (WireMessage, error) {
	var wire WireMessage
	if err := json.Unmarshal([]byte(content), &wire); err != nil {
		return WireMessage{}, fmt.Errorf("unmarshal wire message: %w", err)
	}

	return wire, nil
}

func buildRequestMessage(wire WireMessage) (message.Message, error) {
	requestMessage := message.Message{
		Compress:      wire.Compress,
		OriginalSize:  wire.OriginalSize,
		SessionCrypto: wire.SessionCrypto,
		SessionKey:    &session_key,
		MData:         wire.Data,
	}

	requestMessage.WithCoder(coderClass)

	if requestMessage.Compress {
		if err := requestMessage.DoCompress(); err != nil {
			return message.Message{}, fmt.Errorf("compress request: %w", err)
		}
	}

	return requestMessage, nil
}

func decodeResponseMessage(responseBody []byte) (message.Message, error) {
	responseMessage := message.Message{}
	responseMessage.WithCoder(coderClass)
	if err := responseMessage.Decode(responseBody); err != nil {
		return message.Message{}, fmt.Errorf("decode response: %w", err)
	}

	return responseMessage, nil
}

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
	currentRule = guistructs.Rule{}

	saveConfig()
	rebuildRuleTable()
}

func rebuildRuleTable() {
	// lastRule := privatestate.Rules[len(privatestate.Rules)-1]
	privatestate.RuleTableWidgets = privatestate.RuleTableWidgets[0:0] //clear array

	for _, rule := range appstate.Rules {
		var ruleType string
		if rule.Request {
			ruleType = "req"
		} else {
			ruleType = "res"
		}

		privatestate.RuleTableWidgets = append(privatestate.RuleTableWidgets,
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

	rule := guistructs.Rule{
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

func InitGui(Appstate *guistructs.AppState) {
	coderClass = &coder.Coder{}
	appstate = Appstate
	privatestate = guistructs.PrivateState{}
	loadConfig()
	go handleMessages(appstate, &privatestate)

	w := g.NewMasterWindow("Overview", 1000, 800, 0)
	g.Context.FontAtlas.SetDefaultFont("Consola.ttf", 14)

	editor = g.CodeEditor().ShowWhitespaces(true).LanguageDefinition(g.LanguageDefinitionJSON).Border(true)
	ruleEditor = g.CodeEditor().ShowWhitespaces(true).LanguageDefinition(g.LanguageDefinitionJSON).Border(true)
	creatorEditor = g.CodeEditor().ShowWhitespaces(true).LanguageDefinition(g.LanguageDefinitionJSON).Border(true)
	responseEditor = g.CodeEditor().ShowWhitespaces(true).LanguageDefinition(g.LanguageDefinitionJSON).Border(true)

	// asdRule := Rule{
	// 	Enabled: true,
	// 	Block:   false,
	// 	Name:    "CMD_GET_INFORMATIONLIST2",
	// 	Cmd:     "CMD_GET_INFORMATIONLIST2",
	// 	Request: false,
	// 	JsonPatch: `{
	//   "info_list": [
	//     {
	//       "date": 1779116400,
	//       "important": "TRUE",
	//       "info_id": 13000,
	//       "mes_body": "\u003cI=C=cmn-col-special|** rules are working!! **\u003e\n\nyup\u003e",
	//       "mes_subject": ""
	//     }
	//   ]
	// }`,
	// }
	// appstate.Rules = append(appstate.Rules, asdRule)
	rebuildRuleTable() //show config rule entries

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
		return g.TreeTableRow(key, g.Selectable("null"))
	default:
		return g.TreeTableRow(key, g.Label(fmt.Sprintf("%v", t)))
	}
}

type WireMessage struct {
	Compress      bool            `json:"compress"`
	Data          json.RawMessage `json:"data"`
	OriginalSize  int             `json:"original_size"`
	SessionCrypto bool            `json:"session_crypto"`
}

func sendKojiPro() {
	messageContent := creatorEditor.GetText()

	if err := syncCoderFromSessionKey(); err != nil {
		slog.Error("prepare coder", "err", err)
		return
	}

	wire, err := parseWireMessage(messageContent)
	if err != nil {
		slog.Error("unmarshal wire message", "err", err)
		return
	}

	requestMessage, err := buildRequestMessage(wire)
	if err != nil {
		slog.Error("build request message", "err", err)
		return
	}
	fmt.Println(requestMessage)
	encoded, err := requestMessage.Encode()
	if err != nil {
		slog.Error("encode request", "err", err)
		return
	}
	vals := url.Values{}
	vals.Set("httpMsg", string(encoded))
	toSend := vals.Encode()
	slog.Info(toSend)

	resp, err := handlers.ToKojiPro("/tppstm/main", bytes.NewReader([]byte(toSend)), int64(len(toSend)))
	if err != nil {
		slog.Error("err sendKojiPro", "err", err)
		return
	}
	defer resp.Body.Close()
	slog.Info("kojiPro response status", "status", resp.StatusCode, "contentLength", resp.ContentLength)

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("read kojiPro response", "err", err)
		return
	}

	slog.Info("kojiPro response body", "len", len(responseBody), "body", string(responseBody))
	if len(responseBody) < 8 {
		slog.Error("short kojiPro response", "len", len(responseBody), "body", string(responseBody))
		return
	}

	responseMessage, err := decodeResponseMessage(responseBody)
	if err != nil {
		slog.Error("decode kojiPro response", "err", err)
		return
	}
	fmt.Println(responseMessage)

	responseMessageJson, err := json.Marshal(responseMessage)
	if err != nil {
		slog.Error("marshal response message", "err", err)
		return
	}

	responseEditor.Text(string(responseMessageJson))
}

func loop() {
	g.SingleWindowWithMenuBar().Layout(
		g.MenuBar().Layout(g.Menu("settings").Layout(
			g.Checkbox("show raw messages (not just data)", &showRawMessage),
		)),
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
						Rows(privatestate.RuleTableWidgets...),
				)),
			g.TabItem("Creator").Layout(
				giu.SplitLayout(giu.DirectionVertical, &sashPos,
					g.Column(
						g.Row(g.Button("+"), g.Button("-")),
						g.ListBox(privatestate.MessageStrings).OnChange(func(selectedIndex int) {
							//request list like in postman/insomnia
							fmt.Printf("selected index: %d\n", selectedIndex)
						})),
					g.Column(
						g.Row(
							g.Button("send").OnClick(sendKojiPro),
						),
						g.TabBar().TabItems(
							g.TabItem("editor").Layout(
								creatorEditor,
							),
							g.TabItem("response").Layout(
								g.TabBar().TabItems(
									g.TabItem("JSON").Layout(
										responseEditor,
									),
									g.TabItem("Tree").Layout(
										g.TreeTable().
											Columns(g.TableColumn("Name"), g.TableColumn("Value")).
											Rows(treeListElements...).
											Size(g.Auto, g.Auto),
									),
								),
							),
						),
					),
				),
			),
		),
	)

	if openRuleWindow {
		g.Window("Create Rule").Pos(100, 100).Size(600, 400).Layout(
			g.Column(
				g.Row(
					g.Button("save").OnClick(saveRule),
					g.Button("close").OnClick(func() {
						openRuleWindow = false
						currentRule = guistructs.Rule{}
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
