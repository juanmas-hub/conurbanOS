package globals

type Io_Config struct {
	IpKernel   string `json:"ip_kernel"`
	PortKernel int64  `json:"port_kernel"`
	IpIO       string `json:"ip_io"`
	PortIO     int64  `json:"port_io"`
	LogLevel   string `json:"log_level"`
}

var IoConfig *Io_Config

var NombreIO string

var PidProcesoActual int64 = -1

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
type FinalizacionIO struct {
	PID      int64  `json:"pid"`
	NombreIO string `json:"nombre"`
}

type DesconexionIO struct {
	NombreIO string `json:"nombre"`
	PID      int64  `json:"pid"`
	Ip       string `json:"ip"`
	Puerto   int64  `json:"port"`
}
