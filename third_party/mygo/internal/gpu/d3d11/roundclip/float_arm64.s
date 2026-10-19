//go:build windows

#include "textflag.h"

// Native ABI: X0=this, X1=float bits, X2=target. ARM64 gives the first float
// parameter S0 independently of the integer arguments. Only volatile registers
// are touched; the COM method returns directly to SyscallN's native gateway.
TEXT ·floatThunk(SB), NOSPLIT|NOFRAME, $0-0
	FMOVS R1, F0
	B (R2)

TEXT ·floatThunkAddress(SB), NOSPLIT, $0-8
	MOVD $·floatThunk(SB), R0
	MOVD R0, ret+0(FP)
	RET
