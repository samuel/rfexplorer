package main

import (
	"log"
	"os"

	// _ "net/http/pprof" // register pprof handlers

	"gioui.org/app"
	"gioui.org/unit"

	"github.com/samuel/rfexplorer/rfx"
)

func main() {
	// go func() {
	// 	log.Fatal(http.ListenAndServe(":6060", nil))
	// }()

	port, err := rfx.OpenSerial("/dev/tty.usbserial-0001")
	if err != nil {
		log.Fatal(err)
	}
	rfe, err := rfx.New(port)
	if err != nil {
		log.Fatal(err)
	}
	defer rfe.Close()

	// if err := rfe.SwitchModuleExp(); err != nil {
	// 	log.Fatal(err)
	// }
	// if err := rfe.SetAnalyzerConfig(2475650, 2501300, 0, -120, 0); err != nil {
	// 	log.Fatal(err)
	// }
	// 2.4 GHz Zigbee
	// if err := rfe.SetAnalyzerConfig(2404000, 2481000, 0, -120, 400); err != nil {
	// 	log.Fatal(err)
	// }
	// 2.4 GHz Wi-Fi
	// if err := rfe.SetAnalyzerConfig(2401000, 2495000, 0, -120, 0); err != nil {
	// 	log.Fatal(err)
	// }
	// if err := rfe.SetSteps(512); err != nil {
	// 	log.Fatal(err)
	// }
	// Interesting signal
	// if err := rfe.SetAnalyzerConfig(2420000, 2450000, 0, -120, 0); err != nil {
	// 	log.Fatal(err)
	// }
	// ISM Band (Region 2)
	// if err := rfe.SetAnalyzerConfig(902000, 928000, 0, -120, 0); err != nil {
	// 	log.Fatal(err)
	// }
	// 6 meter amateur radio
	// if err := rfe.SetAnalyzerConfig(50000, 54000, 0, -120, 0); err != nil {
	// 	log.Fatal(err)
	// }
	// 2 meter amateur radio
	// if err := rfe.SetAnalyzerConfig(144000, 148000, 0, -120, 0); err != nil {
	// 	log.Fatal(err)
	// }
	// 1.25 meter amateur radio
	// if err := rfe.SetAnalyzerConfig(222000, 225000, 0, -120, 0); err != nil {
	// 	log.Fatal(err)
	// }
	// 70 centimeters
	// if err := rfe.SetAnalyzerConfig(420000, 450000, 0, -120, 0); err != nil {
	// 	log.Fatal(err)
	// }

	// if err := rfe.SwitchModuleMain(); err != nil {
	// 	log.Fatal(err)
	// }
	// 5 GHz Wi-Fi
	// if err := rfe.SetAnalyzerConfig(5170000, 5835000, 0, -120, 0); err != nil {
	// 	log.Fatal(err)
	// }
	// if err := rfe.SetAnalyzerConfig(5500000, 5700000, 0, -120, 0); err != nil {
	// 	log.Fatal(err)
	// }

	// if err := rfe.SetAnalyzerConfig(433900, 434100, 0, -120, 0); err != nil {
	// 	log.Fatal(err)
	// }

	if err := rfe.SetLCDEnabled(true); err != nil {
		log.Fatal(err)
	}
	if err := rfe.SetScreenDumpEnabled(false); err != nil {
		log.Fatal(err)
	}
	// if err := rfe.Realtime(); err != nil {
	// 	log.Fatal(err)
	// }
	// if err := rfe.SetWifiMode(rfx.WifiModeDisabled); err != nil {
	// 	log.Fatal(err)
	// }

	// if err := rfe.SwitchModuleMain(); err != nil {
	// 	log.Fatal(err)
	// }
	// if err := rfe.SetAnalyzerConfig(5770000, 5835000, 0, -120, 0); err != nil {
	// 	log.Fatal(err)
	// }
	// 5 GHz Wi-Fi
	// if err := rfe.SetAnalyzerConfig(5170000, 5835000, 0, -120, 0); err != nil {
	// 	log.Fatal(err)
	// }

	if err := rfe.SetMaxHold(); err != nil {
		log.Fatal(err)
	}
	// if err := rfe.SetCalculatorMode(rfx.CalculatorModeAvg); err != nil {
	// 	log.Fatal(err)
	// }

	if err := rfe.SwitchModuleExp(); err != nil {
		log.Fatal(err)
	}
	// Trunking radio
	// if err := rfe.SetAnalyzerConfig(850000, 854000, 0, -120, 0); err != nil {
	// 	log.Fatal(err)
	// }
	// 2.4 GHz Wi-Fi
	// if err := rfe.SetAnalyzerConfig(2401000, 2495000, 0, -120, 0); err != nil {
	// 	log.Fatal(err)
	// }
	// Interesting signal
	// if err := rfe.SetAnalyzerConfig(2420000, 2450000, 0, -120, 0); err != nil {
	// 	log.Fatal(err)
	// }
	// 433 OOK
	// if err := rfe.SetAnalyzerConfig(430000, 436000, 0, -120, 0); err != nil {
	// 	log.Fatal(err)
	// }
	// if err := rfe.SetSweepPoints(512); err != nil {
	// 	log.Fatal(err)
	// }

	// if err := rfe.SetCalculatorMode(rfx.CalculatorModeAvg); err != nil {
	// 	log.Fatal(err)
	// }
	// if err := rfe.SetScreenDumpEnabled(false); err != nil {
	// 	log.Fatal(err)
	// }
	if err := rfe.RequestConfig(); err != nil {
		log.Fatal(err)
	}
	if err := rfe.RequestPresets(); err != nil {
		log.Fatal(err)
	}

	ui := NewUI(rfe)

	go func() {
		w := new(app.Window)
		w.Option(app.Title("RF-Explorer"))
		w.Option(app.Size(unit.Dp(1024), unit.Dp(1024)))
		if err := ui.Run(w); err != nil {
			log.Println(err)
			os.Exit(1)
		}
		os.Exit(0)
	}()

	app.Main()
}
