package vehicle

type VehicleInformation struct {
	Vin          string
	EngineNumber string
	CarNumber    string
}

type FourWheelSpeeds struct {
	WheelSpeed_FL float64 //0- 460.74375km/h
	WheelSpeed_FR float64
	WheelSpeed_RL float64
	WheelSpeed_RR float64
}

type FourWheelTyrePressure struct {
	WheelTyrePressure_FL float32 //1.0~4.0Bar(100Kpa)
	WheelTyrePressure_FR float32
	WheelTyrePressure_RL float32
	WheelTyrePressure_RR float32
}

type FourWheelTyreTemperature struct {
	WheelTyreTemperature_FL float32 //-40~125℃
	WheelTyreTemperature_FR float32
	WheelTyreTemperature_RL float32
	WheelTyreTemperature_RR float32
}

type VehicleStatusInfor struct {
	ResidualMileage           int32   //0-1023km
	OdographMeter             int32   //0-0xFFFFFkm
	MaintainMileage           int32   //0- 0x7FFFkm
	Fuel                      int16   //0-127L
	AverageFuleCut            float32 //0-25.5L/100KM
	AverageSpeed              int32   //0-255km/h
	VehicleInstantaneousSpeed float64 //0- 460.74375km/h
	WheelSpeed                FourWheelSpeeds
	TyrePressure              FourWheelTyrePressure
	TyreTemperature           FourWheelTyreTemperature
	SteeringAngularSpeed      int32 //0~1016 deg/s
	SteeringAngle             int32 //-900~900deg
}

type EngineState int8

const (
	EngineOff      EngineState = 0x0
	EngineCranking EngineState = 0x1
	EngineOn       EngineState = 0x2
)

type GearLeverPos int8

const (
	GearLeverPos_P                    GearLeverPos = 0x0
	GearLeverPos_R                    GearLeverPos = 0x1
	GearLeverPos_N                    GearLeverPos = 0x2
	GearLeverPos_D                    GearLeverPos = 0x3
	GearLeverPos_Manual               GearLeverPos = 0x4
	GearLeverPos_Reserved             GearLeverPos = 0x5
	GearLeverPos_IntermediatePosition GearLeverPos = 0x6
	GearLeverPos_Fault                GearLeverPos = 0x7
)

type EngineStatusInfor struct {
	EngineSpeed    float64     //0~8191.875rpm
	EngineRunTime  int32       //s
	VehicleRunTime int32       //s
	EngState       EngineState //off on
	GearLevPos     GearLeverPos
}

type State int8

const (
	Off State = 0x0
	On  State = 0x1
)

type SafetyBeltState struct {
	SafetyBelt_Main State
	SafetyBelt_Vice State
}

type DoorState int8

const (
	DoorOpen     DoorState = 0x0
	DoorClose    DoorState = 0x1
	DoorReserved DoorState = 0x2
	DoorInvaild  DoorState = 0x3
)

type DoorsState struct {
	DriverDoor DoorState
	PassDoor   DoorState
	RLDoor     DoorState
	RRDoor     DoorState
	TrunkDoor  DoorState
}

type CentreControlLockState int8

const (
	CentreControlLockReserved CentreControlLockState = 0x0
	CentreControlLockLocked   CentreControlLockState = 0x1
	CentreControlLockUnlock   CentreControlLockState = 0x2
	CentreControlLockInvalid  CentreControlLockState = 0x3
)

type ElecParkBrakeLampState int8

const (
	ElecParkBrakeLampOff      ElecParkBrakeLampState = 0x0
	ElecParkBrakeLampOn       ElecParkBrakeLampState = 0x1
	ElecParkBrakeLampFlash    ElecParkBrakeLampState = 0x2
	ElecParkBrakeLampReserved ElecParkBrakeLampState = 0x3
)

type HandBrakeState int8

const (
	HandBrakePutdown HandBrakeState = 0x0
	HandBrakePullup  HandBrakeState = 0x1
)

type ParkRadarState int8

const (
	ParkRadarDisplayOff ParkRadarState = 0x0
	ParkRadarDisplayOn  ParkRadarState = 0x1
	ParkRadarSystemErr  ParkRadarState = 0x2
	ParkRadarInvalid    ParkRadarState = 0x3
)

type FrontWiperState int8

const (
	FrontWiperOff       FrontWiperState = 0x0
	FrontWiperLowSpeed  FrontWiperState = 0x1
	FrontWiperHighSpeed FrontWiperState = 0x2
	FrontWiperInterim   FrontWiperState = 0x3
	FrontWiperFault     FrontWiperState = 0x4
	FrontWiperReserved  FrontWiperState = 0x5
	FrontWiperReserved2 FrontWiperState = 0x6
	FrontWiperReserved3 FrontWiperState = 0x7
)

