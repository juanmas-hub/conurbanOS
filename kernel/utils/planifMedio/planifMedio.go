package utils_planifMedio

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"time"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
	general "github.com/sisoputnfrba/tp-golang/kernel/utils/general"
)

// ----- FUNCIONES EXPORTADAS -------

func BloquearProcesoDesdeExecute(proceso globals.Proceso, razon string) {
	// Esta funcion deberia llamarse cuando un proceso en ejecucion llama a IO con la syscall IO (desde syscallController)

	// -- Paso el proceso entre las colas
	// Como la cola de Execute 'no tiene' orden (todos los que estan en execute tienen una cpu ya ejecutando)
	// no se saca el primero de la cola como en las otras funciones

	//log.Print("Se quiere loquear MapaProcesos en BloquearProcesoDesdeExecute")
	globals.MapaProcesosMutex.Lock()
	proceso = general.ActualizarMetricas(proceso, proceso.Estado_Actual)
	proceso.Estado_Actual = globals.BLOCKED

	globals.MapaProcesos[proceso.Pcb.Pid] = proceso
	globals.MapaProcesosMutex.Unlock()
	//log.Print("Se unloquea MapaProcesos en BloquearProcesoDesdeExecute")

	globals.EstadosMutex.Lock()
	pos := general.BuscarProcesoEnExecute(proceso.Pcb.Pid)
	globals.ESTADOS.EXECUTE = append(globals.ESTADOS.EXECUTE[:pos], globals.ESTADOS.EXECUTE[pos+1:]...)
	globals.ESTADOS.BLOCKED = append(globals.ESTADOS.BLOCKED, proceso.Pcb.Pid)
	globals.EstadosMutex.Unlock()

	log.Printf("Bloqueado proceso desde execute PID %d, razon: %s", proceso.Pcb.Pid, razon)

	globals.CantidadSesionesIOMutex.Lock()
	cantidadSesiones := globals.CantidadSesionesIO[proceso.Pcb.Pid]
	globals.CantidadSesionesIOMutex.Unlock()

	// LOG Cambio de Estado: ## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>
	slog.Info(fmt.Sprintf("## (%d) Pasa del estado EXECUTE al estado BLOCKED", proceso.Pcb.Pid))

	// -- Timer hasta ser suspendido
	go timer(globals.KernelConfig.Suspension_time, proceso, cantidadSesiones)

}

// ----- FUNCIONES LOCALES -------

func timer(tiempo int64, proceso globals.Proceso, cantidadSesiones int) {
	defer sigueBloqueado(proceso, cantidadSesiones)
	duracion := time.Duration(tiempo) * time.Millisecond
	time.Sleep(duracion)
}

func sigueBloqueado(proceso globals.Proceso, cantidadSesionesPrevia int) {
	// Si sigue bloqueado (en IO) hay que suspenderlo
	// Para que no siga bloqueado, el proceso tuvo que terminar su IO (lo recibimos como mensaje desde IO, siendo kernel servidor)
	// Cuando kernel reciba de IO el mensaje, ahí le cambiamos el estado

	//log.Print("Se quiere loquear MapaProcesos en sigueBloqueado")
	globals.MapaProcesosMutex.Lock()
	procesoActualmente := globals.MapaProcesos[proceso.Pcb.Pid]
	globals.MapaProcesosMutex.Unlock()
	//log.Print("Se unloquea MapaProcesos en sigueBloqueado")

	// Comparo cantidad de sesiones:
	// 		- Son iguales: es la misma sesion => me fijo si swappeo
	//		- Son distintas: distintas sesiones => no hago nada

	globals.CantidadSesionesIOMutex.Lock()
	log.Printf("Cantidad sesiones actual: %d, previa: %d", globals.CantidadSesionesIO[procesoActualmente.Pcb.Pid], cantidadSesionesPrevia)
	if globals.CantidadSesionesIO[procesoActualmente.Pcb.Pid] == cantidadSesionesPrevia {
		if procesoActualmente.Estado_Actual == globals.BLOCKED {
			// Aviso a memoria que hay que swappear
			avisarSwappeo(procesoActualmente.Pcb.Pid)

			// Cambio de estado
			blockedASuspBlocked(proceso)

			// Libere espacio => llamo a nuevos procesos
			globals.DeDondeSeLlamaMutex.Lock()
			globals.DeDondeSeLlamaPasarProcesosAReady = "Susp Blocked"
			globals.DeDondeSeLlamaMutex.Unlock()
			general.Signal(globals.Sem_PasarProcesoAReady)
		}
	}
	globals.CantidadSesionesIOMutex.Unlock()
}

func blockedASuspBlocked(proceso globals.Proceso) {
	// Muevo el proceso en la colas
	//log.Print("Se quiere loquear MapaProcesos en blockedASuspBlocked")
	globals.MapaProcesosMutex.Lock()
	proceso = general.ActualizarMetricas(proceso, proceso.Estado_Actual)
	proceso.Estado_Actual = globals.SUSP_BLOCKED
	globals.MapaProcesos[proceso.Pcb.Pid] = proceso
	globals.MapaProcesosMutex.Unlock()
	//log.Print("Se unloquea MapaProcesos en blockedASuspBlocked")

	globals.EstadosMutex.Lock()
	pos := general.BuscarProcesoEnBlocked(proceso.Pcb.Pid)
	globals.ESTADOS.BLOCKED = append(globals.ESTADOS.BLOCKED[:pos], globals.ESTADOS.BLOCKED[pos+1:]...)
	globals.ESTADOS.SUSP_BLOCKED = append(globals.ESTADOS.SUSP_BLOCKED, proceso.Pcb.Pid)
	globals.EstadosMutex.Unlock()

	// LOG Cambio de Estado: ## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>
	slog.Info(fmt.Sprintf("## (%d) Pasa del estado BLOCKED al estado SUSP_BLOCKED", proceso.Pcb.Pid))

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
