package utils

import (
	"encoding/json"
	//"fmt"
	"log"
	"net/http"
	"os"
	"bufio"
	"strings"

	globals_memoria "github.com/sisoputnfrba/tp-golang/globals/memoria"
)

func CalcularMock() int {
	PAGE_SIZE := int(globals_memoria.MemoriaConfig.Page_size)
	libres := 0

	for _, estaOcupado := range globals_memoria.MemoriaMarcosOcupados {
		if !estaOcupado {
			libres++
		}
	}
	return libres * PAGE_SIZE
}

func ConsultarMock(w http.ResponseWriter, r *http.Request) {
	mock := CalcularMock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(map[string]int{
		"espacio_libre": mock,
	})
}

func IniciarProceso(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var mensaje globals_memoria.SolicitudIniciarProceso
	err := decoder.Decode(&mensaje)
	if err != nil {
		log.Printf("Error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	log.Println("Me llego para iniciar un proceso")
	log.Printf("%+v\n", mensaje)

	if (CargarProcesoDesdeArchivo(int(mensaje.Pid), mensaje.Archivo_Pseudocodigo) != 0){
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte("notImplemented"))
	}else{
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}
}

func SuspenderProceso(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var mensaje globals_memoria.PidProceso
	err := decoder.Decode(&mensaje)
	if err != nil {
		log.Printf("Error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	log.Printf("Me llego para suspender el proceso de pid: %d", mensaje.Pid)

	// Aca tenes que hacer lo que sea para suspender

	// Hay que swappear las instruccionesss

	delete(globals_memoria.Instrucciones, int(mensaje.Pid))

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func FinalizarProceso(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var mensaje globals_memoria.PidProceso
	err := decoder.Decode(&mensaje)
	if err != nil {
		log.Printf("Error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	log.Printf("Me llego para finalizar el proceso de pid: %d", mensaje.Pid)

	// Aca tenes que hacer lo que sea para finalizar

	// Marcar como libres sus entradas en SWAP

	delete(globals_memoria.Instrucciones, int(mensaje.Pid))

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func MemoryDump(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var mensaje globals_memoria.PidProceso
	err := decoder.Decode(&mensaje)
	if err != nil {
		log.Printf("Error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	log.Printf("Me llego para memory dump el proceso de pid: %d", mensaje.Pid)

	// Aca tenes que hacer lo que sea para finalizar

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func abrirArchivo(filename string) *os.File{
	file, err := os.Open(filename)
	if err != nil {
		log.Println("No se pudo abrir el archivo")
		return nil
	}
	defer file.Close()
	return file
}

func extraerInstrucciones(archivo *os.File) []string{
	var instrucciones []string
	scanner := bufio.NewScanner(archivo)
	for scanner.Scan() {
		linea := strings.TrimSpace(scanner.Text())
		if linea != "" {
			instrucciones = append(instrucciones, linea)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Println("Error al extraer las instrucciones del archivo")
		return nil
	}

	return instrucciones
}

func CargarProcesoDesdeArchivo(pid int, filename string) int {

	if (globals_memoria.Instrucciones[pid] != nil){
		log.Printf("El archivo de pid %d ya tenia sus instrucciones guardadas", pid)
		return 1
	}

	var archivo *os.File = abrirArchivo(filename)
	if (archivo == nil){
		return 1
	}

	globals_memoria.Instrucciones[pid] = extraerInstrucciones(archivo)

	return 0
}