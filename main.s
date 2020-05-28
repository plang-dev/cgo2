#include "textflag.h"

TEXT ·asmgetg(SB), NOSPLIT, $8
    MOVQ 0(TLS), AX
    MOVQ AX, ret+0(FP)
    RET

TEXT ·asmgetcgob(SB), NOSPLIT, $8
    MOVQ 8(TLS), AX
    MOVQ AX, ret+0(FP)
    RET

TEXT ·asmsetcgob(SB), NOSPLIT, $8
    MOVQ p+0(FP), AX
    MOVQ AX, 8(TLS)
    RET

TEXT ·asmcallc(SB), NOSPLIT, $0-16
    MOVQ 8(TLS), CX

    // *gb.pgosp = SP
    MOVQ 0(CX), AX
    MOVQ SP, 0(AX) 

    MOVQ fn+0(FP), AX

    // SP = csp
    MOVQ csp+8(FP), SP 

    // fn()
    CALL AX 

    // SP = *gb.pgosp
    MOVQ 8(TLS), CX
    MOVQ 0(CX), AX
    MOVQ 0(AX), SP

    RET

// System V AMD64 ABI calling conventions register order:
// RDI, RSI, RDX, RCX, R8, R9, [XYZ]MM0–7

TEXT ·asmjmpgowrite(SB), NOSPLIT, $0
    MOVQ 8(TLS), CX

    // *gp.pcsp = SP
    MOVQ 0(CX), AX
    MOVQ SP, 8(AX) 

    // SP = *gb.pgosp
    MOVQ 0(CX), AX
    MOVQ 0(AX), SP

    SUBQ $24, SP
    MOVQ DI, 0(SP)
    MOVQ SI, 8(SP)
    MOVQ DX, 16(SP)
    
    CALL ·gowrite(SB)

    ADDQ $24, SP

    // SP = *gp.pcsp
    MOVQ 8(TLS), CX
    MOVQ 0(CX), AX
    MOVQ 8(AX), SP

    RET

TEXT ·dummy(SB), $0
    RET
    MOVQ 0(TLS), CX
    MOVQ CX, 0(TLS)
    MOVQ $0x1122334455667788, AX
    JMP AX
