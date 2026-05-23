package sessionmanager

import (
	"bytes"
	"compress/flate"
	"compress/zlib"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/unknown321/fuse/abolition"
	"github.com/unknown321/fuse/challengetask"
	"github.com/unknown321/fuse/clusterbuildcost"
	"github.com/unknown321/fuse/clusterparam"
	"github.com/unknown321/fuse/clustersecurity"
	"github.com/unknown321/fuse/coder"
	"github.com/unknown321/fuse/emblem"
	"github.com/unknown321/fuse/equipflag"
	"github.com/unknown321/fuse/equipgrade"
	fobevent "github.com/unknown321/fuse/fobevent/event"
	fobeventreward "github.com/unknown321/fuse/fobevent/reward"
	fobeventtimebonus "github.com/unknown321/fuse/fobevent/timebonus"
	"github.com/unknown321/fuse/fobplaced"
	"github.com/unknown321/fuse/fobrecord"
	"github.com/unknown321/fuse/fobstatus"
	"github.com/unknown321/fuse/fobweaponplacement"
	"github.com/unknown321/fuse/gui"
	"github.com/unknown321/fuse/informationmessage"
	"github.com/unknown321/fuse/intruder"
	"github.com/unknown321/fuse/localbase"
	"github.com/unknown321/fuse/message"
	"github.com/unknown321/fuse/motherbaseparam"
	"github.com/unknown321/fuse/onlinechallengetask"
	onlinechallengetaskplayer "github.com/unknown321/fuse/onlinechallengetask/player"
	"github.com/unknown321/fuse/pfranking"
	"github.com/unknown321/fuse/pfseason"
	"github.com/unknown321/fuse/pfskillstaff"
	"github.com/unknown321/fuse/platform"
	"github.com/unknown321/fuse/player"
	"github.com/unknown321/fuse/playerresource"
	"github.com/unknown321/fuse/playerstatus"
	"github.com/unknown321/fuse/playertask"
	"github.com/unknown321/fuse/ranking/espionageevent"
	"github.com/unknown321/fuse/ranking/pfevent"
	"github.com/unknown321/fuse/sectionstat"
	"github.com/unknown321/fuse/securitylevel"
	"github.com/unknown321/fuse/serveritem"
	"github.com/unknown321/fuse/serverproductparam"
	serverproductparamplayer "github.com/unknown321/fuse/serverproductparam/player"
	"github.com/unknown321/fuse/serverstatus"
	"github.com/unknown321/fuse/servertext"
	"github.com/unknown321/fuse/session"
	"github.com/unknown321/fuse/staffrankrate"
	"github.com/unknown321/fuse/tapeflag"
	"github.com/unknown321/fuse/tppmessage"
	"github.com/unknown321/fuse/user"
	"github.com/unknown321/fuse/util"
)

var override = true

type SignupBonus struct {
	GMP       int
	Resources playerresource.PlayerResource
}

type ManagerOpts struct {
	SignupBonus SignupBonus
}

type SessionManager struct {
	sessions    map[string]*session.Session
	WriteLog    bool
	ManagerOpts ManagerOpts

	Appstate *gui.AppState

	UserRepo                      *user.Repo
	SessionRepo                   *session.Repo
	URLListEntryRepo              *tppmessage.URLListEntryRepo
	PlayerRepo                    *player.Repo
	AbolitionRepo                 *abolition.Repo
	TaskRewardRepo                *challengetask.TaskRewardRepo
	PlayerTaskRepo                *playertask.Repo
	ClusterBuildCostRepo          *clusterbuildcost.Repo
	StaffRankBonusRateRepo        *staffrankrate.Repo
	ServerTextRepo                *servertext.Repo
	EspionageEventRepo            *espionageevent.Repo
	PFEventRepo                   *pfevent.Repo
	FOBEventRewardRepo            *fobeventreward.Repo
	FOBEventTimeBonusRepo         *fobeventtimebonus.Repo
	OnlineChallengeTaskRepo       *onlinechallengetask.Repo
	OnlineChallengeTaskPlayerRepo *onlinechallengetaskplayer.Repo
	ServerProductParamRepo        *serverproductparam.Repo
	ServerProductParamPlayerRepo  *serverproductparamplayer.Repo
	InformationMessageRepo        *informationmessage.Repo
	ServerItemRepo                *serveritem.Repo
	EquipFlagRepo                 *equipflag.Repo
	EquipGradeRepo                *equipgrade.Repo
	TapeFlagRepo                  *tapeflag.Repo
	SecurityLevelRepo             *securitylevel.Repo
	LocalBaseRepo                 *localbase.Repo
	PFSkillStaffRepo              *pfskillstaff.Repo
	ClusterSecurityRepo           *clustersecurity.Repo
	ClusterParamRepo              *clusterparam.Repo
	MotherBaseParamRepo           *motherbaseparam.Repo
	SectionStatRepo               *sectionstat.Repo
	FobRecordRepo                 *fobrecord.Repo
	FobStatusRepo                 *fobstatus.Repo
	PlayerResourceRepo            *playerresource.Repo
	PlayerStatusRepo              *playerstatus.Repo
	FobEventRepo                  *fobevent.Repo
	ServerStatusRepo              *serverstatus.Repo
	PFRankingRepo                 *pfranking.Repo
	PFSeasonRepo                  *pfseason.Repo
	EmblemRepo                    *emblem.Repo
	FOBPlacedRepo                 *fobplaced.Repo
	FOBWeaponPlacementRepo        *fobweaponplacement.Repo
	IntruderRepo                  *intruder.Repo
	LogDir                        string
}