type BackWiperState int8

const (
	BackWiperOff      BackWiperState = 0x0
	BackWiperOn       BackWiperState = 0x1
	BackWiperErr      BackWiperState = 0x2
	BackWiperReserved BackWiperState = 0x3
)

type LampState int8

const (
	LampOff     LampState = 0x0
	LampOn      LampState = 0x1
	LampErr     LampState = 0x2
	LampInvalid LampState = 0x3
)

type CorneringLampState int8

const (
	CorneringLampOpen     CorneringLampState = 0x0
	CorneringLampClose    CorneringLampState = 0x1
	CorneringLampReserved CorneringLampState = 0x2
	CorneringLampInvalid  CorneringLampState = 0x3
)

type CorneringLampsState struct {
	CorneringLampState_Left  CorneringLampState
	CorneringLampState_Right CorneringLampState
}

type AirConditionalState int8

const (
	AirConditionalInactive AirConditionalState = 0x0
	AirConditionalActive   AirConditionalState = 0x1
)

type AirConditionalMode int8

const (
	AirConditionalModeFace             AirConditionalMode = 0x0
	AirConditionalModeFaceAndFoot      AirConditionalMode = 0x1
	AirConditionalModeFoot             AirConditionalMode = 0x2
	AirConditionalModeFootAndDefroster AirConditionalMode = 0x3
)

type AirConditionalFanSpeed int8

const (
	AirConditionalFanSpeedOff AirConditionalFanSpeed = 0x0
	AirConditionalFanSpeed_1  AirConditionalFanSpeed = 0x1
	AirConditionalFanSpeed_2  AirConditionalFanSpeed = 0x2
	AirConditionalFanSpeed_3  AirConditionalFanSpeed = 0x3
	AirConditionalFanSpeed_4  AirConditionalFanSpeed = 0x4
	AirConditionalFanSpeed_5  AirConditionalFanSpeed = 0x5
	AirConditionalFanSpeed_6  AirConditionalFanSpeed = 0x6
	AirConditionalFanSpeed_7  AirConditionalFanSpeed = 0x7
)

type Circulation int8

const (
	CirculationOutside Circulation = 0x0
	CirculationInside  Circulation = 0x1
)

type SeatHeatState int8

const (
	SeatHeatOff             SeatHeatState = 0x0
	SeatHeatHeaterLow       SeatHeatState = 0x1
	SeatHeatHeaterMid       SeatHeatState = 0x2
	SeatHeatHeaterHigh      SeatHeatState = 0x3
	SeatHeatVentilationLow  SeatHeatState = 0x4
	SeatHeatVentilationMid  SeatHeatState = 0x5
	SeatHeatVentilationHigh SeatHeatState = 0x6
	SeatHeatReserved        SeatHeatState = 0x7
)

type MaindriverWindowState int8

const (
	MaindriverWindowClosed        MaindriverWindowState = 0x0
	MaindriverWindowOpened        MaindriverWindowState = 0x1
	MaindriverWindowStoped        MaindriverWindowState = 0x2
	MaindriverWindowAutoUpMov     MaindriverWindowState = 0x3
	MaindriverWindowManualUpMov   MaindriverWindowState = 0x4
	MaindriverWindowAutoDownMov   MaindriverWindowState = 0x5
	MaindriverWindowManualDownMov MaindriverWindowState = 0x6
	MaindriverWindowInvalid       MaindriverWindowState = 0x7
)

type FortifyState int8

const (
	FortifyNo    FortifyState = 0x0
	FortifyFully FortifyState = 0x1
	FortifyHalf  FortifyState = 0x2
	FortifyAlert FortifyState = 0x3
)

type PowerState int8

const (
	PowerOff   PowerState = 0x0
	PowerAcc   PowerState = 0x1
	PowerOn    PowerState = 0x2
	PowerStart PowerState = 0x3
)

type VehicleBodyStateInfor struct {
	SafetyAirBag      int8
	SafetyBelt        SafetyBeltState
	Doors             DoorsState
	CentreControlLock CentreControlLockState
	ElecParkBrakeLamp ElecParkBrakeLampState
	HandBrake         HandBrakeState
	ParkRadar         ParkRadarState
	FrontWiper        FrontWiperState
	BackWiper         BackWiperState
	FrontWaterSpray   State
	FogLamp           State
	HighBeam          LampState
	DippedHeadLight   LampState
	PositionLamp      LampState
	CorneringLamps    CorneringLampsState
	DangerWarnLamp    State
	TemperatureInCar  float32 //-40～+80℃
	TemperatureOutCar float32 //-48~143.5℃
	AirConditional    AirConditionalState
	AirCondMode       AirConditionalMode
	AirCondFanSpeed   AirConditionalFanSpeed
	Circul            Circulation
	AirConditionalTem float32 //17.5-49℃
	FrontFogLamp      LampState
	BackFogLamp       LampState
	SeatHeat          SeatHeatState
	MaindriverWin     MaindriverWindowState
	SkyLight          int32 //% 100%-totally close; 20%-interior open; 0%-total open; 200%-raise
	Fortify           FortifyState
	Power             PowerState
}

