package globals

import "sync"

type Kernel_Config struct {
	Ip_memory           string `json:"ip_memory"`
	Port_memory         int64  `json:"port_memory"`
	Ip_kernel           string `json:"ip_kernel"`
	Port_kernel         int64  `json:"port_kernel"`
	Scheduler_algorithm string `json:"scheduler_algorithm"`
	New_algorithm       string `json:"ready_ingress_algorithm"`
	Alpha               int64  `json:"alpha"`
	Suspension_time     int64  `json:"suspension_time"`
	Log_level           string `json:"log_level"`
	Initial_estimate    int64  `json:"initial_estimate"`
}

var KernelConfig *Kernel_Config

type Mensaje struct {
	Mensaje string `json:"mensaje"`
}

// Mutex
var EstadosMutex sync.Mutex
var MapaProcesosMutex sync.Mutex
var PIDCounterMutex sync.Mutex
var ListaCPUsMutex sync.Mutex
var ListaIOsMutex sync.Mutex
var ProcesosAFinalizarMutex sync.Mutex

// Estructura para comunicarle a Memoria y CPU
type PidJSON struct {
	PID int64 `json:"pid"`
}

type ProcesoAExecutar struct {
	PID int64 `json:"pid"`
	PC  int64 `json:"pc"`
}

// Contador de PID para asignar a nuevos procesos
var PIDCounter int64 = 0

type Handshake struct {
	Nombre string `json:"nombre"`
	IP     string `json:"ip"`
	Puerto int64  `json:"puerto"`
}

type ListaCpu struct {
	Handshake Handshake
	EstaLibre bool
}

type ListaIo struct {
	Handshake             Handshake
	PidProcesoActual      int64 // PID del proceso actual que esta en esta IO
	ColaProcesosEsperando []SyscallIO
}

// Listas
var ListaIOs []ListaIo
var ListaCPUs []ListaCpu

// Constantes
const NEW string = "NEW"
const READY string = "READY"
const EXECUTE string = "EXECUTE"
const BLOCKED string = "BLOCKED"
const SUSP_BLOCKED string = "SUSP_BLOCKED"
const SUSP_READY string = "SUSP_READY"
const EXIT string = "EXIT"

var PLANIFICADOR_LARGO_PLAZO_BLOCKED bool = true

// Semaforos
type Semaforo chan struct{} // es un tipo que ocupa 0 bytes, entonces puedo hacer los semaforos mas eficientes
func CrearSemaforo(maxTareas int) Semaforo {
	semaforo := make(Semaforo, maxTareas)
	for i := 0; i < maxTareas; i++ {
		semaforo <- struct{}{}
	}
	return semaforo
}

var Sem_Cpus = CrearSemaforo(0)

// Empieza en 0:
//		+ Aumenta cuando sea conecta una CPU, o un proceso sale de CPU (hay cpus libres para usar)
//		- Disminuye cuando se elije un proceso a ejecutar

// Estructuras para manejo de procesos

var Sem_ProcesosEnReady = CrearSemaforo(0)

// Es un contador de los procesos que hay en Ready: sirve para que no loopee infinito en el planificador de corto plazo

var Sem_ProcesoAFinalizar = CrearSemaforo(0)
var ProcesosAFinalizar []int64

// Con el semaforo le aviso al planificador de largo plazo que hay un proceso para finalizar
// En el slice le pongo el PID

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

// Solicitud y finalizacion IO
type SolicitudIO struct {
	PID    int64 `json:"pid"`
	Tiempo int64 `json:"tiempo"`
}

type FinalizacionIO struct {
	PID      int64  `json:"pid"`
	NombreIO string `json:"nombre"`
}

// Sycalls
type SyscallIO struct {
	NombreIO  string `json:"nombre_io"`
	NombreCPU string `json:"nombre_cpu"`
	Tiempo    int64  `json:"tiempo"`
	PID       int64  `json:"pid"`
	PC        int64  `json:"pc"`
}

type SyscallExit struct {
	PID       int64  `json:"pid"`
	NombreCPU string `json:"nombre_cpu"`
}

// Solicitud de iniciar proceso
type SolicitudIniciarProceso struct {
	Archivo_Pseudocodigo string
	Tamanio              int64
	Pid                  int64
}

// Devolucion de proceso -- esto no se usa en las syscalls, para esas se usan las structs especificas de cada syscall
// Las constantes strings son para poner el motivo
var FIN_PROCESO string
var REPLANIFICAR_PROCESO string // la verdad nose cuando se usa

type DevolucionProceso struct {
	Motivo     string `json:"string"`
	Pid        int64  `json:"pid"`
	PC         int64  `json:"pc"`
	Nombre_CPU string `json:"nombre_cpu"` // el mismo que se envio en el handshake
}
