package sessionmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/unknown321/fuse/fobtargettype"
	"github.com/unknown321/fuse/message"
	"github.com/unknown321/fuse/player"
	"github.com/unknown321/fuse/tppmessage"
)

func GetCmdGetFobTargetListResponse(ctx context.Context, msg *message.Message, manager *SessionManager, request *tppmessage.CmdGetFobTargetListRequest) tppmessage.CmdGetFobTargetListResponse {
	t := tppmessage.CmdGetFobTargetListResponse{}
	t.CryptoType = tppmessage.CRYPTO_TYPE_COMPOUND
	t.Msgid = tppmessage.CMD_GET_FOB_TARGET_LIST.String()
	t.Result = tppmessage.RESULT_NOERR
	t.Rqid = 0

	t.Type = request.Type

	t.EventPoint = 0 // always 0? TODO might be "points" in the table, most likely esp point
	t.FobDeployDamageParam = tppmessage.FobDeployDamageParam{
		ClusterIndex:   0,               // always 0
		DamageValues:   make([]int, 16), // always 0
		ExpirationDate: 0,               // always 0
		MotherbaseID:   0,               // always 0
	}
	t.ShieldDate = 0 // always 0

	// player status
	{
		playerStatus, err := manager.PlayerStatusRepo.Get(ctx, msg.PlayerID)
		if err != nil {
			slog.Error("player status", "error", err.Error(), "msgid", t.Msgid, "playerID", msg.PlayerID)
			t.Result = tppmessage.RESULT_ERR
			return t
		}

		t.EnableSecurityChallenge = playerStatus.SecurityChallenge
		t.EspPoint = playerStatus.EspionagePoint
	}

	// fob records
	{
		fobRec, err := manager.FobRecordRepo.GetByPlayerID(ctx, msg.PlayerID)
		if err != nil {
			slog.Error("fob record", "error", err.Error(), "msgid", t.Msgid, "playerID", msg.PlayerID)
			t.Result = tppmessage.RESULT_ERR
			return t
		}

		if len(fobRec) < 1 {
			slog.Error("fob record not found", "error", err.Error(), "msgid", t.Msgid, "playerID", msg.PlayerID)
			t.Result = tppmessage.RESULT_ERR
			return t
		}

		t.Lose = fobRec[0].SneakLose
		t.Win = fobRec[0].SneakWin
	}

	var players []player.Player
	var err error
	intrudedFOBID := -1
	if t.Type == fobtargettype.EMERGENCY.String() {
		// TODO must include followers' bases too

		slog.Info("looking for emergency fobs in intruder table", "playerID", msg.PlayerID)
		ii, err := manager.IntruderRepo.GetByOwnerID(ctx, msg.PlayerID)
		if err != nil {
			slog.Error("get intruder", "error", err.Error(), "playerID", msg.PlayerID)
			t.Result = tppmessage.RESULT_ERR
			return t
		}

		// get owner player record
		for _, v := range ii {
			pl, err := manager.PlayerRepo.GetByID(ctx, msg.Platform, v.OwnerID)
			if err != nil {
				slog.Error("get owner player", "error", err.Error(), "ownerID", v.OwnerID)
				t.Result = tppmessage.RESULT_ERR
				return t
			}
			intrudedFOBID = v.MotherBaseID
			players = append(players, pl)
		}
	} else {
		players, err = manager.PlayerRepo.GetAll(ctx)
		if err != nil {
			slog.Error("players", "error", err.Error(), "msgid", t.Msgid, "playerID", msg.PlayerID)
			t.Result = tppmessage.RESULT_ERR
			return t
		}
	}

	for _, p := range players {
		entry := tppmessage.TargetEntry{}

		mbp, err := manager.MotherBaseParamRepo.GetByPlayerID(ctx, p.ID)
		if err != nil {
			slog.Error("mother base param", "error", err.Error(), "msgid", t.Msgid, "playerID", p.ID)
			t.Result = tppmessage.RESULT_ERR
			return t
		}

		for _, v := range mbp {
			if t.Type == fobtargettype.EMERGENCY.String() {
				if v.ID != intrudedFOBID {
					continue
				}
			}

			entry.MotherBaseParam = append(entry.MotherBaseParam, tppmessage.MotherBaseParam{
				AreaID:         0,                           // always 0 here
				ClusterParam:   []tppmessage.ClusterParam{}, // empty here
				ConstructParam: v.ConstructParam,
				FobIndex:       v.FobIndex,
				MotherBaseID:   v.ID,
				PlatformCount:  v.PlatformCount,
				Price:          0, //always 0?
				SecurityRank:   v.SecurityRank,
			})
		}

		entry.AttackerInfo = tppmessage.FobPlayerInfo{
			PlayerID:   0,
			PlayerName: "NotImplement",
			Ugc:        0,
			Xuid:       0,
		}

		fr, err := manager.FobRecordRepo.GetByPlayerID(ctx, p.ID)
		if err != nil {
			slog.Error("players fob record", "error", err.Error(), "msgid", t.Msgid, "playerID", p.ID)
			t.Result = tppmessage.RESULT_ERR
			return t
		}

		if len(fr) < 1 {
			slog.Error("players fob record no record", "msgid", t.Msgid, "playerID", p.ID)
			t.Result = tppmessage.RESULT_ERR
			return t
		}

		playerStatus, err := manager.PlayerStatusRepo.Get(ctx, p.ID)
		if err != nil {
			slog.Error("player status", "error", err.Error(), "msgid", t.Msgid, "playerID", p.ID)
			t.Result = tppmessage.RESULT_ERR
			return t
		}

		resource, err := manager.PlayerResourceRepo.Get(ctx, p.ID, true)
		if err != nil {
			slog.Error("resource", "error", err.Error(), "msgid", t.Msgid, "playerID", p.ID)
			t.Result = tppmessage.RESULT_ERR
			return t
		}
		if len(resource) < 1 {
			slog.Error("resource no record", "msgid", t.Msgid, "playerID", p.ID)
			t.Result = tppmessage.RESULT_ERR
			return t
		}

		entry.OwnerFobRecord = tppmessage.OwnerFobRecord{
			AttackCount:    fr[0].SneakWin + fr[0].SneakLose, // might be wrong
			AttackGmp:      0,                                // TODO only INJURY
			CaptureNuclear: 0,                                // TODO maybe NUCLEAR only?
			CaptureResource: tppmessage.CmdMiningResourceEntry{ // TODO only INJURY
				BioticResource: 0,
				CommonMetal:    0,
				FuelResource:   0,
				MinorMetal:     0,
				PreciousMetal:  0,
			},
			CaptureResourceCount: 0,               // always 0
			CaptureStaff:         0,               // always 0
			CaptureStaffCount:    make([]int, 10), // TODO [0,0,0,0,0,0,0,0,3,0], INJURY, staff per grade
			DateTime:             0,               // INJURY, EVENT
			InjuryStaffCount:     make([]int, 10), // TODO [1,0,0,0,0,0,0,3,12,13], INJURY, staff per grade
			LeftHour:             0,               // TODO FR_ENEMY only, 720,720,720,720,551,298,252,360,720,720, blockade duration? time left to retaliate?
			ProcessingResource: tppmessage.CmdMiningResourceEntry{
				BioticResource: resource[0].Raw.Bio,
				CommonMetal:    resource[0].Raw.CommonMetal,
				FuelResource:   resource[0].Raw.Fuel,
				MinorMetal:     resource[0].Raw.MinorMetal,
				PreciousMetal:  resource[0].Raw.PreciousMetal,
			},
			UsableResource: tppmessage.CmdMiningResourceEntry{
				BioticResource: resource[0].Processed.Bio,
				CommonMetal:    resource[0].Processed.CommonMetal,
				FuelResource:   resource[0].Processed.Fuel,
				MinorMetal:     resource[0].Processed.MinorMetal,
				PreciousMetal:  resource[0].Processed.PreciousMetal,
			},

			StaffCount:     []int{0, 0, 5, 35, 180, 483, 448, 184, 5, 6}, // amount of staff per grade, displayed on fob list page, TODO from database
			SupportCount:   0,                                            // TODO FOLLOW only, amount of times this player supported you?
			SupportedCount: 0,                                            // TODO FOLLOWER only, amount of times you supported this player?
		}

		if t.Type == "INJURY" || t.Type == "TRIAL" {
			entry.OwnerFobRecord.NamePlateID = playerStatus.NamePlateID // INJURY, TRIAL
		}

		// TODO is that a "has nuke" flag?
		if t.Type == fobtargettype.FR_ENEMY.String() || t.Type == fobtargettype.NUCLEAR.String() {
			entry.OwnerFobRecord.Nuclear = 1
		}

		entry.OwnerInfo = tppmessage.FobPlayerInfo{
			Npid: tppmessage.Npid{ // always empty
				Handler: tppmessage.NpidHandler{
					Data:  "",
					Dummy: make([]int, 3),
					Term:  0,
				},
				Opt:      make([]int, 8),
				Reserved: make([]int, 8),
			},
			PlayerID:   p.ID,                                              // must be > 0
			PlayerName: fmt.Sprintf("%d_player%02d", p.PlatformID, p.IDX), // "76561197960287930_player01"
			Ugc:        1,
			Xuid:       p.PlatformID,
		}

		staffCount := 0
		for _, v := range entry.OwnerFobRecord.StaffCount {
			staffCount += v
		}

		entry.OwnerDetailRecord = tppmessage.FobPlayerDetailRecord{
			Enemy: 0, // TODO always 0?
			Espionage: tppmessage.Espionage{
				Lose:    fr[0].SneakLose,
				Score:   playerStatus.EspionagePoint,
				Section: 0, // always 0
				Win:     fr[0].SneakWin,
			},
			Follow:              0,             // TODO 0-1
			Follower:            0,             // TODO 0-1
			Help:                0,             // TODO always 0?
			Hero:                0,             // always 0
			Insurance:           p.IsInsurance, // TODO always 0?
			IsSecurityChallenge: playerStatus.SecurityChallenge,
			LeagueRank: tppmessage.FobRank{
				Grade: p.LeagueGrade,
				Rank:  p.LeagueRank, // PF Rank
				Score: 123,          // TODO ??
			},
			NamePlateID: playerStatus.NamePlateID,
			Online:      0, // TODO check for existing session?
			SneakRank: tppmessage.FobRank{
				Grade: p.FOBGrade,
				Rank:  p.FOBRank, // Espg. Rank
				Score: p.FOBPoint,
			},
			StaffCount: staffCount,
		}

		if t.Type == fobtargettype.NUCLEAR.String() {
			entry.OwnerDetailRecord.Nuclear = 1
		}

		parts, err := manager.EmblemRepo.GetByPlayerID(ctx, p.ID)
		if err != nil {
			slog.Error("emblem", "error", err.Error(), "msgid", t.Msgid, "playerID", p.ID)
			t.Result = tppmessage.RESULT_ERR
			return t
		}

		entry.OwnerDetailRecord.Emblem.Parts = []tppmessage.EmblemPart{}

		for _, part := range parts {
			entry.OwnerDetailRecord.Emblem.Parts = append(entry.OwnerDetailRecord.Emblem.Parts, tppmessage.EmblemPart{
				BaseColor:  part.BaseColor,
				FrameColor: part.FrameColor,
				PositionX:  part.PositionX,
				PositionY:  part.PositionY,
				Rotate:     part.Rotate,
				Scale:      part.Scale,
				TextureTag: part.TextureTag,
			})
		}

		// TODO implement injury
		if t.Type == fobtargettype.INJURY.String() {
			entry.Cluster = 1 // 0-6, type == INJURY only
			entry.IsWin = 1   // 0-1, type == INJURY only
		}

		// TODO figure out param meaning
		if t.Type == fobtargettype.FR_ENEMY.String() {
			entry.IsSneakRestriction = 0 // 0-1, type == FR_ENEMY only
		}

		entry.AttackerEmblem = tppmessage.Emblem{
			Parts: make([]tppmessage.EmblemPart, 0),
		}

		// TODO set attacker for FR_ENEMY

		if t.Type == fobtargettype.EMERGENCY.String() {
			intruders, err := manager.IntruderRepo.GetByOwnerID(ctx, msg.PlayerID)
			if err != nil {
				slog.Error("get intruder", "error", err.Error(), "mbID", entry.MotherBaseParam[0].MotherBaseID, "playerID", msg.PlayerID)
				t.Result = tppmessage.RESULT_ERR
				return t
			}
			for _, ii := range intruders {
				intrPlayer, err := manager.PlayerRepo.GetByID(ctx, msg.Platform, ii.PlayerID)
				if err != nil {
					slog.Error("get intruder player", "error", err.Error(), "mbID", entry.MotherBaseParam[0].MotherBaseID, "playerID", ii.PlayerID)
					t.Result = tppmessage.RESULT_ERR
					return t
				}

				entry.AttackerInfo.PlayerID = intrPlayer.ID
				entry.AttackerInfo.PlayerName = fmt.Sprintf("%d_player%02d", intrPlayer.PlatformID, intrPlayer.IDX) // "76561197960287930_player01"
				entry.AttackerInfo.Xuid = intrPlayer.PlatformID
				entry.AttackerInfo.Ugc = 1
				entry.AttackerSneakRankGrade = intrPlayer.FOBGrade // might be wrong value?

				// TODO emblem
				entry.AttackerEmblem = tppmessage.Emblem{
					Parts: make([]tppmessage.EmblemPart, 0),
				}
			}
		}

		// always empty
		entry.AttackerEspionage = tppmessage.Espionage{}
		entry.AttackerInfo.Npid = tppmessage.Npid{
			Handler: tppmessage.NpidHandler{
				Data:  "",
				Dummy: make([]int, 3),
				Term:  0,
			},
			Opt:      make([]int, 8),
			Reserved: make([]int, 8),
		}
		entry.SneakMode = 0 // always 0

		t.TargetList = append(t.TargetList, entry)
	}

	t.TargetNum = len(t.TargetList)

	return t
}

func HandleCmdGetFobTargetListResponse(message *message.Message, override bool) error {
	if !override {
		return nil
	}

	slog.Info("using overridden version")
	var err error
	//t := GetCmdGetFobTargetListResponse()
	t := tppmessage.CmdGetFobTargetListResponse{}

	message.MData, err = json.Marshal(t)
	if err != nil {
		return fmt.Errorf("cannot marshal: %w", err)
	}

	return nil
}
