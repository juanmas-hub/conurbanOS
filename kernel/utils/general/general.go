package utils_general

// Configuracion y servidores

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

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

func EnviarFinalizacionDeProceso_AMemoria(ip string, puerto int64, pid int64) bool {
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
	if resp.StatusCode == http.StatusOK {
		return true
	}
	return false
}

func EnviarProcesoAEjecutar_ACPU(ip string, puerto int64, pid int64, pc int64) {
	proc := globals.ProcesoAExecutar{
		PID: pid,
		PC:  pc,
	}

	log.Printf("cpu libre elegida ip: %s, port: %d, pid: %d, pc: %d", ip, puerto, pid, pc)
	body, err := json.Marshal(proc)
	if err != nil {
		log.Printf("error codificando proceso a ejecutar: %s", err.Error())
	}

	// Posible problema con el int64 del puerto
	url := fmt.Sprintf("http://%s:%d/dispatchProceso", ip, puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando mensaje a ip:%s puerto:%d, error: %v", ip, puerto, err)
	}

	log.Printf("respuesta del servidor: %s", resp.Status)

	// Guardar PID en la CPU correspondiente
	globals.ListaCPUsMutex.Lock()
	for i := range globals.ListaCPUs {
		if globals.ListaCPUs[i].Handshake.IP == ip && globals.ListaCPUs[i].Handshake.Puerto == puerto {
			globals.ListaCPUs[i].PIDActual = pid
			break
		}
	}
	globals.ListaCPUsMutex.Unlock()
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
	AgregarAInstanciasIOs(handshake)
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

	globals.CantidadSesionesIOMutex.Lock()
	globals.CantidadSesionesIO[pid]++
	globals.CantidadSesionesIOMutex.Unlock()

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
		manejarFinIO(finalizacionIo)
		// LOG : Fin de IO: ## (<PID>) finalizó IO y pasa a READY
		slog.Info(fmt.Sprintf("## (%d) finalizó IO y pasa a READY", finalizacionIo.PID))
	}()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func manejarFinIO(finalizacionIo globals.FinalizacionIO) {
	globals.ListaIOsMutex.Lock()

	io := globals.MapaIOs[finalizacionIo.NombreIO]
	posInstanciaIo := BuscarInstanciaIO(finalizacionIo.NombreIO, finalizacionIo.PID)
	if posInstanciaIo == -1 {
		log.Printf("Error buscando instancia de IO de nombre: %s, con el proceso: %d", finalizacionIo.NombreIO, finalizacionIo.PID)
	}
	instanciaIo := io.Instancias[posInstanciaIo]

	// Cambio el PID del proceso actual
	instanciaIo.PidProcesoActual = -1
	io.Instancias[posInstanciaIo] = instanciaIo

	globals.MapaIOs[finalizacionIo.NombreIO] = io

	// Si hay procesos esperando IO, envio solicitud
	if len(globals.MapaIOs[finalizacionIo.NombreIO].ColaProcesosEsperando) > 0 {
		procesoAIO := globals.MapaIOs[finalizacionIo.NombreIO].ColaProcesosEsperando[0]
		instanciaIo.PidProcesoActual = procesoAIO.PID
		EnviarSolicitudIO(
			instanciaIo.Handshake.IP,
			instanciaIo.Handshake.Puerto,
			procesoAIO.PID,
			procesoAIO.Tiempo,
		)

		// Saco al nuevo proceso de la cola de procesos esperando
		io.ColaProcesosEsperando = io.ColaProcesosEsperando[1:]
	}

	io.Instancias[posInstanciaIo] = instanciaIo
	globals.MapaIOs[finalizacionIo.NombreIO] = io

	globals.ListaIOsMutex.Unlock()

	//log.Print("Se quiere loquear MapaProcesos en manejarFinIO")
	globals.MapaProcesosMutex.Lock()
	proceso := globals.MapaProcesos[finalizacionIo.PID]
	globals.MapaProcesosMutex.Unlock()
	//log.Print("Se unloquea MapaProcesos en manejarFinIO")

	// Si esta en Susp Blocked lo paso a Susp Ready
	if proceso.Estado_Actual == globals.SUSP_BLOCKED {
		SuspBlockedASuspReady(proceso)
	}

	// Si esta en Blocked lo paso Ready (no lo dice el enunciado!¡)
	if proceso.Estado_Actual == globals.BLOCKED {
		BlockedAReady(proceso)
	}
}