func (m *SessionManager) Init(ctx context.Context) error {
	if m.sessions == nil {
		m.sessions = make(map[string]*session.Session)

		sessions, err := m.SessionRepo.GetAll(ctx)
		if err != nil {
			return fmt.Errorf("session get from repo: %w", err)
		}

		for _, s := range sessions {
			pl, err := m.PlayerRepo.GetByID(ctx, s.Platform, s.PlayerID)
			if err != nil {
				slog.Warn("attempted to restore session for non-existing player", "playerID", s.PlayerID)
				continue
			}
			s.PlatformID = pl.PlatformID
			m.sessions[s.ID] = &s
		}

		slog.Info("restored sessions", "count", len(m.sessions))
	}

	return nil
}

func (m *SessionManager) Add(ctx context.Context, sessionID string, sessionKey []byte, userID int, steamID string) error {
	c := coder.Coder{}
	err := c.WithKey(sessionKey)
	if err != nil {
		return fmt.Errorf("cannot initialize coder for session")
	}

	m.sessions[sessionID] = &session.Session{
		CryptoKey: sessionKey,
		Coder:     c,
	}

	if err = m.SessionRepo.Add(ctx, m.sessions[sessionID]); err != nil {
		return err
	}

	slog.Info("added session", "id", sessionID, "key", base64.StdEncoding.EncodeToString(sessionKey), "userID", userID, "steamID", steamID)

	return nil
}

func (m *SessionManager) SetIP(ctx context.Context, sessionID string, inIP int, inPort int, exIP int, exPort int) error {
	if !m.Exists(sessionID) {
		return fmt.Errorf("setIP, session %s doesnt exist", sessionID)
	}

	if err := m.SessionRepo.SetIP(ctx, sessionID, inIP, inPort, exIP, exPort); err != nil {
		return err
	}

	m.sessions[sessionID].ExPort = exPort
	m.sessions[sessionID].ExIp = exIP
	m.sessions[sessionID].InIp = inIP
	m.sessions[sessionID].InPort = inPort

	return nil
}

func (m *SessionManager) Exists(sessionID string) bool {
	_, ok := m.sessions[sessionID]
	if !ok {
		slog.Info("attempted to get nonexistent session", "id", sessionID)
	}
	return ok
}

// Get result is not modifiable
func (m *SessionManager) Get(sessionID string) (*session.Session, error) {
	s, ok := m.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("get, session %s doesnt exist", sessionID)
	}

	return s, nil
}

func (m *SessionManager) GetByUserID(userID int) (*session.Session, error) {
	for _, s := range m.sessions {
		if s.UserID == userID {
			return s, nil
		}
	}

	return nil, fmt.Errorf("no session found")
}

func (m *SessionManager) GetByPlayerID(playerID int) (*session.Session, error) {
	for _, s := range m.sessions {
		if s.PlayerID == playerID {
			return s, nil
		}
	}

	return nil, fmt.Errorf("no session found")
}

func (m *SessionManager) SetPlayerID(ctx context.Context, sessionID string, playerID int) error {
	if !m.Exists(sessionID) {
		return fmt.Errorf("setPlayerID, session %s doesnt exist", sessionID)
	}

	v := m.sessions[sessionID]
	v.PlayerID = playerID
	m.sessions[sessionID] = v

	if err := m.SessionRepo.SetPlayerID(ctx, sessionID, playerID); err != nil {
		return err
	}

	return nil
}

func (m *SessionManager) Update(ctx context.Context, sessionID string) error {
	if !m.Exists(sessionID) {
		return fmt.Errorf("update, session %s doesnt exist", sessionID)
	}

	m.sessions[sessionID].Timestamp = time.Now().Unix()

	if err := m.SessionRepo.SetTimestamp(ctx, sessionID, m.sessions[sessionID].Timestamp); err != nil {
		return err
	}

	slog.Info("updated session", "id", sessionID)

	return nil
}

func (m *SessionManager) Remove(sessionID string) {
	s, ok := m.sessions[sessionID]
	if !ok {
		slog.Error("attempted to remove nonexistent session", "session", sessionID)
		return
	}

	slog.Debug("removed session", "session", sessionID, "userID", s.UserID)
	delete(m.sessions, sessionID)
}

