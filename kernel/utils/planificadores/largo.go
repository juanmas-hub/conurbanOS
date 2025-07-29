package planificadores

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"sync"
	"time"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
	general "github.com/sisoputnfrba/tp-golang/kernel/utils/general"
)

func IniciarPlanificadorLargoPlazo(archivo string, tamanio int64) {
	// Espera el Enter en otra rutina asi se puede abrir el servidor
	reader := bufio.NewReader(os.Stdin)
	slog.Info(" ---- Planificador de largo plazo en STOP, presionar ENTER: ")
	for {
		text, _ := reader.ReadString('\n')
		//slog.Debug(text)

		if text == "\n" {
			globals.PLANIFICADOR_LARGO_PLAZO_BLOCKED = false
			break
		}
	}

	go EjecutarPlanificadorCortoPlazo()
	go pasar_procesos_a_ready()

	CrearProcesoNuevo(archivo, tamanio) // Primer proceso
}

func pasar_procesos_a_ready() {
	// Esta funcion se ejecuta cuando llega un proceso a NEW, a EXIT, a SUSP_BLOCKED y a SUSP_READY
	// Voy a intentar pasar la mayor cantidad de procesos que pueda mientras memoria tenga espacio

	for {

		//slog.Info("# Esperando pasar procesos a Ready")
		globals.WaitPasarProcesoAReady()
		//slog.Info("# Intentando pasar procesos a Ready")

		globals.SuspReadyMutex.Lock()

		pasar_desde_susp_ready()
		susp_empty := len(globals.Cola_susp_ready) == 0

		globals.SuspReadyMutex.Unlock()

		if susp_empty {
			pasar_desde_new()
		}
	}
}

func CrearProcesoNuevo(archivo string, tamanio int64) {

	pid := asignar_pid()
	agregar_a_mapa_mutex(pid)
	agregar_a_mapa_procesos(pid)
	crear_estructuras_proceso(pid, archivo, tamanio)
	agregar_a_new(pid)

	globals.SignalPasarProcesoAReady()

}

func FinalizarProceso(pid int64, estadoAnterior string) {

	enviar_finalizacion_a_memoria(globals.KernelConfig.Ip_memory, globals.KernelConfig.Port_memory, pid)
	eliminar_proceso_de(pid, estadoAnterior)

	globals.ProcesosMutex[pid].Lock()
	actualizar_metricas(pid, estadoAnterior)
	globals.ProcesosMutex[pid].Unlock()

	log_exit(pid, estadoAnterior)
	eliminar_de_mapa_procesos(pid)
	globals.SignalPasarProcesoAReady()

}

// Auxiliares

func eliminar_de_mapa_procesos(pid int64) {
	globals.ProcesosMutex[pid].Lock()
	delete(globals.MapaProcesos, pid)
	globals.ProcesosMutex[pid].Unlock()
}

func agregar_a_mapa_mutex(pid int64) {
	m, ok := globals.ProcesosMutex[pid]
	if !ok {
		m = &sync.Mutex{}
		globals.ProcesosMutex[pid] = m
	}
}

func agregar_a_mapa_procesos(pid int64) {
	globals.ProcesosMutex[pid].Lock()
	p, ok := globals.MapaProcesos[pid]
	if !ok {
		p = &globals.Proceso{}
		globals.MapaProcesos[pid] = p
	}
	globals.ProcesosMutex[pid].Unlock()
}

func crear_estructuras_proceso(pid int64, archivo string, tamanio int64) {
	proceso := &globals.Proceso{
		Pcb: globals.PCB{
			Pid: pid,
			PC:  0,
			// Las metricas las inicializa en 0
		},
		Estado_Actual:        globals.NEW,
		Rafaga:               nil,
		UltimoCambioDeEstado: time.Now(),
		Tamaño:               tamanio,
		Archivo_Pseudocodigo: archivo,
	}

	if globals.KernelConfig.Scheduler_algorithm != "FIFO" {
		rafaga := globals.Rafagas{
			Raf_Ant:  0,
			Est_Ant:  0,
			Est_Sgte: float64(globals.KernelConfig.Initial_estimate),
		}

		proceso.Rafaga = &rafaga
	}

	globals.ProcesosMutex[pid].Lock()
	globals.MapaProcesos[pid] = proceso
	globals.ProcesosMutex[pid].Unlock()
}

func asignar_pid() int64 {
	globals.PIDCounterMutex.Lock()

	pid := globals.PIDCounter
	globals.PIDCounter++

	globals.PIDCounterMutex.Unlock()

	return pid
}