type BrakeFluidState int8

const (
	BrakeFluidNormal BrakeFluidState = 0x0
	BrakeFluidLow    BrakeFluidState = 0x1
)

type EngineFaultState int8

const (
	EngineFaultOff      EngineFaultState = 0x0
	EngineFaultOn       EngineFaultState = 0x1
	EngineFaultReserved EngineFaultState = 0x2
	EngineFaultInvalid  EngineFaultState = 0x3
)

type EPSState int8

const (
	EPS_OK    EPSState = 0x0
	EPS_Fault EPSState = 0x1
)

type EPBState int8

const (
	EPBLampOff       EPBState = 0x0
	EPBLampOn        EPBState = 0x1
	EPBLampReserved1 EPBState = 0x2
	EPBLampReserved2 EPBState = 0x3
)

type FaultMode int8

const (
	NoFault FaultMode = 0x0
	Fault   FaultMode = 0x1
)

type PanoramaCamera struct {
	ForntCam FaultMode
	RearCam  FaultMode
	LeftCam  FaultMode
	RightCam FaultMode
}

type RRSState int8

const (
	RRSNoInstalled RRSState = 0x0
	RRSFailed      RRSState = 0x1
	RRSNormal      RRSState = 0x2
	RRSReserved    RRSState = 0x3
)

type RRSensor struct {
	F1Sensor RRSState
	F2Sensor RRSState
	R1Sensor RRSState
	R2Sensor RRSState
	R3Sensor RRSState
	R4Sensor RRSState
}

type VehicleFault struct {
	PositionLamp         LampState
	BrakeFluid           BrakeFluidState
	BatteryVoltage       float32
	BatteryChargeVoltage float32
	EngineFault          EngineFaultState
	OpenDoorIllegal      FortifyState
	TotalCarFortify      FortifyState
	EPS                  EPSState
	EPB                  EPBState
	ABS                  FaultMode
	EBD                  FaultMode
	ESC                  FaultMode
	Camera               PanoramaCamera
	RRS                  RRSensor
	TCU                  FaultMode
}

type TBoxInfor struct {
	Key          string
	TBoxSerailNo string
	VIN          string
	ICCID        string
}

type User struct {
	UserID    string
	PIN       string
	Authority string
}

type Instruction int16

const (
	Instruction_RemoteControlVehicle_TrumpetOpen        Instruction = 0x0
	Instruction_RemoteControlVehicle_TrumpetClose       Instruction = 0x1
	Instruction_RemoteControlVehicle_LampOpen           Instruction = 0x2
	Instruction_RemoteControlVehicle_LampClose          Instruction = 0x3
	Instruction_RemoteControlVehicle_EngineStart        Instruction = 0x4
	Instruction_RemoteControlVehicle_EngineStop         Instruction = 0x5
	Instruction_RemoteControlVehicle_AirConOpen         Instruction = 0x6
	Instruction_RemoteControlVehicle_AirConClose        Instruction = 0x7
	Instruction_RemoteControlVehicle_FrontDefrostOpen   Instruction = 0x8
	Instruction_RemoteControlVehicle_FrontDefrostClose  Instruction = 0x9
	Instruction_RemoteControlVehicle_BackDefrostOpen    Instruction = 0xA
	Instruction_RemoteControlVehicle_BackDefrostClose   Instruction = 0xB
	Instruction_RemoteControlVehicle_SeatHeatOpen       Instruction = 0xC
	Instruction_RemoteControlVehicle_SeatHeatClose      Instruction = 0xD
	Instruction_RemoteControlVehicle_SkyLightOpen       Instruction = 0xE
	Instruction_RemoteControlVehicle_SkyLightClose      Instruction = 0xF
	Instruction_RemoteControlVehicle_MainDriverWinOpen  Instruction = 0x10
	Instruction_RemoteControlVehicle_MainDriverWinClose Instruction = 0x11
	Instruction_RemoteControlVehicle_LockVehicle        Instruction = 0x12
	Instruction_RemoteControlVehicle_UnlockVehicle      Instruction = 0x13
	Instruction_RemoteControlVehicle_FaultDiagnosis     Instruction = 0x14
)

type GPS struct {
	Latitude  float64
	Longitude float64
	Altitude  float64
	Speed     float64
	Bearing   float64
	Accuracy  float64
	Time      int64
}

type VehiclePosition struct {
	Gps GPS
}
