package globals

type Io_Config struct {
	IpKernel   string `json:"ip_kernel"`
	PortKernel int64  `json:"port_kernel"`
	IpIO       int64  `json:"ip_io"`
	PortIO     int64  `json:"port_io"`
	LogLevel   string `json:"log_level"`
}

var IoConfig *Io_Config

type Mensaje struct {
	Mensaje string `json:"mensaje"`
}

type HandshakeIO struct {
	Nombre string `json:"nombre"`
	IP     string `json:"ip"`
	Puerto int64  `json:"puerto"`
}

type SolicitudIO struct {
	PID    int64 `json:"pid"`
	Tiempo int64 `json:"tiempo"`
}
