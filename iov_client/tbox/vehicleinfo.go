package tbox

import (
	//"hcxy/iov/log/logger"
	"hcxy/iov/util"
	"time"
)

type VehicleInfor struct {
	Version uint32
	Time    uint32

	IsLocation uint8
	Latitude   uint32
	Longitude  uint32
	Heading    uint16
	Speed      uint16

	EngineSpeedRPM     uint16
	VehicleSpeedVSOSig uint16

	IdleSpeedStatus     byte //1
	ACRequest           byte //1
	ParkBrake           byte //1
	EngineRunningStatus byte //1
	St_gearLeverPos     byte //4

	AirBagFailSts byte //1
	T_BOX_Fault   byte //1
	EMS_Fault     byte //1
	EPS_Fault     byte //1
	ESC_Fault     byte //1
	St_TCU        byte //1
	ABS_Fault     byte //1
	EBD_Fault     byte //1

	EPB_Fault            byte //1
	ICU_Fault            byte //1
	PassSeatBeltStatus   byte //2
	DriverSeatBeltStatus byte //2
	Alarm_Mode           byte //2

	CrashOutputSts byte
	LF_Pressure    byte
	RF_Pressure    byte
	RR_Pressure    byte
	LR_Pressure    byte

	FrontWiperGear  byte //3
	FrontSprayState byte //1
	RearWiperGear   byte //3
	RearSprayState  byte //1

	Sunroofstatus       byte //2
	DriverDoorStatus    byte //1
	PassengerDoorStatus byte //1
	RearRightDoorStatus byte //1
	RearLeftDoorStatus  byte //1
	BackDoorStatus      byte //1
	FLDoorLock_Status   byte //1

	FRDoorLock_Status byte //1
	RLDoorLock_Status byte //1
	RRDoorLock_Status byte //1
	HoodStatus        byte //1
	TrunkLock_Status  byte //1
	PositionLampState byte //1
	BrakeLiquidLow    byte //1
	BatteryVoltageLow byte //1

	KeyPosition        byte //2
	TurnRightStatus    byte //1
	TurnLeftStatus     byte //1
	HighBeamStatus     byte //1
	LowBeamStatus      byte //1
	FrontFogStatus     byte //1
	RearFogStatus      byte //1
	TotalOdometer_km   uint32
	FuelVolume         byte
	BatteryVoltage     byte
	FuelConsumption    uint16
	OutsideTemperature uint16

	EngineCoolantTemperature byte
	FuelAlarm                byte //1
	Dragging                 byte //1
	ChargeStatusLight        byte //1
	Turnover                 byte //1
	MaintenanceLightStatus   byte //1
	VHSDataValidity          byte //1
	EngineTrouble            byte //1
	EngineCoolantTempHigh    byte //1

	SkylightState byte

	HazardWarningLampState byte //1
	ConditionMode          byte //1
	IllegalOpenTheDoor     byte //1
	SharpSlowdown          byte //1
	CallipersFault         byte //1
	Scratches              byte //1
	Collision              byte //1
	AirTouchkey            byte //1

	SunIllumination byte

	vehicleState               byte
	Reverse2                   byte //2
	IsPreconditioning          byte //2
	IsCharging                 byte //2
	IsPluggedIn                byte //2
	BulbFailureCount           byte
	BulbFailure1               byte
	BulbFailure2               byte
	BulbFailure3               byte
	BulbFailure4               byte
	BulbFailure5               byte
	BulbFailure6               byte
	BulbFailure7               byte
	SAS_SteeringAngle          uint16
	SAS_SteeringAngleSpeed     uint16
	ClutchState                byte //1
	BootState                  byte //1
	GsensorStatus              byte //1
	RapidAcceleration          byte //1
	SharpTurn                  byte //1
	Reverse3                   byte //1
	NetworkType                byte //2
	SignalStrength             byte
	Mileage                    uint16
	AverageVelocity            uint16
	LeftFrontSpeed             uint16
	RightFrontSpeed            uint16
	LeftRearSpeed              uint16
	RightRearSpeed             uint16
	EngineRunningtime          int //24bit
	TotalRunningtime           uint32
	BodyLightState             byte //2bit
	CentralLockState           byte //2bit
	FootBrakeState             byte //2bit
	RadarState                 byte //2bit
	FrontDefrostState          byte //1bit
	RearDefrostState           byte //1bit
	SeatWarmingState           byte //3bit
	MainWindowPositionAndState byte //3bit

	InsideTemperature      uint16
	MileageBetweenServices uint16
	AirWinCondition        byte
	AirWinRateGear         byte
	AirLoopMode            byte
	AirCurrSetTemperature  uint16
	Accelerator            byte
}

const (
	SwitchStateOn  byte = 0x0
	SwitchStateOff byte = 0x1
)

const (
	StateInvalid byte = 0x0
	StateNormal  byte = 0x1
)

