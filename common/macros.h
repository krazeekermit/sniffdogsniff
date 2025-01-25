#ifndef MACROS_H
#define MACROS_H

#define packed_struct struct __attribute__((__packed__))

#define STREAM_HEX(O, A, L) \
{ \
    int i; \
    for (i = 0; i < L; i++) { \
        uint8_t hi = (A[i] >> 4) & 0x0f; \
        uint8_t lo = (A[i] & 0x0f); \
        O << (char) (hi > 9 ? ('a' + (hi - 10)) : ('0' + hi)); \
        O << (char) (lo > 9 ? ('a' + (lo - 10)) : ('0' + lo)); \
    } \
}

#define GOTO_IF(COND, LABEL, RET, RETV) \
if (COND) \
{\
    RET = RETV; \
    goto LABEL; \
}

#endif // MACROS_H
