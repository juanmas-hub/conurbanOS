package globals

type Memoria_Config struct {
	Port_memory      int64  `json:"port_memory"`
	Memory_size      int64  `json:"memory_size"`
	Page_size        int64  `json:"page_size"`
	Entries_per_page int64  `json:"entries_per_page"`
	Number_of_levels int64  `json:"number_of_levels"`
	Memory_delay     int64  `json:"memory_delay"`
	Swap_delay       int64  `json:"swap_delay"`
	Swapfile_path    string `json:"swapfile_path"`
	Log_level        string `json:"log_level"`
	Dump_path        string `json:"dump_path"`
}

var MemoriaConfig *Memoria_Config

type Mensaje struct {
	Mensaje string `json:"mensaje"`
}

type Memoria_Metrica struct {
	AccesosTablas            int `json:"accesos_tablas"`
	InstruccionesSolicitadas int `json:"instrucciones_solicitadas"`
	BajadasSwap              int `json:"bajadas_swap"`
	SubidasMemoria           int `json:"subidas_memoria"`
	LecturasMemoria          int `json:"lecturas_memoria"`
	EscriturasMemoria        int `json:"escrituras_memoria"`
}

// Entrada de una tabla de páginas
type EntradaTablaPagina struct {
	Pagina         int
	Marco          int
	SiguienteNivel *TablaDePaginas // Ya se inicializa por defecto como null
}

type TablaDePaginas struct {
	Entradas []EntradaTablaPagina // 4*64=256
}

type Manager map[int]*TablaDePaginas

var ProcessManager *Manager

var Memoria []byte

var MemoriaMarcosOcupados []bool

// Estructura donde recibo para inicializar proceso
type SolicitudIniciarProceso struct {
	Archivo_Pseudocodigo string
	Tamanio              int64
	Pid                  int64
}

// Estructura para recibir PID
type PidProceso struct {
	Pid int64 `json:"pid"`
}

type Pseudocodigo map[int][]string

var Instrucciones Pseudocodigo
