package gui

import (
	"encoding/json"
	"log/slog"
	"os"
)

func saveConfig() {
	ruleArrayJson, err := json.Marshal(appstate.Rules)
	if err != nil {
		slog.Error("error while marshalling rules", err)
	}
	err = os.WriteFile("config.json", ruleArrayJson, 0644)
	if err != nil {
		slog.Error("error while writing config", err)
	}
}

func loadConfig() {
	ruleArrayJson, err := os.ReadFile("config.json")
	if err != nil {
		slog.Error("error reading config file", err)
	}
	var ruleArray []Rule
	err = json.Unmarshal(ruleArrayJson, ruleArray)
	if err != nil {
		slog.Error("error while unmarshalling rules", err)
	}
	appstate.Rules = ruleArray
}
