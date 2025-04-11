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
