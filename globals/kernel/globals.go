package globals

type Kernel_Config struct {
	Ip_memory           string `json:"ip_memory"`
	Port_memory         int64  `json:"port_memory"`
	Port_kernel         int64  `json:"port_kernel"`
	Scheduler_algorithm string `json:"scheduler_algorithm"`
	New_algorithm       string `json:"new_algorithm"`
	Alpha               string `json:"alpha"`
	Suspension_time     int64  `json:"suspension_time"`
	Log_level           string `json:"log_level"`
}

var KernelConfig *Kernel_Config

type Mensaje struct {
	Mensaje string `json:"mensaje"`
}

type HandshakeIO struct {
	Nombre string `json:"nombre"`
	IP     string `json:"ip"`
	Puerto int    `json:"puerto"`
}

// Constantes
const NEW string = "NEW"
const READY string = "READY"
const EXECUTE string = "EXECUTE"
const BLOCKED string = "BLOCKED"
const SUSP_BLOCKED string = "SUSP_BLOCKED"
const SUSP_READY string = "SUSP_READY"
const EXIT string = "EXIT"

// Estructuras para manejo de procesos

type Metricas struct {
	New          int64
	Ready        int64
	Execute      int64
	Blocked      int64
	Susp_Ready   int64
	Susp_Blocked int64
}
type PCB struct {
	Pid int64
	PC  int64
	ME  Metricas
	MT  Metricas
}

type Rafagas struct {
	Est_Ant  int64
	Raf_Ant  int64
	Est_Sgte int64
}

type Proceso struct {
	Pcb           PCB
	Estado_Actual string
	Rafaga        *Rafagas
}

type Proceso_Nuevo struct {
	Archivo_Pseudocodigo string
	Tamaño               int64
	Proceso              Proceso
}

var MapaProcesos map[int64]Proceso = make(map[int64]Proceso)

// Estructuras para los estados

type Execute_Struct struct {
	Libre bool
	Pid   int64
}
type Estados struct {
	NEW          []Proceso_Nuevo
	READY        []int
	EXECUTE      []Execute_Struct
	BLOCKED      []int
	SUSP_BLOCKED []int
	SUSP_READY   []int
}

var ESTADOS Estados
