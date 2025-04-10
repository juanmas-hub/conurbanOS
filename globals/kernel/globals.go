package globals

type Kernel_Config struct {
	Ip_memory           string `json:"ip_memory"`
	Port_memory         int    `json:"port_memory"`
	Port_kernel         int    `json:"port_kernel"`
	Scheduler_algorithm string `json:"scheduler_algorithm"`
	New_algorithm       string `json:"new_algorithm"`
	Alpha               string `json:"alpha"`
	Suspension_time     int    `json:"suspension_time"`
	Log_level           string `json:"log_level"`
}

var KernelConfig *Kernel_Config
