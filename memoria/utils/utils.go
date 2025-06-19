package utils

import (
	//"encoding/json"
	//"fmt"
	"bufio"
	"log"
	//"net/http"
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
