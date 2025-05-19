package utils_general

// Configuracion y servidores

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

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

func Wait(semaforo globals.Semaforo) {
	<-semaforo
}

func Signal(semaforo globals.Semaforo) {
	semaforo <- struct{}{}
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

func EnviarFinalizacionDeProceso_AMemoria(ip string, puerto int64, pid int64) {
	mensaje := globals.PidJSON{PID: pid}
	body, err := json.Marshal(mensaje)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	// Posible problema con el int64 del puerto
	url := fmt.Sprintf("http://%s:%d/finalizarProceso", ip, puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando mensaje a ip:%s puerto:%d", ip, puerto)
	}

	log.Printf("respuesta del servidor: %s", resp.Status)
}

func EnviarProcesoAEjecutar_ACPU(ip string, puerto int64, pid int64, pc int64) {
	proc := globals.ProcesoAExecutar{
		PID: pid,
		PC:  pc,
	}

	body, err := json.Marshal(proc)
	if err != nil {
		log.Printf("error codificando proceso a ejecutar: %s", err.Error())
	}

	// Posible problema con el int64 del puerto
	url := fmt.Sprintf("http://%s:%d/dispatchProceso", ip, puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando mensaje a ip:%s puerto:%d", ip, puerto)
	}

	log.Printf("respuesta del servidor: %s", resp.Status)
}

func EnviarInterrupcionACPU(ip string, puerto int64, pid int64) {
	mensaje := globals.PidJSON{PID: pid}
	body, err := json.Marshal(mensaje)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	// Posible problema con el int64 del puerto
	url := fmt.Sprintf("http://%s:%d/interrumpir", ip, puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando interrupción a ip:%s puerto:%d", ip, puerto)
	}
	log.Printf("Interrupción enviada a CPU - PID: %d", pid)
	log.Printf("respuesta de la CPU: %s", resp.Status)
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
	var handshake globals.Handshake
	err := decoder.Decode(&handshake)
	if err != nil {
		log.Printf("Error al decodificar handshake: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar handshake"))
		return
	}

	log.Println("Me llego un handshake de IO")
	log.Printf("%+v\n", handshake)

	globals.ListaIOsMutex.Lock()
	AgregarAListaIOs(handshake)
	globals.ListaIOsMutex.Unlock()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func RecibirHandshakeCPU(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var handshake globals.Handshake
	err := decoder.Decode(&handshake)
	if err != nil {
		log.Printf("Error al decodificar handshake: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar handshake"))
		return
	}

	log.Println("Me llego un handshake de CPU")
	log.Printf("%+v\n", handshake)

	globals.ListaCPUsMutex.Lock()
	AgregarAListaCPUs(handshake)
	globals.ListaCPUsMutex.Unlock()

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

	log.Printf("Solicitud IO enviada al modulo IO - PID: %d, Tiempo: %dms", pid, tiempo)
	log.Printf("Respuesta del modulo IO: %s", resp.Status)
}

