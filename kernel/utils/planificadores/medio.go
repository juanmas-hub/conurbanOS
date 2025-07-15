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
func EjecutarPlanificadorMedioPlazo(proceso globals.Proceso, razon string) bool {

	general.LogIntentoLockeo("MapaProcesos", "EjecutarPlanificadorMedioPlazo")
	globals.MapaProcesosMutex.Lock()
	general.LogLockeo("MapaProcesos", "EjecutarPlanificadorMedioPlazo")
	general.LogIntentoLockeo("Estados", "EjecutarPlanificadorMedioPlazo")
	globals.EstadosMutex.Lock()
	general.LogLockeo("Estados", "EjecutarPlanificadorMedioPlazo")

	if !CambiarEstado(proceso.Pcb.Pid, globals.EXECUTE, globals.BLOCKED) {
		globals.EstadosMutex.Unlock()
		globals.MapaProcesosMutex.Unlock()
		return false
	}

	globals.CantidadSesionesIOMutex.Lock()
	cantidadSesiones := globals.CantidadSesionesIO[proceso.Pcb.Pid]
	globals.CantidadSesionesIOMutex.Unlock()

	globals.EstadosMutex.Unlock()
	general.LogUnlockeo("Estados", "EjecutarPlanificadorMedioPlazo")
	globals.MapaProcesosMutex.Unlock()
	general.LogUnlockeo("MapaProcesos", "EjecutarPlanificadorMedioPlazo")

	// -- Timer hasta ser suspendido

	go func() {
		time.Sleep(time.Duration(globals.KernelConfig.Suspension_time) * time.Millisecond)
		sigueBloqueado(proceso, cantidadSesiones)
	}()

	slog.Debug(fmt.Sprintf("Proceso %d bloqueado, arranco el timer", proceso.Pcb.Pid))

	return true

}

func sigueBloqueado(proceso globals.Proceso, cantidadSesionesPrevia int) {
	// Si sigue bloqueado (en IO) hay que suspenderlo
	// Para que no siga bloqueado, el proceso tuvo que terminar su IO (lo recibimos como mensaje desde IO, siendo kernel servidor)
	// Cuando kernel reciba de IO el mensaje, ahí le cambiamos el estado

	slog.Debug(fmt.Sprintf("Termino el timer del proceso %d", proceso.Pcb.Pid))

	general.LogIntentoLockeo("MapaProcesos", "sigueBloqueado")
	globals.MapaProcesosMutex.Lock()
	general.LogLockeo("MapaProcesos", "sigueBloqueado")

	procesoActualmente, presente := globals.MapaProcesos[proceso.Pcb.Pid]
	if !presente {
		slog.Debug(fmt.Sprintf("PID %d no se encontro en MapaProcesos en sigueBloqueado. Probablemente finalizo", proceso.Pcb.Pid))
		globals.MapaProcesosMutex.Unlock()
		general.LogUnlockeo("MapaProcesos", "sigueBloqueado")

		return
	}

	// Comparo cantidad de sesiones:
	// 		- Son iguales: es la misma sesion => me fijo si swappeo
	//		- Son distintas: distintas sesiones => no hago nada

	globals.CantidadSesionesIOMutex.Lock()
	cantidadSesionesActual := globals.CantidadSesionesIO[procesoActualmente.Pcb.Pid]
	globals.CantidadSesionesIOMutex.Unlock()

	log.Printf("Cantidad sesiones actual: %d, previa: %d", cantidadSesionesActual, cantidadSesionesPrevia)

	if cantidadSesionesActual == cantidadSesionesPrevia && procesoActualmente.Estado_Actual == globals.BLOCKED {

		// Cambio de estado
		general.LogIntentoLockeo("Estados", "sigueBloqueado")
		globals.EstadosMutex.Lock()
		general.LogLockeo("Estados", "sigueBloqueado")
		ok := CambiarEstado(procesoActualmente.Pcb.Pid, globals.BLOCKED, globals.SUSP_BLOCKED)
		globals.EstadosMutex.Unlock()
		general.LogUnlockeo("Estados", "sigueBloqueado")

		if ok {
			// Aviso a memoria que hay que swappear
			slog.Debug(fmt.Sprint("Hay que swappear proceso: ", procesoActualmente.Pcb.Pid))
			general.AvisarSwappeo(procesoActualmente.Pcb.Pid)
			slog.Debug(fmt.Sprint("Ya termino el aviso de swappeo a memoria del proceso: ", procesoActualmente.Pcb.Pid))

			// Libere espacio => llamo a nuevos procesos
			globals.DeDondeSeLlamaMutex.Lock()
			globals.DeDondeSeLlamaPasarProcesosAReady = "Susp Blocked"
			globals.DeDondeSeLlamaMutex.Unlock()
			globals.SignalPasarProcesoAReady()
		}
	}

	globals.MapaProcesosMutex.Unlock()
	general.LogUnlockeo("MapaProcesos", "sigueBloqueado")
}
