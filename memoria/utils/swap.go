package utils

import (
	//"encoding/json"
	"fmt"
	"io"
	"log/slog"

	//"net/http"
	"os"

	globals_memoria "github.com/sisoputnfrba/tp-golang/globals/memoria"
)

/*func moverseAPaginaSWAP(pagina globals_memoria.Pagina, archivo *os.File) int {
	var direccion int64 = int64(pagina.IndiceSwapAsignado * int(globals_memoria.MemoriaConfig.Page_size))

	// Posicionarse en la dirección deseada
	_, err := archivo.Seek(direccion, 0)
	if err != nil {
		log.Printf("Error al posicionarse en el archivo: %v", err)
		return 1
	}
	return 0
}*/

func moverseAPaginaSWAP(inicioSwap int, numeroDePagina int, archivo *os.File) int {
	pageSize := globals_memoria.MemoriaConfig.Page_size
	direccion := int64((inicioSwap + numeroDePagina) * int(pageSize))

	_, err := archivo.Seek(direccion, 0)
	if err != nil {
		slog.Debug(fmt.Sprintf("Error al posicionarse en el archivo: %v", err))
		return 1
	}
	return 0
}

/*
func leerPaginaSWAP(pagina globals_memoria.Pagina, archivo *os.File) string {

	if moverseAPaginaSWAP(pagina, archivo) == 1 {
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
}*/

func leerPaginaSWAP(inicioSwap int, numeroDePagina int, archivo *os.File) string {

	globals_memoria.ArchivoSwapMutex.Lock()
	defer globals_memoria.ArchivoSwapMutex.Unlock()

	if moverseAPaginaSWAP(inicioSwap, numeroDePagina, archivo) == 1 {
		return ""
	}

	// Leer los 64 bytes
	buffer := make([]byte, int(globals_memoria.MemoriaConfig.Page_size))
	_, err := archivo.Read(buffer)
	if err != nil {
		slog.Debug(fmt.Sprintf("error al leer del archivo: %v", err))
		return ""
	}

	return string(buffer)
}

/*
func eliminarPaginasSWAP(pid int) []globals_memoria.PaginaDTO {
	var archivo *os.File = abrirArchivoBinario()
	var paginasSwap *[]globals_memoria.Pagina = &globals_memoria.Procesos[pid].PaginasSWAP
	var pageSize int = int(globals_memoria.MemoriaConfig.Page_size)
	var paginasDTO []globals_memoria.PaginaDTO
	var nuevasPaginasSwapDisponibles []globals_memoria.Pagina

	var paginaDTO globals_memoria.PaginaDTO

	for i := 0; i < len(*paginasSwap); i++ {

		paginaDTO.Contenido = leerPaginaSWAP((*paginasSwap)[i], archivo)
		log.Print(paginaDTO.Contenido)
		paginaDTO.Entrada = (*paginasSwap)[i].EntradaAsignada
		log.Print(paginaDTO.Entrada)

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
		(*paginasSwap)[i].IndiceSwapAsignado = -1
	}
	globals_memoria.ListaPaginasSwapDisponibles = append(globals_memoria.ListaPaginasSwapDisponibles, nuevasPaginasSwapDisponibles...)

	*paginasSwap = nil

	return paginasDTO
}*/

func eliminarPaginasSWAP(pid int) []globals_memoria.Pagina {
	var archivo *os.File = abrirArchivoBinario()
	defer archivo.Close()

	var paginaActual globals_memoria.Pagina
	var paginas []globals_memoria.Pagina
	pageSize := globals_memoria.MemoriaConfig.Page_size
	inicioSwap := globals_memoria.Procesos[pid].InicioSWAP
	cantidadPaginas := globals_memoria.Procesos[pid].CantidadDePaginas

	for i := 0; i < cantidadPaginas; i++ {

		paginaActual.NumeroDePagina = i
		paginaActual.Contenido = []byte(leerPaginaSWAP(inicioSwap, paginaActual.NumeroDePagina, archivo))

		paginas = append(paginas, paginaActual)

		// Volver a posicionarse para sobrescribir
		if moverseAPaginaSWAP(inicioSwap, paginaActual.NumeroDePagina, archivo) == 1 {
			slog.Debug(fmt.Sprintf("Error al reposicionarse para sobrescribir la página SWAP %d en el indice %v", inicioSwap+paginaActual.NumeroDePagina, i))
			return nil
		}

		// Sobrescribir con ceros
		ceros := make([]byte, pageSize)
		_, err := archivo.Write(ceros)
		if err != nil {
			slog.Debug(fmt.Sprintf("error al sobrescribir la página: %v", err))
			return nil
		}

		// Marco los indices como libres
		globals_memoria.IndicesSWAPOcupadosMutex.Lock()
		for i := 0; i < cantidadPaginas; i++ {
			if inicioSwap+i < len(globals_memoria.IndicesSWAPOcupados) {
				globals_memoria.IndicesSWAPOcupados[inicioSwap+i] = false
			}
		}
		globals_memoria.IndicesSWAPOcupadosMutex.Unlock()

	}

	return paginas
}

/*func obtenerIndiceSwapDisponible() int {
	var IndiceSwapDisponible int

	if len(globals_memoria.ListaPaginasSwapDisponibles) > 0 {
		IndiceSwapDisponible = globals_memoria.ListaPaginasSwapDisponibles[0].IndiceSwapAsignado
		globals_memoria.ListaPaginasSwapDisponibles = globals_memoria.ListaPaginasSwapDisponibles[1:] // eliminar primera
	} else {
		IndiceSwapDisponible = globals_memoria.ProximoIndiceSwap
		globals_memoria.ProximoIndiceSwap++
	}

	return IndiceSwapDisponible
}*/