func FinalizacionIO(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var finalizacionIo globals.FinalizacionIO
	err := decoder.Decode(&finalizacionIo)
	if err != nil {
		log.Printf("Error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	log.Printf("Finalizo el IO del PID: %d", finalizacionIo.PID)
	log.Printf("%+v\n", finalizacionIo)

	go func() {
		globals.ListaIOsMutex.Lock()

		// Elimino de la cola
		posIo, _ := ObtenerIO(finalizacionIo.NombreIO)
		globals.ListaIOs[posIo].PidProcesoActual = -1
		globals.ListaIOs[posIo].ColaProcesosEsperando = globals.ListaIOs[posIo].ColaProcesosEsperando[1:]
		log.Println("Length cola procesos esperando en: " + finalizacionIo.NombreIO + ": " + strconv.Itoa(len(globals.ListaIOs[posIo].ColaProcesosEsperando)))

		// Si hay procesos esperando IO, envio solicitud
		if len(globals.ListaIOs[posIo].ColaProcesosEsperando) > 0 {
			procesoAIO := globals.ListaIOs[posIo].ColaProcesosEsperando[0]
			globals.ListaIOs[posIo].PidProcesoActual = procesoAIO.PID
			EnviarSolicitudIO(
				globals.ListaIOs[posIo].Handshake.IP,
				globals.ListaIOs[posIo].Handshake.Puerto,
				procesoAIO.PID,
				procesoAIO.Tiempo,
			)

		}

		globals.ListaIOsMutex.Unlock()

		globals.MapaProcesosMutex.Lock()
		proceso := globals.MapaProcesos[finalizacionIo.PID]
		globals.MapaProcesosMutex.Unlock()

		// Si esta en Susp Blocked lo paso a Susp Ready
		if proceso.Estado_Actual == globals.SUSP_BLOCKED {
			globals.MapaProcesosMutex.Lock()
			proceso.Estado_Actual = globals.SUSP_READY
			globals.MapaProcesos[finalizacionIo.PID] = proceso
			globals.MapaProcesosMutex.Unlock()

			pos := BuscarProcesoEnSuspBlocked(proceso.Pcb.Pid)

			globals.EstadosMutex.Lock()
			globals.ESTADOS.SUSP_BLOCKED = append(globals.ESTADOS.SUSP_BLOCKED[:pos], globals.ESTADOS.SUSP_BLOCKED[pos+1:]...)
			globals.ESTADOS.SUSP_READY = append(globals.ESTADOS.SUSP_READY, proceso.Pcb.Pid)
			globals.EstadosMutex.Unlock()
		}

		// Si esta en Blocked lo paso Ready (no lo dice el enunciado!¡)
		if proceso.Estado_Actual == globals.BLOCKED {
			globals.MapaProcesosMutex.Lock()
			proceso.Estado_Actual = globals.BLOCKED
			globals.MapaProcesos[finalizacionIo.PID] = proceso
			globals.MapaProcesosMutex.Unlock()

			pos := BuscarProcesoEnBlocked(proceso.Pcb.Pid)

			globals.EstadosMutex.Lock()
			globals.ESTADOS.BLOCKED = append(globals.ESTADOS.SUSP_BLOCKED[:pos], globals.ESTADOS.SUSP_BLOCKED[pos+1:]...)
			globals.ESTADOS.READY = append(globals.ESTADOS.SUSP_READY, proceso.Pcb.Pid)
			globals.EstadosMutex.Unlock()
		}
	}()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func BuscarProcesoEnBlocked(pid int64) int64 {
	globals.EstadosMutex.Lock()
	colaBlocked := globals.ESTADOS.BLOCKED
	globals.EstadosMutex.Unlock()

	var posicion int64

	for indice, valor := range colaBlocked {
		if valor == pid {
			posicion = int64(indice)
			break
		}
	}

	return posicion
}

func BuscarProcesoEnExecute(pid int64) int64 {
	globals.EstadosMutex.Lock()
	colaExecute := globals.ESTADOS.EXECUTE
	globals.EstadosMutex.Unlock()

	var posicion int64

	for indice, valor := range colaExecute {
		if valor == pid {
			posicion = int64(indice)
			break
		}
	}

	return posicion
}

func BuscarProcesoEnNew(pid int64) int64 {
	globals.EstadosMutex.Lock()
	colaNew := globals.ESTADOS.NEW
	globals.EstadosMutex.Unlock()

	var posicion int64

	for indice, valor := range colaNew {
		if valor.Proceso.Pcb.Pid == pid {
			posicion = int64(indice)
			break
		}
	}

	return posicion
}

func BuscarProcesoEnSuspBlocked(pid int64) int64 {
	globals.EstadosMutex.Lock()
	colaSuspBlocked := globals.ESTADOS.SUSP_BLOCKED
	globals.EstadosMutex.Unlock()

	var posicion int64

	for indice, valor := range colaSuspBlocked {
		if valor == pid {
			posicion = int64(indice)
			break
		}
	}

	return posicion
}

func BuscarProcesoEnSuspReady(pid int64) int64 {
	globals.EstadosMutex.Lock()
	colaSuspReady := globals.ESTADOS.SUSP_READY
	globals.EstadosMutex.Unlock()

	var posicion int64

	for indice, valor := range colaSuspReady {
		if valor == pid {
			posicion = int64(indice)
			break
		}
	}

	return posicion
}

func BuscarProcesoEnReady(pid int64) int64 {
	globals.EstadosMutex.Lock()
	colaReady := globals.ESTADOS.READY
	globals.EstadosMutex.Unlock()

	var posicion int64

	for indice, valor := range colaReady {
		if valor == pid {
			posicion = int64(indice)
			break
		}
	}

	return posicion
}

func DesconexionIO(w http.ResponseWriter, r *http.Request) {
	// Cuando se desconecta un IO, se pasa a exit el proceso que estaba en el IO.

	decoder := json.NewDecoder(r.Body)
	var desconexionIO globals.FinalizacionIO
	err := decoder.Decode(&desconexionIO)
	if err != nil {
		log.Printf("Error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	globals.ListaIOsMutex.Lock()
	posIo, _ := ObtenerIO(desconexionIO.NombreIO)
	io := globals.ListaIOs[posIo]

	// Elimino de lista IOs
	globals.ListaIOs = append(globals.ListaIOs[:posIo], globals.ListaIOs[posIo+1:]...)

	globals.ListaIOsMutex.Unlock()

	log.Printf("Se desconecto el IO: %s, que tenia el proceso de PID: %d", io.Handshake.Nombre, desconexionIO.PID)

	if desconexionIO.PID != -1 {
		// Finalizo proceso
		ProcesoAExit(desconexionIO.PID)

		// Nose que hay que hacer con los de la cola de esa IO (no dice el enunciado)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func ProcesoAExit(pid int64) {
	globals.ProcesosAFinalizarMutex.Lock()
	globals.ProcesosAFinalizar = append(globals.ProcesosAFinalizar, pid)
	globals.ProcesosAFinalizarMutex.Unlock()
	Signal(globals.Sem_ProcesoAFinalizar)
}

func ObtenerIO(nombre string) (int64, bool) {
	// Hay que buscar por nombre en la ListaIOs

	return 0, true
}

func AgregarAListaIOs(handshake globals.Handshake) {
	elementoAAgregar := globals.ListaIo{
		Handshake:             handshake,
		PidProcesoActual:      -1,
		ColaProcesosEsperando: nil,
	}
	globals.ListaIOs = append(globals.ListaIOs, elementoAAgregar)
}

func AgregarAListaCPUs(handshake globals.Handshake) {
	elementoAAgregar := globals.ListaCpu{
		Handshake: handshake,
		EstaLibre: true,
	}
	globals.ListaCPUs = append(globals.ListaCPUs, elementoAAgregar)
	Signal(globals.Sem_Cpus)
}

func BuscarCpu(nombre string) int {
	var posCpu int
	encontrado := false
	globals.ListaCPUsMutex.Lock()
	for i := range globals.ListaCPUs {
		if globals.ListaCPUs[i].Handshake.Nombre == nombre {
			posCpu = i
			encontrado = true
		}
	}

	globals.ListaCPUsMutex.Unlock()

	if encontrado {
		return posCpu
	} else {
		// Si devuelve esto es que se desconecto la CPU en el medio. Hay q ser mala persona
		log.Println("No se encontro la CPU en la devolucion")
		return -1
	}
}
