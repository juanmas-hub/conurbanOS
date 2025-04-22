package utils

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
)

func IniciarConfiguracion(filePath string) *globals.Kernel_Config {
	var config *globals.Kernel_Config
	configFile, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer configFile.Close()

	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)

	return config
}

func EnviarMensajeAMemoria(ip string, puerto int64, mensajeTxt string) {
	mensaje := globals.Mensaje{Mensaje: mensajeTxt}
	body, err := json.Marshal(mensaje)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	// Posible problema con el int64 del puerto
	url := fmt.Sprintf("http://%s:%d/mensajeDeKernel", ip, puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando mensaje a ip:%s puerto:%d", ip, puerto)
	}

	log.Printf("respuesta del servidor: %s", resp.Status)
}

func RecibirMensajeDeCpu(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var mensaje globals.Mensaje
	err := decoder.Decode(&mensaje)
	if err != nil {
		log.Printf("Error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	log.Println("Me llego un mensaje de CPU")
	log.Printf("%+v\n", mensaje)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func RecibirMensajeDeIo(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var mensaje globals.Mensaje
	err := decoder.Decode(&mensaje)
	if err != nil {
		log.Printf("Error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	log.Println("Me llego un mensaje de IO")
	log.Printf("%+v\n", mensaje)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func RecibirHandshakeIO(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var handshake globals.HandshakeIO
	err := decoder.Decode(&handshake)
	if err != nil {
		log.Printf("Error al decodificar handshake: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar handshake"))
		return
	}

	log.Println("Me llego un handshake de IO")
	log.Printf("%+v\n", handshake)

	globals.HandshakesIO = append(globals.HandshakesIO, handshake)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func RecibirHandshakeCPU(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var handshake globals.HandshakeIO
	err := decoder.Decode(&handshake)
	if err != nil {
		log.Printf("Error al decodificar handshake: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar handshake"))
		return
	}

	log.Println("Me llego un handshake de CPU")
	log.Printf("%+v\n", handshake)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// Todavia esta funcion no se usa (correctamente)
func EnviarSolicitudIO(ipIO string, puertoIO int64, pid int64, tiempo int64) {

	solicitud := globals.SolicitudIO{
		PID:    pid,
		Tiempo: tiempo,
	}

	body, err := json.Marshal(solicitud)
	if err != nil {
		log.Printf("Error codificando la solicitud IO: %s", err.Error())
	}

	url := fmt.Sprintf("http://%s:%d/solicitudDeIo", ipIO, puertoIO)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("Error enviando solicitud IO a ipIO:%s puertoIO:%d", ipIO, puertoIO)
	}

	log.Printf("Solicitud IO terminada al modulo IO - PID: %d, Tiempo: %dms", pid, tiempo)
	log.Printf("Respuesta del modulo IO: %s", resp.Status)
}

func IniciarPlanificadorLargoPlazo(archivo string, tamanio int64) {
	// Espera el Enter en otra rutina asi se puede abrir el servidor

	reader := bufio.NewReader(os.Stdin)
	log.Println("Planificador de largo plazo en STOP, presionar ENTER: ")
	for {
		text, _ := reader.ReadString('\n')
		log.Print(text)

		if text == "\n" {
			break
		}
	}

	CrearProcesoNuevo(archivo, tamanio)
}

func CrearProcesoNuevo(archivo string, tamanio int64) {
	pid := globals.PIDCounter
	globals.PIDCounter++

	proceso := globals.Proceso{
		Pcb: globals.PCB{
			Pid: pid,
			PC:  0,
			// Las metricas las inicializa en 0
		},
		Estado_Actual: globals.NEW,
		Rafaga:        nil,
	}

	procesoNuevo := globals.Proceso_Nuevo{
		Archivo_Pseudocodigo: archivo,
		Tamaño:               tamanio,
		Proceso:              proceso,
	}

	globals.ESTADOS.NEW = append(globals.ESTADOS.NEW, procesoNuevo)

	if globals.KernelConfig.New_algorithm == "PMCP" {
		OrdenarNewPorTamanio()
	}

	PasarProcesosAReady()
}

func OrdenarNewPorTamanio() {

	// Con ordenar por tamaño (mas chicho primero) ya el algoritmo PMCP estaria hecho (creo)
	sort.Slice(globals.ESTADOS.NEW, func(i, j int) bool {
		return globals.ESTADOS.NEW[i].Tamaño < globals.ESTADOS.NEW[j].Tamaño
	})
}

func PasarProcesosAReady() {
	// Esta funcion deberia llamarse cuando llega un proceso a NEW, a EXIT, a SUSP_BLOCKED y (SUSP_READY ???)
	// Voy a intentar pasar la mayor cantidad de procesos que pueda mientras memoria tenga espacio
	// Primero me fijo en SUSP READY y despues en NEW --- nose si esta bien hacerlo asi

	var lenghtSUSP_READY = len(globals.ESTADOS.SUSP_READY)
	for {
		proceso := globals.MapaProcesos[globals.ESTADOS.SUSP_READY[0]]
		if SolicitarInicializarProcesoAMemoria_DesdeSUSP_READY(proceso) == false {
			break
		}

		proceso.Estado_Actual = globals.READY
		globals.MapaProcesos[proceso.Pcb.Pid] = proceso
		globals.ESTADOS.SUSP_READY = globals.ESTADOS.SUSP_READY[1:]
		lenghtSUSP_READY--
		globals.ESTADOS.READY = append(globals.ESTADOS.READY, proceso.Pcb.Pid)
	}

	if lenghtSUSP_READY == 0 {
		for {
			if SolicitarInicializarProcesoAMemoria_DesdeNEW(globals.ESTADOS.NEW[0]) == false {
				break
			}

			procesoEnNew := globals.ESTADOS.NEW[0]
			procesoEnReady := globals.Proceso{
				Pcb:           procesoEnNew.Proceso.Pcb,
				Estado_Actual: globals.READY,
				Rafaga:        nil,
			}
			globals.MapaProcesos[procesoEnReady.Pcb.Pid] = procesoEnReady
			globals.ESTADOS.NEW = globals.ESTADOS.NEW[1:]
			globals.ESTADOS.READY = append(globals.ESTADOS.READY, procesoEnReady.Pcb.Pid)
		}
	}
}

func SolicitarInicializarProcesoAMemoria_DesdeNEW(proceso globals.Proceso_Nuevo) bool {
	// Se pudo iniciarlizar => devuelve true
	// No se pudo inicializar => devuelve false
	return true
}

func SolicitarInicializarProcesoAMemoria_DesdeSUSP_READY(proceso globals.Proceso) bool {
	// Se pudo iniciarlizar => devuelve true
	// No se pudo inicializar => devuelve false
	return true
}