const (
	SeatWarmingStateOff             byte = 0x0
	SeatWarmingStateHeaterLow       byte = 0x1
	SeatWarmingStateHeaterMid       byte = 0x2
	SeatWarmingStateHeaterHigh      byte = 0x3
	SeatWarmingStateVentilationLow  byte = 0x4
	SeatWarmingStateVentilationMid  byte = 0x5
	SeatWarmingStateVentilationHigh byte = 0x6
)

const (
	MainWindowStateClosed           byte = 0x0
	MainWindowStateOpened           byte = 0x1
	MainWindowStateStopped          byte = 0x2
	MainWindowStateAutoUpMoving     byte = 0x3
	MainWindowStateManualUpMoving   byte = 0x4
	MainWindowStateAutoDownMoving   byte = 0x5
	MainWindowStateManualDownMoving byte = 0x6
	MainWindowStateInvalid          byte = 0x7
)

type FaultAlarm struct {
	LightReminder         byte //1bit
	BrakeLiquidLow        byte //1bit
	BatteryVoltageLow     byte //1bit
	EngineTrouble         byte //1bit
	EngineCoolantTempHigh byte //1bit
	IllegalOpenDoor       byte //1bit
	EPStrouble            byte //1bit
	EPBtrouble            byte //1bit
	ABStrouble            byte //1bit
	EBDtrouble            byte //1bit
	ESCtrouble            byte //1bit
	TCUtrouble            byte //1bit
	SRStrouble            byte //1bit
	SRSOut                byte //1bit
	Crash                 byte //1bit
	Scratches             byte //1bit
	LeftTurnLampTrouble   byte //1bit
	RightTurnLamp         byte //1bit
	PositionLamp          byte //1bit
	DippedHeadlight       byte //1bit
	HighBeam              byte //1bit
	FrontFogLamp          byte //1bit
	RearFogLamp           byte //1bit
	Dragging              byte //1bit
	Turnover              byte //1bit
	reverse               byte //7bit
}

const (
	LocationFailure byte = 0x0
	LocationSuccess byte = 0x1
)

const (
	NotInIdle byte = 0x0
	InIdle    byte = 0x1
)

const (
	AirConditionerCloses byte = 0x0
	AirConditionerOpens  byte = 0x1
)

const (
	NotParked byte = 0x0
	Parked    byte = 0x1
)

const (
	EngineOff byte = 0x0
	EngineOn  byte = 0x1
)

const (
	GearLeverPos_P                    byte = 0x00
	GearLeverPos_L                    byte = 0x01
	GearLeverPos_Manual               byte = 0x02
	GearLeverPos_IntermediatePosition byte = 0x03
	GearLeverPos_1stManualGearEngaged byte = 0x04
	GearLeverPos_D                    byte = 0x05
	GearLeverPos_N                    byte = 0x06
	GearLeverPos_R                    byte = 0x07
	GearLeverPos_Reserved1            byte = 0x08
	GearLeverPos_2ndManualGearEngaged byte = 0x09
	GearLeverPos_3rdGearEngaged       byte = 0x0A
	GearLeverPos_4thManualGearEngaged byte = 0x0B
	GearLeverPos_5thManualGearEngaged byte = 0x0C
	GearLeverPos_6thManualGearEngaged byte = 0x0D
	GearLeverPos_Reserved2            byte = 0x0E
	GearLeverPos_Fault                byte = 0x0F
)

const (
	NoFault byte = 0x0
	Fault   byte = 0x1
)

const (
	PassSeatBeltStatusNotConfig byte = 0x0
	PassSeatBeltStatusFault     byte = 0x1
	PassSeatBeltStatusNotBulked byte = 0x2
	PassSeatBeltStatusBulked    byte = 0x3
)

const (
	DriverSeatBeltStatusUnused     byte = 0x0
	DriverSeatBeltStatusUnknown    byte = 0x0
	DriverSeatBeltStatusUninserted byte = 0x0
	DriverSeatBeltStatusInserted   byte = 0x0
)

const (
	AlarmModeUndefences byte = 0x0
	AlarmModeDefences   byte = 0x1
	AlarmModeAlarm      byte = 0x2
)

const (
	CrashOutputStatusNotPop byte = 0x0
	CrashOutputStatusPop    byte = 0x1
)

