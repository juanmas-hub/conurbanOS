package utils_general

// Configuracion y servidores

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
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

		globals.MapaProcesosMutex.Lock()
		proceso := globals.MapaProcesos[finalizacionIo.PID]
		globals.MapaProcesosMutex.Unlock()

		// Si esta en Susp Blocked lo paso a Susp Ready
		if proceso.Estado_Actual == globals.SUSP_BLOCKED {
			globals.MapaProcesosMutex.Lock()
			proceso = ActualizarMetricas(proceso, proceso.Estado_Actual)
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
			proceso = ActualizarMetricas(proceso, proceso.Estado_Actual)
			proceso.Estado_Actual = globals.READY
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

// Dada una cola y un PID, busca el proceso en la cola y devuelve la posicion.
func buscarProcesoEnColaIO(cola []globals.SyscallIO, pid int64) int {
	return 0
}

// Cuando se cambia de estado. Se tiene que llamar con el mutex del mapa proceso LOCKEADO, y antes de cambiar el estado al nuevo. Devuelve el proceso con las metricas cambiadas.
func ActualizarMetricas(proceso globals.Proceso, estadoAnterior string) globals.Proceso {
	// Falta hacer MT
	ahora := time.Now()

	switch estadoAnterior {
	case globals.NEW:
		ME := proceso.Pcb.ME
		ME.New++
		proceso.Pcb.ME = ME
		MT := proceso.Pcb.MT
		MT.New = MT.New + ahora.Sub(proceso.UltimoCambioDeEstado)
		proceso.UltimoCambioDeEstado = ahora
		return proceso
	case globals.READY:
		ME := proceso.Pcb.ME
		ME.Ready++
		proceso.Pcb.ME = ME
		MT := proceso.Pcb.MT
		MT.Ready = MT.Ready + ahora.Sub(proceso.UltimoCambioDeEstado)
		proceso.UltimoCambioDeEstado = ahora
		return proceso
	case globals.EXECUTE:
		ME := proceso.Pcb.ME
		ME.Execute++
		proceso.Pcb.ME = ME
		MT := proceso.Pcb.MT
		MT.Execute = MT.Execute + ahora.Sub(proceso.UltimoCambioDeEstado)
		proceso.UltimoCambioDeEstado = ahora
		return proceso
	case globals.BLOCKED:
		ME := proceso.Pcb.ME
		ME.Blocked++
		proceso.Pcb.ME = ME
		MT := proceso.Pcb.MT
		MT.Blocked = MT.Blocked + ahora.Sub(proceso.UltimoCambioDeEstado)
		proceso.UltimoCambioDeEstado = ahora
		return proceso
	case globals.SUSP_BLOCKED:
		ME := proceso.Pcb.ME
		ME.Susp_Blocked++
		proceso.Pcb.ME = ME
		MT := proceso.Pcb.MT
		MT.Susp_Blocked = MT.Susp_Blocked + ahora.Sub(proceso.UltimoCambioDeEstado)
		proceso.UltimoCambioDeEstado = ahora
		return proceso
	case globals.SUSP_READY:
		ME := proceso.Pcb.ME
		ME.Susp_Blocked++
		proceso.Pcb.ME = ME
		MT := proceso.Pcb.MT
		MT.Susp_Ready = MT.Susp_Ready + ahora.Sub(proceso.UltimoCambioDeEstado)
		proceso.UltimoCambioDeEstado = ahora
		return proceso
	default:
		// No deberia entrar nunca aca
		return proceso
	}
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
	globals.MapaProcesosMutex.Lock()
	proceso := globals.MapaProcesos[pid]
	proceso.Pcb.PC = pc
	globals.MapaProcesosMutex.Unlock()
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
