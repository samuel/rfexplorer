// Package web serves a browser spectrum display for an RF Explorer. Each
// browser connects over a WebSocket and gets its own subscription, so a slow
// client loses sweeps without affecting the others or the device.
package web

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"samuelks.com/rfx"
)

//go:embed static
var staticFiles embed.FS

const (
	// commandTimeout bounds a device command requested by a client.
	commandTimeout = 10 * time.Second
	// writeTimeout drops a client that stops reading.
	writeTimeout = 10 * time.Second
	// clientBuffer is the number of messages queued per client before the
	// oldest sweeps are dropped.
	clientBuffer = 16
)

// Server is the HTTP handler for the web UI.
type Server struct {
	dev    *rfx.Device
	logger *slog.Logger
	mux    *http.ServeMux
}

// New returns a server for dev. A nil logger discards messages.
func New(dev *rfx.Device, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	s := &Server{dev: dev, logger: logger, mux: http.NewServeMux()}
	static, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(err) // the embedded directory always exists
	}
	s.mux.Handle("GET /", http.FileServerFS(static))
	s.mux.HandleFunc("GET /api/state", s.handleState)
	s.mux.HandleFunc("GET /api/presets", s.handlePresets)
	s.mux.HandleFunc("GET /ws", s.handleStream)
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// Messages sent to clients. Frequencies are in Hz.
type (
	infoMsg struct {
		Type       string `json:"type"` // "info"
		Main       string `json:"main"`
		Expansion  string `json:"expansion"`
		Firmware   string `json:"firmware"`
		MaxPresets int    `json:"max_presets"`
	}
	configMsg struct {
		Type       string     `json:"type"` // "config"
		Start      int64      `json:"start"`
		Stop       int64      `json:"stop"`
		Step       int64      `json:"step"`
		Points     int        `json:"points"`
		AmpTop     int        `json:"amp_top"`
		AmpBottom  int        `json:"amp_bottom"`
		RBW        int64      `json:"rbw"`
		Calculator string     `json:"calculator"`
		Module     string     `json:"module"`
		Boards     []boardMsg `json:"boards"`
	}
	boardMsg struct {
		Module   string `json:"module"`
		Model    string `json:"model"`
		MinFreq  int64  `json:"min_freq"` // zero if unknown
		MaxFreq  int64  `json:"max_freq"`
		MaxSpan  int64  `json:"max_span"`
		Reported bool   `json:"reported"`
	}
	sweepMsg struct {
		Type    string    `json:"type"` // "sweep"
		Start   int64     `json:"start"`
		Step    int64     `json:"step"`
		Samples []float32 `json:"samples"`
	}
	resultMsg struct {
		Type  string `json:"type"` // "result"
		Cmd   string `json:"cmd"`
		Error string `json:"error,omitempty"`
	}
)

// command is a request from a client.
type command struct {
	Cmd string `json:"cmd"` // "sweep", "calculator", "hold" or "resume"
	// For "sweep".
	Start     rfx.Frequency `json:"start"`
	Stop      rfx.Frequency `json:"stop"`
	AmpTop    *int          `json:"amp_top"`
	AmpBottom *int          `json:"amp_bottom"`
	// For "calculator".
	Mode rfx.CalculatorMode `json:"mode"`
}

func newInfoMsg(info rfx.DeviceInfo) infoMsg {
	return infoMsg{
		Type:       "info",
		Main:       info.Main.String(),
		Expansion:  info.Expansion.String(),
		Firmware:   info.Firmware,
		MaxPresets: info.MaxPresets(),
	}
}

