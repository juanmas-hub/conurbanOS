package planificadores

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
)

func Enviar_proceso_a_cpu(ip string, puerto int64, pid int64, pc int64, nombre string) {
	proc := globals.ProcesoAExecutar{
		PID: pid,
		PC:  pc,
	}

	//log.Printf("cpu libre elegida ip: %s, port: %d, pid: %d, pc: %d", ip, puerto, pid, pc)
	body, err := json.Marshal(proc)
	if err != nil {
		slog.Debug(fmt.Sprintf("error codificando proceso a ejecutar: %s", err.Error()))
	}

	// Posible problema con el int64 del puerto
	url := fmt.Sprintf("http://%s:%d/dispatchProceso", ip, puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		slog.Debug(fmt.Sprintf("error enviando mensaje a ip:%s puerto:%d, error: %v", ip, puerto, err))
	}

	slog.Debug(fmt.Sprintf("Proceso PID %d enviado a %s, respuesta: %s", pid, nombre, resp.Status))

	// Guardar PID en la CPU correspondiente
	globals.ListaCPUsMutex.Lock()
	for i := range globals.ListaCPUs {
		if globals.ListaCPUs[i].Handshake.IP == ip && globals.ListaCPUs[i].Handshake.Puerto == puerto {
			globals.ListaCPUs[i].PIDActual = pid
			break
		}
	}
	globals.ListaCPUsMutex.Unlock()
}

func enviar_finalizacion_a_memoria(ip string, puerto int64, pid int64) bool {
	mensaje := globals.PidJSON{PID: pid}
	body, err := json.Marshal(mensaje)
	if err != nil {
		slog.Debug(fmt.Sprintf("error codificando mensaje: %s", err.Error()))
	}

	// Posible problema con el int64 del puerto
	url := fmt.Sprintf("http://%s:%d/finalizarProceso", ip, puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		slog.Debug(fmt.Sprintf("error enviando mensaje a ip:%s puerto:%d", ip, puerto))
	}

	//slog.Debug(fmt.Sprintf("Finalizacion PID %d enviada a memoria, respuesta: %s", pid, resp.Status))

	if resp.StatusCode == http.StatusOK {
		return true
	}
	return false
}

func enviar_reanudar_proceso_a_memoria(pid int64) bool {
	// Se pudo iniciarlizar => devuelve true
	// No se pudo inicializar => devuelve false
	mensaje := globals.PidJSON{PID: pid}

	body, err := json.Marshal(mensaje)
	if err != nil {
		slog.Debug(fmt.Sprintf("error codificando mensaje: %s", err.Error()))
	}

	url := fmt.Sprintf("http://%s:%d/reanudarProceso", globals.KernelConfig.Ip_memory, globals.KernelConfig.Port_memory)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		slog.Debug(fmt.Sprintf("error enviando mensaje a ip:%s puerto:%d", globals.KernelConfig.Ip_memory, globals.KernelConfig.Port_memory))
	}

	//slog.Debug(fmt.Sprintf("respuesta del servidor: %s", resp.Status))

	if resp.Status == "200 OK" {
		return true
	}

	return false
}

func enviar_inicializar_proceso_a_memoria(proceso globals.Proceso) bool {
	// Se pudo iniciarlizar => devuelve true
	// No se pudo inicializar => devuelve false

	mensaje := globals.SolicitudIniciarProceso{
		Archivo_Pseudocodigo: proceso.Archivo_Pseudocodigo,
		Tamanio:              proceso.Tamaño,
		Pid:                  proceso.Pcb.Pid,
		Susp:                 false,
	}
	body, err := json.Marshal(mensaje)
	if err != nil {
		slog.Debug(fmt.Sprintf("error codificando mensaje: %s", err.Error()))
	}

	// Posible problema con el int64 del puerto
	url := fmt.Sprintf("http://%s:%d/iniciarProceso", globals.KernelConfig.Ip_memory, globals.KernelConfig.Port_memory)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		slog.Debug(fmt.Sprintf("error enviando mensaje a ip:%s puerto:%d", globals.KernelConfig.Ip_memory, globals.KernelConfig.Port_memory))
	}

	//slog.Debug(fmt.Sprintf("respuesta del servidor: %s", resp.Status))

	if resp.Status == "200 OK" {
		return true
	}

	return false
}

func enviar_suspension_proceso_a_memoria(pid int64) {
	mensaje := globals.PidJSON{PID: pid}
	body, err := json.Marshal(mensaje)
	if err != nil {
		slog.Debug(fmt.Sprintf("error codificando mensaje: %s", err.Error()))
	}

	// Posible problema con el int64 del puerto
	url := fmt.Sprintf("http://%s:%d/suspenderProceso", globals.KernelConfig.Ip_memory, globals.KernelConfig.Port_memory)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		slog.Debug(fmt.Sprintf("error enviando mensaje a ip:%s puerto:%d", globals.KernelConfig.Ip_memory, globals.KernelConfig.Port_memory))
	}

	slog.Debug(fmt.Sprintf("Enviado aviso de swappeo de PID %d a memoria, resp: %s", pid, resp.Status))
}

func enviar_interrupcion_a_cpu(ip string, puerto int64, nombre string, pid int64) (*globals.RespuestaInterrupcion, error) {
	mensaje := globals.PidJSON{PID: pid}
	body, err := json.Marshal(mensaje)
	if err != nil {
		slog.Debug(fmt.Sprintf("error codificando mensaje: %s", err.Error()))
	}

	// Posible problema con el int64 del puerto
	url := fmt.Sprintf("http://%s:%d/interrumpir", ip, puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		slog.Debug(fmt.Sprintf("error enviando interrupción a ip:%s puerto:%d", ip, puerto))
	}

	//slog.Debug(fmt.Sprintf("Interrupcion enviada a CPU: %s, resp: %s", nombre, resp.Status))

	// Respuesta de CPU
	var respuesta globals.RespuestaInterrupcion
	if err := json.NewDecoder(resp.Body).Decode(&respuesta); err != nil {
		slog.Debug(fmt.Sprintf("error decodificando respuesta de la CPU: %s", err.Error()))
		return nil, err
	}
	return &respuesta, nil
}