func (vi *VehicleInfor) GetVehicleInforData() []byte {
	var oneByte byte
	vehicleInfor := VehicleInfor{}
	vehicleInforData := make([]byte, 0, 1024)

	vehicleInfor.Version = 0xFF000103
	tmp, _ := util.Uint32ToByteSlice(vehicleInfor.Version)
	vehicleInforData = append(vehicleInforData, tmp...)

	vehicleInfor.Time = uint32(time.Now().Unix())
	tmp, _ = util.Uint32ToByteSlice(vehicleInfor.Time)
	vehicleInforData = append(vehicleInforData, tmp...)

	vehicleInfor.IsLocation = LocationSuccess
	vehicleInforData = append(vehicleInforData, vehicleInfor.IsLocation)

	vehicleInfor.Latitude = 30 * 1000000
	tmp, _ = util.Uint32ToByteSlice(vehicleInfor.Latitude)
	vehicleInforData = append(vehicleInforData, tmp...)

	vehicleInfor.Longitude = 170 * 1000000
	tmp, _ = util.Uint32ToByteSlice(vehicleInfor.Longitude)
	vehicleInforData = append(vehicleInforData, tmp...)

	vehicleInfor.Heading = 300
	tmp, _ = util.Uint16ToByteSlice(vehicleInfor.Heading)
	vehicleInforData = append(vehicleInforData, tmp...)

	vehicleInfor.Speed = 60 / 10
	tmp, _ = util.Uint16ToByteSlice(vehicleInfor.Speed)
	vehicleInforData = append(vehicleInforData, tmp...)

	vehicleInfor.EngineSpeedRPM = 3000
	tmp, _ = util.Uint16ToByteSlice(vehicleInfor.EngineSpeedRPM)
	vehicleInforData = append(vehicleInforData, tmp...)

	vehicleInfor.VehicleSpeedVSOSig = 60 / 10
	tmp, _ = util.Uint16ToByteSlice(vehicleInfor.VehicleSpeedVSOSig)
	vehicleInforData = append(vehicleInforData, tmp...)

	vehicleInfor.IdleSpeedStatus = NotInIdle
	vehicleInfor.ACRequest = AirConditionerOpens
	vehicleInfor.ParkBrake = NotParked
	vehicleInfor.EngineRunningStatus = EngineOn
	vehicleInfor.St_gearLeverPos = GearLeverPos_D

	oneByte |= vehicleInfor.IdleSpeedStatus
	oneByte |= (vehicleInfor.ACRequest << 1)
	oneByte |= (vehicleInfor.ParkBrake << 2)
	oneByte |= (vehicleInfor.EngineRunningStatus << 3)
	oneByte |= (vehicleInfor.St_gearLeverPos << 4)
	vehicleInforData = append(vehicleInforData, oneByte)

	vehicleInfor.AirBagFailSts = NoFault
	vehicleInfor.T_BOX_Fault = NoFault
	vehicleInfor.EMS_Fault = NoFault
	vehicleInfor.EPS_Fault = NoFault
	vehicleInfor.ESC_Fault = NoFault
	vehicleInfor.St_TCU = NoFault
	vehicleInfor.ABS_Fault = NoFault
	vehicleInfor.EBD_Fault = NoFault

	oneByte |= vehicleInfor.AirBagFailSts
	oneByte |= (vehicleInfor.T_BOX_Fault << 1)
	oneByte |= (vehicleInfor.EMS_Fault << 2)
	oneByte |= (vehicleInfor.EPS_Fault << 3)
	oneByte |= (vehicleInfor.ESC_Fault << 4)
	oneByte |= (vehicleInfor.St_TCU << 5)
	oneByte |= (vehicleInfor.ABS_Fault << 6)
	oneByte |= (vehicleInfor.EBD_Fault << 7)
	vehicleInforData = append(vehicleInforData, oneByte)

	vehicleInfor.EPB_Fault = NoFault
	vehicleInfor.ICU_Fault = NoFault
	vehicleInfor.PassSeatBeltStatus = PassSeatBeltStatusBulked
	vehicleInfor.DriverSeatBeltStatus = DriverSeatBeltStatusInserted
	vehicleInfor.Alarm_Mode = AlarmModeUndefences

	oneByte |= vehicleInfor.EPB_Fault
	oneByte |= (vehicleInfor.ICU_Fault << 1)
	oneByte |= (vehicleInfor.PassSeatBeltStatus << 2)
	oneByte |= (vehicleInfor.DriverSeatBeltStatus << 4)
	oneByte |= (vehicleInfor.Alarm_Mode << 6)
	vehicleInforData = append(vehicleInforData, oneByte)

	vehicleInfor.CrashOutputSts = CrashOutputStatusPop
	vehicleInforData = append(vehicleInforData, vehicleInfor.CrashOutputSts)

	vehicleInfor.LF_Pressure = 2 * 10
	vehicleInforData = append(vehicleInforData, vehicleInfor.LF_Pressure)

	vehicleInfor.RF_Pressure = 2 * 10
	vehicleInforData = append(vehicleInforData, vehicleInfor.RF_Pressure)

	vehicleInfor.RR_Pressure = 2 * 10
	vehicleInforData = append(vehicleInforData, vehicleInfor.RR_Pressure)

	vehicleInfor.LR_Pressure = 2 * 10
	vehicleInforData = append(vehicleInforData, vehicleInfor.LR_Pressure)

	return nil
}
