package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"os"
	"strconv"
	"strings"
	"text/template"

	"github.com/unknown321/fuse/challengetask"
	"github.com/unknown321/fuse/tppmessage"
)

type entry struct {
	LangID string `xml:"LangId,attr"`
	Value  string `xml:"Value,attr"`
}

type LangFile struct {
	Entries []entry `xml:"Entries>Entry"`
}

func FromLangFile() []challengetask.TaskRewardEntry {
	f, err := os.ReadFile("./tpp_challenge_task.eng.lng2.xml")
	if err != nil {
		panic(err)
	}

	ll := &LangFile{}
	if err = xml.Unmarshal(f, ll); err != nil {
		panic(err)
	}

	var res []challengetask.TaskRewardEntry

	for _, v := range ll.Entries {
		nopref := strings.TrimPrefix(v.LangID, "challenge_task_")
		id, _ := strconv.ParseInt(nopref, 10, 32)
		ww := challengetask.TaskRewardEntry{
			ID:     int(id),
			Name:   v.Value,
			Reward: tppmessage.CmdGetChallengeTaskRewardsReward{},
		}
		res = append(res, ww)
	}

	return res
}

type jsonEntry struct {
	Reward struct {
		BottomType int `json:"bottom_type"`
		Rate       int
		Section    int
		Type       int
		Value      int
	}
	TaskID int `json:"task_id"`
}

type jsonStruct struct {
	TaskList []jsonEntry `json:"task_list"`
}

var temp = `package sessionmanager

import "github.com/unknown321/fuse/tppmessage"

var ChallengeTaskRewardEntries = []TaskRewardEntry{
{{- range . }}
	{
		ID: {{ .ID }},
		Name: ` + "`" + `{{ .Name }}` + "`" + `,
		Reward: tppmessage.CmdGetChallengeTaskRewardsReward{
			BottomType: {{ .Reward.BottomType }},
			Rate:       {{ .Reward.Rate }},
			Section:    {{ .Reward.Section }},
			Type:       {{ .Reward.Type }},
			Value:      {{ .Reward.Value }},
		},
	},
{{- end }}
}
`

func FromJson() *jsonStruct {
	data, err := os.ReadFile("./values.json")
	if err != nil {
		panic(err)
	}

	jj := &jsonStruct{}
	if err = json.Unmarshal(data, jj); err != nil {
		panic(err)
	}

	return jj
}

func main() {
	entries := FromLangFile()
	jj := FromJson()

	for i := range entries {
		for _, v := range jj.TaskList {
			if entries[i].ID == v.TaskID {
				entries[i].Reward.BottomType = v.Reward.BottomType
				entries[i].Reward.Rate = v.Reward.Rate
				entries[i].Reward.Section = v.Reward.Section
				entries[i].Reward.Type = v.Reward.Type
				entries[i].Reward.Value = v.Reward.Value
				break
			}
		}
	}

	t, err := template.New("a").Parse(temp)
	if err != nil {
		panic(err)
	}

	o := []byte{}
	res := bytes.NewBuffer(o)
	if err = t.Execute(res, entries); err != nil {
		panic(err)
	}

	if err = os.WriteFile("../challenge_tasks.go", res.Bytes(), 0644); err != nil {
		panic(err)
	}
}
