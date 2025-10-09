package sessionmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/unknown321/fuse/message"
	"github.com/unknown321/fuse/tppmessage"
)

func HandleCmdGetFobTargetListRequest(ctx context.Context, msg *message.Message, manager *SessionManager) error {
	var err error
	t := tppmessage.CmdGetFobTargetListRequest{}
	err = json.Unmarshal(msg.MData, &t)
	if err != nil {
		return fmt.Errorf("cannot unmarshal: %w", err)
	}

	slog.Info("CMD_GET_FOB_TARGET_LIST", "type", t.Type)

	data := FromJSON(ctx, t.Msgid)
	if data != nil {
		msg.MData = data
		return nil
	}

	d := GetCmdGetFobTargetListResponse(ctx, msg, manager, &t)
	d.Type = t.Type

	msg.MData, err = json.Marshal(d)
	if err != nil {
		return fmt.Errorf("cannot marshal: %w", err)
	}

	return nil
}
