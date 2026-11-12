package rfx

import "math"

// ConvertmW2dBm returns dBM given mW.
func ConvertmW2dBm(mW float64) float64 {
	return 10.0 * math.Log10(mW)
}

// ConvertWatt2dBm returns dBM given W.
func ConvertWatt2dBm(mW float64) float64 {
	return 30.0 + ConvertmW2dBm(mW)
}

// ConvertdBm2dBuV returns dBuV given dBm.
func ConvertdBm2dBuV(dBm float64) float64 {
	return dBm + 107.0
}

// ConvertdBuV2dBm returns dBm given dBuV.
func ConvertdBuV2dBm(dBuV float64) float64 {
	return dBuV - 107.0
}

// ConvertdBm2mW returns mW given dBm.
func ConvertdBm2mW(dBm float64) float64 {
	return math.Pow(10, dBm/10.0)
}

// ConvertdBm2Watt returns W given dBm.
func ConvertdBm2Watt(dBm float64) float64 {
	return ConvertdBm2mW(dBm) / 1000.0
}

// DecorateSerialNumber returns the serial number formatted as xxxx-xxxx-xxxx-xxxx
// If the serial number is invalid then an empty string is returned.
func DecorateSerialNumber(sn string) string {
	if len(sn) != 16 {
		return ""
	}
	return sn[:4] + "-" + sn[4:8] + "-" + sn[8:12] + "-" + sn[12:16]
}
