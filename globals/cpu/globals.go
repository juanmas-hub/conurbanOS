package globals

type Cpu_Config struct {
	Port_cpu          int64  `json:"port_cpu"`
	Ip_memory         string `json:"ip_memory"`
	Port_memory       int64  `json:"port_memory"`
	Ip_kernel         string `json:"ip_kernel"`
	Port_kernel       int64  `json:"port_kernel"`
	Tlb_entries       int64  `json:"tlb_entries"`
	Tlb_replacement   string `json:"tlb_replacement"`
	Cache_entries     int64  `json:"cache_entries"`
	Cache_replacement string `json:"cache_replacement"`
	Cache_delay       int64  `json:"cache_delay"`
	Log_level         string `json:"log_level"`
}

var CpuConfig *Cpu_Config

type Mensaje struct {
	Mensaje string `json:"mensaje"`
}

type HandshakeCPU struct {
	Nombre string `json:"nombre"`
	IP     string `json:"ip"`
	Puerto int64  `json:"puerto"`
}

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

type SolicitudInstruccion struct {
	Pid int64 `json:"pid"`
	PC  int64 `json:"pc"`
}

type InstruccionDecodificada struct {
	Nombre     string
	Parametros []string
}

type ProcesoAExecutar struct {
	PID int64 `json:"PID"`
	PC  int64 `json:"PC"`
}
