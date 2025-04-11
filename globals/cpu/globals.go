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