func SuspBlockedASuspReady(proceso globals.Proceso) {
	//log.Print("Se quiere loquear MapaProcesos en SuspBlockedASuspReady")
	globals.MapaProcesosMutex.Lock()
	proceso = ActualizarMetricas(proceso, proceso.Estado_Actual)
	proceso.Estado_Actual = globals.SUSP_READY
	globals.MapaProcesos[proceso.Pcb.Pid] = proceso
	globals.MapaProcesosMutex.Unlock()
	//log.Print("Se unloquea MapaProcesos en SuspBlockedASuspReady")

	pos := BuscarProcesoEnSuspBlocked(proceso.Pcb.Pid)

	globals.EstadosMutex.Lock()
	globals.ESTADOS.SUSP_BLOCKED = append(globals.ESTADOS.SUSP_BLOCKED[:pos], globals.ESTADOS.SUSP_BLOCKED[pos+1:]...)
	globals.ESTADOS.SUSP_READY = append(globals.ESTADOS.SUSP_READY, proceso.Pcb.Pid)
	globals.EstadosMutex.Unlock()

	log.Printf("Proceso de PID %d fue movido de Susp Blocked a Susp Ready", proceso.Pcb.Pid)

	globals.DeDondeSeLlamaMutex.Lock()
	globals.DeDondeSeLlamaPasarProcesosAReady = "SUSP READY"
	globals.DeDondeSeLlamaMutex.Unlock()
	Signal(globals.Sem_PasarProcesoAReady)

	// LOG Cambio de Estado: ## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>
	slog.Info(fmt.Sprintf("## (%d) Pasa del estado SUSP_BLOCKED al estado SUSP_READY", proceso.Pcb.Pid))
}

func BlockedAReady(proceso globals.Proceso) {
	//log.Print("Se quiere loquear MapaProcesos en BlockedAReady")
	globals.MapaProcesosMutex.Lock()
	proceso = ActualizarMetricas(proceso, proceso.Estado_Actual)
	proceso.Estado_Actual = globals.READY
	globals.MapaProcesos[proceso.Pcb.Pid] = proceso
	globals.MapaProcesosMutex.Unlock()
	//log.Print("Se unloquea MapaProcesos en BlockedAReady")

	pos := BuscarProcesoEnBlocked(proceso.Pcb.Pid)

	globals.EstadosMutex.Lock()
	globals.ESTADOS.BLOCKED = append(globals.ESTADOS.BLOCKED[:pos], globals.ESTADOS.BLOCKED[pos+1:]...)
	globals.ESTADOS.READY = append(globals.ESTADOS.SUSP_READY, proceso.Pcb.Pid)
	globals.EstadosMutex.Unlock()

	log.Printf("Proceso de PID %d fue movido de Blocked a Ready", proceso.Pcb.Pid)

	log.Printf("cantidad de procesos en READY: %+v", len(globals.ESTADOS.READY))

	NotificarProcesoEnReady(globals.NotificadorDesalojo)
	Signal(globals.Sem_ProcesosEnReady) // Nuevo proceso en ready

	// LOG Cambio de Estado: ## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>
	slog.Info(fmt.Sprintf("## (%d) Pasa del estado BLOCKED al estado READY", proceso.Pcb.Pid))
}

