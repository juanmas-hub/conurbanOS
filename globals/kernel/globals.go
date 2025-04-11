package globals

type Kernel_Config struct {
	Ip_memory           string `json:"ip_memory"`
	Port_memory         int64  `json:"port_memory"`
	Port_kernel         int64  `json:"port_kernel"`
	Scheduler_algorithm string `json:"scheduler_algorithm"`
	New_algorithm       string `json:"new_algorithm"`
	Alpha               string `json:"alpha"`
	Suspension_time     int64  `json:"suspension_time"`
	Log_level           string `json:"log_level"`
}

var KernelConfig *Kernel_Config

type Mensaje struct {
	Mensaje string `json:"mensaje"`
}
