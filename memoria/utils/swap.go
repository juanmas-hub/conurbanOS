package utils

import (
	//"encoding/json"
	//"fmt"
	"log"
	//"net/http"
	"os"

	globals_memoria "github.com/sisoputnfrba/tp-golang/globals/memoria"
)

func moverseAPaginaSWAP(pagina int, archivo *os.File) int {
	direccion := int64(pagina * int(globals_memoria.MemoriaConfig.Page_size))

	// Posicionarse en la dirección deseada
	_, err := archivo.Seek(direccion, 0)
	if err != nil {
		log.Printf("Error al posicionarse en el archivo: %v", err)
		return 1
	}
	return 0
}


func leerPaginaSWAP(pagina int, archivo *os.File) string {

	if (moverseAPaginaSWAP(pagina, archivo) == 1) {
		return ""
	}

	// Leer los 64 bytes
	buffer := make([]byte, int(globals_memoria.MemoriaConfig.Page_size))
	_, err := archivo.Read(buffer)
	if err != nil {
		log.Printf("error al leer del archivo: %v", err)
		return ""
	}

	return string(buffer)
}

func eliminarPaginaSWAP(pid int, pagina int, archivo *os.File) int {

	if (moverseAPaginaSWAP(pagina, archivo) == 1) {
		return 1
	}

	// Sobrescribir con ceros
	ceros := make([]byte, int(globals_memoria.MemoriaConfig.Page_size))
	_, err := archivo.Write(ceros)
	if err != nil {
		log.Printf("error al sobrescribir la página: %v", err)
		return 1
	}

	globals_memoria.ListaPaginasSwapDisponibles = append(globals_memoria.ListaPaginasSwapDisponibles, pagina)

	for i := 0; i < len(globals_memoria.PaginasSwapProceso[pid]); i++{
		if (globals_memoria.PaginasSwapProceso[pid])[i] == pagina {
			globals_memoria.PaginasSwapProceso[pid] = 
			append(globals_memoria.PaginasSwapProceso[pid][:i], globals_memoria.PaginasSwapProceso[pid][i+1:]...)
			break
		} 
	}



	return 0
}

func abrirArchivoBinario() *os.File{
	var ruta string = globals_memoria.MemoriaConfig.Swapfile_path

	archivo, err := os.OpenFile(ruta, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil
	}
	return archivo
}

func escribirPaginaSWAP(dato string, pagina int, archivo *os.File) int{ // dato de tamanio 64

	if (moverseAPaginaSWAP(pagina, archivo) == 1) {
		return 1
	}

	// Escribir el dato como bytes
	_, err := archivo.Write([]byte(dato))
	if err != nil {
		log.Printf("error al escribir en el archivo: %v", err)
		return 1
	}

	return 0
}

func escribirEnSWAP(pid int, datos []string, archivo *os.File) int {
	var tamanioPagina int = int(globals_memoria.MemoriaConfig.Page_size)

	for i := 0; i < len(datos); i++ {
		var dato string = datos[i]
		var bytesDato []byte = []byte(dato)

		// Calcular cantidad de páginas necesarias (con redondeo hacia arriba)
		cantidadPaginasNecesarias := (len(bytesDato) + tamanioPagina - 1) / tamanioPagina

		for j := 0; j < cantidadPaginasNecesarias; j++ {
			// Calcular el inicio y fin del segmento
			inicio := j * tamanioPagina
			fin := inicio + tamanioPagina
			if fin > len(bytesDato) {
				fin = len(bytesDato)
			}

			// Extraer segmento
			var segmento []byte = bytesDato[inicio:fin]
			// Asegurar que mida exactamente tamanioPagina (rellenar con ceros si es necesario)
			if len(segmento) < tamanioPagina {
				padding := make([]byte, tamanioPagina-len(segmento))
				segmento = append(segmento, padding...) // Uso ellipsis (...)
			}

			// Obtener página disponible
			var paginaDisponible int

			if len(globals_memoria.ListaPaginasSwapDisponibles) > 0 {
				paginaDisponible = globals_memoria.ListaPaginasSwapDisponibles[0]
				globals_memoria.ListaPaginasSwapDisponibles = globals_memoria.ListaPaginasSwapDisponibles[1:] // eliminar primera
			} else {
				paginaDisponible = globals_memoria.ProximaPaginaSwap
				globals_memoria.ProximaPaginaSwap++
			}

			// Escribir segmento en la página asignada
			err := escribirPaginaSWAP(string(segmento), paginaDisponible, archivo)
			if err != 0 {
				log.Printf("Error al escribir en página %d: %v", paginaDisponible, err)
				return 1
			}
			globals_memoria.PaginasSwapProceso[pid] = append(globals_memoria.PaginasSwapProceso[pid], paginaDisponible)

		}
	}

	return 0
}
