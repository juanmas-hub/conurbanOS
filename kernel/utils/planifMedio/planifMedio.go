package utils_planifMedio

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
	general "github.com/sisoputnfrba/tp-golang/kernel/utils/general"
	pl "github.com/sisoputnfrba/tp-golang/kernel/utils/planifLargo"
)

// ----- FUNCIONES EXPORTADAS -------

func BloquearProcesoDesdeExecute(proceso globals.Proceso) {
	// Esta funcion deberia llamarse cuando un proceso en ejecucion llama a IO con la syscall IO (desde syscallController)

	// -- Paso el proceso entre las colas
	// Como la cola de Execute 'no tiene' orden (todos los que estan en execute tienen una cpu ya ejecutando)
	// no se saca el primero de la cola como en las otras funciones

	globals.MapaProcesosMutex.Lock()
	proceso = general.ActualizarMetricas(proceso, proceso.Estado_Actual)
	proceso.Estado_Actual = globals.BLOCKED

	globals.MapaProcesos[proceso.Pcb.Pid] = proceso
	globals.MapaProcesosMutex.Unlock()

	globals.EstadosMutex.Lock()
	pos := general.BuscarProcesoEnExecute(proceso.Pcb.Pid)
	globals.ESTADOS.EXECUTE = append(globals.ESTADOS.EXECUTE[:pos], globals.ESTADOS.EXECUTE[pos+1:]...)
	globals.ESTADOS.BLOCKED = append(globals.ESTADOS.BLOCKED, proceso.Pcb.Pid)
	globals.EstadosMutex.Unlock()

	// -- Timer hasta ser suspendido
	go timer(globals.KernelConfig.Suspension_time, proceso)

}

// ----- FUNCIONES LOCALES -------

func timer(tiempo int64, proceso globals.Proceso) {
	defer sigueBloqueado(proceso)
	duracion := time.Duration(tiempo) * time.Millisecond
	time.Sleep(duracion)
}

func sigueBloqueado(proceso globals.Proceso) {
	// Si sigue bloqueado (en IO) hay que suspenderlo
	// Para que no siga bloqueado, el proceso tuvo que terminar su IO (lo recibimos como mensaje desde IO, siendo kernel servidor)
	// Cuando kernel reciba de IO el mensaje, ahí le cambiamos el estado

	globals.MapaProcesosMutex.Lock()
	procesoActualmente := globals.MapaProcesos[proceso.Pcb.Pid]
	globals.MapaProcesosMutex.Unlock()

	if procesoActualmente.Estado_Actual == globals.BLOCKED {
		// Aviso a memoria que hay que swappear
		avisarSwappeo(procesoActualmente.Pcb.Pid)

		// Cambio de estado
		blockedASuspBlocked(proceso)

		// Libere espacio => llamo a nuevos procesos
		pl.PasarProcesosAReady()
	}
}

func blockedASuspBlocked(proceso globals.Proceso) {
	// Muevo el proceso en la colas
	globals.MapaProcesosMutex.Lock()
	proceso = general.ActualizarMetricas(proceso, proceso.Estado_Actual)
	proceso.Estado_Actual = globals.SUSP_BLOCKED
	globals.MapaProcesos[proceso.Pcb.Pid] = proceso
	globals.MapaProcesosMutex.Unlock()

	globals.EstadosMutex.Lock()
	pos := general.BuscarProcesoEnBlocked(proceso.Pcb.Pid)
	globals.ESTADOS.BLOCKED = append(globals.ESTADOS.BLOCKED[:pos], globals.ESTADOS.BLOCKED[pos+1:]...)
	globals.ESTADOS.SUSP_BLOCKED = append(globals.ESTADOS.SUSP_BLOCKED, proceso.Pcb.Pid)
	globals.EstadosMutex.Unlock()

}

func avisarSwappeo(pid int64) {
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

	log.Printf("respuesta del servidor: %s", resp.Status)
}