func (m *SessionManager) HandleRequest(ctx context.Context, msg *message.Message, pltf platform.Platform) error {
	var err error
	switch msg.MsgID {
	case tppmessage.CMD_AUTH_STEAMTICKET:
		err = HandleCmdAuthSteamticketRequest(msg)
	case tppmessage.CMD_REQAUTH_HTTPS:
		err = HandleCmdReqAuthHTTPSRequest(ctx, msg, m, pltf)
	case tppmessage.CMD_SEND_IPANDPORT:
		err = HandleCmdSendIPAndPortRequest(ctx, msg, m)
	case tppmessage.CMD_GET_PLAYERLIST:
		err = HandleCmdGetPlayerlistRequest(ctx, msg, m)
	case tppmessage.CMD_SET_CURRENTPLAYER:
		err = HandleCmdSetCurrentplayerRequest(ctx, msg, m)
	case tppmessage.CMD_CREATE_PLAYER:
		err = HandleCmdCreatePlayerRequest(ctx, msg, m)
	case tppmessage.CMD_GDPR_CHECK:
		err = HandleCmdGDPRCheckRequest(msg)
	case tppmessage.CMD_GET_INFORMATIONLIST2:
		err = HandleCmdGetInformationlist2Request(ctx, msg, m)
	case tppmessage.CMD_GET_URLLIST:
		err = HandleCmdGetUrllistRequest(ctx, msg, m.URLListEntryRepo)
	case tppmessage.CMD_GET_SVRLIST:
		err = HandleCmdGetSvrListRequest(msg)
	case tppmessage.CMD_GET_SVRTIME:
		err = HandleCmdGetSvrTimeRequest(msg)
	case tppmessage.CMD_SYNC_MOTHER_BASE:
		err = HandleCmdSyncMotherBaseRequest(ctx, msg, m)
	case tppmessage.CMD_SYNC_RESOURCE:
		err = HandleCmdSyncResourceRequest(ctx, msg, m)
	case tppmessage.CMD_SYNC_SOLDIER_BIN:
		err = HandleCmdSyncSoldierBinRequest(ctx, msg, m)
	case tppmessage.CMD_MINING_RESOURCE:
		err = HandleCmdMiningResourceRequest(ctx, msg, m)
	case tppmessage.CMD_GET_FOB_STATUS:
		err = HandleCmdGetFobStatusRequest(ctx, msg, m)
	case tppmessage.CMD_GET_ONLINE_PRISON_LIST:
		err = HandleCmdGetOnlinePrisonListRequest(ctx, msg)
	case tppmessage.CMD_GET_FOB_DAMAGE:
		err = HandleCmdGetFobDamageRequest(ctx, msg, m)
	case tppmessage.CMD_GET_FOB_TARGET_LIST:
		err = HandleCmdGetFobTargetListRequest(ctx, msg, m)
	case tppmessage.CMD_GET_FOB_TARGET_DETAIL:
		err = HandleCmdGetFobTargetDetailRequest(ctx, msg, m)
	case tppmessage.CMD_ABORT_MOTHER_BASE:
		err = HandleCmdAbortMotherBaseRequest(msg)
	case tppmessage.CMD_SNEAK_MOTHER_BASE:
		err = HandleCmdSneakMotherBaseRequest(ctx, msg, m)
	case tppmessage.CMD_ACTIVE_SNEAK_MOTHER_BASE:
		err = HandleCmdActiveSneakMotherBaseRequest(msg)
	case tppmessage.CMD_GET_FOB_EVENT_LIST:
		err = HandleCmdGetFobEventListRequest(ctx, msg, m)
	case tppmessage.CMD_SEND_SNEAK_RESULT:
		err = HandleCmdSendSneakResultRequest(ctx, msg, m)
	case tppmessage.CMD_OPEN_WORMHOLE:
		err = HandleCmdOpenWormholeRequest(msg)
	case tppmessage.CMD_GET_ABOLITION_COUNT:
		err = HandleCmdGetAbolitionCountRequest(ctx, msg, m)
	case tppmessage.CMD_SYNC_LOADOUT:
		err = HandleCmdSyncLoadoutRequest(msg)
	case tppmessage.CMD_GET_COMBAT_DEPLOY_RESULT:
		err = HandleCmdGetCombatDeployResultRequest(ctx, msg)
	case tppmessage.CMD_GET_SERVER_ITEM_LIST:
		err = HandleCmdGetServerItemListRequest(ctx, msg, m)
	case tppmessage.CMD_GET_SERVER_ITEM:
		err = HandleCmdGetServerItemRequest(ctx, msg, m)
	case tppmessage.CMD_CHECK_SERVER_ITEM_CORRECT:
		err = HandleCmdCheckServerItemCorrectRequest(msg)
	case tppmessage.CMD_GET_CHALLENGE_TASK_REWARDS:
		err = HandleCmdGetChallengeTaskRewardsRequest(ctx, msg, m)
	case tppmessage.CMD_GET_CHALLENGE_TASK_TARGET_VALUES:
		err = HandleCmdGetChallengeTaskTargetValuesRequest(ctx, msg, m)
	case tppmessage.CMD_GET_RANKING:
		err = HandleCmdGetRankingRequest(ctx, msg, m)
	case tppmessage.CMD_GET_MBCOIN_REMAINDER:
		err = HandleCmdGetMBCoinRemainderRequest(ctx, msg, m)
	case tppmessage.CMD_GET_OWN_FOB_LIST:
		err = HandleCmdGetOwnFobListRequest(ctx, msg, m)
	case tppmessage.CMD_GET_PURCHASABLE_AREA_LIST:
		err = HandleCmdGetPurchasableAreaListRequest(ctx, msg, m)
	case tppmessage.CMD_GET_SECURITY_INFO:
		err = HandleCmdGetSecurityInfoRequest(ctx, msg, m)
	case tppmessage.CMD_UPDATE_SESSION:
		err = HandleCmdUpdateSessionRequest(ctx, msg, m)
	case tppmessage.CMD_SEND_MISSION_RESULT:
		err = HandleCmdSendMissionResultRequest(msg)
	case tppmessage.CMD_SEND_BOOT:
		err = HandleCmdSendBootRequest(ctx, msg)
	case tppmessage.CMD_GET_FOB_NOTICE:
		err = HandleCmdGetFobNoticeRequest(ctx, msg, m)
	case tppmessage.CMD_GET_FOB_PARAM:
		err = HandleCmdGetFobParamRequest(ctx, msg)
	case tppmessage.CMD_GET_LOGIN_PARAM:
		err = HandleCmdGetLoginParamRequest(ctx, msg, m)
	case tppmessage.CMD_GET_SECURITY_SETTING_PARAM:
		err = HandleCmdGetSecuritySettingParamRequest(ctx, msg, m)
	case tppmessage.CMD_GET_RESOURCE_PARAM:
		err = HandleCmdGetResourceParamRequest(ctx, msg, m)
	case tppmessage.CMD_CHECK_DEFENCE_MOTHERBASE:
		err = HandleCmdCheckDefenceMotherbaseRequest(msg)
	case tppmessage.CMD_SYNC_EMBLEM:
		err = HandleCmdSyncEmblemRequest(ctx, msg, m)
	case tppmessage.CMD_GET_WORMHOLE_LIST:
		err = HandleCmdGetWormholeListRequest(ctx, msg, m)
	//case tppmessage.CMD_ADD_FOLLOW:
	//case tppmessage.CMD_APPROVE_STEAM_SHOP:
	//case tppmessage.CMD_CALC_COST_FOB_DEPLOY_REPLACE:
	//case tppmessage.CMD_CALC_COST_TIME_REDUCTION:
	//case tppmessage.CMD_CANCEL_COMBAT_DEPLOY:
	//case tppmessage.CMD_CANCEL_COMBAT_DEPLOY_SINGLE:
	//case tppmessage.CMD_CANCEL_SHORT_PFLEAGUE:
	//case tppmessage.CMD_CHECK_CONSUME_TRANSACTION:
	//case tppmessage.CMD_CHECK_SHORT_PFLEAGUE_ENTERABLE:
	//case tppmessage.CMD_COMMIT_CONSUME_TRANSACTION:
	//case tppmessage.CMD_CONSUME_RESERVE:
	//case tppmessage.CMD_CREATE_NUCLEAR:
	//case tppmessage.CMD_DELETE_FOLLOW:
	//case tppmessage.CMD_DELETE_TROOPS_LIST:
	//case tppmessage.CMD_DEPLOY_FOB_ASSIST:
	//case tppmessage.CMD_DEPLOY_MISSION:
	//case tppmessage.CMD_DESTRUCT_NUCLEAR:
	//case tppmessage.CMD_DESTRUCT_ONLINE_NUCLEAR:
	//case tppmessage.CMD_DEVELOP_SERVER_ITEM:
	//case tppmessage.CMD_DEVELOP_WEPON:
	//case tppmessage.CMD_ELAPSE_COMBAT_DEPLOY:
	//case tppmessage.CMD_ENTER_SHORT_PFLEAGUE:
	//case tppmessage.CMD_EXCHANGE_FOB_EVENT_POINT:
	//case tppmessage.CMD_EXCHANGE_LEAGUE_POINT2:
	//case tppmessage.CMD_EXCHANGE_LEAGUE_POINT:
	//case tppmessage.CMD_EXTEND_PLATFORM:
	//case tppmessage.CMD_GET_CAMPAIGN_DIALOG_LIST:
	//case tppmessage.CMD_GET_COMBAT_DEPLOY_LIST:
	//case tppmessage.CMD_GET_CONTRIBUTE_PLAYER_LIST:
	//case tppmessage.CMD_GET_DAILY_REWARD:
	//case tppmessage.CMD_GET_DEVELOPMENT_PROGRESS:
	//case tppmessage.CMD_GET_ENTITLEMENT_ID_LIST:
	//case tppmessage.CMD_GET_FOB_DEPLOY_LIST:
	//case tppmessage.CMD_GET_FOB_EVENT_DETAIL:
	//case tppmessage.CMD_GET_FOB_EVENT_POINT_EXCHANGE_PARAMS:
	//case tppmessage.CMD_GET_FOB_REWARD_LIST:
	//case tppmessage.CMD_GET_LEAGUE_RESULT:
	//case tppmessage.CMD_GET_NEXT_MAINTENANCE:
	//case tppmessage.CMD_GET_ONLINE_DEVELOPMENT_PROGRESS:
	//case tppmessage.CMD_GET_PAY_ITEM_LIST:
	//case tppmessage.CMD_GET_PF_DETAIL_PARAMS:
	//case tppmessage.CMD_GET_PF_POINT_EXCHANGE_PARAMS:
	//case tppmessage.CMD_GET_PLATFORM_CONSTRUCTION_PROGRESS:
	//case tppmessage.CMD_GET_PLAYER_PLATFORM_LIST:
	//case tppmessage.CMD_GET_PREVIOUS_SHORT_PFLEAGUE_RESULT:
	//case tppmessage.CMD_GET_PURCHASE_HISTORY:
	//case tppmessage.CMD_GET_PURCHASE_HISTORY_NUM:
	//case tppmessage.CMD_GET_RENTAL_LOADOUT_LIST:
	//case tppmessage.CMD_GET_SECURITY_PRODUCT_LIST:
	//case tppmessage.CMD_GET_SHOP_ITEM_NAME_LIST:
	//case tppmessage.CMD_GET_SHORT_PFLEAGUE_RESULT:
	//case tppmessage.CMD_GET_SNEAK_TARGET_LIST:
	//case tppmessage.CMD_GET_STEAM_SHOP_ITEM_LIST:
	//case tppmessage.CMD_GET_TROOPS_LIST:
	//case tppmessage.CMD_INVALID:
	//case tppmessage.CMD_NOTICE_SNEAK_MOTHER_BASE:
	//case tppmessage.CMD_OPEN_STEAM_SHOP:
	//case tppmessage.CMD_PURCHASE_FIRST_FOB:
	//case tppmessage.CMD_PURCHASE_FOB:
	//case tppmessage.CMD_PURCHASE_NUCLEAR_COMPLETION:
	//case tppmessage.CMD_PURCHASE_ONLINE_DEPLOYMENT_COMPLETION:
	//case tppmessage.CMD_PURCHASE_ONLINE_DEVELOPMENT_COMPLETION:
	//case tppmessage.CMD_PURCHASE_PLATFORM_CONSTRUCTION:
	//case tppmessage.CMD_PURCHASE_RESOURCES_PROCESSING:
	//case tppmessage.CMD_PURCHASE_SECURITY_SERVICE:
	//case tppmessage.CMD_PURCHASE_SEND_TROOPS_COMPLETION:
	//case tppmessage.CMD_PURCHASE_WEPON_DEVELOPMENT_COMPLETION:
	//case tppmessage.CMD_RELOCATE_FOB:
	//case tppmessage.CMD_RENTAL_LOADOUT:
	//case tppmessage.CMD_REQAUTH_SESSIONSVR:
	//case tppmessage.CMD_REQUEST_RELIEF:
	//case tppmessage.CMD_RESET_MOTHER_BASE:
	//case tppmessage.CMD_SALE_RESOURCE:
	//case tppmessage.CMD_SEND_DEPLOY_INJURE:
	//case tppmessage.CMD_SEND_HEARTBEAT:
	//case tppmessage.CMD_SEND_NUCLEAR:
	//case tppmessage.CMD_SEND_ONLINE_CHALLENGE_TASK_STATUS:
	//case tppmessage.CMD_SEND_SUSPICION_PLAY_DATA:
	//case tppmessage.CMD_SEND_TROOPS:
	//case tppmessage.CMD_SET_SECURITY_CHALLENGE:
	//case tppmessage.CMD_SPEND_SERVER_WALLET:
	//case tppmessage.CMD_START_CONSUME_TRANSACTION:
	//case tppmessage.CMD_SYNC_RESET:
	//case tppmessage.CMD_SYNC_SOLDIER_DIFF:
	//case tppmessage.CMD_USE_PF_ITEM:
	//case tppmessage.CMD_USE_SHORT_PF_ITEM:
	default:
		slog.Error("unknown command", "command", msg.MsgID.String())
		msg.MData = []byte(fmt.Sprintf(`{"crypto_type": "COMPOUND","flowid": null,"msgid": "%s", "result": "ERR", "rqid": 0, "xuid":null}`, msg.MsgID.String()))
		msg.Compress = false
	}

	return err
}

