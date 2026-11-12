package rfx

// func (m Model) IsWifi5GHz(self):
// 	"""True if 6G/6G+ is in 5GHz WiFi band, false if 2.4GHz
// 	"""
// 	return ((self.ActiveModel == RFE_Common.eModel.MODEL_6G) or (self.ActiveModel == RFE_Common.eModel.MODEL_6G_PLUS)) and (self.StartFrequencyMHZ >= 5160)

// func (m Model) IsMWSUB3G() bool {
// 	// Property true if the WSUB3G connected is actually a MWSUB3G. This enables us to work with MWSUB3G same as WSUB3G except
// // in specific places where MWSUB3G code is required. Use this function anytime to know if a WSUB3G is actually a MWSUB3G
// 	return ((self.m_eMainBoardModel == RFE_Common.eModel.MODEL_WSUB3G) and (self.m_eExpansionBoardModel == RFE_Common.eModel.MODEL_NONE))
// }

// @property
// def IsMW5G(self):
// 	return (self.m_eMainBoardModel == RFE_Common.eModel.MODEL_W5G3G or self.m_eMainBoardModel == RFE_Common.eModel.MODEL_W5G4G or self.m_eMainBoardModel == RFE_Common.eModel.MODEL_W5G5G)

// func (m Model) IsMainboardAnalyzerPlus() bool {
// 	// Return true if the device mainboard is a PLUS model, regardless the expansion model connected (if any)
// 	return m == ModelWSUB1GPlus || m == ModelAudioPro ||
// 	return self.MainBoardModel == RFE_Common.eModel.MODEL_WSUB1G_PLUS or self.IsAudioPro or self.IsMW5G
// }

// @property
// def IsExpansion4G6GPlus(self):
// 	"""Returns true if there is an expansion 4G+ or 6G+ board installed (note it may not be the active band)
// 	"""
// 	return (self.m_eExpansionBoardModel == RFE_Common.eModel.MODEL_4G_PLUS) or (self.m_eExpansionBoardModel == RFE_Common.eModel.MODEL_6G_PLUS)

// @property
// def IsExpansionPlus(self):
// 	"""Returns true if there is an expansion PLUS board installed (note it may not be the active band)
// 	"""
// 	return (self.m_eExpansionBoardModel == RFE_Common.eModel.MODEL_2400_PLUS) or self.IsExpansion4G6GPlus

// @property
// def IsActive4G6GPlus(self):
// 	"""Returns true if 4G+ or 6G+ boards are active
// 	"""
// 	return (self.m_eActiveModel == RFE_Common.eModel.MODEL_4G_PLUS) or (self.m_eActiveModel == RFE_Common.eModel.MODEL_6G_PLUS)

// @property
// def IsHighResAvailable(self):
// 	"""Returns true if devices allow high resolution mode for sweeps
// 	"""
// 	#High resolution is not supported in Wifi mode
// 	return ((self.m_eActiveModel == RFE_Common.eModel.MODEL_WSUB1G_PLUS or self.IsActive4G6GPlus or self.IsAudioPro or self.IsMWSUB3G or self.IsMW5G) and self.m_eMode != RFE_Common.eMode.MODE_WIFI_ANALYZER)

// @property
// def IsInputStageAvailable(self):
// 	"""Returns true if device allows choose input stage
// 	"""
// 	return ((self.m_eActiveModel == RFE_Common.eModel.MODEL_WSUB1G_PLUS) or (self.m_eActiveModel == RFE_Common.eModel.MODEL_2400_PLUS) or\
// 			self.IsAudioPro or self.IsMWSUB3G) or self.IsActive4G6GPlus
