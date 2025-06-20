package globals

type Memoria_Config struct {
	Port_memory      int64  `json:"port_memory"`
	Memory_size      int64  `json:"memory_size"`
	Ip_memory        string `json:"ip_memory"`
	Page_size        int64  `json:"page_size"`
	Entries_per_page int64  `json:"entries_per_page"`
	Number_of_levels int64  `json:"number_of_levels"`
	Memory_delay     int64  `json:"memory_delay"`
	Swap_delay       int64  `json:"swap_delay"`
	Swapfile_path    string `json:"swapfile_path"`
	Log_level        string `json:"log_level"`
	Dump_path        string `json:"dump_path"`
	Scripts_path     string `json:"scripts_path"`
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

type MetricasMap map[int]*Memoria_Metrica

var Metricas *MetricasMap

// Entrada de una tabla de páginas
type EntradaTablaPagina struct {
	Pagina         int
	Marco          int
	SiguienteNivel *TablaDePaginas // Ya se inicializa por defecto como null
}

type TablaDePaginas struct {
	Entradas []EntradaTablaPagina
}

type Manager map[int]*TablaDePaginas

var ProcessManager *Manager

var Memoria []byte

var MemoriaMarcosOcupados []bool


type PaginaDTO struct{
	Contenido string
	Entrada *EntradaTablaPagina
} 

type Pagina struct{
	IndiceAsignado int
	IndiceSwapAsignado int
	EntradaAsignada *EntradaTablaPagina
}

type Proceso struct {
	Pseudocodigo []string
	MarcosAsignados []Pagina
	Suspendido bool
	PaginasSWAP []Pagina
}

type ProcesosMap map[int]*Proceso

var Procesos ProcesosMap

type IniciarProcesoDTO struct {
	ArchivoPseudocodigo string `json:"archivo_pseudocodigo"`
	Tamanio              int64 `json:"tamanio"`
	Pid                  int64 `json:"pid"`
}

type PidDTO struct {
	Pid int64 `json:"pid"`
}

type InstruccionDTO struct {
	Pid int64 `json:"pid"`
	Pc  int64 `json:"pc"`
}

type LecturaDTO struct {
	Pid int64 `json:"pid"`
	Posicion int64 `json:"posicion"`
	Tamanio int64 `json:"tamanio"`
}

type EscrituraDTO struct {
	Pid int64 `json:"pid"`
	Posicion int64 `json:"posicion"`
	Dato string `json:"dato"`
}
type TablaDTO struct {
	Pid int64 `json:"pid"`
	Indices []int `json:"indices"`
}

type ConsultaPaginaDTO struct {
	Pid int64 `json:"pid"`
	PrimerIndice int64 `json:"primer_indice"`
}

type LeerPaginaDTO struct {
	IndicePagina int64 `json:"indice_pagina"`
}

type ActualizarPaginaDTO struct {
	IndicePagina int64 `json:"indice_pagina"`
	Dato []byte `json:"dato"`
}

var ListaPaginasSwapDisponibles []Pagina

var ProximoIndiceSwap int