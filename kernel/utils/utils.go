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

func EnviarProcesoAEjecutar_ACPU(ip string, puerto int64, pid int64) {
	/*mensaje := globals.PidJSON{PID: pid}
	body, err := json.Marshal(mensaje)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	// Posible problema con el int64 del puerto
	url := fmt.Sprintf("http://%s:%d/dispatchProceso", ip, puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando mensaje a ip:%s puerto:%d", ip, puerto)
	}

	log.Printf("respuesta del servidor: %s", resp.Status)*/
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

	globals.HandshakesMutex.Lock()
	globals.HandshakesIO = append(globals.HandshakesIO, handshake)
	globals.HandshakesMutex.Unlock()

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

	globals.HandshakesMutex.Lock()
	globals.HandshakesCPU = append(globals.HandshakesCPU, handshake)
	globals.HandshakesMutex.Unlock()

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
			globals.PLANIFICADOR_LARGO_PLAZO_BLOCKED = false
			break
		}
	}

	CrearProcesoNuevo(archivo, tamanio)
}

func EjecutarPlanificadorCortoPlazo() {

	if globals.KernelConfig.Scheduler_algorithm == "FIFO" {
		globals.EstadosMutex.Lock()

		procesoAEjecutar := globals.ESTADOS.READY[0]
		ip, port := ElegirCPUlibre()
		EnviarProcesoAEjecutar_ACPU(ip, port, procesoAEjecutar)

		globals.MapaProcesosMutex.Lock()

		ReadyAExecute(globals.MapaProcesos[procesoAEjecutar])
		log.Printf("Proceso agregado a EXEC. Ahora tiene %d procesos", len(globals.ESTADOS.EXECUTE))

		globals.EstadosMutex.Unlock()
		globals.MapaProcesosMutex.Unlock()
	}

	if globals.KernelConfig.Scheduler_algorithm == "SJF" {
		globals.EstadosMutex.Lock()
		// SJF SIN DESALOJO (Se elige al proceso que tenga la rafaga estimada mas corta)
		// sort.SLice compara pares de elementos (i y j) si i < j -> true, si j < i -> false
		sort.Slice(globals.ESTADOS.READY, func(i, j int) bool {
			pidI := globals.ESTADOS.READY[i]
			pidJ := globals.ESTADOS.READY[j]

			// De cada par de procesos sacamos la rafaga que tiene cada uno
			rafagaI := globals.MapaProcesos[pidI].Rafaga
			rafagaJ := globals.MapaProcesos[pidJ].Rafaga
			// Si la rafagaI es menor, la ponemos antes
			return rafagaI.Est_Sgte < rafagaJ.Est_Sgte
		})

		procesoAEjecutar := globals.ESTADOS.READY[0]
		ip, port := ElegirCPUlibre()
		EnviarProcesoAEjecutar_ACPU(ip, port, procesoAEjecutar)
		globals.MapaProcesosMutex.Lock()
		ReadyAExecute(globals.MapaProcesos[procesoAEjecutar])
		log.Printf("Proceso agregado a EXEC. Ahora tiene %d procesos", len(globals.ESTADOS.EXECUTE))
		globals.EstadosMutex.Unlock()
		globals.MapaProcesosMutex.Unlock()
	}

	if globals.KernelConfig.Scheduler_algorithm == "SRT" {
		// Con desalojo
		// No se como sería esto. Capaz hay q hacer una funcion aparte porque se llamaria en momentos distintos

		if len(globals.ESTADOS.EXECUTE) > 0 {
			pidEnExec := globals.ESTADOS.EXECUTE[0]
			rafagaExec := globals.MapaProcesos[pidEnExec].Rafaga.Est_Sgte
			rafagaNuevo := globals.MapaProcesos[globals.ESTADOS.READY[0]].Rafaga.Est_Sgte

			if rafagaNuevo < rafagaExec {
				// InterrupcionACpu(pidEnExec)
			}
		}

	}
}

func ActualizarEstimado(pid int64, rafagaReal int64) {
	// En desarrollo
	//proceso := globals.MapaProcesos[pid]
	//alpha := globals.KernelConfig.Alpha
	//ant := proceso.Rafaga.Est_Sgte

	// Est(n+1) =  R(n) + (1-) Est(n) ;    [0,1]

	//globals.MapaProcesos[pid] = proceso
}

func ElegirCPUlibre() (string, int64) {
	// Hay que hacerlo. Seguramente haya que cambiar HandshakesCPU para indicar cual esta libre

	return globals.HandshakesCPU[0].IP, globals.HandshakesCPU[0].Puerto
}

