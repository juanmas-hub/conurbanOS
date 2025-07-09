package utils_planifLargo

import (
	"fmt"
	"log"
	"log/slog"
	"sort"
	"time"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
	estados "github.com/sisoputnfrba/tp-golang/kernel/utils/estados"
	general "github.com/sisoputnfrba/tp-golang/kernel/utils/general"
)

func ordenarNewPorTamanio() {

	// Con ordenar por tamaño (mas chicho primero) ya el algoritmo PMCP estaria hecho (creo)
	sort.Slice(globals.ESTADOS.NEW, func(i, j int) bool {
		return globals.ESTADOS.NEW[i].Tamaño < globals.ESTADOS.NEW[j].Tamaño
	})
}

func finalizarProceso(pid int64) {

	//log.Print("Se quiere loquear MapaProcesos en finalizarProceso x1")
	globals.MapaProcesosMutex.Lock()
	proceso, ok := globals.MapaProcesos[pid]
	globals.MapaProcesosMutex.Unlock()
	//log.Print("Se unloquea MapaProcesos en finalizarProceso x1")
	if !ok {
		log.Printf("No se encontró el proceso con PID %d", pid)
		return
	}

	// Mando el PID
	ok = general.EnviarFinalizacionDeProceso_AMemoria(globals.KernelConfig.Ip_memory, globals.KernelConfig.Port_memory, pid)

	if !ok {
		log.Printf("Memoria no pudo finalizar el proceso PID %d.", proceso.Pcb.Pid)
		return
	}

	// Elimino de la cola y mando a exit
	estados.ProcesoAExit(proceso)
	estados.EliminarProcesoDeSuCola(pid, proceso.Estado_Actual)

	// Elimino del mapa procesos
	//log.Print("Se quiere loquear MapaProcesos en finalizarProceso x2")
	globals.MapaProcesosMutex.Lock()
	delete(globals.MapaProcesos, pid)
	globals.MapaProcesosMutex.Unlock()
	//log.Print("Se unloquea MapaProcesos en finalizarProceso x2")

	// Iniciar nuevos procesos
	globals.DeDondeSeLlamaMutex.Lock()
	globals.DeDondeSeLlamaPasarProcesosAReady = "Exit"
	globals.DeDondeSeLlamaMutex.Unlock()
	//general.Signal(globals.Sem_PasarProcesoAReady)
	globals.SignalPasarProcesoAReady()
}

func CrearProcesoNuevo(archivo string, tamanio int64) {

	globals.PIDCounterMutex.Lock()

	pid := globals.PIDCounter
	globals.PIDCounter++

	globals.PIDCounterMutex.Unlock()

	proceso := globals.Proceso{
		Pcb: globals.PCB{
			Pid: pid,
			PC:  0,
			// Las metricas las inicializa en 0
		},
		Estado_Actual:        globals.NEW,
		Rafaga:               nil,
		UltimoCambioDeEstado: time.Now(),
	}

	if globals.KernelConfig.Scheduler_algorithm != "FIFO" {
		rafaga := globals.Rafagas{
			Raf_Ant:  0,
			Est_Ant:  globals.KernelConfig.Initial_estimate,
			Est_Sgte: 0,
		}

		proceso.Rafaga = &rafaga
	}

	globals.MapaProcesosMutex.Lock()
	globals.MapaProcesos[pid] = proceso
	globals.MapaProcesosMutex.Unlock()

	procesoNuevo := globals.Proceso_Nuevo{
		Archivo_Pseudocodigo: archivo,
		Tamaño:               tamanio,
		Proceso:              proceso,
	}

	//log.Print("Se quiere bloquear en CrearProcesoNuevo")
	globals.EstadosMutex.Lock()
	log.Print("Se bloqueo en CrearProcesoNuevo")

	//log.Printf("Agregando proceso a NEW. Cantidad actual: %d", len(globals.ESTADOS.NEW))

	// Aca no hay metricas que actualizar
	globals.ESTADOS.NEW = append(globals.ESTADOS.NEW, procesoNuevo)

	// LOG Creación de Proceso: “## (<PID>) Se crea el proceso - Estado: NEW”
	slog.Info(fmt.Sprintf("## (%d) Se crea el proceso - Estado: NEW", pid))

	//log.Printf("Después de agregar, NEW tiene %d procesos", len(globals.ESTADOS.NEW))
	if globals.KernelConfig.New_algorithm == "PMCP" {
		ordenarNewPorTamanio()
		log.Print("NEW despues de ordenarlo: ", globals.ESTADOS.NEW)
	}

	//log.Print("Se quiere desbloquear en CrearProcesoNuevo")
	globals.EstadosMutex.Unlock()
	log.Print("Se desbloqueo en CrearProcesoNuevo")

	globals.DeDondeSeLlamaMutex.Lock()
	globals.DeDondeSeLlamaPasarProcesosAReady = "New"
	globals.DeDondeSeLlamaMutex.Unlock()
	//general.Signal(globals.Sem_PasarProcesoAReady)
	globals.SignalPasarProcesoAReady()

}
