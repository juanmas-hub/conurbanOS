package utils_planifLargo

import (
	"bufio"
	"log"
	"os"
	"sort"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
	general "github.com/sisoputnfrba/tp-golang/kernel/utils/general"
	cp "github.com/sisoputnfrba/tp-golang/kernel/utils/planifCorto"
)

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

	CrearProcesoNuevo(archivo, tamanio)
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
		Estado_Actual: globals.NEW,
		Rafaga:        nil,
	}

	procesoNuevo := globals.Proceso_Nuevo{
		Archivo_Pseudocodigo: archivo,
		Tamaño:               tamanio,
		Proceso:              proceso,
	}

	globals.EstadosMutex.Lock()
	log.Printf("Agregando proceso a NEW. Cantidad actual: %d", len(globals.ESTADOS.NEW))

	globals.ESTADOS.NEW = append(globals.ESTADOS.NEW, procesoNuevo)
	log.Printf("Después de agregar, NEW tiene %d procesos", len(globals.ESTADOS.NEW))
	if globals.KernelConfig.New_algorithm == "PMCP" {
		OrdenarNewPorTamanio()
	}
	globals.EstadosMutex.Unlock()

	// Si se crea un proceso nuevo antes de que aprete Enter, se agrega a NEW pero no se pasan procesos a READY
	if globals.PLANIFICADOR_LARGO_PLAZO_BLOCKED == false {
		PasarProcesosAReady()
	}
}

func OrdenarNewPorTamanio() {

	// Con ordenar por tamaño (mas chicho primero) ya el algoritmo PMCP estaria hecho (creo)
	sort.Slice(globals.ESTADOS.NEW, func(i, j int) bool {
		return globals.ESTADOS.NEW[i].Tamaño < globals.ESTADOS.NEW[j].Tamaño
	})
}

func PasarProcesosAReady() {
	// Esta funcion deberia llamarse cuando llega un proceso a NEW, a EXIT, a SUSP_BLOCKED y (SUSP_READY ???)
	// Voy a intentar pasar la mayor cantidad de procesos que pueda mientras memoria tenga espacio
	// Primero me fijo en SUSP READY y despues en NEW --- nose si esta bien hacerlo asi

	globals.EstadosMutex.Lock()
	globals.MapaProcesosMutex.Lock()

	var lenghtSUSP_READY = len(globals.ESTADOS.SUSP_READY)
	for lenghtSUSP_READY > 0 {
		proceso := globals.MapaProcesos[globals.ESTADOS.SUSP_READY[0]]
		if SolicitarInicializarProcesoAMemoria_DesdeSUSP_READY(proceso) == false {
			break
		}

		SuspReadyAReady(proceso)
		lenghtSUSP_READY--
	}

	if lenghtSUSP_READY == 0 {

		for len(globals.ESTADOS.NEW) > 0 {
			procesoNuevo := globals.ESTADOS.NEW[0]

			if SolicitarInicializarProcesoAMemoria_DesdeNEW(procesoNuevo) == false {
				break
			}

			NewAReady(procesoNuevo)
		}
	}

	globals.EstadosMutex.Unlock()
	globals.MapaProcesosMutex.Unlock()
}

func SolicitarInicializarProcesoAMemoria_DesdeNEW(proceso globals.Proceso_Nuevo) bool {
	// Se pudo iniciarlizar => devuelve true
	// No se pudo inicializar => devuelve false
	return true
}

func SolicitarInicializarProcesoAMemoria_DesdeSUSP_READY(proceso globals.Proceso) bool {
	// Se pudo iniciarlizar => devuelve true
	// No se pudo inicializar => devuelve false
	return true
}

func FinalizarProceso(pid int64) {
	proceso, ok := globals.MapaProcesos[pid]
	if !ok {
		log.Printf("No se encontró el proceso con PID %d", pid)
		return
	}

	// Mando el PID
	general.EnviarFinalizacionDeProceso_AMemoria(globals.KernelConfig.Ip_memory, globals.KernelConfig.Port_memory, pid)

	// Confirmación de la memoria aca...
	// Me parece que la confirmacion es por la misma funcion que por la que mandas el mensaje (memoria no tiene ip y port del kernel)
	// Que pasa si no puede finalizarlo? O no puede pasar eso?
	RecibirConfirmacionDeMemoria(proceso.Pcb.Pid)

	delete(globals.MapaProcesos, pid)
	log.Printf("El PCB del proceso con PID %d fue liberado", pid)

	// Me imagino que hay que eliminarlo de de las colas tambien, o no?
	// Diria yo que ya esta eliminado de las colas, esta funcion se llamaria cuando un proceso pasa a exit, y en todos
	// los cambios de estado los sacamos de la cola anterior

	// Iniciar nuevos procesos
	PasarProcesosAReady()

	// Loguear metricas de estado
}

func RecibirConfirmacionDeMemoria(pid int64) bool {

	return true
}

// Funciones para no hacer tanto quilombo en pasar procesos de un estado a otro

func NewAReady(proceso globals.Proceso_Nuevo) {

	procesoEnReady := globals.Proceso{
		Pcb:           proceso.Proceso.Pcb,
		Estado_Actual: globals.READY,
		Rafaga:        nil,
	}
	globals.MapaProcesos[procesoEnReady.Pcb.Pid] = procesoEnReady
	globals.ESTADOS.NEW = globals.ESTADOS.NEW[1:]
	globals.ESTADOS.READY = append(globals.ESTADOS.READY, procesoEnReady.Pcb.Pid)

	log.Printf("cantidad de procesos en READY: %+v", len(globals.ESTADOS.READY))

}

func SuspReadyAReady(proceso globals.Proceso) {

	proceso.Estado_Actual = globals.READY
	globals.MapaProcesos[proceso.Pcb.Pid] = proceso
	globals.ESTADOS.SUSP_READY = globals.ESTADOS.SUSP_READY[1:]
	globals.ESTADOS.READY = append(globals.ESTADOS.READY, proceso.Pcb.Pid)

}
