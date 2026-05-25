package gui

import (
	"bytes"
	"compress/flate"
	"compress/zlib"
	"encoding/base64"
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
	"github.com/unknown321/fuse/util"
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
		if len(appstate.MessageLog) == 0 {
			slog.Warn("message channel received but message log is empty")
			continue
		}
		latestMessage := appstate.MessageLog[len(appstate.MessageLog)-1]

		if latestMessage.SessionKey != nil {
			session_key = *latestMessage.SessionKey
			coderClass.WithKey([]byte(session_key))
		}
		slog.Info("7")

		if msgType == 0 {
			privatestate.MessageStrings = append(privatestate.MessageStrings, "req - "+latestMessage.MsgID.String())
		} else {
			privatestate.MessageStrings = append(privatestate.MessageStrings, "res - "+latestMessage.MsgID.String())
		}
		giu.Update()
		slog.Info("8")
	}
}

var (
	editor           *g.CodeEditorWidget
	creatorEditor    *g.CodeEditorWidget
	responseEditor   *g.CodeEditorWidget
	ruleEditor       *g.CodeEditorWidget
	treeListElements []*g.TreeTableRowWidget

	sashPos        = float32(320)
	editorSize     = float32(320)
	showRawMessage = false

	creatorCompress        = false
	creatorSessionCrypto   = true
	currentSelectedContent int
	currentRequestName     string
	currentRequestID       int
)

var currentRule guistructs.Rule

func syncOuterCoder() error {
	if coderClass == nil {
		coderClass = &coder.Coder{}
	}

	if err := coderClass.WithKey(nil); err != nil {
		return fmt.Errorf("cannot initialize outer coder: %w", err)
	}

	return nil
}

func buildRequestMessage(content string) (message.Message, error) {
	var probe message.Message
	probe.MData = []byte(content)
	if err := probe.GetDataType(); err != nil {
		return message.Message{}, fmt.Errorf("cannot determine msgid from inner payload: %w", err)
	}

	requestMessage := message.Message{
		Compress:      creatorCompress,
		SessionCrypto: creatorSessionCrypto,
		SessionKey:    &session_key,
		MsgID:         probe.MsgID,
		IsRequest:     true,
		MData:         []byte(content),
	}

	requestMessage.OriginalSize = len(requestMessage.MData)

	innerPayload := requestMessage.MData
	if requestMessage.Compress {
		var compressed bytes.Buffer
		writer, err := zlib.NewWriterLevel(&compressed, flate.BestCompression)
		if err != nil {
			return message.Message{}, fmt.Errorf("cannot create zlib writer: %w", err)
		}
		if _, err := writer.Write(innerPayload); err != nil {
			_ = writer.Close()
			return message.Message{}, fmt.Errorf("cannot compress inner payload: %w", err)
		}
		if err := writer.Close(); err != nil {
			return message.Message{}, fmt.Errorf("cannot finish compression: %w", err)
		}
		innerPayload = compressed.Bytes()
	}

	if requestMessage.SessionCrypto {
		if session_key == "" {
			return message.Message{}, fmt.Errorf("session crypto is enabled but session key is empty")
		}
		sessionCoder := coder.Coder{}
		if err := sessionCoder.WithKey([]byte(session_key)); err != nil {
			return message.Message{}, fmt.Errorf("cannot initialize session coder: %w", err)
		}
		innerPayload = sessionCoder.EncodeBlowfish(innerPayload)
	}

	if requestMessage.Compress || requestMessage.SessionCrypto {
		encoded := base64.StdEncoding.EncodeToString(innerPayload)
		lines := util.SplitByteString([]byte(encoded), 76)
		innerPayload = bytes.Join(lines, []byte("\r\n"))
		innerPayload = append(innerPayload, []byte("\r\n")...)
	}

	requestMessage.MData = innerPayload
	requestMessage.WithCoder(coderClass)

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

func deleteRule() {
	openRuleWindow = false
	for i, rule := range appstate.Rules {
		if currentRule.Name == rule.Name {
			appstate.Rules = append(appstate.Rules[:i], appstate.Rules[i+1:]...)
		}
	}

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
	creatorEditor = g.CodeEditor().ShowWhitespaces(true).Border(true)
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
	updateRequestList()

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

func sendKojiPro() {
	messageContent := creatorEditor.GetText()

	if err := syncOuterCoder(); err != nil {
		slog.Error("prepare coder", "err", err)
		return
	}

	requestMessage, err := buildRequestMessage(messageContent)
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

	// responseMessageJson, err := json.Marshal(responseMessage)
	// if err != nil {
	// 	slog.Error("marshal response message", "err", err)
	// 	return
	// }

	prettyJson, _ := prettyPrint(responseMessage)

	responseEditor.Text(prettyJson)
}

func createRequest() {
	request := guistructs.Request{
		Name:          "request",
		Body:          "",
		SessionCrypto: false,
		Compress:      false,
	}
	appstate.Requests = append(appstate.Requests, request)

	saveConfig()

	openRequest(len(appstate.Requests) - 1)
	updateRequestList()
}

func saveRequest() {
	appstate.Requests[currentRequestID] = guistructs.Request{
		Name:          currentRequestName,
		Body:          creatorEditor.GetText(),
		SessionCrypto: creatorSessionCrypto,
		Compress:      creatorCompress,
	}

	saveConfig()
	updateRequestList()
	// appstate.Requests = append(appstate.Requests, request)
}

func openRequest(index int) {
	slog.Info("opening", index)
	request := appstate.Requests[index]

	currentRequestName = request.Name
	creatorEditor.Text(request.Body)
	creatorCompress = request.Compress
	creatorSessionCrypto = request.SessionCrypto
}

func deleteRequest() {
	appstate.Requests = append(appstate.Requests[:currentRequestID], appstate.Requests[currentRequestID+1:]...)
	if len(appstate.Requests) == 0 {
		createRequest()
	}

	saveConfig()
	updateRequestList()
}

func updateRequestList() {
	privatestate.RequestStrings = privatestate.RequestStrings[0:0] //clear array
	for _, request := range appstate.Requests {
		privatestate.RequestStrings = append(privatestate.RequestStrings, request.Name)
	}

	if len(appstate.Requests) == 1 {
		openRequest(0)
	}

	giu.Update()
}

func loop() {
	g.SingleWindowWithMenuBar().Layout(
		g.MenuBar().Layout(g.Menu("settings").Layout(
			g.Checkbox("show raw messages (not just data)", &showRawMessage),
		)),
		g.TabBar().TabItems(
			g.TabItem("Requests").Layout(
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
						g.Row(
							g.Button("+").OnClick(createRequest),
							g.Button("-").OnClick(deleteRequest),
							g.Button("import from requests"),
						),
						g.ListBox(privatestate.RequestStrings).OnChange(openRequest)),
					g.Column(
						g.Row(
							g.Label("name:"),
							g.InputText(&currentRequestName),
							g.Button("save").OnClick(saveRequest),
						),
						g.Row(
							g.Button("send").OnClick(sendKojiPro),
							g.Checkbox("compress", &creatorCompress),
							g.Checkbox("session crypto", &creatorSessionCrypto),
						),

						giu.SplitLayout(giu.DirectionVertical, &editorSize,
							g.Column(g.Label("request"), creatorEditor),
							g.Column(g.Label("response"), responseEditor),
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
					g.Button("delete").OnClick(deleteRule),
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
