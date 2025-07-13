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

/*
type MetricasMap map[int]*Memoria_Metrica

var Metricas *MetricasMap
*/

var MetricasMap map[int]Memoria_Metrica

var Memoria []byte
var MemoriaMarcosOcupados []bool

/*
// Entrada de una tabla de páginas
type EntradaTablaPagina struct {
	Nivel          int
	Marco          int
	Presencia      int
	Uso            int
	Modificado     int
	SiguienteNivel *TablaDePaginas // Ya se inicializa por defecto como null
}

type TablaDePaginas struct {
	Entradas []EntradaTablaPagina
}

type Manager map[int]*TablaDePaginas

var Tablas *Manager


type PaginaDTO struct {
	Contenido string
	Entrada   *EntradaTablaPagina
}

type Pagina struct {
	Pid				   int
	//NumeroDePagina	   int
	IndiceAsignado     int
	IndiceSwapAsignado int
	EntradaAsignada    *EntradaTablaPagina
}

type Proceso struct {
	Pseudocodigo    []string
	MarcosAsignados []Pagina
	Suspendido      bool
	PaginasSWAP     []Pagina
}

type ProcesosMap map[int]*Proceso

var Procesos ProcesosMap

*/

type IniciarProcesoDTO struct {
	ArchivoPseudocodigo string `json:"archivo_pseudocodigo"`
	Tamanio             int64  `json:"tamanio"`
	Pid                 int64  `json:"pid"`
}

type PidDTO struct {
	Pid int64 `json:"pid"`
}

type InstruccionDTO struct {
	Pid int64 `json:"pid"`
	Pc  int64 `json:"pc"`
}

type LecturaDTO struct {
	Pid      int64 `json:"pid"`
	Posicion int64 `json:"posicion"`
	Tamanio  int64 `json:"tamanio"`
}

type EscrituraDTO struct {
	Pid             int64  `json:"pid"`
	DireccionFisica int64  `json:"posicion"`
	Dato            string `json:"dato"`
}
type TablaDTO struct {
	Pid     int64 `json:"pid"`
	Indices []int `json:"entradas"`
}

type ConsultaPaginaDTO struct {
	Pid          int64 `json:"pid"`
	PrimerIndice int64 `json:"primer_indice"`
}

type LeerPaginaDTO struct {
	DireccionFisica int64 `json:"direccion_fisica"`
}

type ActualizarPaginaDTO struct {
	Pid             int64  `json:"pid"`
	DireccionFisica int64  `json:"direccion_fisica"`
	Contenido       []byte `json:"contenido"`
}

var Prueba string

/*
var ListaPaginasSwapDisponibles []Pagina

var ProximoIndiceSwap int
*/

var IndicesSWAPOcupados []bool

// NUEVA OPCION
type Proceso struct {
	Pseudocodigo      []string
	Suspendido        bool
	TablaDePaginas    TablaPaginas
	InicioSWAP        int // guarda el indice donde inicia el proceso en swap
	CantidadDePaginas int
}

var Procesos map[int]Proceso // mapeado por PID

type TablaPaginas struct {
	Entradas []EntradaTP
}

type EntradaTP struct {
	NumeroDePagina int           // -1 en niveles intermedios
	NumeroDeFrame  int           // -1 en niveles intermedios
	SiguienteNivel *TablaPaginas // nil en el ultimo nivel
}

type PaginaEnSwap struct {
	Pid            int
	NumeroDePagina int
	contenido      []byte
}

// Lo usuaria para swappear
type Pagina struct {
	NumeroDePagina int
	Contenido      []byte
}

type PaginaLinkeada struct {
	NumeroDePagina int
	NumeroDeFrame  int
}

/*

Inicialización del proceso
- Le asignas los frames
- Creas la estructura Proceso
- Creas la tabla de Paginas
	- En el ultimo nivel le pones el numero de pagina y el numero de frame que le asociaste

Para acceder a una pagina:
- Recorres la TP
- Obtenes el numero de frame
- Haces la cuenta para tener la posicion de memoria

Suspensión de proceso:
- Pones el bool de suspendido en true
- La tabla de paginas la dejas como está
- Eliminas el contenido de la memoria (recorres la TP y por cada pagina tomas el frame, y lo llenas de 0s. Otra opcion seria tomar el inicio del proceso, su tamaño, volver a calcular los frames q necesita y llenarlos de 0s)
- Escribis en SWAP

Des-suspensión de proceso:
- Te fijas que el proceso entra en la memoria
- Creas una estructura auxiliar {pagina, frame, contenido}
- Obtenes los frames libres y los guardas en la estructura auxiliar (rellenas solo el numero de frame)
- Lees de SWAP y lo guardas en la estructura auxiliar (rellenas solo numero de pagina y contenido)
- Hasta aca ya tendrías todo linkeado: numero de pagina, con un numero de frame y su contenido
- Escribis en memoria
- Actualizas la TP con los nuevos numeros de frame

Finalización de proceso:
- Llenas de 0s la memoria en los marcos que tiene el proceso
- ELiminas el proceso y su TP

Acceso a tabla de páginas:
- ???

Acceso a espacio de usuario:
- Leer:
	- Me pasan DF, leo, y devuelvo el contenido
- Escribir:
	- Me pasan DF
	- Escribo

Leer Página completa:
- Recibo una DF (tiene que ser el byte 0 de la pagina, supongo que hay que chequearlo)
- Leo la pagina completa
- Devuelvo el contenido

Actualizar página completa:
- Recibo una DF (tiene que ser el byte 0 de la pagina, supongo que hay que chequearlo)
- Escribo la pagina

Memory Dump:
- Recorres la TP y escribis en el archivo DUMP


*** DUDAS
- Bits:
	- Bit de uso: no veo para q usarlo
	- Bit de Modificado: cuando un proceso vuelve de swap a memoria, las entradas de swap se borran,
		entonces no tendria sentido tenerlo, porque solo voy a tener el contenido en un lugar
	- Bit Presencia: si la pagina está en memoria (suspendido = false), todas las paginas estan en memoria. Si está suspendido, este bit no importa porque todas las paginas estan en swap

- Nose que vendria a ser "Acceso a TP"
*/
