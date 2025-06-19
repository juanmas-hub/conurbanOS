package utils

import (
	//"encoding/json"
	//"fmt"
	"log"
	//"net/http"
	"os"

	globals_memoria "github.com/sisoputnfrba/tp-golang/globals/memoria"
)

func moverseAPaginaSWAP(pagina globals_memoria.Pagina, archivo *os.File) int {
	direccion := int64(pagina.IndiceSwapAsignado * int(globals_memoria.MemoriaConfig.Page_size))

	// Posicionarse en la dirección deseada
	_, err := archivo.Seek(direccion, 0)
	if err != nil {
		log.Printf("Error al posicionarse en el archivo: %v", err)
		return 1
	}
	return 0
}


func leerPaginaSWAP(pagina globals_memoria.Pagina, archivo *os.File) string {

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

func eliminarPaginasSWAP(pid int) []globals_memoria.PaginaDTO {
	var archivo *os.File = abrirArchivoBinario()
	var paginasSwap *[]globals_memoria.Pagina = &globals_memoria.Procesos[pid].PaginasSWAP
	var pageSize int = int(globals_memoria.MemoriaConfig.Page_size)
	var paginasDTO []globals_memoria.PaginaDTO
	var nuevasPaginasSwapDisponibles []globals_memoria.Pagina

	var paginaDTO globals_memoria.PaginaDTO

	for i:=0; i<len(*paginasSwap); i++ {

		paginaDTO.Contenido = leerPaginaSWAP((*paginasSwap)[i], archivo)
		paginaDTO.Entrada = (*paginasSwap)[i].EntradaAsignada

		paginasDTO = append(paginasDTO, paginaDTO)

		// Volver a posicionarse para sobrescribir
		if moverseAPaginaSWAP((*paginasSwap)[i], archivo) == 1 {
			log.Printf("Error al reposicionarse para sobrescribir la página SWAP %d en el indice %v", (*paginasSwap)[i].IndiceSwapAsignado, i)
			return nil
		}

		// Sobrescribir con ceros
		ceros := make([]byte, pageSize)
		_, err := archivo.Write(ceros)
		if err != nil {
			log.Printf("error al sobrescribir la página: %v", err)
			return nil
		}

		nuevasPaginasSwapDisponibles = append(nuevasPaginasSwapDisponibles, (*paginasSwap)[i])
	}
	globals_memoria.ListaPaginasSwapDisponibles = append(globals_memoria.ListaPaginasSwapDisponibles, nuevasPaginasSwapDisponibles...)
	*paginasSwap = nil

	return paginasDTO
}

func abrirArchivoBinario() *os.File{
	var ruta string = globals_memoria.MemoriaConfig.Swapfile_path

	archivo, err := os.OpenFile(ruta, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil
	}
	return archivo
}

func obtenerIndiceSwapDisponible()int{
	var IndiceSwapDisponible int

	if len(globals_memoria.ListaPaginasSwapDisponibles) > 0 {
		IndiceSwapDisponible = globals_memoria.ListaPaginasSwapDisponibles[0].IndiceSwapAsignado
		globals_memoria.ListaPaginasSwapDisponibles = globals_memoria.ListaPaginasSwapDisponibles[1:] // eliminar primera
	} else {
		IndiceSwapDisponible = globals_memoria.ProximoIndiceSwap
		globals_memoria.ProximoIndiceSwap++
	}

	return IndiceSwapDisponible
}

func escribirPaginaSWAP(dato string, pagina globals_memoria.Pagina, archivo *os.File) int{ // dato de tamanio 64

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

func escribirEnSWAP(pid int, paginasDTO []globals_memoria.PaginaDTO) int {
	var archivo *os.File = abrirArchivoBinario()

	for i := 0; i < len(paginasDTO); i++ {
		var paginaDTO globals_memoria.PaginaDTO 
		var paginaSwapDisponible globals_memoria.Pagina

		paginaDTO = paginasDTO[i]

		// Transferimos los campos
		paginaSwapDisponible.IndiceSwapAsignado = obtenerIndiceSwapDisponible()
		paginaSwapDisponible.EntradaAsignada = paginaDTO.Entrada
		paginaSwapDisponible.IndiceAsignado = -1

		// Escribir contenido en la página asignada
		err := escribirPaginaSWAP(paginaDTO.Contenido, paginaSwapDisponible, archivo)
		if err != 0 {
			log.Printf("Error al escribir en página SWAP %d: %v", paginaSwapDisponible.IndiceSwapAsignado, err)
			return 1
		}
		globals_memoria.Procesos[pid].PaginasSWAP = append(globals_memoria.Procesos[pid].PaginasSWAP, paginaSwapDisponible)

	}
		return 0
	}

	