func CrearProcesoNuevo(archivo string, tamanio int64) {

	globals.PIDCounterMutex.Lock()

	pid := globals.PIDCounter
	globals.PIDCounter++

	globals.PIDCounterMutex.Unlock()

	log.Printf("Creando nuevo proceso con PID %d y tamaño %d\n", pid, tamanio)
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

	globals.EstadosMutex.Lock()
	log.Printf("Agregando proceso a NEW. Cantidad actual: %d", len(globals.ESTADOS.NEW))

	globals.ESTADOS.NEW = append(globals.ESTADOS.NEW, procesoNuevo)
	log.Printf("Después de agregar, NEW tiene %d procesos", len(globals.ESTADOS.NEW))
	if globals.KernelConfig.New_algorithm == "PMCP" {
		OrdenarNewPorTamanio()
	}
	globals.EstadosMutex.Unlock()

	// Si se crea un proceso nuevo antes de que aprete Enter, se agrega a NEW pero no se pasan procesos a READY
	if globals.PLANIFICADOR_LARGO_PLAZO_BLOCKED == false {
		PasarProcesosAReady()
	}
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

	globals.EstadosMutex.Lock()
	globals.MapaProcesosMutex.Lock()

	var lenghtSUSP_READY = len(globals.ESTADOS.SUSP_READY)
	for lenghtSUSP_READY > 0 {
		proceso := globals.MapaProcesos[globals.ESTADOS.SUSP_READY[0]]
		if SolicitarInicializarProcesoAMemoria_DesdeSUSP_READY(proceso) == false {
			break
		}

		SuspReadyAReady(proceso)
		lenghtSUSP_READY--
	}

	if lenghtSUSP_READY == 0 {

		for len(globals.ESTADOS.NEW) > 0 {
			procesoNuevo := globals.ESTADOS.NEW[0]

			if SolicitarInicializarProcesoAMemoria_DesdeNEW(procesoNuevo) == false {
				break
			}

			NewAReady(procesoNuevo)
		}
	}

	globals.EstadosMutex.Unlock()
	globals.MapaProcesosMutex.Unlock()

	// Esto es solo para probar si funciona, hay que ver en que momentos se llama a esa funcion
	go EjecutarPlanificadorCortoPlazo()
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

func FinalizarProceso(pid int64) {
	proceso, ok := globals.MapaProcesos[pid]
	if !ok {
		log.Printf("No se encontró el proceso con PID %d", pid)
		return
	}

	// Mando el PID
	EnviarFinalizacionDeProceso_AMemoria(globals.KernelConfig.Ip_memory, globals.KernelConfig.Port_memory, pid)

	// Confirmación de la memoria aca...
	// Me parece que la confirmacion es por la misma funcion que por la que mandas el mensaje (memoria no tiene ip y port del kernel)
	// Que pasa si no puede finalizarlo? O no puede pasar eso?
	RecibirConfirmacionDeMemoria(proceso.Pcb.Pid)

	delete(globals.MapaProcesos, pid)
	log.Printf("El PCB del proceso con PID %d fue liberado", pid)

	// Me imagino que hay que eliminarlo de de las colas tambien, o no?
	// Diria yo que ya esta eliminado de las colas, esta funcion se llamaria cuando un proceso pasa a exit, y en todos
	// los cambios de estado los sacamos de la cola anterior

	// Iniciar nuevos procesos
	PasarProcesosAReady()

	// Loguear metricas de estado
}

func RecibirConfirmacionDeMemoria(pid int64) bool {

	return true
}

// Funciones para no hacer tanto quilombo en pasar procesos de un estado a otro

func NewAReady(proceso globals.Proceso_Nuevo) {

	procesoEnReady := globals.Proceso{
		Pcb:           proceso.Proceso.Pcb,
		Estado_Actual: globals.READY,
		Rafaga:        nil,
	}
	globals.MapaProcesos[procesoEnReady.Pcb.Pid] = procesoEnReady
	globals.ESTADOS.NEW = globals.ESTADOS.NEW[1:]
	globals.ESTADOS.READY = append(globals.ESTADOS.READY, procesoEnReady.Pcb.Pid)

	log.Printf("cantidad de procesos en READY: %+v", len(globals.ESTADOS.READY))

}

func SuspReadyAReady(proceso globals.Proceso) {

	proceso.Estado_Actual = globals.READY
	globals.MapaProcesos[proceso.Pcb.Pid] = proceso
	globals.ESTADOS.SUSP_READY = globals.ESTADOS.SUSP_READY[1:]
	globals.ESTADOS.READY = append(globals.ESTADOS.READY, proceso.Pcb.Pid)

}

func ReadyAExecute(proceso globals.Proceso) {
	// Esto funcionaría para FIFO y SJF. Nose si SRT

	proceso.Estado_Actual = globals.EXECUTE
	globals.MapaProcesos[proceso.Pcb.Pid] = proceso
	globals.ESTADOS.READY = globals.ESTADOS.READY[1:]
	globals.ESTADOS.EXECUTE = append(globals.ESTADOS.EXECUTE, proceso.Pcb.Pid)
}
