package server

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/unknown321/fuse/coder"
	"github.com/unknown321/fuse/gui"
	guistructs "github.com/unknown321/fuse/gui_structs"
	"github.com/unknown321/fuse/handlers"
	"github.com/unknown321/fuse/sessionmanager"
)

const DsnURIDefault = "./fuse.dat"

func Start(baseURL string, listenAddr string, platform string, writeLog bool, passThrough bool, dsnURI string, bonus sessionmanager.SignupBonus) {
	appstate := guistructs.AppState{
		MessageChannel: make(chan int),
	}
	go StartServer(baseURL, listenAddr, platform, writeLog, passThrough, dsnURI, bonus, &appstate)
	gui.InitGui(&appstate)
}

func StartServer(baseURL string, listenAddr string, platform string, writeLog bool, passThrough bool, dsnURI string, bonus sessionmanager.SignupBonus, appstate *guistructs.AppState) {
	now := time.Now().Unix()
	if writeLog {
		err := os.MkdirAll("./log/"+strconv.Itoa(int(now)), 0755)

		if err != nil {
			slog.Error("cannot create logdir")
			return
		}
	}

	manager := sessionmanager.SessionManager{
		Appstate: appstate,
		WriteLog: writeLog,
		LogDir:   strconv.FormatInt(now, 10),
		ManagerOpts: sessionmanager.ManagerOpts{
			SignupBonus: bonus,
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		slog.Info("request", "url", req.URL)
		w.WriteHeader(404)
		_, _ = w.Write([]byte("not found\n"))
	})

	c := coder.Coder{}
	err := c.WithKey(nil)
	if err != nil {
		slog.Error("cannot create coder", "error", err.Error())
		return
	}

	gh := handlers.GateHandler{}
	gh.WithManager(&manager)
	gh.WithCoder(&c)
	if dsnURI == "" {
		dsnURI = DsnURIDefault
	}
	err = gh.DBConnect(dsnURI)
	if err != nil {
		slog.Error("cannot connect to database", "uri", dsnURI, "error", err.Error())
		return
	}
	slog.Info("db connected")
	ctx := context.Background()
	err = gh.InitDB(ctx, baseURL, platform)

	if err != nil {
		slog.Error("cannot create schema", "error", err.Error())
		return
	}

	if err = gh.Seed(ctx); err != nil {
		slog.Info("seed", "warn", err.Error(), "but", "it's ok")
	}

	gh.PassThrough = passThrough
	gh.FromJSON = false

	mux.HandleFunc("/tppstm/gate", gh.Handle)
	mux.HandleFunc("/tppstm/main", gh.Handle)
	mux.HandleFunc("/tppstmweb/eula/eula.var", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("eula\n"))
	})
	mux.HandleFunc("/tppstmweb/coin/coin.var", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("coin terms\n"))
	})
	mux.HandleFunc("/tppstmweb/privacy/privacy.var", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("privacy\n"))
	})
	mux.HandleFunc("/tppstmweb/gdpr/privacy.var", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("privacy\n"))
	})
	mux.HandleFunc("/tppstmweb/privacy_jp/privacy.var", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("privacy jp\n"))
	})
	mux.HandleFunc("/tppstmweb/privacy_ccpa/privacy.var", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("privacy jp\n"))
	})

	slog.Info("serving...")
	err = http.ListenAndServe(listenAddr, mux)
	if err != nil {
		slog.Error("cannot listen and serve", "error", err.Error())
		return
	}
}
