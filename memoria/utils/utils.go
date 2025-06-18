package utils

import (
	"encoding/json"
	//"fmt"
	"bufio"
	"log"
	"net/http"
	"os"
	"strings"

	//globals "github.com/sisoputnfrba/tp-golang/globals/memoria"
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

	var enviado struct {
		Mock int `json:"mock"`
	}

	enviado.Mock = mock

	jsonData, err := json.Marshal(enviado)

	if (err != nil){
		log.Printf("Error al codificar el mock a JSON: %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error interno del servidor"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)

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

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func abrirArchivo(filename string) *os.File {

	var rutaArchivo string = globals_memoria.MemoriaConfig.Scripts_path + filename + ".txt"

	log.Println("Intentando acceder a la direccion: ", rutaArchivo)

	file, err := os.Open(rutaArchivo)
	if err != nil {
		log.Println("No se pudo abrir el archivo: ", err)
		return nil
	}
	return file
}

func extraerInstrucciones(archivo *os.File) []string {
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

func ObtenerInstruccionesDesdeArchivo(filename string) []string {

	var archivo *os.File = abrirArchivo(filename)
	if archivo == nil {
		return nil
	}

	var instrucciones []string = extraerInstrucciones(archivo)

	return instrucciones
}

// FUNCIONES RELACIONADAS A PROCESOS
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

	// Aca empieza la logica
	var pid int = int(mensaje.Pid)

	if AlmacenarProceso(pid, mensaje.Archivo_Pseudocodigo) != nil {
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte("notImplemented"))
	} else {
		log.Println("Proceso iniciado con exito: ", globals_memoria.Procesos[pid])

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

	// Aca empieza la logica

	var pid int = int(mensaje.Pid)

	var paginas []string
	
	paginas = eliminarPaginasFisicas(pid)

	escribirEnSWAP(pid, paginas)

	globals_memoria.Procesos[pid].Suspendido = true

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

	// Aca empieza la logica
	var pid int = int(mensaje.Pid)

	if globals_memoria.Procesos[pid].Suspendido {
		eliminarPaginasSWAP(pid)
	}else {
		eliminarPaginasFisicas(pid)
	}
	
	delete(globals_memoria.Procesos, pid)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func ReanudarProceso(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var mensaje globals_memoria.PidProceso
	err := decoder.Decode(&mensaje)
	if err != nil {
		log.Printf("Error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	log.Println("Solicitud para reanudar proceso con swap")
	log.Printf("%+v\n", mensaje.Pid)

	// Aca empieza la logica
	var pid int = int(mensaje.Pid)
	var paginasNecesarias int = len(globals_memoria.Procesos[pid].PaginasSWAP)

	if paginasNecesarias != 0 {
		var paginas []string
		var marcosDisponibles []int = buscarMarcosDisponibles(paginasNecesarias)
		if marcosDisponibles == nil{
			// error no hay suficiente espacio
		} 
		paginas = eliminarPaginasSWAP(pid)
		escribirPaginas(paginas, marcosDisponibles)
	}
	
	

	log.Printf("Proceso %d reanudado correctamente", mensaje.Pid)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// FUNCIONES RELACIONADAS A INSTRUCCIONES
func EnviarInstruccion(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var mensaje globals_memoria.SolicitudInstruccion
	err := decoder.Decode(&mensaje)
	if err != nil {
		log.Printf("Error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	log.Printf("Solicitud de instruccion de PID: %d y PC: %d", mensaje.Pid, mensaje.Pc)
	log.Printf("%+v\n", mensaje.Pid)

	instruccion := globals_memoria.Procesos[int(mensaje.Pid)].Pseudocodigo[mensaje.Pc]
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

func AccederEspacioUsuarioLectura(w http.ResponseWriter, r *http.Request){
	decoder := json.NewDecoder(r.Body)
	var mensaje globals_memoria.SolicitudLectura
	err := decoder.Decode(&mensaje)
	if err != nil {
		log.Printf("Error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	var leido string = leer(int(mensaje.Posicion), int(mensaje.Tamanio))

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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}

func AccederEspacioUsuarioEscritura(w http.ResponseWriter, r *http.Request){
	
}