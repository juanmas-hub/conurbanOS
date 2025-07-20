package general

import (
	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
)

func ActualizarPC(pid int64, pc int64) {

	proceso := globals.MapaProcesos[pid]
	proceso.Pcb.PC = pc

}

func NotificarReplanifSRT() {
	select {
	case globals.SrtReplanificarChan <- struct{}{}:
	default:
	}
}
