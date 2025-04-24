package globals

import "sync"

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

// Mutex
var EstadosMutex sync.Mutex
var MapaProcesosMutex sync.Mutex
var PIDCounterMutex sync.Mutex
var HandshakesMutex sync.Mutex

// Estructura para comunicarle a Memoria y CPU
type PidJSON struct {
	PID int64 `json:"pid"`
}

// Contador de PID para asignar a nuevos procesos
var PIDCounter int64 = 0

type HandshakeIO struct {
	Nombre string `json:"nombre"`
	IP     string `json:"ip"`
	Puerto int64  `json:"puerto"`
}

// Lista de los IO disponibles
var HandshakesIO []HandshakeIO
var HandshakesCPU []HandshakeIO

// Constantes
const NEW string = "NEW"
const READY string = "READY"
const EXECUTE string = "EXECUTE"
const BLOCKED string = "BLOCKED"
const SUSP_BLOCKED string = "SUSP_BLOCKED"
const SUSP_READY string = "SUSP_READY"
const EXIT string = "EXIT"

var PLANIFICADOR_LARGO_PLAZO_BLOCKED bool = true

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

type Estados struct {
	NEW          []Proceso_Nuevo
	READY        []int64
	EXECUTE      []int64
	BLOCKED      []int64
	SUSP_BLOCKED []int64
	SUSP_READY   []int64
}

var ESTADOS Estados

// Solicitud a IO
type SolicitudIO struct {
	PID    int64 `json:"pid"`
	Tiempo int64 `json:"tiempo"`
}
