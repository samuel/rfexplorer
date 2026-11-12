package rfx

const (
	posInternalCalibrated6G            = 134 // start position for 6G model
	posInternalCalibratedMWSUB3G       = 0   // start position for MWSUB3G model
	posInternalCalibratedWSUB1G        = 15  // start position for WSUB1G model
	posEndInternalCalibratedWSUB1G     = 58  // end position for WSUB1G, that is, POS_INTERNAL_CALIBRATED_WSUB1G = 15 + WSUB1G calibration data size = 43
	posInternalCalibratedWSUB1GPlus    = 0   // start position for WSUB1G+ model
	posEndInternalCalibratedWSUB1GPlus = 258 // end position for WSUB1G+, that is, POS_INTERNAL_CALIBRATED_WSUB1G_PLUS = 0 + WSUB1G calibration data size = 258 ((R1+R2+R3)86 x 3)
)
