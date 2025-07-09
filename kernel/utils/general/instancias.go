package utils_general

import (
	"log"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
)

func agregarAInstanciasIOs(handshake globals.Handshake) {
	elementoAAgregar := globals.InstanciaIO{
		Handshake:        handshake,
		PidProcesoActual: -1,
	}
	io, existe := globals.MapaIOs[handshake.Nombre]
	if !existe {
		io = globals.EntradaMapaIO{}
	}
	io.Instancias = append(io.Instancias, elementoAAgregar)
	globals.MapaIOs[handshake.Nombre] = io
}

func agregarAListaCPUs(handshake globals.Handshake) {
	elementoAAgregar := globals.ListaCpu{
		Handshake: handshake,
		EstaLibre: true,
	}
	globals.ListaCPUs = append(globals.ListaCPUs, elementoAAgregar)

	switch globals.KernelConfig.Scheduler_algorithm {
	case "FIFO", "SJF":
		Signal(globals.Sem_Cpus)
	case "SRT":
		NotificarReplanifSRT()
	}
}

func BuscarCpu(nombre string) int {
	var posCpu int
	encontrado := false
	for i := range globals.ListaCPUs {
		if globals.ListaCPUs[i].Handshake.Nombre == nombre {
			posCpu = i
			encontrado = true
			break
		}
	}

	if encontrado {
		return posCpu
	} else {
		// Si devuelve esto es que se desconecto la CPU en el medio. Hay q ser mala persona
		log.Println("No se encontro la CPU en la devolucion")
		return -1
	}
}

func BuscarCpuPorPID(pid int64) (string, int64, string, bool) {
	globals.ListaCPUsMutex.Lock()
	defer globals.ListaCPUsMutex.Unlock()

	for _, cpu := range globals.ListaCPUs {
		if !cpu.EstaLibre && cpu.PIDActual == pid {
			return cpu.Handshake.IP, cpu.Handshake.Puerto, cpu.Handshake.Nombre, true
		}
	}
	return "", 0, "", false
}

// Dado el nombre del IO y una IP y Puerto, busca la instancia de IO que tiene ese IP, y devuelve su posicion en la cola. Se llama con Lista IOs muteada.
func buscarPosInstanciaIO(nombreIO string, ip string, puerto int64) int {

	io := globals.MapaIOs[nombreIO]

	for i := range io.Instancias {
		if io.Instancias[i].Handshake.Puerto == puerto && io.Instancias[i].Handshake.IP == ip {
			return i
		}
	}

	return -2

}

// Dado un nombre IO, y un PID, busca la instancia donde esta ejecutando ese proceso. Retorna posicion en cola de instancias. Se llama con Lista IO muteada
func BuscarInstanciaIO(nombreIO string, pid int64) int {

	io := globals.MapaIOs[nombreIO]

	for i := range io.Instancias {
		if io.Instancias[i].PidProcesoActual == pid {
			return i
		}
	}

	return -1
}

// Dada un nombre de IO, busca una instancia libre. Devuelve la instancia, la posicion en la cola de instancias y si hay instancia libre. Se llama con Lista IO muteada
func BuscarInstanciaIOLibre(nombreIO string) (globals.InstanciaIO, int, bool) {
	var instancia globals.InstanciaIO

	io := globals.MapaIOs[nombreIO]

	for i := range io.Instancias {
		if io.Instancias[i].PidProcesoActual == -1 {
			// Esta libre
			return io.Instancias[i], i, true
		}
	}

	return instancia, -1, false
}

// Dado un nombre de IO, devuelve si existe. Se llama con Lista IO muteada.
func VerificarExistenciaIO(nombreIO string) bool {
	_, existe := globals.MapaIOs[nombreIO]
	return existe
}
