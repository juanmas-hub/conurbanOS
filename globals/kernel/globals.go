package globals

import (
	"sync"
	"time"
)

type Kernel_Config struct {
	Ip_memory           string  `json:"ip_memory"`
	Port_memory         int64   `json:"port_memory"`
	Ip_kernel           string  `json:"ip_kernel"`
	Port_kernel         int64   `json:"port_kernel"`
	Scheduler_algorithm string  `json:"scheduler_algorithm"`
	New_algorithm       string  `json:"ready_ingress_algorithm"`
	Alpha               float64 `json:"alpha"`
	Suspension_time     int64   `json:"suspension_time"`
	Log_level           string  `json:"log_level"`
	Initial_estimate    int64   `json:"initial_estimate"`
}

var KernelConfig *Kernel_Config

type Mensaje struct {
	Mensaje string `json:"mensaje"`
}

// Mutex
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

type HandshakeIO struct {
	NombreIO        string `json:"nombre_io"`
	NombreInstancia string `json:"nombre_instancia"`
	IP              string `json:"ip"`
	Puerto          int64  `json:"puerto"`
}

type ListaCpu struct {
	Handshake Handshake
	EstaLibre bool
	PIDActual int64
}

type EntradaMapaIO struct {
	Instancias            []InstanciaIO
	ColaProcesosEsperando []SyscallIO
}

type InstanciaIO struct {
	Handshake        HandshakeIO
	PidProcesoActual int64 // PID del proceso actual que esta en esta IO | Si es -1 no hay procesos
}

// Listas
var ListaCPUs []ListaCpu

var MapaIOs map[string]EntradaMapaIO = make(map[string]EntradaMapaIO) // mapa de las IOs conectadas, con clave nombre

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
	return make(Semaforo, maxTareas) // canal buffered, pero vacío inicialmente
}

var Sem_Cpus = CrearSemaforo(1000)

// Empieza en 0:
//		+ Aumenta cuando sea conecta una CPU, o un proceso sale de CPU (hay cpus libres para usar)
//		- Disminuye cuando se elije un proceso a ejecutar

// Estructuras para manejo de procesos

var Sem_ProcesosEnReady = CrearSemaforo(1000)

// Es un contador de los procesos que hay en Ready: sirve para que no loopee infinito en el planificador de corto plazo

var Sem_ProcesoAFinalizar = CrearSemaforo(0)
var ProcesosAFinalizar []int64

// var Sem_PasarProcesoAReady = CrearSemaforo(0)
var Sem_PasarProcesoAReady chan struct{} = make(chan struct{}, 1000) // buffer grande = acumulador de signals
func SignalPasarProcesoAReady() {
	Sem_PasarProcesoAReady <- struct{}{}
}

func WaitPasarProcesoAReady() {
	<-Sem_PasarProcesoAReady
}

// Con el semaforo le aviso al planificador de largo plazo que hay un proceso para finalizar
// En el slice le pongo el PID

var DeDondeSeLlamaPasarProcesosAReady string = ""
var DeDondeSeLlamaMutex sync.Mutex

// Mapa PID:Cantidad. La cantidad de sesiones de IO indica cuantas veces fue a IO. Se usa para controlar el timer en planificador medio. Se aumenta en EnviarSolicitudIO
var CantidadSesionesIO map[int64]int = make(map[int64]int)
var CantidadSesionesIOMutex sync.Mutex

// SRT
var SrtReplanificarChan = make(chan struct{}, 1) // buffered para no bloquear

type MetricasEstado struct {
	New          int64
	Ready        int64
	Execute      int64
	Blocked      int64
	Susp_Ready   int64
	Susp_Blocked int64
}

type MetricasTiempo struct {
	New          time.Duration
	Ready        time.Duration
	Execute      time.Duration
	Blocked      time.Duration
	Susp_Ready   time.Duration
	Susp_Blocked time.Duration
}
type PCB struct {
	Pid int64
	PC  int64
	ME  MetricasEstado
	MT  MetricasTiempo
}

type Rafagas struct {
	Est_Ant  float64
	Raf_Ant  float64
	Est_Sgte float64
}

type Proceso struct {
	Pcb                  PCB
	Estado_Actual        string
	Rafaga               *Rafagas
	UltimoCambioDeEstado time.Time
	Archivo_Pseudocodigo string
	Tamaño               int64
}

var MapaProcesos map[int64]*Proceso = make(map[int64]*Proceso)
var ProcesosMutex map[int64]*sync.Mutex = make(map[int64]*sync.Mutex)

type RespuestaInterrupcion struct {
	PC int64 `json:"pc"`
}

// Estructuras para los estados

var Cola_new []int64
var Cola_ready []int64
var Cola_execute []int64
var Cola_blocked []int64
var Cola_susp_blocked []int64
var Cola_susp_ready []int64

var NewMutex sync.Mutex
var ReadyMutex sync.Mutex
var ExecuteMutex sync.Mutex
var BlockedMutex sync.Mutex
var SuspBlockedMutex sync.Mutex
var SuspReadyMutex sync.Mutex

// Solicitud y finalizacion IO
type SolicitudIO struct {
	PID    int64 `json:"pid"`
	Tiempo int64 `json:"tiempo"`
}

type FinalizacionIO struct {
	PID             int64  `json:"pid"`
	NombreIO        string `json:"nombre_io"`
	NombreInstancia string `json:"nombre_instancia"`
}

type DesconexionIO struct {
	NombreIO        string `json:"nombre"`
	NombreInstancia string `json:"nombre_instancia"`
	PID             int64  `json:"pid"`
	Ip              string `json:"ip"`
	Puerto          int64  `json:"port"`
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

type SyscallDump struct {
	PID       int64  `json:"pid"`
	PC        int64  `json:"pc"`
	NombreCPU string `json:"nombre_cpu"`
}

type SyscallInit struct {
	Tamanio     int64  `json:"tamanio"`
	Archivo     string `json:"archivo"`
	Nombre_CPU  string `json:"nombre_cpu"`
	Pid_proceso int64  `json:"pid_proceso"` // pid del proceso que ejecuta la syscall
	Pc          int64  `json:"pc"`          // pc actualizado
}

// Solicitud de iniciar proceso
type SolicitudIniciarProceso struct {
	Archivo_Pseudocodigo string `json:"archivo_pseudocodigo"`
	Tamanio              int64  `json:"tamanio"`
	Pid                  int64  `json:"pid"`
	Susp                 bool   `json:"susp"`
}
