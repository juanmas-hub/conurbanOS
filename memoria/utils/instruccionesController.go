package utils

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	//"fmt"
	//"bufio"

	"net/http"

	//"os"
	//"strings"

	//globals "github.com/sisoputnfrba/tp-golang/globals/memoria"
	globals_memoria "github.com/sisoputnfrba/tp-golang/globals/memoria"
)

func ConsultarMock(w http.ResponseWriter, r *http.Request) {
	mock := CalcularMock()

	var enviado struct {
		Mock int `json:"mock"`
	}

	enviado.Mock = mock

	jsonData, err := json.Marshal(enviado)

	if err != nil {
		slog.Debug(fmt.Sprintf("Error al codificar el mock a JSON: %s", err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error interno del servidor"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)

}

/*
	func EnviarInstruccion(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		var mensaje globals_memoria.InstruccionDTO
		err := decoder.Decode(&mensaje)
		if err != nil {
			log.Printf("Error al decodificar mensaje: %s\n", err.Error())
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Error al decodificar mensaje"))
			return
		}

		log.Printf("Solicitud de instruccion de PID: %d y PC: %d", mensaje.Pid, mensaje.Pc)
		log.Printf("%+v\n", mensaje.Pid)

		var instruccion string = globals_memoria.Procesos[int(mensaje.Pid)].Pseudocodigo[mensaje.Pc]
		var delayMem int64 = globals_memoria.MemoriaConfig.Memory_delay

		time.Sleep(time.Duration(delayMem) * time.Millisecond)
		if (*globals_memoria.Metricas)[int(mensaje.Pid)] == nil {
			(*globals_memoria.Metricas)[int(mensaje.Pid)] = &globals_memoria.Memoria_Metrica{
				AccesosTablas:            0,
				InstruccionesSolicitadas: 0,
				BajadasSwap:              0,
				SubidasMemoria:           0,
				LecturasMemoria:          0,
				EscriturasMemoria:        0,
			}
		}

		(*globals_memoria.Metricas)[int(mensaje.Pid)].InstruccionesSolicitadas++

		log.Printf("## PID: %d - Obtener instrucción: %d - Instrucción: %s", mensaje.Pid, mensaje.Pc, instruccion)

		var enviado struct {
			Instruccion string `json:"instruccion"`
		}
		enviado.Instruccion = instruccion
		jsonData, err := json.Marshal(enviado)
		if err != nil {
			log.Printf("Error al codificar la instruccion a JSON: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Error interno del servidor"))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonData)
	}
*/
func EnviarInstruccion(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var mensaje globals_memoria.InstruccionDTO
	err := decoder.Decode(&mensaje)
	if err != nil {
		slog.Debug(fmt.Sprintf("Error al decodificar mensaje: %s\n", err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	slog.Debug(fmt.Sprintf("Solicitud de instruccion de PID: %d y PC: %d", mensaje.Pid, mensaje.Pc))
	slog.Debug(fmt.Sprintf("%+v\n", mensaje.Pid))

	var instruccion string = globals_memoria.Procesos[int(mensaje.Pid)].Pseudocodigo[mensaje.Pc]
	var delayMem int64 = globals_memoria.MemoriaConfig.Memory_delay

	time.Sleep(time.Duration(delayMem) * time.Millisecond)

	metricas, existe := globals_memoria.MetricasMap[int(mensaje.Pid)]

	if !existe {
		metricas = globals_memoria.Memoria_Metrica{
			AccesosTablas:            0,
			InstruccionesSolicitadas: 0,
			BajadasSwap:              0,
			SubidasMemoria:           0,
			LecturasMemoria:          0,
			EscriturasMemoria:        0,
		}
	}
	metricas.InstruccionesSolicitadas++
	globals_memoria.MetricasMap[int(mensaje.Pid)] = metricas

	slog.Info(fmt.Sprintf("## PID: %d - Obtener instrucción: %d - Instrucción: %s", mensaje.Pid, mensaje.Pc, instruccion))

	var enviado struct {
		Instruccion string `json:"instruccion"`
	}
	enviado.Instruccion = instruccion
	jsonData, err := json.Marshal(enviado)
	if err != nil {
		slog.Debug(fmt.Sprintf("Error al codificar la instruccion a JSON: %s", err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error interno del servidor"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}

/*
func AccederEspacioUsuarioLectura(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var mensaje globals_memoria.LecturaDTO
	err := decoder.Decode(&mensaje)
	if err != nil {
		log.Printf("Error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	var pid int = int(mensaje.Pid)
	var posicion int = int(mensaje.Posicion)
	var tamanio int = int(mensaje.Tamanio)
	var delayMem int64 = globals_memoria.MemoriaConfig.Memory_delay

	time.Sleep(time.Duration(delayMem) * time.Millisecond)
	(*globals_memoria.Metricas)[pid].LecturasMemoria++
	var leido string = leer(posicion, tamanio)

	var enviado struct {
		Dato string `json:"dato"`
	}
	enviado.Dato = leido
	jsonData, err := json.Marshal(enviado)
	if err != nil {
		log.Printf("Error al codificar la instruccion a JSON: %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error interno del servidor"))
		return
	}

	log.Printf("## PID: %d - Lectura - Dir. Física: %d - Tamaño: %d", mensaje.Pid, mensaje.Posicion, mensaje.Tamanio)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}*/

func AccederEspacioUsuarioLectura(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var mensaje globals_memoria.LecturaDTO
	err := decoder.Decode(&mensaje)
	if err != nil {
		slog.Debug(fmt.Sprintf("Error al decodificar mensaje: %s\n", err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	var pid int = int(mensaje.Pid)
	proceso := globals_memoria.Procesos[pid]
	var direccionFisica int = int(mensaje.Posicion)
	var tamanio int = int(mensaje.Tamanio)

	var delayMem int64 = globals_memoria.MemoriaConfig.Memory_delay
	time.Sleep(time.Duration(delayMem) * time.Millisecond)

	metricas := globals_memoria.MetricasMap[pid]
	metricas.LecturasMemoria++
	globals_memoria.MetricasMap[pid] = metricas

	var leido string = leer(direccionFisica, tamanio)

	var enviado struct {
		Dato string `json:"dato"`
	}
	enviado.Dato = leido
	jsonData, err := json.Marshal(enviado)
	if err != nil {
		slog.Debug(fmt.Sprintf("Error al codificar la instruccion a JSON: %s", err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error interno del servidor"))
		return
	}

	slog.Info(fmt.Sprintf("## PID: %d - Lectura - Dir. Física: %d - Tamaño: %d", mensaje.Pid, mensaje.Posicion, mensaje.Tamanio))

	globals_memoria.Procesos[pid] = proceso

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}

/*
func AccederEspacioUsuarioEscritura(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var mensaje globals_memoria.EscrituraDTO
	err := decoder.Decode(&mensaje)
	if err != nil {
		log.Printf("Error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	var pid int = int(mensaje.Pid)
	var posicion int = int(mensaje.Posicion)
	var dato string = mensaje.Dato
	var delayMem int64 = globals_memoria.MemoriaConfig.Memory_delay

	time.Sleep(time.Duration(delayMem) * time.Millisecond)
	if escribir(posicion, dato) < 0 {
		log.Printf("Error al escribir en la posicion %v", int(mensaje.Posicion))
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Error al escribir en la posicion"))
		return
	}

	(*globals_memoria.Metricas)[pid].EscriturasMemoria++
	log.Printf("## PID: %d - Lectura - Dir. Física: %d - Tamaño: %d", mensaje.Pid, mensaje.Posicion, len(mensaje.Dato))

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}*/

func AccederEspacioUsuarioEscritura(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var mensaje globals_memoria.EscrituraDTO
	err := decoder.Decode(&mensaje)
	if err != nil {
		//log.Printf("Error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	var pid int = int(mensaje.Pid)
	proceso := globals_memoria.Procesos[pid]
	var direccionFisica int = int(mensaje.DireccionFisica)
	var dato string = mensaje.Dato
	var delayMem int64 = globals_memoria.MemoriaConfig.Memory_delay

	time.Sleep(time.Duration(delayMem) * time.Millisecond)
	if escribir(direccionFisica, dato) < 0 {
		//log.Printf("Error al escribir en la posicion %v", direccionFisica)
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Error al escribir en la posicion"))
		return
	}

	metricas := globals_memoria.MetricasMap[pid]
	metricas.EscriturasMemoria++
	globals_memoria.MetricasMap[pid] = metricas

	slog.Info(fmt.Sprintf("## PID: %d - Escritura - Dir. Física: %d - Tamaño: %d", mensaje.Pid, direccionFisica, len(mensaje.Dato)))
	slog.Debug(fmt.Sprint("Memoria despues de escribir: ", globals_memoria.Memoria))

	globals_memoria.Procesos[pid] = proceso

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