func newConfigMsg(c rfx.SweepConfig, boards []rfx.Board) configMsg {
	bm := make([]boardMsg, len(boards))
	for i, b := range boards {
		bm[i] = boardMsg{
			Module:   b.Module.String(),
			Model:    b.Model.String(),
			MinFreq:  int64(b.Min),
			MaxFreq:  int64(b.Max),
			MaxSpan:  int64(b.MaxSpan),
			Reported: b.Reported,
		}
	}
	return configMsg{
		Type:       "config",
		Start:      int64(c.Start),
		Stop:       int64(c.Stop()),
		Step:       int64(c.Step),
		Points:     c.Points,
		AmpTop:     c.AmpTop,
		AmpBottom:  c.AmpBottom,
		RBW:        int64(c.RBW),
		Calculator: c.Calculator.String(),
		Module:     c.Module().String(),
		Boards:     bm,
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleState(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, struct {
		Info   infoMsg   `json:"info"`
		Config configMsg `json:"config"`
	}{newInfoMsg(s.dev.Info()), newConfigMsg(s.dev.Config(), s.dev.Boards())})
}

func (s *Server) handlePresets(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), commandTimeout)
	defer cancel()
	presets, err := s.dev.Presets(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	used := []rfx.Preset{}
	for _, p := range presets {
		if p.Name != "" {
			used = append(used, p)
		}
	}
	writeJSON(w, used)
}

func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		s.logger.Info("websocket upgrade failed", "error", err)
		return
	}
	defer conn.CloseNow() //nolint:errcheck // closing an already failed connection

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	sub := s.dev.Subscribe(rfx.SubscribeOptions{Kinds: rfx.KindSweep | rfx.KindConfig, Buffer: clientBuffer})
	defer sub.Close()

	// Results of commands are sent by the writer so writes never overlap.
	results := make(chan resultMsg, 4)
	go func() {
		defer cancel()
		s.readCommands(ctx, conn, results)
	}()

	send := func(v any) error {
		wctx, wcancel := context.WithTimeout(ctx, writeTimeout)
		defer wcancel()
		return wsjson.Write(wctx, conn, v)
	}
	if err := send(newInfoMsg(s.dev.Info())); err != nil {
		return
	}
	if err := send(newConfigMsg(s.dev.Config(), s.dev.Boards())); err != nil {
		return
	}
	for {
		var v any
		select {
		case <-ctx.Done():
			conn.Close(websocket.StatusNormalClosure, "") //nolint:errcheck // the client may already be gone
			return
		case res := <-results:
			v = res
		case msg, ok := <-sub.C():
			if !ok {
				reason := "device disconnected"
				if err := s.dev.Err(); err != nil {
					reason = err.Error()
				}
				conn.Close(websocket.StatusGoingAway, reason) //nolint:errcheck // the client may already be gone
				return
			}
			switch msg := msg.(type) {
			case *rfx.Sweep:
				v = sweepMsg{Type: "sweep", Start: int64(msg.Start), Step: int64(msg.Step), Samples: msg.Samples}
			case *rfx.SweepConfig:
				v = newConfigMsg(*msg, s.dev.Boards())
			default:
				continue
			}
		}
		if err := send(v); err != nil {
			s.logger.Info("websocket write failed", "error", err)
			return
		}
	}
}

// readCommands runs the client's commands one at a time until the
// connection closes.
func (s *Server) readCommands(ctx context.Context, conn *websocket.Conn, results chan<- resultMsg) {
	for {
		var cmd command
		if err := wsjson.Read(ctx, conn, &cmd); err != nil {
			var closeErr websocket.CloseError
			if !errors.As(err, &closeErr) && ctx.Err() == nil {
				s.logger.Info("websocket read failed", "error", err)
			}
			return
		}
		res := resultMsg{Type: "result", Cmd: cmd.Cmd}
		if err := s.run(ctx, cmd); err != nil {
			res.Error = err.Error()
		}
		select {
		case results <- res:
		case <-ctx.Done():
			return
		}
	}
}

func (s *Server) run(ctx context.Context, cmd command) error {
	ctx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()
	switch cmd.Cmd {
	case "sweep":
		cfg := s.dev.Config()
		req := rfx.SweepRequest{Start: cmd.Start, Stop: cmd.Stop, AmpTop: cfg.AmpTop, AmpBottom: cfg.AmpBottom}
		if cmd.AmpTop != nil {
			req.AmpTop = *cmd.AmpTop
		}
		if cmd.AmpBottom != nil {
			req.AmpBottom = *cmd.AmpBottom
		}
		_, err := s.dev.SetSweep(ctx, req)
		return err
	case "calculator":
		return s.dev.SetCalculator(ctx, cmd.Mode)
	case "hold":
		return s.dev.Hold(ctx)
	case "resume":
		_, err := s.dev.Resume(ctx)
		return err
	}
	return fmt.Errorf("unknown command %q", cmd.Cmd)
}