func (m *SessionManager) HandleResponseFromKojiPro(ctx context.Context, message *message.Message) error {
	var err error
	switch message.MsgID {
	//case tppmessage.CMD_AUTH_STEAMTICKET:
	case tppmessage.CMD_REQAUTH_HTTPS:
		err = HandleCmdReqAuthHTTPSResponse(ctx, message, m)
	//case tppmessage.CMD_SEND_IPANDPORT:
	//case tppmessage.CMD_GET_PLAYERLIST:
	//case tppmessage.CMD_SET_CURRENTPLAYER:
	//case tppmessage.CMD_CREATE_PLAYER:
	//case tppmessage.CMD_GDPR_CHECK:
	//case tppmessage.CMD_REQAUTH_SESSIONSVR:
	//case tppmessage.CMD_SEND_HEARTBEAT:
	//case tppmessage.CMD_NOTICE_SNEAK_MOTHER_BASE:
	//case tppmessage.CMD_GET_INFORMATIONLIST2:
	case tppmessage.CMD_GET_URLLIST:
		err = HandleCmdGetUrllistResponse(ctx, message, override, m.URLListEntryRepo)
		//case tppmessage.CMD_GET_SVRLIST:
		//case tppmessage.CMD_GET_SVRTIME:
		//case tppmessage.CMD_SYNC_MOTHER_BASE:
		//case tppmessage.CMD_SYNC_RESOURCE:
		//case tppmessage.CMD_SYNC_SOLDIER_BIN:
		//case tppmessage.CMD_SYNC_SOLDIER_DIFF:
		//case tppmessage.CMD_SYNC_EMBLEM:
		//case tppmessage.CMD_CREATE_NUCLEAR:
		//case tppmessage.CMD_DESTRUCT_NUCLEAR:
		//case tppmessage.CMD_SEND_NUCLEAR:
		//case tppmessage.CMD_MINING_RESOURCE:
		//case tppmessage.CMD_SYNC_RESET:
		//case tppmessage.CMD_RESET_MOTHER_BASE:
		//case tppmessage.CMD_GET_FOB_STATUS:
		//case tppmessage.CMD_GET_ONLINE_PRISON_LIST:
		//case tppmessage.CMD_GET_FOB_DAMAGE:
		//case tppmessage.CMD_GET_FOB_TARGET_LIST:
		//case tppmessage.CMD_GET_FOB_TARGET_DETAIL:
		//case tppmessage.CMD_ABORT_MOTHER_BASE:
		//case tppmessage.CMD_SNEAK_MOTHER_BASE:
		//case tppmessage.CMD_ACTIVE_SNEAK_MOTHER_BASE:
		//case tppmessage.CMD_REQUEST_RELIEF:
		//case tppmessage.CMD_GET_FOB_REWARD_LIST:
		//case tppmessage.CMD_GET_FOB_EVENT_LIST:
		//case tppmessage.CMD_GET_FOB_EVENT_DETAIL:
		//case tppmessage.CMD_SEND_SNEAK_RESULT:
		//case tppmessage.CMD_OPEN_WORMHOLE:
		//case tppmessage.CMD_GET_WORMHOLE_LIST:
		//case tppmessage.CMD_GET_ABOLITION_COUNT:
		//case tppmessage.CMD_GET_CONTRIBUTE_PLAYER_LIST:
		//case tppmessage.CMD_GET_RENTAL_LOADOUT_LIST:
		//case tppmessage.CMD_SYNC_LOADOUT:
		//case tppmessage.CMD_RENTAL_LOADOUT:
		//case tppmessage.CMD_GET_COMBAT_DEPLOY_LIST:
		//case tppmessage.CMD_DEPLOY_MISSION:
		//case tppmessage.CMD_CANCEL_COMBAT_DEPLOY:
		//case tppmessage.CMD_CANCEL_COMBAT_DEPLOY_SINGLE:
		//case tppmessage.CMD_ELAPSE_COMBAT_DEPLOY:
		//case tppmessage.CMD_GET_COMBAT_DEPLOY_RESULT:
		//case tppmessage.CMD_SEND_DEPLOY_INJURE:
		//case tppmessage.CMD_GET_FOB_DEPLOY_LIST:
		//case tppmessage.CMD_DEPLOY_FOB_ASSIST:
		//case tppmessage.CMD_CALC_COST_FOB_DEPLOY_REPLACE:
		//case tppmessage.CMD_GET_SERVER_ITEM_LIST:
		//case tppmessage.CMD_GET_SERVER_ITEM:
		//case tppmessage.CMD_DEVELOP_SERVER_ITEM:
		//case tppmessage.CMD_SPEND_SERVER_WALLET:
		//case tppmessage.CMD_CHECK_SERVER_ITEM_CORRECT:
		//case tppmessage.CMD_DESTRUCT_ONLINE_NUCLEAR:
		//case tppmessage.CMD_GET_CHALLENGE_TASK_REWARDS:
		//case tppmessage.CMD_GET_CHALLENGE_TASK_TARGET_VALUES:
		//case tppmessage.CMD_SEND_ONLINE_CHALLENGE_TASK_STATUS:
		//case tppmessage.CMD_GET_PF_DETAIL_PARAMS:
		//case tppmessage.CMD_GET_LEAGUE_RESULT:
		//case tppmessage.CMD_EXCHANGE_LEAGUE_POINT:
		//case tppmessage.CMD_GET_PF_POINT_EXCHANGE_PARAMS:
		//case tppmessage.CMD_EXCHANGE_LEAGUE_POINT2:
		//case tppmessage.CMD_USE_PF_ITEM:
		//case tppmessage.CMD_GET_RANKING:
		//case tppmessage.CMD_GET_NEXT_MAINTENANCE:
		//case tppmessage.CMD_CHECK_SHORT_PFLEAGUE_ENTERABLE:
		//case tppmessage.CMD_ENTER_SHORT_PFLEAGUE:
		//case tppmessage.CMD_CANCEL_SHORT_PFLEAGUE:
		//case tppmessage.CMD_GET_SHORT_PFLEAGUE_RESULT:
		//case tppmessage.CMD_GET_PREVIOUS_SHORT_PFLEAGUE_RESULT:
		//case tppmessage.CMD_USE_SHORT_PF_ITEM:
		//case tppmessage.CMD_GET_MBCOIN_REMAINDER:
		//case tppmessage.CMD_GET_OWN_FOB_LIST:
		//case tppmessage.CMD_GET_PURCHASABLE_AREA_LIST:
		//case tppmessage.CMD_PURCHASE_FOB:
		//case tppmessage.CMD_PURCHASE_FIRST_FOB:
		//case tppmessage.CMD_RELOCATE_FOB:
		//case tppmessage.CMD_GET_PAY_ITEM_LIST:
		//case tppmessage.CMD_PURCHASE_RESOURCES_PROCESSING:
		//case tppmessage.CMD_PURCHASE_PLATFORM_CONSTRUCTION:
		//case tppmessage.CMD_PURCHASE_SECURITY_SERVICE:
		//case tppmessage.CMD_PURCHASE_SEND_TROOPS_COMPLETION:
		//case tppmessage.CMD_PURCHASE_WEPON_DEVELOPMENT_COMPLETION:
		//case tppmessage.CMD_PURCHASE_NUCLEAR_COMPLETION:
		//case tppmessage.CMD_PURCHASE_ONLINE_DEPLOYMENT_COMPLETION:
		//case tppmessage.CMD_PURCHASE_ONLINE_DEVELOPMENT_COMPLETION:
		//case tppmessage.CMD_GET_SECURITY_INFO:
		//case tppmessage.CMD_GET_SECURITY_PRODUCT_LIST:
		//case tppmessage.CMD_GET_DEVELOPMENT_PROGRESS:
		//case tppmessage.CMD_GET_ONLINE_DEVELOPMENT_PROGRESS:
		//case tppmessage.CMD_GET_PLATFORM_CONSTRUCTION_PROGRESS:
		//case tppmessage.CMD_SEND_TROOPS:
		//case tppmessage.CMD_GET_TROOPS_LIST:
		//case tppmessage.CMD_DELETE_TROOPS_LIST:
		//case tppmessage.CMD_EXTEND_PLATFORM:
		//case tppmessage.CMD_DEVELOP_WEPON:
		//case tppmessage.CMD_CALC_COST_TIME_REDUCTION:
		//case tppmessage.CMD_CONSUME_RESERVE:
		//case tppmessage.CMD_CHECK_CONSUME_TRANSACTION:
		//case tppmessage.CMD_START_CONSUME_TRANSACTION:
		//case tppmessage.CMD_COMMIT_CONSUME_TRANSACTION:
		//case tppmessage.CMD_GET_PURCHASE_HISTORY_NUM:
		//case tppmessage.CMD_GET_PURCHASE_HISTORY:
		//case tppmessage.CMD_GET_ENTITLEMENT_ID_LIST:
		//case tppmessage.CMD_GET_SHOP_ITEM_NAME_LIST:
		//case tppmessage.CMD_OPEN_STEAM_SHOP:
		//case tppmessage.CMD_APPROVE_STEAM_SHOP:
		//case tppmessage.CMD_GET_STEAM_SHOP_ITEM_LIST:
		//case tppmessage.CMD_ADD_FOLLOW:
		//case tppmessage.CMD_DELETE_FOLLOW:
		//case tppmessage.CMD_UPDATE_SESSION:
		//case tppmessage.CMD_SEND_MISSION_RESULT:
		//case tppmessage.CMD_SEND_BOOT:
		//case tppmessage.CMD_GET_FOB_NOTICE:
		//case tppmessage.CMD_GET_DAILY_REWARD:
		//case tppmessage.CMD_GET_CAMPAIGN_DIALOG_LIST:
		//case tppmessage.CMD_GET_PLAYER_PLATFORM_LIST:
		//case tppmessage.CMD_GET_FOB_PARAM:
		//case tppmessage.CMD_GET_LOGIN_PARAM:
		//case tppmessage.CMD_GET_SECURITY_SETTING_PARAM:
		//case tppmessage.CMD_GET_RESOURCE_PARAM:
		//case tppmessage.CMD_SALE_RESOURCE:
		//case tppmessage.CMD_CHECK_DEFENCE_MOTHERBASE:
		//case tppmessage.CMD_SEND_SUSPICION_PLAY_DATA:
		//case tppmessage.CMD_SET_SECURITY_CHALLENGE:
		//case tppmessage.CMD_GET_FOB_EVENT_POINT_EXCHANGE_PARAMS:
		//case tppmessage.CMD_EXCHANGE_FOB_EVENT_POINT:
		//case tppmessage.CMD_GET_SNEAK_TARGET_LIST:
		//case tppmessage.CMD_INVALID:
	}

	return err
}