// Cuando se cambia de estado. Se tiene que llamar con el mutex del mapa proceso LOCKEADO, y antes de cambiar el estado al nuevo. Devuelve el proceso con las metricas cambiadas.
func ActualizarMetricas(proceso globals.Proceso, estadoAnterior string) globals.Proceso {
	// Falta hacer MT
	ahora := time.Now()

	ME := proceso.Pcb.ME
	MT := proceso.Pcb.MT
	tiempoEnEstado := ahora.Sub(proceso.UltimoCambioDeEstado)

	switch estadoAnterior {
	case globals.NEW:
		ME.New++
		MT.New += tiempoEnEstado
	case globals.READY:
		ME.Ready++
		MT.Ready += tiempoEnEstado
	case globals.EXECUTE:
		ME.Execute++
		MT.Execute += tiempoEnEstado
	case globals.BLOCKED:
		ME.Blocked++
		MT.Blocked += tiempoEnEstado
	case globals.SUSP_BLOCKED:
		ME.Susp_Blocked++
		MT.Susp_Blocked += tiempoEnEstado
	case globals.SUSP_READY:
		ME.Susp_Ready++
		MT.Susp_Ready += tiempoEnEstado
	default:
		// No deberia entrar nunca aca
	}

	proceso.Pcb.ME = ME
	proceso.Pcb.MT = MT
	proceso.UltimoCambioDeEstado = ahora

	return proceso
}

// Se llama con estados mutex lockeado
func BuscarProcesoEnBlocked(pid int64) int64 {
	colaBlocked := globals.ESTADOS.BLOCKED

	var posicion int64

	for indice, valor := range colaBlocked {
		if valor == pid {
			posicion = int64(indice)
			break
		}
	}

	return posicion
}

