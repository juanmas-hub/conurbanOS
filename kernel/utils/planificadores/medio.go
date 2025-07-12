package planificadores

import (
	"fmt"
	"log"
	"log/slog"
	"time"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
	general "github.com/sisoputnfrba/tp-golang/kernel/utils/general"
)

// Se llama cuando un proceso de execute se bloquea (IO o DUMP)
func EjecutarPlanificadorMedioPlazo(proceso globals.Proceso, razon string) {

	ExecuteABlocked(proceso, razon)

	globals.CantidadSesionesIOMutex.Lock()
	cantidadSesiones := globals.CantidadSesionesIO[proceso.Pcb.Pid]
	globals.CantidadSesionesIOMutex.Unlock()

	// -- Timer hasta ser suspendido

	go func() {
		time.Sleep(time.Duration(time.Duration(globals.KernelConfig.Suspension_time).Milliseconds()))
		sigueBloqueado(proceso, cantidadSesiones)
	}()

	slog.Debug(fmt.Sprintf("Proceso %d suspendido, arranco el timer", proceso.Pcb.Pid))

}

func sigueBloqueado(proceso globals.Proceso, cantidadSesionesPrevia int) {
	// Si sigue bloqueado (en IO) hay que suspenderlo
	// Para que no siga bloqueado, el proceso tuvo que terminar su IO (lo recibimos como mensaje desde IO, siendo kernel servidor)
	// Cuando kernel reciba de IO el mensaje, ahí le cambiamos el estado

	slog.Debug(fmt.Sprintf("Termino el timer del proceso %d", proceso.Pcb.Pid))

	globals.MapaProcesosMutex.Lock()
	procesoActualmente := globals.MapaProcesos[proceso.Pcb.Pid]
	globals.MapaProcesosMutex.Unlock()

	// Comparo cantidad de sesiones:
	// 		- Son iguales: es la misma sesion => me fijo si swappeo
	//		- Son distintas: distintas sesiones => no hago nada

	cantidadSesionesActual := globals.CantidadSesionesIO[procesoActualmente.Pcb.Pid]

	log.Printf("Cantidad sesiones actual: %d, previa: %d", cantidadSesionesActual, cantidadSesionesPrevia)

	if cantidadSesionesActual == cantidadSesionesPrevia && procesoActualmente.Estado_Actual == globals.BLOCKED {
		// Aviso a memoria que hay que swappear
		slog.Debug(fmt.Sprint("Hay que swappear proceso: ", procesoActualmente.Pcb.Pid))
		general.AvisarSwappeo(procesoActualmente.Pcb.Pid)

		// Cambio de estado
		globals.MapaProcesosMutex.Lock()
		BlockedASuspBlocked(proceso)
		globals.MapaProcesosMutex.Unlock()

		// Libere espacio => llamo a nuevos procesos
		globals.DeDondeSeLlamaMutex.Lock()
		globals.DeDondeSeLlamaPasarProcesosAReady = "Susp Blocked"
		globals.DeDondeSeLlamaMutex.Unlock()
		globals.SignalPasarProcesoAReady()
	}
}
