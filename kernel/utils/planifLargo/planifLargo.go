package utils_planifLargo

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"time"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
	general "github.com/sisoputnfrba/tp-golang/kernel/utils/general"
	cp "github.com/sisoputnfrba/tp-golang/kernel/utils/planifCorto"
)

// ----- FUNCIONES EXPORTADAS -------

func IniciarPlanificadorLargoPlazo(archivo string, tamanio int64) {
	// Espera el Enter en otra rutina asi se puede abrir el servidor
	reader := bufio.NewReader(os.Stdin)
	log.Println("Planificador de largo plazo en STOP, presionar ENTER: ")
	for {
		text, _ := reader.ReadString('\n')
		log.Print(text)

		if text == "\n" {
			globals.PLANIFICADOR_LARGO_PLAZO_BLOCKED = false
			break
		}
	}
	// El planificador de corto plazo se ejecuta aca porque no tiene sentido ejecutarlo si no pueden entrar procesos
	go cp.EjecutarPlanificadorCortoPlazo()

	go escucharFinalizacionesDeProcesos()

	go PasarProcesosAReady()

	CrearProcesoNuevo(archivo, tamanio) // Primer proceso
}

func CrearProcesoNuevo(archivo string, tamanio int64) {

	globals.PIDCounterMutex.Lock()

	pid := globals.PIDCounter
	globals.PIDCounter++

	globals.PIDCounterMutex.Unlock()

	log.Printf("Creando nuevo proceso con PID %d y tamaño %d\n", pid, tamanio)
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

	procesoNuevo := globals.Proceso_Nuevo{
		Archivo_Pseudocodigo: archivo,
		Tamaño:               tamanio,
		Proceso:              proceso,
	}

	globals.EstadosMutex.Lock()
	log.Printf("Agregando proceso a NEW. Cantidad actual: %d", len(globals.ESTADOS.NEW))

	// Aca no hay metricas que actualizar
	globals.ESTADOS.NEW = append(globals.ESTADOS.NEW, procesoNuevo)
	log.Printf("Después de agregar, NEW tiene %d procesos", len(globals.ESTADOS.NEW))
	if globals.KernelConfig.New_algorithm == "PMCP" {
		ordenarNewPorTamanio()
	}
	globals.EstadosMutex.Unlock()

	// Si se crea un proceso nuevo antes de que aprete Enter, se agrega a NEW pero no se pasan procesos a READY
	if globals.PLANIFICADOR_LARGO_PLAZO_BLOCKED == false {
		globals.DeDondeSeLlamaMutex.Lock()
		globals.DeDondeSeLlamaPasarProcesosAReady = "New"
		globals.DeDondeSeLlamaMutex.Unlock()
		general.Signal(globals.Sem_PasarProcesoAReady)
	}

	// LOG Creación de Proceso: “## (<PID>) Se crea el proceso - Estado: NEW”
	slog.Info(fmt.Sprintf("## (%d) Se crea el proceso - Estado: NEW", pid))
}

func PasarProcesosAReady() {
	// Ejecuta en un hilo
	// Esta funcion se ejecuta cuando llega un proceso a NEW, a EXIT, a SUSP_BLOCKED y a SUSP_READY
	// Voy a intentar pasar la mayor cantidad de procesos que pueda mientras memoria tenga espacio

	for {
		general.Wait(globals.Sem_PasarProcesoAReady)

		log.Print("Intentando pasar procesos a ready porque llego un proceso a: ", globals.DeDondeSeLlamaPasarProcesosAReady)

		globals.EstadosMutex.Lock()

		var lenghtSUSP_READY = len(globals.ESTADOS.SUSP_READY)
		for lenghtSUSP_READY > 0 {
			pid := globals.ESTADOS.SUSP_READY[0]
			if solicitarInicializarProcesoAMemoria_DesdeSUSP_READY(pid) == false {
				break
			}

			//log.Print("Se quiere loquear MapaProcesos en PasarProcesosAReady x1")
			globals.MapaProcesosMutex.Lock()
			//log.Print("Se loquea MapaProcesos en PasarProcesosAReady x1")
			proceso := globals.MapaProcesos[pid]
			globals.MapaProcesosMutex.Unlock()
			//log.Print("Se unloquea MapaProcesos en PasarProcesosAReady x1")
			suspReadyAReady(proceso)
			lenghtSUSP_READY--
		}

		if lenghtSUSP_READY == 0 {

			for len(globals.ESTADOS.NEW) > 0 {
				procesoNuevo := globals.ESTADOS.NEW[0]

				if solicitarInicializarProcesoAMemoria_DesdeNEW(procesoNuevo) == false {
					break
				}

				newAReady(procesoNuevo)

			}
		}

		globals.EstadosMutex.Unlock()
	}
}

