package utils_planifMedio

import (
	"log"
	"time"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
	estados "github.com/sisoputnfrba/tp-golang/kernel/utils/estados"
	general "github.com/sisoputnfrba/tp-golang/kernel/utils/general"
)

// Se llama cuando un proceso de execute se bloquea (IO o DUMP)
func EjecutarPlanificadorMedioPlazo(proceso globals.Proceso, razon string) {

	estados.ExecuteABlocked(proceso, razon)

	globals.CantidadSesionesIOMutex.Lock()
	cantidadSesiones := globals.CantidadSesionesIO[proceso.Pcb.Pid]
	globals.CantidadSesionesIOMutex.Unlock()

	// -- Timer hasta ser suspendido
	go timer(globals.KernelConfig.Suspension_time, proceso, cantidadSesiones)

}

func timer(tiempo int64, proceso globals.Proceso, cantidadSesiones int) {
	defer sigueBloqueado(proceso, cantidadSesiones)
	duracion := time.Duration(tiempo) * time.Millisecond
	time.Sleep(duracion)
}

func sigueBloqueado(proceso globals.Proceso, cantidadSesionesPrevia int) {
	// Si sigue bloqueado (en IO) hay que suspenderlo
	// Para que no siga bloqueado, el proceso tuvo que terminar su IO (lo recibimos como mensaje desde IO, siendo kernel servidor)
	// Cuando kernel reciba de IO el mensaje, ahí le cambiamos el estado

	globals.MapaProcesosMutex.Lock()
	procesoActualmente := globals.MapaProcesos[proceso.Pcb.Pid]
	globals.MapaProcesosMutex.Unlock()

	// Comparo cantidad de sesiones:
	// 		- Son iguales: es la misma sesion => me fijo si swappeo
	//		- Son distintas: distintas sesiones => no hago nada

	globals.CantidadSesionesIOMutex.Lock()
	log.Printf("Cantidad sesiones actual: %d, previa: %d", globals.CantidadSesionesIO[procesoActualmente.Pcb.Pid], cantidadSesionesPrevia)
	if globals.CantidadSesionesIO[procesoActualmente.Pcb.Pid] == cantidadSesionesPrevia {
		if procesoActualmente.Estado_Actual == globals.BLOCKED {
			// Aviso a memoria que hay que swappear
			general.AvisarSwappeo(procesoActualmente.Pcb.Pid)

			// Cambio de estado
			estados.BlockedASuspBlocked(proceso)

			// Libere espacio => llamo a nuevos procesos
			globals.DeDondeSeLlamaMutex.Lock()
			globals.DeDondeSeLlamaPasarProcesosAReady = "Susp Blocked"
			globals.DeDondeSeLlamaMutex.Unlock()
			general.Signal(globals.Sem_PasarProcesoAReady)
		}
	}
	globals.CantidadSesionesIOMutex.Unlock()
}
