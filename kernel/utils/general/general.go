package utils_general

// Configuracion y servidores

import (
	"encoding/json"
	"log"
	"os"
	"time"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
)

func Wait(semaforo globals.Semaforo) {
	<-semaforo
}

func Signal(semaforo globals.Semaforo) {
	semaforo <- struct{}{}
}

func IniciarConfiguracion(filePath string) *globals.Kernel_Config {
	var config *globals.Kernel_Config
	configFile, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer configFile.Close()

	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)

	return config
}

// Mandando PID, se finaliza ese proceso.
func FinalizarProceso(pid int64) {
	globals.ProcesosAFinalizarMutex.Lock()
	globals.ProcesosAFinalizar = append(globals.ProcesosAFinalizar, pid)
	globals.ProcesosAFinalizarMutex.Unlock()

	Signal(globals.Sem_ProcesoAFinalizar)

}

// Mandando nombre del CPU, se libera. Aumenta el semaforo de Semaforos de CPU, entonces el planificador corto replanifica.
func LiberarCPU(nombreCPU string) {
	globals.ListaCPUsMutex.Lock()
	posCpu := BuscarCpu(nombreCPU)
	globals.ListaCPUs[posCpu].EstaLibre = true
	globals.ListaCPUsMutex.Unlock()
	switch globals.KernelConfig.Scheduler_algorithm {
	case "FIFO", "SJF":
		Signal(globals.Sem_Cpus)
	case "SRT":
		NotificarReplanifSRT()
	}
}

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
	case globals.SUSP_BLOCKED:
		ME.Susp_Blocked++
		MT.Susp_Blocked += tiempoEnEstado
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
