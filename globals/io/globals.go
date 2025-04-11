package globals

type Io_Config struct {
	IpKernel   string `json:"ip_kernel"`
	PortKernel int64  `json:"port_kernel"`
	PortIO     int64  `json:"port_io"`
	LogLevel   int64  `json:"log_level"`
}

var IoConfig *Io_Config
