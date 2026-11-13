package web

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"samuelks.com/rfx"
	"samuelks.com/rfx/rfxtest"
)

const (
	testSetup  = "#C2-M:006,005,01.34\r\n"
	testConfig = "#C2-F:2400000,0855854,0000,-120,0112,1,000,0015000,2700000,0600000,00510,0000,000\r\n"
)

func handler(cmd []byte) []byte {
	switch {
	case string(cmd) == "C0":
		return []byte(testSetup + testConfig)
	case bytes.HasPrefix(cmd, []byte("C+")):
		return append(append([]byte("#C+"), cmd[2]), "\r\n"...)
	case string(cmd) == "CP\x00":
		rec := make([]byte, 35) // "$P" length:0x20 then 32 bytes
		copy(rec, "$P\x20\x02\x01Remote")
		rec[19], rec[20] = 0x78, 0x69 // 27000 kHz
		rec[23], rec[24] = 0x60, 0x6d // 28000 kHz
		rec[28], rec[29], rec[30] = 0xec, 0x88, 10
		return append(append(rec, "\r\n"...), "#PCK\r\n"...)
	}
	return nil
}

func newTestServer(t *testing.T) (*httptest.Server, *rfxtest.Device) {
	t.Helper()
	port := rfxtest.NewDevice(handler)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	dev, err := rfx.New(ctx, port)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = dev.Close() }) //nolint:errcheck // test cleanup
	srv := httptest.NewServer(New(dev, nil))
	t.Cleanup(srv.Close)
	return srv, port
}

func TestStream(t *testing.T) {
	srv, port := newTestServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/ws", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.CloseNow() //nolint:errcheck // test cleanup

	next := func() map[string]any {
		t.Helper()
		var m map[string]any
		if err := wsjson.Read(ctx, conn, &m); err != nil {
			t.Fatal(err)
		}
		return m
	}
	if m := next(); m["type"] != "info" || m["main"] != "6G" || m["firmware"] != "1.34" {
		t.Fatalf("first message %v", m)
	}
	if m := next(); m["type"] != "config" || m["start"] != float64(2400e6) || m["points"] != float64(112) {
		t.Fatalf("second message %v", m)
	}

	port.Send(append(append([]byte("$S\x70"), bytes.Repeat([]byte{100}, 112)...), "\r\n"...))
	m := next()
	if m["type"] != "sweep" || m["start"] != float64(2400e6) || len(m["samples"].([]any)) != 112 || m["samples"].([]any)[0] != float64(-50) {
		t.Fatalf("sweep message %v", m)
	}

	if err := wsjson.Write(ctx, conn, map[string]any{"cmd": "calculator", "mode": "MaxHold"}); err != nil {
		t.Fatal(err)
	}
	for m = next(); m["type"] != "result"; m = next() {
	}
	if m["cmd"] != "calculator" || m["error"] != nil {
		t.Fatalf("result %v", m)
	}
	if err := wsjson.Write(ctx, conn, map[string]any{"cmd": "sweep", "start": "2500MHz", "stop": "2400MHz"}); err != nil {
		t.Fatal(err)
	}
	for m = next(); m["type"] != "result"; m = next() {
	}
	if m["cmd"] != "sweep" || m["error"] == nil {
		t.Fatalf("invalid range result %v", m)
	}
}

func TestAPI(t *testing.T) {
	srv, _ := newTestServer(t)
	get := func(path string) []byte {
		t.Helper()
		res, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close() //nolint:errcheck // test
		body, err := io.ReadAll(res.Body)
		if err != nil || res.StatusCode != http.StatusOK {
			t.Fatalf("GET %s: %d %s %v", path, res.StatusCode, body, err)
		}
		return body
	}

	var presets []rfx.Preset
	if err := json.Unmarshal(get("/api/presets"), &presets); err != nil {
		t.Fatal(err)
	}
	if len(presets) != 1 || presets[0].Name != "Remote" || presets[0].Start != 27*rfx.MHz {
		t.Fatalf("presets %+v", presets)
	}

	var state struct {
		Info   infoMsg   `json:"info"`
		Config configMsg `json:"config"`
	}
	if err := json.Unmarshal(get("/api/state"), &state); err != nil {
		t.Fatal(err)
	}
	if state.Info.Expansion != "WSUB3G" || state.Config.Stop <= state.Config.Start {
		t.Fatalf("state %+v", state)
	}

	if page := get("/"); !bytes.Contains(page, []byte("app.js")) {
		t.Fatal("index page not served")
	}
	if js := get("/app.js"); !bytes.Contains(js, []byte("WebSocket")) {
		t.Fatal("script not served")
	}
}
