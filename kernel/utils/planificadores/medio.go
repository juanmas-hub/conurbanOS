package planificadores

import (
	"fmt"
	"log/slog"
	"time"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
)

// Se llama cuando un proceso de execute se bloquea (IO o DUMP)
func BloquearProceso(pid int64) bool {

	ok := execute_a_blocked(pid)
	if !ok {
		return false
	}

	globals.CantidadSesionesIOMutex.Lock()
	cantidadSesiones := globals.CantidadSesionesIO[pid]
	globals.CantidadSesionesIOMutex.Unlock()

	go func() {
		time.Sleep(time.Duration(globals.KernelConfig.Suspension_time) * time.Millisecond)
		sigueBloqueado(pid, cantidadSesiones)
	}()

	return true
}

/*
func EjecutarPlanificadorMedioPlazo(proceso globals.Proceso, razon string) bool {

	if !CambiarEstado(proceso.Pcb.Pid, globals.EXECUTE, globals.BLOCKED) {
		globals.MapaProcesosMutex.Unlock()
		return false
	}

	globals.CantidadSesionesIOMutex.Lock()
	cantidadSesiones := globals.CantidadSesionesIO[proceso.Pcb.Pid]
	globals.CantidadSesionesIOMutex.Unlock()

	// -- Timer hasta ser suspendido

	go func() {
		time.Sleep(time.Duration(globals.KernelConfig.Suspension_time) * time.Millisecond)
		sigueBloqueado(proceso, cantidadSesiones)
	}()

	//slog.Debug(fmt.Sprintf("Proceso %d bloqueado, arranco el timer", proceso.Pcb.Pid))

	return true

}*/

/*
func sigueBloqueado(proceso globals.Proceso, cantidadSesionesPrevia int) {
	// Si sigue bloqueado (en IO) hay que suspenderlo
	// Para que no siga bloqueado, el proceso tuvo que terminar su IO (lo recibimos como mensaje desde IO, siendo kernel servidor)
	// Cuando kernel reciba de IO el mensaje, ahí le cambiamos el estado

	//slog.Debug(fmt.Sprintf("Termino el timer del proceso %d", proceso.Pcb.Pid))

	globals.MapaProcesosMutex.Lock()

	procesoActualmente, presente := globals.MapaProcesos[proceso.Pcb.Pid]
	if !presente {
		//slog.Debug(fmt.Sprintf("PID %d no se encontro en MapaProcesos en sigueBloqueado. Probablemente finalizo", proceso.Pcb.Pid))
		globals.MapaProcesosMutex.Unlock()

		return
	}

	// Comparo cantidad de sesiones:
	// 		- Son iguales: es la misma sesion => me fijo si swappeo
	//		- Son distintas: distintas sesiones => no hago nada

	globals.CantidadSesionesIOMutex.Lock()
	cantidadSesionesActual := globals.CantidadSesionesIO[procesoActualmente.Pcb.Pid]
	globals.CantidadSesionesIOMutex.Unlock()

	//log.Printf("Cantidad sesiones actual: %d, previa: %d", cantidadSesionesActual, cantidadSesionesPrevia)

	globals.MapaProcesosMutex.Unlock()
	general.LogUnlockeo("MapaProcesos", "sigueBloqueado")

	if cantidadSesionesActual == cantidadSesionesPrevia && procesoActualmente.Estado_Actual == globals.BLOCKED {

		// Cambio de estado
		globals.MapaProcesosMutex.Lock()
		ok := CambiarEstado(procesoActualmente.Pcb.Pid, globals.BLOCKED, globals.SUSP_BLOCKED)
		globals.MapaProcesosMutex.Unlock()

		if ok {
			// Aviso a memoria que hay que swappear
			//slog.Debug(fmt.Sprint("Hay que swappear proceso: ", procesoActualmente.Pcb.Pid))
			general.AvisarSwappeo(procesoActualmente.Pcb.Pid)
			//slog.Debug(fmt.Sprint("Ya termino el aviso de swappeo a memoria del proceso: ", procesoActualmente.Pcb.Pid))

			// Libere espacio => llamo a nuevos procesos
			globals.SignalPasarProcesoAReady()
		}
	}

}*/

func sigueBloqueado(pid int64, cantidadSesionesPrevia int) {

	globals.ProcesosMutex[pid].Lock()
	procesoActualmente, presente := globals.MapaProcesos[pid]
	if !presente {
		globals.ProcesosMutex[pid].Unlock()
		return
	}

	// Comparo cantidad de sesiones:
	// 		- Son iguales: es la misma sesion => me fijo si swappeo
	//		- Son distintas: distintas sesiones => no hago nada

	globals.CantidadSesionesIOMutex.Lock()
	cantidadSesionesActual := globals.CantidadSesionesIO[procesoActualmente.Pcb.Pid]
	globals.CantidadSesionesIOMutex.Unlock()

	if cantidadSesionesActual == cantidadSesionesPrevia && procesoActualmente.Estado_Actual == globals.BLOCKED {

		enviar_suspension_proceso_a_memoria(procesoActualmente.Pcb.Pid)
		blocked_a_susp_blocked(pid)
		globals.SignalPasarProcesoAReady()
	}

	globals.ProcesosMutex[pid].Unlock()

}

// Auxiliares

func actualizar_rafagas(proceso *globals.Proceso, rafagaReal float64) {

	slog.Debug(fmt.Sprintf("Actualizando estimado: %d", proceso.Pcb.Pid))
	slog.Debug(fmt.Sprintf("Rafaga real: %f", rafagaReal))

	alpha := globals.KernelConfig.Alpha
	est_ant := proceso.Rafaga.Est_Sgte

	proceso.Rafaga.Est_Ant = est_ant
	proceso.Rafaga.Raf_Ant = rafagaReal
	proceso.Rafaga.Est_Sgte = rafagaReal*alpha + est_ant*(1-alpha)
	// Est(n+1) =  R(n) + (1-) Est(n) ;    [0,1]

	slog.Debug(fmt.Sprintf("Rafaga actualizada de PID %d: %f", proceso.Pcb.Pid, proceso.Rafaga))
}
