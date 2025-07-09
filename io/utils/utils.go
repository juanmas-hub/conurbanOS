package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	globals "github.com/sisoputnfrba/tp-golang/globals/io"
)

func IniciarConfiguracion(filePath string) *globals.Io_Config {

	var config *globals.Io_Config
	configFile, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer configFile.Close()

	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)

	return config
}

func HandshakeAKernel(ip string, puerto int64, nombreIO string, ipIO string, puertoIO int64) {

	handshake := globals.HandshakeIO{
		Nombre: nombreIO,
		IP:     ipIO,
		Puerto: puertoIO,
	}
	body, err := json.Marshal(handshake)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	url := fmt.Sprintf("http://%s:%d/handshakeIO", ip, puerto)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))

	if err != nil {
		log.Printf("error en el handshake a ip:%s puerto:%d", ip, puerto)
	}

	log.Printf("respuesta del servidor (handshake): %s", resp.Status)

}

// Todavia esta funcion no se usa
func RecibirSolicitudDeKernel(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var solicitud globals.SolicitudIO
	err := decoder.Decode(&solicitud)
	if err != nil {
		log.Printf("Error al decodificar solicitud de IO: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar solicitud de IO"))
		return
	}

	log.Println("Me llego solicitud de IO")
	log.Printf("%+v\n", solicitud)

	globals.PidProcesoActual = solicitud.PID

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
		PID:      pid,
		NombreIO: globals.NombreIO,
	}
	body, err := json.Marshal(mensaje)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	globals.PidProcesoActual = -1

	// Posible problema con el int64 del puerto
	url := fmt.Sprintf("http://%s:%d/finalizacionIO", ip, puerto)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))

	if err != nil {
		log.Printf("error enviando mensaje a ip:%s puerto:%d", ip, puerto)
	}

	log.Printf("respuesta del servidor: %s", resp.Status)

}

func Desconectar(ip string, puerto int64, pid int64) {

	mensaje := globals.DesconexionIO{
		NombreIO: globals.NombreIO,
		PID:      pid,
		Ip:       globals.IoConfig.IpIO,
		Puerto:   globals.IoConfig.PortIO,
	}
	body, err := json.Marshal(mensaje)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	// Posible problema con el int64 del puerto
	url := fmt.Sprintf("http://%s:%d/desconexionIO", ip, puerto)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))

	if err != nil {
		log.Printf("error enviando mensaje a ip:%s puerto:%d", ip, puerto)
	}

	log.Printf("respuesta del servidor: %s", resp.Status)

}
