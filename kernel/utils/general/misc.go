package general

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
)

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

	for _, cpu := range globals.ListaCPUs {
		if !cpu.EstaLibre && cpu.PIDActual == pid {
			return cpu.Handshake.IP, cpu.Handshake.Puerto, cpu.Handshake.Nombre, true
		}
	}
	return "", 0, "", false
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

func EnviarInterrupcionACPU(ip string, puerto int64, nombre string, pid int64) (*globals.RespuestaInterrupcion, error) {
	mensaje := globals.PidJSON{PID: pid}
	body, err := json.Marshal(mensaje)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	// Posible problema con el int64 del puerto
	url := fmt.Sprintf("http://%s:%d/interrumpir", ip, puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando interrupción a ip:%s puerto:%d", ip, puerto)
	}

	slog.Debug(fmt.Sprintf("Interrupcion enviada a CPU: %s, resp: %s", nombre, resp.Status))

	// Respuesta de CPU
	var respuesta globals.RespuestaInterrupcion
	if err := json.NewDecoder(resp.Body).Decode(&respuesta); err != nil {
		log.Printf("error decodificando respuesta de la CPU: %s", err.Error())
		return nil, err
	}
	return &respuesta, nil
}

func EnviarDumpMemory(pid int64) bool {
	mensaje := globals.PidJSON{PID: pid}
	body, err := json.Marshal(mensaje)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	// Posible problema con el int64 del puerto
	url := fmt.Sprintf("http://%s:%d/memoryDump", globals.KernelConfig.Ip_memory, globals.KernelConfig.Port_memory)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando mensaje a ip:%s puerto:%d", globals.KernelConfig.Ip_memory, globals.KernelConfig.Port_memory)
	}

	slog.Debug(fmt.Sprintf("Enviado DUMP MEMORY a memoria, resp: %s", resp.Status))

	if resp.StatusCode == http.StatusOK {
		return true
	}
	return false
}

func EnviarSolicitudIO(ipIO string, puertoIO int64, pid int64, tiempo int64) {

	solicitud := globals.SolicitudIO{
		PID:    pid,
		Tiempo: tiempo,
	}

	body, err := json.Marshal(solicitud)
	if err != nil {
		log.Printf("Error codificando la solicitud IO: %s", err.Error())
	}

	url := fmt.Sprintf("http://%s:%d/solicitudDeIo", ipIO, puertoIO)

	slog.Debug(fmt.Sprintf("Solicitud IO enviada al modulo IO - PID: %d, Tiempo: %dms", pid, tiempo))

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("Error enviando solicitud IO a ipIO:%s puertoIO:%d", ipIO, puertoIO)
	}

	slog.Debug(fmt.Sprintf("Solicitud IO enviada al modulo IO - PID: %d, Tiempo: %dms, respuesta: %s", pid, tiempo, resp.Status))

	globals.CantidadSesionesIOMutex.Lock()
	globals.CantidadSesionesIO[pid]++
	globals.CantidadSesionesIOMutex.Unlock()
}

func AvisarSwappeo(pid int64) {
	mensaje := globals.PidJSON{PID: pid}
	body, err := json.Marshal(mensaje)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	// Posible problema con el int64 del puerto
	url := fmt.Sprintf("http://%s:%d/suspenderProceso", globals.KernelConfig.Ip_memory, globals.KernelConfig.Port_memory)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando mensaje a ip:%s puerto:%d", globals.KernelConfig.Ip_memory, globals.KernelConfig.Port_memory)
	}

	slog.Debug(fmt.Sprintf("Enviado aviso de swappeo de PID %d a memoria, resp: %s", pid, resp.Status))
}

func LogLockeo(semaforo string, funcion string) {
	slog.Debug(fmt.Sprintf("Se lockeo %s en %s", semaforo, funcion))
}

func LogIntentoLockeo(semaforo string, funcion string) {
	slog.Debug(fmt.Sprintf("Se quiere lockear %s en %s", semaforo, funcion))
}

func LogUnlockeo(semaforo string, funcion string) {
	slog.Debug(fmt.Sprintf("Se unlockeo %s en %s", semaforo, funcion))
}

func EstaEnCola(cola []int64, pid int64) bool {
	for _, val := range cola {
		if val == pid {
			return true
		}
	}
	return false
}
