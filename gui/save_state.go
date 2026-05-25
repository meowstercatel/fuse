package gui

import (
	"encoding/json"
	"log/slog"
	"os"

	guistructs "github.com/unknown321/fuse/gui_structs"
)

func saveConfig() {
	AppStateJson, err := json.Marshal(appstate)
	if err != nil {
		slog.Error("error while marshalling appstate", err)
	}
	err = os.WriteFile("config.json", AppStateJson, 0644)
	if err != nil {
		slog.Error("error while writing config", err)
	}
}

func loadConfig() {
	AppState, err := os.ReadFile("config.json")
	if err != nil {
		slog.Error("error reading config file", err)
	}
	var AppStateConfig guistructs.AppState
	err = json.Unmarshal(AppState, &AppStateConfig)
	if err != nil {
		slog.Error("error while unmarshalling appstate", err)
	}
	if appstate != nil {
		AppStateConfig.MessageChannel = appstate.MessageChannel
		*appstate = AppStateConfig
		appstate.MessageChannel = AppStateConfig.MessageChannel
	}
}
