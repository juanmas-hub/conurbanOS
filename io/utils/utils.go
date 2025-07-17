package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	globals "github.com/sisoputnfrba/tp-golang/globals/io"
)

func IniciarConfiguracion(filePath string) *globals.Io_Config {

	var config *globals.Io_Config
	configFile, err := os.Open(filePath)
	if err != nil {
		log.Println("error: ", err)
		log.Println(err.Error())
	}
	defer configFile.Close()

	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)

	return config
}

func HandshakeAKernel(ip string, puerto int64, nombreIO string, ipIO string, puertoIO int64) {

	handshake := globals.HandshakeIO{
		NombreIO:        nombreIO,
		NombreInstancia: globals.NombreInstancia,
		IP:              ipIO,
		Puerto:          puertoIO,
	}
	body, err := json.Marshal(handshake)
	if err != nil {
		slog.Debug(fmt.Sprintf("error codificando mensaje: %s", err.Error()))
	}

	url := fmt.Sprintf("http://%s:%d/handshakeIO", ip, puerto)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))

	if err != nil {
		slog.Debug(fmt.Sprintf("error en el handshake a ip:%s puerto:%d", ip, puerto))
	}

	slog.Debug(fmt.Sprintf("respuesta del servidor (handshake): %s", resp.Status))

}

func RecibirSolicitudDeKernel(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var solicitud globals.SolicitudIO
	err := decoder.Decode(&solicitud)
	if err != nil {
		slog.Debug(fmt.Sprintf("Error al decodificar solicitud de IO: %s\n", err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar solicitud de IO"))
		return
	}

	slog.Debug(fmt.Sprint("Me llego solicitud de IO"))
	slog.Debug(fmt.Sprintf("%+v\n", solicitud))

	globals.PidProcesoActual = solicitud.PID

	slog.Info(fmt.Sprintf("## PID: %d - Inicio de IO - Tiempo: %d", solicitud.PID, solicitud.Tiempo))

	go USleep(solicitud.Tiempo, solicitud.PID)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func USleep(tiempo int64, pid int64) {
	defer EnviarFinalizacionIOAKernel(globals.IoConfig.IpKernel, globals.IoConfig.PortKernel, pid)
	duracion := time.Duration(tiempo) * time.Millisecond
	time.Sleep(duracion)
}

func EnviarFinalizacionIOAKernel(ip string, puerto int64, pid int64) {

	mensaje := globals.FinalizacionIO{
		PID:             pid,
		NombreIO:        globals.IoConfig.NombreIO,
		NombreInstancia: globals.NombreInstancia,
	}

	slog.Info(fmt.Sprintf("## PID: %d - Fin de IO", pid))

	body, err := json.Marshal(mensaje)
	if err != nil {
		slog.Debug(fmt.Sprintf("error codificando mensaje: %s", err.Error()))
	}

	globals.PidProcesoActual = -1

	// Posible problema con el int64 del puerto
	url := fmt.Sprintf("http://%s:%d/finalizacionIO", ip, puerto)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))

	if err != nil {
		slog.Debug(fmt.Sprintf("error enviando mensaje a ip:%s puerto:%d", ip, puerto))
	}

	slog.Debug(fmt.Sprintf("respuesta del servidor: %s", resp.Status))

}

func Desconectar(ip string, puerto int64, pid int64) {

	mensaje := globals.DesconexionIO{
		NombreInstancia: globals.NombreInstancia,
		NombreIO:        globals.IoConfig.NombreIO,
		PID:             pid,
		Ip:              globals.IoConfig.IpIO,
		Puerto:          globals.IoConfig.PortIO,
	}
	body, err := json.Marshal(mensaje)
	if err != nil {
		slog.Debug(fmt.Sprintf("error codificando mensaje: %s", err.Error()))
	}

	// Posible problema con el int64 del puerto
	url := fmt.Sprintf("http://%s:%d/desconexionIO", ip, puerto)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))

	if err != nil {
		slog.Debug(fmt.Sprintf("error enviando mensaje a ip:%s puerto:%d", ip, puerto))
	}

	slog.Debug(fmt.Sprintf("respuesta del servidor: %s", resp.Status))

}
