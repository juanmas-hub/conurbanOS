package procesos

import (
	"log"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
	estados "github.com/sisoputnfrba/tp-golang/kernel/utils/estados"
	general "github.com/sisoputnfrba/tp-golang/kernel/utils/general"
)

func FinalizarProceso(pid int64) {
	globals.MapaProcesosMutex.Lock()
	proceso, ok := globals.MapaProcesos[pid]
	globals.MapaProcesosMutex.Unlock()
	if !ok {
		log.Printf("No se encontró el proceso con PID %d", pid)
		return
	}

	// Enviar a memoria
	ok = general.EnviarFinalizacionDeProceso_AMemoria(globals.KernelConfig.Ip_memory, globals.KernelConfig.Port_memory, pid)
	if !ok {
		log.Printf("Memoria no pudo finalizar el proceso PID %d.", pid)
		return
	}

	// Mover a EXIT y eliminar de su cola
	estados.ProcesoAExit(proceso)

	globals.EstadosMutex.Lock()
	estados.EliminarProcesoDeSuCola(pid, proceso.Estado_Actual)
	globals.EstadosMutex.Unlock()

	// Eliminar del mapa de procesos
	globals.MapaProcesosMutex.Lock()
	delete(globals.MapaProcesos, pid)
	globals.MapaProcesosMutex.Unlock()

	// Señal para ready
	globals.DeDondeSeLlamaMutex.Lock()
	globals.DeDondeSeLlamaPasarProcesosAReady = "Exit"
	globals.DeDondeSeLlamaMutex.Unlock()
	globals.SignalPasarProcesoAReady()
}
