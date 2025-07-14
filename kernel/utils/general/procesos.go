package general

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"time"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
)

func ActualizarPC(pid int64, pc int64) {

	//log.Print("Se quiere loquear MapaProcesos en ActualizarPC")
	globals.MapaProcesosMutex.Lock()
	proceso := globals.MapaProcesos[pid]
	proceso.Pcb.PC = pc
	globals.MapaProcesos[pid] = proceso
	globals.MapaProcesosMutex.Unlock()
	//log.Print("Se unloquea MapaProcesos en ActualizarPC")
}

func NotificarReplanifSRT() {
	select {
	case globals.SrtReplanificarChan <- struct{}{}:
	default:
	}
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
		slog.Debug(fmt.Sprint("Actualizada metrica BLOCKED a: ", MT.Blocked))
	case globals.SUSP_BLOCKED:
		ME.Susp_Blocked++
		MT.Susp_Blocked += tiempoEnEstado
		slog.Debug(fmt.Sprint("Actualizada metrica SUSP_BLOCKED a: ", MT.Susp_Blocked))
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
