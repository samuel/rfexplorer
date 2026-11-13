package rfx

// Positions of each model's table within the embedded calibration data.
const (
	posInternalCalibrated6G            = 134 // start position for 6G model
	posInternalCalibratedMWSUB3G       = 0   // start position for MWSUB3G model
	posInternalCalibratedWSUB1G        = 15  // start position for WSUB1G model
	posEndInternalCalibratedWSUB1G     = 58  // end position for WSUB1G: 15 + 43 values
	posInternalCalibratedWSUB1GPlus    = 0   // start position for WSUB1G+ model
	posEndInternalCalibratedWSUB1GPlus = 258 // end position for WSUB1G+: 3 ranges of 86 values
)

// calibrationOffsetsDB returns the amplitude offsets in dB for model from the
// embedded calibration data. Each value is a signed byte in units of 0.5 dB.
// https://github.com/RFExplorer/RFExplorer-for-Python/blob/ea25c90c500bdc2b78b146903ce65ff97a0abe69/RFExplorer/RFExplorer.py#L1115
func calibrationOffsetsDB(model Model, data []byte) []float64 {
	var start int
	end := len(data)
	switch model {
	case Model6G:
		start = posInternalCalibrated6G
	case ModelWSUB1G:
		start = posInternalCalibratedWSUB1G
		end = posEndInternalCalibratedWSUB1G
	case ModelWSUB1GPlus:
		start = posInternalCalibratedWSUB1GPlus
		end = posEndInternalCalibratedWSUB1GPlus
	case ModelWSUB3G:
		start = posInternalCalibratedMWSUB3G
	}
	end = min(end, len(data))
	if start >= end {
		return nil
	}
	offsets := make([]float64, 0, end-start)
	for _, v := range data[start:end] {
		offsets = append(offsets, float64(int8(v))/2.0)
	}
	return offsets
}
