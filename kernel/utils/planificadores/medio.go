package planificadores

import (
	"fmt"
	"log/slog"
	"time"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
)

// Se llama cuando un proceso de execute se bloquea (IO o DUMP)
func BloquearProceso(pid int64) bool {

	//slog.Info("Se llamo a bloquear proceso")

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

	//slog.Debug(fmt.Sprintf("Actualizando estimado: %d", proceso.Pcb.Pid))
	//slog.Debug(fmt.Sprintf("Rafaga real: %f", rafagaReal))

	alpha := globals.KernelConfig.Alpha
	est_ant := proceso.Rafaga.Est_Sgte

	proceso.Rafaga.Est_Ant = est_ant
	proceso.Rafaga.Raf_Ant = rafagaReal
	proceso.Rafaga.Est_Sgte = rafagaReal*alpha + est_ant*(1-alpha)
	// Est(n+1) =  R(n) + (1-) Est(n) ;    [0,1]

	//slog.Debug(fmt.Sprintf("Rafaga actualizada de PID %d: %f", proceso.Pcb.Pid, proceso.Rafaga))

	slog.Info(fmt.Sprintf("SJF/SRT == > Rafaga actualizada de: %d. Rafaga Anterior: %f. Rafaga Estimada: %f", proceso.Pcb.Pid, rafagaReal, proceso.Rafaga.Est_Sgte))

}