// Se llama con estados mutex lockeado
func BuscarProcesoEnExecute(pid int64) int64 {
	colaExecute := globals.ESTADOS.EXECUTE

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
	var desconexionIO globals.DesconexionIO
	err := decoder.Decode(&desconexionIO)
	if err != nil {
		log.Printf("Error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	globals.ListaIOsMutex.Lock()
	io := globals.MapaIOs[desconexionIO.NombreIO]

	pidProceso := desconexionIO.PID

	// Saco la instancia de la cola de instancias
	posInstancia := BuscarPosInstanciaIO(desconexionIO.NombreIO, desconexionIO.Ip, desconexionIO.Puerto)
	if posInstancia == -2 {
		log.Printf("Error buscando la instancia de IO de IP: %s, puerto: %d, que tendría el proceso: %d", desconexionIO.Ip, desconexionIO.Puerto, pidProceso)
	}
	io.Instancias = append(io.Instancias[:posInstancia], io.Instancias[posInstancia+1:]...)

	// Si habia proceso ejecutando
	if pidProceso != -1 {

		// Finalizo proceso que esta ejecuando en esa IO
		FinalizarProceso(pidProceso)
	}

	// Si no quedan mas instancias
	if len(io.Instancias) == 0 {
		// Finalizo todos los procesos de la cola esperando esa IO
		for i := range io.ColaProcesosEsperando {
			FinalizarProceso(io.ColaProcesosEsperando[i].PID)
		}
	}

	globals.MapaIOs[desconexionIO.NombreIO] = io
	globals.ListaIOsMutex.Unlock()

	log.Printf("Se desconecto el IO: %s, que tenia el proceso de PID: %d", desconexionIO.NombreIO, pidProceso)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func AgregarAInstanciasIOs(handshake globals.Handshake) {
	elementoAAgregar := globals.InstanciaIO{
		Handshake:        handshake,
		PidProcesoActual: -1,
	}
	io, existe := globals.MapaIOs[handshake.Nombre]
	if !existe {
		io = globals.EntradaMapaIO{}
	}
	io.Instancias = append(io.Instancias, elementoAAgregar)
	globals.MapaIOs[handshake.Nombre] = io
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
	for i := range globals.ListaCPUs {
		if globals.ListaCPUs[i].Handshake.Nombre == nombre {
			posCpu = i
			encontrado = true
			break
		}
	}

	if encontrado {
		return posCpu
	} else {
		// Si devuelve esto es que se desconecto la CPU en el medio. Hay q ser mala persona
		log.Println("No se encontro la CPU en la devolucion")
		return -1
	}
}

func BuscarCpuPorPID(pid int64) (string, int64, bool) {
	globals.ListaCPUsMutex.Lock()
	defer globals.ListaCPUsMutex.Unlock()

	for _, cpu := range globals.ListaCPUs {
		if !cpu.EstaLibre && cpu.PIDActual == pid {
			return cpu.Handshake.IP, cpu.Handshake.Puerto, true
		}
	}
	return "", 0, false
}

// Mandando PID, se finaliza ese proceso.
func FinalizarProceso(pid int64) {
	globals.ProcesosAFinalizarMutex.Lock()
	globals.ProcesosAFinalizar = append(globals.ProcesosAFinalizar, pid)
	globals.ProcesosAFinalizarMutex.Unlock()
	Signal(globals.Sem_ProcesoAFinalizar)
}

func NotificarProcesoEnReady(notificador chan struct{}) {
	select {
	case notificador <- struct{}{}: // intento mandar la señal
	default:
		// si el canal ya tiene una señal, no hago nada para no bloquear ni saturar
	}
}

// Mandando nombre del CPU, se libera. Aumenta el semaforo de Semaforos de CPU, entonces el planificador corto replanifica.
func LiberarCPU(nombreCPU string) {
	globals.ListaCPUsMutex.Lock()
	posCpu := BuscarCpu(nombreCPU)
	globals.ListaCPUs[posCpu].EstaLibre = true
	globals.ListaCPUsMutex.Unlock()
	Signal(globals.Sem_Cpus)
}

func ActualizarPC(pid int64, pc int64) {

	//log.Print("Se quiere loquear MapaProcesos en ActualizarPC")
	globals.MapaProcesosMutex.Lock()
	proceso := globals.MapaProcesos[pid]
	proceso.Pcb.PC = pc
	globals.MapaProcesos[pid] = proceso
	globals.MapaProcesosMutex.Unlock()
	//log.Print("Se unloquea MapaProcesos en ActualizarPC")
}

func EnviarDumpMemory(pid int64) bool {
	mensaje := globals.PidJSON{PID: pid}
	body, err := json.Marshal(mensaje)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	// Posible problema con el int64 del puerto
	url := fmt.Sprintf("http://%s:%d/memoryDump", globals.KernelConfig.Ip_memory, globals.KernelConfig.Port_memory)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando mensaje a ip:%s puerto:%d", globals.KernelConfig.Ip_memory, globals.KernelConfig.Port_memory)
	}

	log.Printf("respuesta del servidor: %s", resp.Status)
	if resp.StatusCode == http.StatusOK {
		return true
	}
	return false
}

// Dado el nombre del IO y una IP y Puerto, busca la instancia de IO que tiene ese IP, y devuelve su posicion en la cola. Se llama con Lista IOs muteada.
func BuscarPosInstanciaIO(nombreIO string, ip string, puerto int64) int {

	io := globals.MapaIOs[nombreIO]

	for i := range io.Instancias {
		if io.Instancias[i].Handshake.Puerto == puerto && io.Instancias[i].Handshake.IP == ip {
			return i
		}
	}

	return -2

}

// Dado un nombre IO, y un PID, busca la instancia donde esta ejecutando ese proceso. Retorna posicion en cola de instancias. Se llama con Lista IO muteada
func BuscarInstanciaIO(nombreIO string, pid int64) int {

	io := globals.MapaIOs[nombreIO]

	for i := range io.Instancias {
		if io.Instancias[i].PidProcesoActual == pid {
			return i
		}
	}

	return -1
}