func (m *SessionManager) Handle(ctx context.Context, message *message.Message) error {
	// TODO platform to variable
	message.Platform = platform.Steam

	if message.Compress || message.SessionCrypto {
		message.MData = bytes.ReplaceAll(message.MData, []byte("\\r\\n"), nil)
		dst := make([]byte, base64.StdEncoding.DecodedLen(len(message.MData)))
		slog.Debug("data", "", fmt.Sprintf("%s", message.MData))
		_, err := base64.StdEncoding.Decode(dst, message.MData)
		if err != nil {
			return fmt.Errorf("cannot base64 decode message mdata: %w", err)
		}
		message.MData = dst
	}

	slog.Debug("decode", "crypto", message.SessionCrypto)
	if message.SessionCrypto {
		sess, err := m.Get(*message.SessionKey)
		if err != nil {
			return fmt.Errorf("cannot get session to decrypt message: %w", err)
		}

		message.MData = sess.Coder.DecodeBlowfish(message.MData)
		message.PlayerID = sess.PlayerID
		message.UserID = sess.UserID
		message.PlatformID = sess.PlatformID
	}

	slog.Debug("decode", "compress", message.Compress)
	if message.Compress {
		reader, err2 := zlib.NewReader(bytes.NewReader(message.MData))
		if err2 != nil {
			return fmt.Errorf("cannot create zlib reader: %w", err2)
		}
		defer reader.Close()

		var b []byte
		buf := bytes.NewBuffer(b)

		_, err := io.Copy(buf, reader)
		if err != nil {
			return fmt.Errorf("cannot copy compressed data back: %w", err)
		}

		message.MData = buf.Bytes()
	}

	slog.Debug("mdata", "len", len(message.MData), "want", message.OriginalSize, "status", message.OriginalSize == len(message.MData))

	message.MData = message.MData[:message.OriginalSize]

	slog.Debug("decoded inner message", "data", message.MData)

	err := message.GetDataType()
	if err != nil {
		return fmt.Errorf("cannot decode message data: %w", err)
	}

	if m.WriteLog {
		if err = message.ToFile(fmt.Sprintf("log/%s", m.LogDir)); err != nil {
			slog.Warn("cannot save message to file", "error", err.Error())
		}
	}

	message.Data = string(message.MData)
	m.Appstate.MessageLog = append(m.Appstate.MessageLog, *message)
	if message.IsRequest {
		m.Appstate.MessageChannel <- 0
	} else {
		m.Appstate.MessageChannel <- 1
	}

	slog.Info("handling message", "type", message.MsgID, "override", override, "request", message.IsRequest)
	if message.IsRequest {
		// TODO platform to var
		err = m.HandleRequest(ctx, message, platform.Steam)
	} else {
		err = m.HandleResponseFromKojiPro(ctx, message)
	}

	if err != nil {
		return fmt.Errorf("cannot handle message %s: %w", message.MsgID, err)
	}

	message.OriginalSize = len(message.MData)
	slog.Debug("set new original size", "value", message.OriginalSize)

	slog.Debug("encode", "compress", message.Compress)
	if message.Compress {
		var b bytes.Buffer
		writer, err := zlib.NewWriterLevel(&b, flate.BestCompression)
		if err != nil {
			return fmt.Errorf("cannot create zlib writer: %w", err)
		}

		defer writer.Close()
		if _, err = writer.Write(message.MData); err != nil {
			return fmt.Errorf("cannot compress: %w", err)
		}

		err = writer.Flush()
		if err != nil {
			return fmt.Errorf("cannot flush compressed data: %w", err)
		}

		_ = writer.Close()

		message.MData = b.Bytes()
	}

	slog.Debug("encode", "crypto", message.SessionCrypto)
	if message.SessionCrypto {
		sess, err := m.Get(*message.SessionKey)
		if err != nil {
			return fmt.Errorf("cannot get session to decrypt message: %w", err)
		}

		padLen := 8 - len(message.MData)&7
		if padLen <= 8 && padLen > 0 {
			padding := make([]byte, padLen)
			slog.Debug("added padding", "size", padLen)
			for i := range padding {
				padding[i] = byte(padLen)
			}
			message.MData = append(message.MData, padding...)
		}

		message.MData = sess.Coder.EncodeBlowfish(message.MData)
	}

	if message.Compress || message.SessionCrypto {
		message.MData = []byte(base64.StdEncoding.EncodeToString(message.MData))
		lines := util.SplitByteString(message.MData, 76)
		message.MData = bytes.Join(lines, []byte("\r\n"))
		message.MData = append(message.MData, []byte("\r\n")...)
	}

	return nil
}
