package rfx

// import (
// 	"fmt"
// 	"sync/atomic"
// )

// type Analyzer struct {
// 	rf     *RFExplorer
// 	config atomic.Value // *CurrentConfigPacket
// 	ch     chan AnalyzerMessage
// }

// type AnalyzerMessage interface{}

// type SamplesMessage struct {
// 	Samples []Sample
// }

// type Sample struct {
// 	FreqHZ int
// 	Amp    int
// }

// func NewAnalyzer(device string) (*Analyzer, error) {
// 	rf, err := New("/dev/tty.SLAB_USBtoUART")
// 	if err != nil {
// 		return nil, err
// 	}
// 	// Initial setup and fetch config
// 	if err := rf.RequestConfig(); err != nil {
// 		return nil, err
// 	}
// 	a := &Analyzer{
// 		rf: rf,
// 		ch: make(chan AnalyzerMessage, 16),
// 	}
// setupLoop:
// 	for {
// 		pkt, ok := <-rf.Chan()
// 		if !ok {
// 			rf.Close()
// 			return nil, fmt.Errorf("rfx: failed to get current config")
// 		}
// 		switch pkt := pkt.(type) {
// 		case *CurrentConfigPacket:
// 			a.config.Store(pkt)
// 			break setupLoop
// 		}
// 	}
// 	go a.readLoop()
// 	return a, nil
// }

// func (a *Analyzer) Close() error {
// 	return a.rf.Close()
// }

// func (a *Analyzer) Chan() chan AnalyzerMessage {
// 	return a.ch
// }

// func (a *Analyzer) Config() *CurrentConfigPacket {
// 	return a.config.Load()
// }

// func (a *Analyzer) readLoop() {
// 	for {
// 		pkt := <-a.rf.Chan()
// 		switch pkt := pkt.(type) {
// 		case *CurrentConfigPacket:
// 			a.config.Store(pkt)
// 			a.ch <- pkt
// 		case *rfx.SweepDataPacket:
// 		}
// 	}
// }