// Devuelve la posicion donde empezaria a escribir
func obtenerIndiceSwapDisponible(cantPaginasNecesarias int) int {
	// Buscar un bloque libre contiguo
	globals_memoria.IndicesSWAPOcupadosMutex.Lock()
	defer globals_memoria.IndicesSWAPOcupadosMutex.Unlock()
	for i := 0; i <= len(globals_memoria.IndicesSWAPOcupados)-cantPaginasNecesarias; i++ {
		encontrado := true
		for j := 0; j < cantPaginasNecesarias; j++ {
			if globals_memoria.IndicesSWAPOcupados[i+j] {
				encontrado = false
				break
			}
		}
		if encontrado {
			// Marcar como ocupado
			for j := 0; j < cantPaginasNecesarias; j++ {
				globals_memoria.IndicesSWAPOcupados[i+j] = true
			}
			return i
		}
	}

	// No hay bloque libre → agregamos al final
	indiceInicio := len(globals_memoria.IndicesSWAPOcupados)

	for i := 0; i < cantPaginasNecesarias; i++ {
		globals_memoria.IndicesSWAPOcupados = append(globals_memoria.IndicesSWAPOcupados, true)
	}

	return indiceInicio

}

/*
func escribirPaginaSWAP(dato string, pagina globals_memoria.Pagina, archivo *os.File) int { // dato de tamanio 64

	if moverseAPaginaSWAP(pagina, archivo) == 1 {
		return 1
	}

	// Escribir el dato como bytes
	_, err := archivo.Write([]byte(dato))
	if err != nil {
		log.Printf("error al escribir en el archivo: %v", err)
		return 1
	}

	return 0
}*/

/*
func escribirEnSWAP(pid int, paginasDTO []globals_memoria.PaginaDTO) int {
	var archivo *os.File = abrirArchivoBinario()

	for i := 0; i < len(paginasDTO); i++ {
		var paginaDTO globals_memoria.PaginaDTO
		var paginaSwapDisponible globals_memoria.Pagina

		paginaDTO = paginasDTO[i]
		slog.Debug(fmt.Sprintf("Pagina DTO que llega a escribirEnSwap: %+v", paginaDTO))

		// Transferimos los campos
		paginaSwapDisponible.IndiceSwapAsignado = obtenerIndiceSwapDisponible()
		paginaSwapDisponible.EntradaAsignada = paginaDTO.Entrada
		paginaSwapDisponible.IndiceAsignado = -1

		// Escribir contenido en la página asignada
		err := escribirPaginaSWAP(paginaDTO.Contenido, paginaSwapDisponible, archivo)
		if err != 0 {
			log.Printf("Error al escribir en página SWAP %d: %v", paginaSwapDisponible.IndiceSwapAsignado, err)
			return -1
		}

		slog.Debug(fmt.Sprintf("Pagina swap que se guarda en slice del proceso %d: %+v", pid, paginaDTO))

		globals_memoria.Procesos[pid].PaginasSWAP = append(globals_memoria.Procesos[pid].PaginasSWAP, paginaSwapDisponible)
	}

	// Leer todo el archivo
	data, err := io.ReadAll(archivo)
	if err != nil {
		log.Fatal(err)
	}

	slog.Debug(fmt.Sprintf("Leí %d bytes\n", len(data)))

	// Mostrar los primeros 10 bytes en hexadecimal
	//for i := 0; i < 10 && i < len(data); i++ {
	//	fmt.Printf("%02X ", data[i])
	//}
	slog.Debug(fmt.Sprint("Data leida de SWAP: ", data))

	return 0
}*/

func escribirEnSWAP(pid int, paginas []globals_memoria.Pagina) int {
	archivo := abrirArchivoBinario()
	if archivo == nil {
		slog.Debug(fmt.Sprintf("No se pudo abrir el archivo de SWAP"))
		return -1
	}
	defer archivo.Close()

	pageSize := int(globals_memoria.MemoriaConfig.Page_size)

	// Obtener una posición libre en el archivo SWAP
	inicio := obtenerIndiceSwapDisponible(len(paginas))

	for _, pagina := range paginas {

		puntero := (inicio + pagina.NumeroDePagina) * pageSize
		_, err := archivo.Seek(int64(puntero), io.SeekStart)
		if err != nil {
			slog.Debug(fmt.Sprintf("Error al posicionarse en SWAP en %d: %v", puntero, err))
			return -1
		}

		// Escribir contenido
		n, err := archivo.Write(pagina.Contenido)
		if err != nil || n != pageSize {
			slog.Debug(fmt.Sprintf("Error al escribir en SWAP en %d: %v", puntero, err))
			return -1
		}

		// Cuando guardo la primer pagina, guardo su posicion como inicio del swap
		if pagina.NumeroDePagina == 0 {
			globals_memoria.ProcesosMutex[pid].Lock()
			proceso := globals_memoria.Procesos[pid]
			proceso.InicioSWAP = inicio
			globals_memoria.Procesos[pid] = proceso
			globals_memoria.ProcesosMutex[pid].Unlock()
		}

		slog.Debug(fmt.Sprintf("SWAP: PID %d, Página %d guardada en índice %d ", pid, pagina.NumeroDePagina, inicio+pagina.NumeroDePagina))
	}

	return 0
}