// Hay una funcion FinalizarProceso en utils general que no pude poner aca, pero tiene que
// ver con el planificador de largo plazo

// ------- FUNCIONES LOCALES ---------

func ordenarNewPorTamanio() {

	// Con ordenar por tamaño (mas chicho primero) ya el algoritmo PMCP estaria hecho (creo)
	sort.Slice(globals.ESTADOS.NEW, func(i, j int) bool {
		return globals.ESTADOS.NEW[i].Tamaño < globals.ESTADOS.NEW[j].Tamaño
	})
}

func solicitarInicializarProcesoAMemoria_DesdeNEW(proceso globals.Proceso_Nuevo) bool {
	// Se pudo iniciarlizar => devuelve true
	// No se pudo inicializar => devuelve false

	mensaje := globals.SolicitudIniciarProceso{
		Archivo_Pseudocodigo: proceso.Archivo_Pseudocodigo,
		Tamanio:              proceso.Tamaño,
		Pid:                  proceso.Proceso.Pcb.Pid,
		Susp:                 false,
	}
	body, err := json.Marshal(mensaje)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	// Posible problema con el int64 del puerto
	url := fmt.Sprintf("http://%s:%d/iniciarProceso", globals.KernelConfig.Ip_memory, globals.KernelConfig.Port_memory)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando mensaje a ip:%s puerto:%d", globals.KernelConfig.Ip_memory, globals.KernelConfig.Port_memory)
	}

	log.Printf("respuesta del servidor: %s", resp.Status)

	if resp.Status == "200 OK" {
		return true
	}

	return false
}

func solicitarInicializarProcesoAMemoria_DesdeSUSP_READY(pid int64) bool {
	// Se pudo iniciarlizar => devuelve true
	// No se pudo inicializar => devuelve false
	mensaje := globals.PidJSON{PID: pid}

	body, err := json.Marshal(mensaje)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	url := fmt.Sprintf("http://%s:%d/reanudarProceso", globals.KernelConfig.Ip_memory, globals.KernelConfig.Port_memory)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando mensaje a ip:%s puerto:%d", globals.KernelConfig.Ip_memory, globals.KernelConfig.Port_memory)
	}

	log.Printf("respuesta del servidor: %s", resp.Status)

	if resp.Status == "200 OK" {
		return true
	}

	return false
}

func escucharFinalizacionesDeProcesos() {
	// Queda escuchando en un hilo los procesos que terminan
	for {
		general.Wait(globals.Sem_ProcesoAFinalizar)
		globals.ProcesosAFinalizarMutex.Lock()
		pid := globals.ProcesosAFinalizar[0]
		globals.ProcesosAFinalizar = globals.ProcesosAFinalizar[1:]
		globals.ProcesosAFinalizarMutex.Unlock()
		go finalizarProceso(pid)
	}
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
	eliminarDeSuCola(pid, proceso.Estado_Actual)
	procesoAExit(proceso)

	// Elimino del mapa procesos
	//log.Print("Se quiere loquear MapaProcesos en finalizarProceso x2")
	globals.MapaProcesosMutex.Lock()
	delete(globals.MapaProcesos, pid)
	globals.MapaProcesosMutex.Unlock()
	//log.Print("Se unloquea MapaProcesos en finalizarProceso x2")
	log.Printf("El PCB del proceso con PID %d fue liberado", pid)

	// Iniciar nuevos procesos
	globals.DeDondeSeLlamaMutex.Lock()
	globals.DeDondeSeLlamaPasarProcesosAReady = "Exit"
	globals.DeDondeSeLlamaMutex.Unlock()
	general.Signal(globals.Sem_PasarProcesoAReady)
}

