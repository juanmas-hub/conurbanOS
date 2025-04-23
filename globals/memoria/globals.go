package globals

type Memoria_Config struct {
	Port_memory      int64  `json:"port_memory"`
	Memory_size      int64  `json:"memory_size"`
	Page_size        int64  `json:"page_size"`
	Entries_per_page int64  `json:"entries_per_page"`
	Number_of_levels int64  `json:"number_of_levels"`
	Memory_delay     int64  `json:"memory_delay"`
	Swapfile_path    string `json:"swapfile_path"`
	Swap_delay       int64  `json:"swap_delay"`
	Log_level        string `json:"log_level"`
	Dump_path        string `json:"dump_path"`
}

var MemoriaConfig *Memoria_Config

type Mensaje struct {
	Mensaje string `json:"mensaje"`
}

type Memoria_Metrica struct {
    AccesosTablas         	 int `json:"accesos_tablas"`
    InstruccionesSolicitadas int `json:"instrucciones_solicitadas"`
    BajadasSwap           	 int `json:"bajadas_swap"`
    SubidasMemoria        	 int `json:"subidas_memoria"`
    LecturasMemoria       	 int `json:"lecturas_memoria"`
    EscriturasMemoria     	 int `json:"escrituras_memoria"`
}

// Entrada de una tabla de páginas
type EntradaTablaPagina struct {
    Presente      bool  // ¿Está en RAM?
	PaginaVirtual int	// No es necesario pues el indice de la entrada en la tabla de paginas ya lo indica
    Marco         int   // Número de marco en RAM
    DireccionSwap int   // Dirección en disco si fue swappeada
}

// Nivel 2
type TablaSecundaria []EntradaTablaPagina

// Nivel 1
type DirectorioPaginas []*TablaSecundaria



 