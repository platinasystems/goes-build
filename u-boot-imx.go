package main

type QuadSPIConfig struct {
	DQSLoopback                         uint32
	HoldDelay                           uint32
	R1                                  [2]uint32
	DeviceQuadModeEn                    uint32
	DeviceCmd                           uint32
	WriteCmdIpcr                        uint32
	WriteEnableIpcr                     uint32
	ChipSelectHoldTime                  uint32
	ChipSelectSetupTime                 uint32
	SerialFlashA1Size                   uint32
	SerialFlashA2Size                   uint32
	SerialFlashB1Size                   uint32
	SerialFlashB2Size                   uint32
	SerialClockFrequency                uint32
	BusyBitOffset                       uint32
	ModeOfOperationOfSerialFlash        uint32
	SerialFlashPortBSelection           uint32
	DualDataRateModeEnable              uint32
	DataStrobeSignalEnableInSerialFlash uint32
	ParallelModeEnable                  uint32
	CS1OnPortA                          uint32
	CS1OnPortB                          uint32
	FullSpeedPhaseSelection             uint32
	FullSpeedDelaySelection             uint32
	DDRSamplingPoint                    uint32
	LUTProgramSequence                  [64]uint32
	ReadStatusIpcr                      uint32
	EnableDqsPhase                      uint32
	R2                                  [9]uint32
	DqsPadSettingOverride               uint32
	SclkPacSettingOverride              uint32
	DataPadSettingOverride              uint32
	CsPadSettingOverride                uint32
	DqsLoopbackInternal                 uint32
	DqsPhaseSel                         uint32
	DqsFaDelayChainSel                  uint32
	DqsFbDelayChainSel                  uint32
	SclkFaDelayChainSel                 uint32
	SclkFbDelayChainSel                 uint32
	R3                                  [16]uint32
	Tag                                 uint32
}

func bmcQuadSPIConfig() (c QuadSPIConfig) {
	c.DeviceQuadModeEn = 0x01
	c.DeviceCmd = 0x8282
	c.WriteCmdIpcr = 0x03000002
	c.WriteEnableIpcr = 0x02000000
	c.ChipSelectHoldTime = 0x3
	c.ChipSelectSetupTime = 0x3
	c.SerialFlashA1Size = 0x08000000
	c.SerialFlashB1Size = 0x08000000
	c.ModeOfOperationOfSerialFlash = 0x4
	c.DualDataRateModeEnable = 0x1
	c.LUTProgramSequence[0] = 0x2a1804ed
	c.LUTProgramSequence[1] = 0x0e082e01
	c.LUTProgramSequence[2] = 0x24003a04
	c.LUTProgramSequence[3] = 0x0
	c.LUTProgramSequence[4] = 0x1c010405
	c.LUTProgramSequence[5] = 0x00002400
	c.LUTProgramSequence[6] = 0x0
	c.LUTProgramSequence[7] = 0x0
	c.LUTProgramSequence[8] = 0x24000406
	c.LUTProgramSequence[9] = 0x0
	c.LUTProgramSequence[10] = 0x0
	c.LUTProgramSequence[11] = 0x0
	c.LUTProgramSequence[12] = 0x20010401
	c.LUTProgramSequence[13] = 0x00002400
	c.LUTProgramSequence[14] = 0x0
	c.LUTProgramSequence[15] = 0x0
	c.LUTProgramSequence[16] = 0x1c010435
	c.LUTProgramSequence[17] = 0x00002400
	c.ReadStatusIpcr = 0x01000001
	c.Tag = 0xc0ffee01

	return
}