func log_exit(pid int64, estadoAnterior string) {

	globals.ProcesosMutex[pid].Lock()
	proceso := globals.MapaProcesos[pid]
	globals.ProcesosMutex[pid].Unlock()

	// LOG Cambio de Estado: ## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>
	slog.Info(fmt.Sprintf("## (%d) Pasa del estado %s al estado %s", pid, estadoAnterior, globals.EXIT))

	// LOG Fin de Proceso: ## (<PID>) - Finaliza el proceso
	slog.Info(fmt.Sprintf("## (%d) - Finaliza el proceso", proceso.Pcb.Pid))

	// Counts
	newCount := proceso.Pcb.ME.New
	readyCount := proceso.Pcb.ME.Ready
	execCount := proceso.Pcb.ME.Execute
	blockedCount := proceso.Pcb.ME.Blocked
	suspblockedCount := proceso.Pcb.ME.Susp_Blocked
	suspreadyCount := proceso.Pcb.ME.Susp_Ready

	// Times
	newTimes := proceso.Pcb.MT.New.Milliseconds()
	readyTimes := proceso.Pcb.MT.Ready.Milliseconds()
	execTimes := proceso.Pcb.MT.Execute.Milliseconds()
	blockedTimes := proceso.Pcb.MT.Blocked.Milliseconds()
	suspblockedTimes := proceso.Pcb.MT.Susp_Blocked.Milliseconds()
	suspreadyTimes := proceso.Pcb.MT.Susp_Ready.Milliseconds()

	// LOG Métricas de Estado: ## (<PID>) - Métricas de estado: NEW (NEW_COUNT) (NEW_TIME), READY (READY_COUNT) (READY_TIME), …
	slog.Info(fmt.Sprintf("## (%d) - Métricas de estado: NEW %d %dms, READY %d %dms, EXECUTE %d %dms, BLOCKED %d %dms, SUSP_BLOCKED %d %dms, SUSP_READY %d %dms", proceso.Pcb.Pid, newCount, newTimes, readyCount, readyTimes, execCount, execTimes, blockedCount, blockedTimes, suspblockedCount, suspblockedTimes, suspreadyCount, suspreadyTimes))

}

func ordenar_susp_ready() {
	// Con ordenar por tamaño (mas chicho primero) ya el algoritmo PMCP estaria hecho
	sort.Slice(globals.Cola_susp_ready, func(i, j int) bool {

		pidI := int64(globals.Cola_susp_ready[i])
		pidJ := int64(globals.Cola_susp_ready[j])

		globals.ProcesosMutex[pidI].Lock()
		globals.ProcesosMutex[pidJ].Lock()
		b := globals.MapaProcesos[pidI].Tamaño < globals.MapaProcesos[pidJ].Tamaño
		globals.ProcesosMutex[pidJ].Unlock()
		globals.ProcesosMutex[pidI].Unlock()
		return b
	})
	//slog.Info(fmt.Sprint("ordenando new: ", globals.Cola_susp_ready))
}

func ordenar_new() {

	sort.Slice(globals.Cola_new, func(i, j int) bool {

		pidI := int64(globals.Cola_new[i])
		pidJ := int64(globals.Cola_new[j])

		globals.ProcesosMutex[pidI].Lock()
		globals.ProcesosMutex[pidJ].Lock()
		b := globals.MapaProcesos[pidI].Tamaño < globals.MapaProcesos[pidJ].Tamaño
		globals.ProcesosMutex[pidJ].Unlock()
		globals.ProcesosMutex[pidI].Unlock()
		return b
	})

	//slog.Info(fmt.Sprint("ordenando new: ", globals.Cola_new))
}

func pasar_desde_susp_ready() {

	for len(globals.Cola_susp_ready) > 0 {

		if globals.KernelConfig.New_algorithm == "PMCP" {
			ordenar_susp_ready()
		}

		//slog.Info(fmt.Sprint("Sussp ready; ", globals.Cola_susp_ready))

		pid := globals.Cola_susp_ready[0]

		if !enviar_reanudar_proceso_a_memoria(pid) {
			break
		}
		susp_ready_a_ready(pid)
		Avisar_proceso_en_ready()

	}

}

func pasar_desde_new() {

	globals.NewMutex.Lock()

	for len(globals.Cola_new) > 0 {
		//slog.Info("entrando al ciclo de nwe")

		// Si hay procesos en Susp Ready, tienen prioridad
		/*globals.SuspReadyMutex.Lock()
		if len(globals.Cola_susp_ready) > 0 {
			globals.SignalPasarProcesoAReady()
			globals.SuspReadyMutex.Unlock()
			break
		} else {
			globals.SuspReadyMutex.Unlock()
		}*/

		if globals.KernelConfig.New_algorithm == "PMCP" {
			ordenar_new()
		}

		pid := globals.Cola_new[0]

		globals.ProcesosMutex[pid].Lock()
		proceso := globals.MapaProcesos[pid]
		globals.ProcesosMutex[pid].Unlock()

		if !enviar_inicializar_proceso_a_memoria(*proceso) {
			break
		}

		new_a_ready(pid)
		Avisar_proceso_en_ready()

	}

	globals.NewMutex.Unlock()
}

func Avisar_proceso_en_ready() {
	switch globals.KernelConfig.Scheduler_algorithm {
	case "FIFO", "SJF":
		general.Signal(globals.Sem_ProcesosEnReady)
	case "SRT":
		general.NotificarReplanifSRT()
	}
}
