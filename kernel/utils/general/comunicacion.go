package utils_general

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
)

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

	slog.Debug(fmt.Sprintf("Finalizacion PID %d enviada a memoria, respuesta: %s", pid, resp.Status))

	if resp.StatusCode == http.StatusOK {
		return true
	}
	return false
}

func EnviarProcesoAEjecutar_ACPU(ip string, puerto int64, pid int64, pc int64, nombre string) {
	proc := globals.ProcesoAExecutar{
		PID: pid,
		PC:  pc,
	}

	//log.Printf("cpu libre elegida ip: %s, port: %d, pid: %d, pc: %d", ip, puerto, pid, pc)
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

	slog.Debug(fmt.Sprintf("Proceso PID %d enviado a %s, respuesta: %s", pid, nombre, resp.Status))

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

func EnviarInterrupcionACPU(ip string, puerto int64, nombre string, pid int64) (*globals.RespuestaInterrupcion, error) {
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

	slog.Debug(fmt.Sprintf("Interrupcion enviada a CPU: %s, resp: %s", nombre, resp.Status))

	// Respuesta de CPU
	var respuesta globals.RespuestaInterrupcion
	if err := json.NewDecoder(resp.Body).Decode(&respuesta); err != nil {
		log.Printf("error decodificando respuesta de la CPU: %s", err.Error())
		return nil, err
	}
	return &respuesta, nil
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

	globals.ListaIOsMutex.Lock()
	slog.Debug(fmt.Sprintf("Se levantó una nueva IO: %s", handshake.Nombre))
	agregarAInstanciasIOs(handshake)
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

	globals.ListaCPUsMutex.Lock()
	slog.Debug(fmt.Sprintf("Se levantó una nueva CPU: %s, IP: %s, Puerto: %d", handshake.Nombre, handshake.IP, handshake.Puerto))
	agregarAListaCPUs(handshake)
	globals.ListaCPUsMutex.Unlock()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

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

	slog.Debug(fmt.Sprintf("Solicitud IO enviada al modulo IO - PID: %d, Tiempo: %dms", pid, tiempo))

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("Error enviando solicitud IO a ipIO:%s puertoIO:%d", ipIO, puertoIO)
	}

	slog.Debug(fmt.Sprintf("Solicitud IO enviada al modulo IO - PID: %d, Tiempo: %dms, respuesta: %s", pid, tiempo, resp.Status))

	globals.CantidadSesionesIOMutex.Lock()
	globals.CantidadSesionesIO[pid]++
	globals.CantidadSesionesIOMutex.Unlock()
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

	slog.Debug(fmt.Sprintf("Enviado DUMP MEMORY a memoria, resp: %s", resp.Status))

	if resp.StatusCode == http.StatusOK {
		return true
	}
	return false
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
	globals.ListaIOsMutex.Unlock()

	pidProceso := desconexionIO.PID

	slog.Debug(fmt.Sprintf("Se desconecto el IO: %s, que tenia el proceso de PID: %d", desconexionIO.NombreIO, pidProceso))

	// Saco la instancia de la cola de instancias
	globals.ListaIOsMutex.Lock()
	posInstancia := buscarPosInstanciaIO(desconexionIO.NombreIO, desconexionIO.Ip, desconexionIO.Puerto)
	globals.ListaIOsMutex.Unlock()

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

	globals.ListaIOsMutex.Lock()
	globals.MapaIOs[desconexionIO.NombreIO] = io
	globals.ListaIOsMutex.Unlock()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func AvisarSwappeo(pid int64) {
	mensaje := globals.PidJSON{PID: pid}
	body, err := json.Marshal(mensaje)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	// Posible problema con el int64 del puerto
	url := fmt.Sprintf("http://%s:%d/suspenderProceso", globals.KernelConfig.Ip_memory, globals.KernelConfig.Port_memory)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando mensaje a ip:%s puerto:%d", globals.KernelConfig.Ip_memory, globals.KernelConfig.Port_memory)
	}

	slog.Debug(fmt.Sprintf("Enviado aviso de swappeo de PID %d a memoria, resp: %s", pid, resp.Status))
}

func SolicitarInicializarProcesoAMemoria_DesdeNEW(proceso globals.Proceso_Nuevo) bool {
	// Se pudo iniciarlizar => devuelve true
	// No se pudo inicializar => devuelve false

	mensaje := globals.SolicitudIniciarProceso{
		Archivo_Pseudocodigo: proceso.Archivo_Pseudocodigo,
		Tamanio:              proceso.Tamaño,
		Pid:                  proceso.Proceso.Pcb.Pid,
		Susp:                 false,
	}
	body, err := json.Marshal(mensaje)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	// Posible problema con el int64 del puerto
	url := fmt.Sprintf("http://%s:%d/iniciarProceso", globals.KernelConfig.Ip_memory, globals.KernelConfig.Port_memory)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando mensaje a ip:%s puerto:%d", globals.KernelConfig.Ip_memory, globals.KernelConfig.Port_memory)
	}

	log.Printf("respuesta del servidor: %s", resp.Status)

	if resp.Status == "200 OK" {
		return true
	}

	return false
}

func SolicitarInicializarProcesoAMemoria_DesdeSUSP_READY(pid int64) bool {
	// Se pudo iniciarlizar => devuelve true
	// No se pudo inicializar => devuelve false
	mensaje := globals.PidJSON{PID: pid}

	body, err := json.Marshal(mensaje)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	url := fmt.Sprintf("http://%s:%d/reanudarProceso", globals.KernelConfig.Ip_memory, globals.KernelConfig.Port_memory)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando mensaje a ip:%s puerto:%d", globals.KernelConfig.Ip_memory, globals.KernelConfig.Port_memory)
	}

	log.Printf("respuesta del servidor: %s", resp.Status)

	if resp.Status == "200 OK" {
		return true
	}

	return false
}