func eliminarDeSuCola(pid int64, estadoActual string) {
	// Busco la cola correspondiente y elimino el proceso
	globals.EstadosMutex.Lock()
	switch estadoActual {
	case globals.BLOCKED:
		pos := general.BuscarProcesoEnBlocked(pid)
		globals.ESTADOS.BLOCKED = append(globals.ESTADOS.BLOCKED[:pos], globals.ESTADOS.BLOCKED[pos+1:]...)
	case globals.EXECUTE:
		pos := general.BuscarProcesoEnExecute(pid)
		globals.ESTADOS.EXECUTE = append(globals.ESTADOS.EXECUTE[:pos], globals.ESTADOS.EXECUTE[pos+1:]...)
	case globals.NEW:
		pos := general.BuscarProcesoEnNew(pid)
		globals.ESTADOS.NEW = append(globals.ESTADOS.NEW[:pos], globals.ESTADOS.NEW[pos+1:]...)
	case globals.SUSP_BLOCKED:
		pos := general.BuscarProcesoEnSuspBlocked(pid)
		globals.ESTADOS.SUSP_BLOCKED = append(globals.ESTADOS.SUSP_BLOCKED[:pos], globals.ESTADOS.SUSP_BLOCKED[pos+1:]...)
	case globals.SUSP_READY:
		pos := general.BuscarProcesoEnSuspReady(pid)
		globals.ESTADOS.SUSP_READY = append(globals.ESTADOS.SUSP_READY[:pos], globals.ESTADOS.SUSP_READY[pos+1:]...)
	case globals.READY:
		pos := general.BuscarProcesoEnReady(pid)
		globals.ESTADOS.READY = append(globals.ESTADOS.READY[:pos], globals.ESTADOS.READY[pos+1:]...)
	default:
		log.Printf("Error eliminando proceso PID: %d de su cola en EliminarDeSuCola", pid)
	}
	globals.EstadosMutex.Unlock()
}

func procesoAExit(proceso globals.Proceso) {
	// Actualizamos metricas
	globals.MapaProcesosMutex.Lock()
	proceso = general.ActualizarMetricas(proceso, proceso.Estado_Actual)
	globals.MapaProcesosMutex.Unlock()

	// LOG Cambio de Estado: ## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>
	slog.Info(fmt.Sprintf("## (%d) Pasa del estado %s al estado EXIT", proceso.Pcb.Pid, proceso.Estado_Actual))

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

func newAReady(proceso globals.Proceso_Nuevo) {
	// Mutex de estados y mapa procesos estan lockeados (pasarProcesosAReady)

	procesoEnReady := globals.Proceso{
		Pcb:                  proceso.Proceso.Pcb,
		Estado_Actual:        globals.READY,
		Rafaga:               proceso.Proceso.Rafaga,
		UltimoCambioDeEstado: proceso.Proceso.UltimoCambioDeEstado,
	}

	//log.Print("Se quiere loquear MapaProcesos en newAReady")
	globals.MapaProcesosMutex.Lock()
	//log.Print("Se loquea MapaProcesos en newAReady")
	procesoEnReady = general.ActualizarMetricas(procesoEnReady, globals.NEW)
	globals.MapaProcesos[procesoEnReady.Pcb.Pid] = procesoEnReady
	globals.ESTADOS.NEW = globals.ESTADOS.NEW[1:]
	globals.ESTADOS.READY = append(globals.ESTADOS.READY, procesoEnReady.Pcb.Pid)
	globals.MapaProcesosMutex.Unlock()
	//log.Print("Se unloquea MapaProcesos en newAReady")

	log.Printf("cantidad de procesos en READY: %+v", len(globals.ESTADOS.READY))

	general.NotificarProcesoEnReady(globals.NotificadorDesalojo)
	general.Signal(globals.Sem_ProcesosEnReady) // Nuevo proceso en ready

	// LOG Cambio de Estado: ## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>
	slog.Info(fmt.Sprintf("## (%d) Pasa del estado NEW al estado READY", procesoEnReady.Pcb.Pid))
}

func suspReadyAReady(proceso globals.Proceso) {

	//log.Print("Se quiere loquear MapaProcesos en suspReadyAReady")
	globals.MapaProcesosMutex.Lock()
	proceso = general.ActualizarMetricas(proceso, proceso.Estado_Actual)
	proceso.Estado_Actual = globals.READY
	globals.MapaProcesos[proceso.Pcb.Pid] = proceso
	globals.ESTADOS.SUSP_READY = globals.ESTADOS.SUSP_READY[1:]
	globals.ESTADOS.READY = append(globals.ESTADOS.READY, proceso.Pcb.Pid)
	globals.MapaProcesosMutex.Unlock()
	//log.Print("Se unloquea MapaProcesos en suspReadyAReady")

	general.NotificarProcesoEnReady(globals.NotificadorDesalojo)
	general.Signal(globals.Sem_ProcesosEnReady) // Nuevo proceso en ready

	// LOG Cambio de Estado: ## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>
	slog.Info(fmt.Sprintf("## (%d) Pasa del estado SUSP_READY al estado READY", proceso.Pcb.Pid))
}
